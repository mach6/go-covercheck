package output_test

import (
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

const inspectFunctionsSource = `package p

func Hit() {
	println()
}

func (s *S[K]) Miss() {
	println("never")
}

func Empty() {}
`

func inspectFunctionsConfig(t *testing.T) *config.Config {
	t.Helper()
	restoreNoColor(t)
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Inspect = true
	cfg.NoColor = true
	color.NoColor = true
	return cfg
}

func inspectFunctionsProfile(t *testing.T, name string) *cover.Profile {
	t.Helper()
	return &cover.Profile{
		FileName: test.CreateTempFile(t, name, inspectFunctionsSource),
		Blocks: []cover.ProfileBlock{
			{StartLine: 3, StartCol: 13, EndLine: 5, EndCol: 2, NumStmt: 1, Count: 1},
			{StartLine: 7, StartCol: 21, EndLine: 9, EndCol: 2, NumStmt: 1, Count: 0},
		},
	}
}

func TestInspect_ListsUncoveredFunctions(t *testing.T) {
	cfg := inspectFunctionsConfig(t)
	p := inspectFunctionsProfile(t, "inspect_fn.go")

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		require.NoError(t, output.InspectUncoveredLines([]*cover.Profile{p}, cfg))
	})

	require.Empty(t, stderr)
	// Miss has uncovered lines and Empty has no block at all; Hit ran.
	require.Contains(t, stdout, "Uncovered functions: (*S[K]).Miss (line 7), Empty (line 11)")
	require.NotContains(t, stdout, "Hit (line")
	// The function header sits under the file header, before the code hunks.
	require.Less(t, strings.Index(stdout, "--- "+p.FileName), strings.Index(stdout, "Uncovered functions:"))
	require.Less(t, strings.Index(stdout, "Uncovered functions:"), strings.Index(stdout, "@@ Lines 7-9 @@"))
	require.Contains(t, stdout, `println("never")`)
}

func TestInspect_FileWithOnlyUncoveredFunctionsIsListed(t *testing.T) {
	cfg := inspectFunctionsConfig(t)
	p := &cover.Profile{FileName: test.CreateTempFile(t, "inspect_empty.go", "package p\n\nfunc Empty() {}\n")}

	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		require.NoError(t, output.InspectUncoveredLines([]*cover.Profile{p}, cfg))
	})

	require.Contains(t, stdout, "--- "+p.FileName)
	require.Contains(t, stdout, "Uncovered functions: Empty (line 3)")
	require.NotContains(t, stdout, "@@")
	require.NotContains(t, stdout, "No uncovered")
}

func TestInspect_FullyCoveredFileHasNoFunctionHeader(t *testing.T) {
	cfg := inspectFunctionsConfig(t)
	p := &cover.Profile{
		FileName: test.CreateTempFile(t, "inspect_hit.go", "package p\n\nfunc Hit() {\n\tprintln()\n}\n"),
		Blocks:   []cover.ProfileBlock{{StartLine: 3, StartCol: 12, EndLine: 5, EndCol: 2, NumStmt: 1, Count: 1}},
	}

	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		require.NoError(t, output.InspectUncoveredLines([]*cover.Profile{p}, cfg))
	})

	require.Contains(t, stdout, "No uncovered lines found!")
	require.NotContains(t, stdout, "Uncovered functions")
}

func TestInspect_InspectFileAndContextWithFunctions(t *testing.T) {
	cfg := inspectFunctionsConfig(t)
	first := inspectFunctionsProfile(t, "inspect_first.go")
	second := inspectFunctionsProfile(t, "inspect_second.go")
	cfg.InspectFiles = []string{"inspect_first.go"}
	cfg.InspectContext = 0

	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		require.NoError(t, output.InspectUncoveredLines([]*cover.Profile{first, second}, cfg))
	})

	require.Contains(t, stdout, "--- "+first.FileName)
	require.NotContains(t, stdout, second.FileName)
	require.Equal(t, 1, strings.Count(stdout, "Uncovered functions:"), "only the selected file's functions")
	// Context only affects the code hunks: with none, Hit's lines stay out.
	require.NotContains(t, stdout, "println()\n")
	require.Contains(t, stdout, `println("never")`)
}

func TestInspect_UnreadableSourceHasNoFunctionNames(t *testing.T) {
	cfg := inspectFunctionsConfig(t)
	p := &cover.Profile{
		FileName: "does/not/exist.go",
		Blocks:   []cover.ProfileBlock{{StartLine: 4, StartCol: 1, EndLine: 4, EndCol: 10, NumStmt: 1, Count: 0}},
	}

	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		require.NoError(t, output.InspectUncoveredLines([]*cover.Profile{p}, cfg))
	})

	require.Contains(t, stdout, "--- does/not/exist.go")
	require.Contains(t, stdout, "@@ Lines 4-4 @@")
	require.NotContains(t, stdout, "Uncovered functions")
}

func TestInspect_FunctionsListedWhenUncoveredOutputIsSuppressed(t *testing.T) {
	cfg := inspectFunctionsConfig(t)
	cfg.NoUncoveredLines = true
	cfg.NoUncoveredFunctions = true
	p := inspectFunctionsProfile(t, "inspect_suppressed.go")

	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		require.NoError(t, output.InspectUncoveredLines([]*cover.Profile{p}, cfg))
	})

	require.Contains(t, stdout, "Uncovered functions: (*S[K]).Miss (line 7), Empty (line 11)")
}
