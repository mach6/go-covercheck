package functions_test

import (
	"os"
	"strings"
	"testing"

	"github.com/mach6/go-covercheck/pkg/functions"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

// testdata/fixture.coverage.out was produced by `go test -coverprofile` against
// testdata/fixture.go, so these assertions pin behavior to real cover output.
func TestCollect_RealProfile(t *testing.T) {
	profiles, err := cover.ParseProfiles("testdata/fixture.coverage.out")
	require.NoError(t, err)
	require.Len(t, profiles, 1)

	src, err := os.ReadFile("testdata/fixture.go")
	require.NoError(t, err)

	fns, err := functions.Collect(profiles[0], strings.Split(string(src), "\n"))
	require.NoError(t, err)

	covered := map[int][]bool{}
	for _, fn := range fns {
		covered[fn.StartLine] = append(covered[fn.StartLine], fn.Covered)
	}
	require.Equal(t, map[int][]bool{
		5:  {true},        // generic method Push, called
		7:  {false},       // generic method Pop, not called
		14: {true},        // Empty: zero-statement block still records a hit
		16: {false},       // EmptyUncalled
		19: {true},        // Outer: called even though its closure is not
		28: {true},        // generic func Map
		37: {false, true}, // a and b share a line; only b is called
	}, covered, "package-level closure on line 26 must not be counted")

	total, hits := functions.Coverage(fns)
	require.Equal(t, 8, total)
	require.Equal(t, 5, hits)
}

func TestCollect_SameLineFunctionsUseColumns(t *testing.T) {
	src := []string{
		"package p",
		"func a() int { return 1 }; func b() int { return 2 }",
	}
	p := &cover.Profile{FileName: "p.go", Blocks: []cover.ProfileBlock{
		{StartLine: 2, StartCol: 14, EndLine: 2, EndCol: 26, NumStmt: 1, Count: 0},
		{StartLine: 2, StartCol: 41, EndLine: 2, EndCol: 53, NumStmt: 1, Count: 3},
	}}

	fns, err := functions.Collect(p, src)
	require.NoError(t, err)
	require.Len(t, fns, 2)
	require.False(t, fns[0].Covered)
	require.True(t, fns[1].Covered)
}

func TestCollect_SkipsBodylessDeclarations(t *testing.T) {
	src := []string{
		"package p",
		"func asm() int",
		"func goFunc() {}",
	}
	fns, err := functions.Collect(&cover.Profile{FileName: "p.go"}, src)
	require.NoError(t, err)
	require.Len(t, fns, 1)
	require.Equal(t, 3, fns[0].StartLine)
	require.False(t, fns[0].Covered)
}

func TestCollect_MethodsAndClosures(t *testing.T) {
	src := []string{
		"package p",         // 1
		"type T struct{}",   // 2
		"func (T) V() {",    // 3
		"\tgo func() {",     // 4
		"\t\tprintln()",     // 5
		"\t}()",             // 6
		"}",                 // 7
		"func (*T) P() {}",  // 8
		"func init() {}",    // 9
		"func init() {}",    // 10
		"var _ = func() {}", // 11
		"func _() {}",       // 12
	}
	// Only the closure body inside V executed; V itself is still covered
	// because the closure's block lies within V's extent.
	p := &cover.Profile{FileName: "p.go", Blocks: []cover.ProfileBlock{
		{StartLine: 3, StartCol: 15, EndLine: 4, EndCol: 12, NumStmt: 1, Count: 0},
		{StartLine: 4, StartCol: 12, EndLine: 6, EndCol: 3, NumStmt: 1, Count: 1},
		{StartLine: 11, StartCol: 17, EndLine: 11, EndCol: 18, NumStmt: 0, Count: 1},
	}}

	fns, err := functions.Collect(p, src)
	require.NoError(t, err)
	total, hits := functions.Coverage(fns)
	require.Equal(t, 5, total, "V, P, two init and _; the var closure is not counted")
	require.Equal(t, 1, hits)
	require.True(t, fns[0].Covered)
}

func TestCollect_ZeroCountBlocksDoNotCover(t *testing.T) {
	src := []string{"package p", "func f() { println() }"}
	p := &cover.Profile{FileName: "p.go", Blocks: []cover.ProfileBlock{
		{StartLine: 2, StartCol: 10, EndLine: 2, EndCol: 23, NumStmt: 1, Count: 0},
	}}

	fns, err := functions.Collect(p, src)
	require.NoError(t, err)
	total, hits := functions.Coverage(fns)
	require.Equal(t, 1, total)
	require.Equal(t, 0, hits)
}

func TestCollect_NoFunctions(t *testing.T) {
	fns, err := functions.Collect(&cover.Profile{FileName: "p.go"}, []string{"package p", "var x = 1"})
	require.NoError(t, err)
	require.Empty(t, fns)
	total, hits := functions.Coverage(fns)
	require.Zero(t, total)
	require.Zero(t, hits)
}

func TestCollect_UnparsableSource(t *testing.T) {
	_, err := functions.Collect(&cover.Profile{FileName: "p.go"}, nil)
	require.Error(t, err)

	_, err = functions.Collect(&cover.Profile{FileName: "p.go"}, []string{"package p", "func {"})
	require.Error(t, err)
}
