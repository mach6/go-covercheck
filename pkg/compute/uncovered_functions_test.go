package compute //nolint:testpackage

import (
	"encoding/json"
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
	"gopkg.in/yaml.v3"
)

func TestCollectResults_UncoveredFunctions(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()

	r, _ := CollectResults(functionProfiles(""), cfg)

	// alpha.go: One ran, Two did not.
	require.Equal(t, "Two:7", fileByName(t, r, "testdata/alpha/alpha.go").UncoveredFunctions)
	require.Equal(t, "Three:3", fileByName(t, r, "testdata/alpha/three.go").UncoveredFunctions)
	// beta.go: the zero-statement method ran, so nothing is listed.
	require.Empty(t, fileByName(t, r, "testdata/beta/beta.go").UncoveredFunctions)
	// Packages carry no per-function list, like uncovered lines.
	for _, p := range r.ByPackage {
		require.Empty(t, p.UncoveredFunctions)
	}
}

func TestCollectResults_UncoveredFunctionsSuppressed(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()
	cfg.NoUncoveredFunctions = true

	r, _ := CollectResults(functionProfiles(""), cfg)
	for _, f := range r.ByFile {
		require.Empty(t, f.UncoveredFunctions)
	}
	require.Equal(t, "1/2", fileByName(t, r, "testdata/alpha/alpha.go").Functions, "counts are unaffected")
}

func TestCollectResults_UncoveredFunctionsAreIndependentOfUncoveredLines(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()

	cfg.NoUncoveredLines = true
	r, _ := CollectResults(functionProfiles(""), cfg)
	alpha := fileByName(t, r, "testdata/alpha/alpha.go")
	require.Empty(t, alpha.UncoveredLines)
	require.Equal(t, "Two:7", alpha.UncoveredFunctions)

	cfg.NoUncoveredLines = false
	cfg.NoUncoveredFunctions = true
	r, _ = CollectResults(functionProfiles(""), cfg)
	alpha = fileByName(t, r, "testdata/alpha/alpha.go")
	require.NotEmpty(t, alpha.UncoveredLines)
	require.Empty(t, alpha.UncoveredFunctions)
}

func TestCollectResults_UncoveredFunctionsMissingOrBrokenSource(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()

	block := []cover.ProfileBlock{{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 1, Count: 0}}
	profiles := []*cover.Profile{
		{FileName: "does/not/exist.go", Mode: "set", Blocks: block},
		{FileName: "testdata/broken/broken.go", Mode: "set", Blocks: block},
	}
	r, _ := CollectResults(profiles, cfg)
	require.Len(t, r.ByFile, 2)
	for _, f := range r.ByFile {
		require.Empty(t, f.UncoveredFunctions, f.File)
	}
}

func TestResults_UncoveredFunctionsStructuredFields(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()
	r, _ := CollectResults(functionProfiles(""), cfg)

	j, err := json.Marshal(r)
	require.NoError(t, err)
	require.Contains(t, string(j), `"uncoveredFunctions":"Two:7"`)

	y, err := yaml.Marshal(r)
	require.NoError(t, err)
	require.Contains(t, string(y), "uncoveredFunctions: Two:7")

	// omitempty: a file with nothing to list has no key at all.
	r, _ = CollectResults(functionProfiles("")[2:], cfg)
	j, err = json.Marshal(r)
	require.NoError(t, err)
	require.NotContains(t, string(j), "uncoveredFunctions")
}
