package functions_test

import (
	"os"
	"strings"
	"testing"

	"github.com/mach6/go-covercheck/pkg/functions"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func TestCollect_Names(t *testing.T) {
	src := []string{
		"package p",                       // 1
		"func Plain() {}",                 // 2
		"func (T) Value() {}",             // 3
		"func (t T) NamedValue() {}",      // 4
		"func (*T) Pointer() {}",          // 5
		"func (t *T) NamedPointer() {}",   // 6
		"func (s *S[K]) Generic() {}",     // 7
		"func (s S[K, V]) TwoParams() {}", // 8
		"func (s *S[_]) Blank() {}",       // 9
		"func (s (*T)) Paren() {}",        // 10
		"func Gen[T any](v T) {}",         // 11
	}
	fns, err := functions.Collect(&cover.Profile{FileName: "p.go"}, src)
	require.NoError(t, err)

	names := make([]string, 0, len(fns))
	for _, fn := range fns {
		names = append(names, fn.Name)
	}
	require.Equal(t, []string{
		"Plain",
		"T.Value",
		"T.NamedValue",
		"(*T).Pointer",
		"(*T).NamedPointer",
		"(*S[K]).Generic",
		"S[K,V].TwoParams",
		"(*S[_]).Blank",
		"(*T).Paren",
		"Gen",
	}, names)
}

func TestFormatUncovered_RealProfile(t *testing.T) {
	profiles, err := cover.ParseProfiles("testdata/fixture.coverage.out")
	require.NoError(t, err)
	src, err := os.ReadFile("testdata/fixture.go")
	require.NoError(t, err)
	fns, err := functions.Collect(profiles[0], strings.Split(string(src), "\n"))
	require.NoError(t, err)

	// Outer is covered even though its closure never ran, and Empty is
	// covered by its zero-statement block, so neither is listed.
	require.Equal(t, "(*Stack[T]).Pop:7, EmptyUncalled:16, a:37", functions.FormatUncovered(fns))
	require.Len(t, functions.Uncovered(fns), 3)
}

func TestFormatUncovered_SameLineAndOrder(t *testing.T) {
	src := []string{
		"package p",
		"func a() int { return 1 }; func b() int { return 2 }",
		"func c() {}",
	}
	fns, err := functions.Collect(&cover.Profile{FileName: "p.go"}, src)
	require.NoError(t, err)
	require.Equal(t, "a:2, b:2, c:3", functions.FormatUncovered(fns))

	// Executing b leaves a and c.
	p := &cover.Profile{FileName: "p.go", Blocks: []cover.ProfileBlock{
		{StartLine: 2, StartCol: 41, EndLine: 2, EndCol: 53, NumStmt: 1, Count: 1},
	}}
	fns, err = functions.Collect(p, src)
	require.NoError(t, err)
	require.Equal(t, "a:2, c:3", functions.FormatUncovered(fns))
}

func TestFormatUncovered_ClosureCountsTowardEnclosing(t *testing.T) {
	src := []string{
		"package p",
		"func Outer() {",
		"\tgo func() {",
		"\t\tprintln()",
		"\t}()",
		"}",
		"func Other() {}",
	}
	p := &cover.Profile{FileName: "p.go", Blocks: []cover.ProfileBlock{
		{StartLine: 3, StartCol: 12, EndLine: 5, EndCol: 3, NumStmt: 1, Count: 1},
	}}
	fns, err := functions.Collect(p, src)
	require.NoError(t, err)
	require.Equal(t, "Other:7", functions.FormatUncovered(fns))
}

func TestFormatUncovered_NoneUncovered(t *testing.T) {
	require.Empty(t, functions.FormatUncovered(nil))
	require.Empty(t, functions.Uncovered(nil))
	require.Empty(t, functions.FormatUncovered([]functions.Function{{Name: "A", StartLine: 1, Covered: true}}))
}
