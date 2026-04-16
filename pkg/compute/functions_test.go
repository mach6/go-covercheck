package compute_test

import (
	"os"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func writeTempGoFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.go")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestGetFunctionCoverageForFile(t *testing.T) {
	// Test with a Go file that doesn't exist - error should be propagated
	total, covered, err := compute.GetFunctionCoverageForFile("nonexistent.go", nil)
	require.Error(t, err)
	require.Equal(t, 0, total)
	require.Equal(t, 0, covered)

	// Test with a non-Go file
	total, covered, err = compute.GetFunctionCoverageForFile("readme.txt", nil)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Equal(t, 0, covered)
}

func TestGetFunctionCoverageForFile_Success(t *testing.T) {
	src := "package p\nfunc A() {}\nfunc B() {}\n"
	path := writeTempGoFile(t, src)

	// Parse first to get actual line numbers.
	fns, err := compute.CountFunctionsInFile(path)
	require.NoError(t, err)
	require.Len(t, fns, 2)

	// Provide a block that covers only the first function.
	blocks := []cover.ProfileBlock{
		{StartLine: fns[0].StartLine, Count: 1},
	}

	total, covered, err := compute.GetFunctionCoverageForFile(path, blocks)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Equal(t, 1, covered)
}

func TestGetFunctionCoverageForFile_NoCoverage(t *testing.T) {
	src := "package p\nfunc A() {}\nfunc B() {}\n"
	path := writeTempGoFile(t, src)

	total, covered, err := compute.GetFunctionCoverageForFile(path, nil)
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Equal(t, 0, covered)
}

func TestCountFunctionsInFile(t *testing.T) {
	// Test with a non-existent file
	functions, err := compute.CountFunctionsInFile("nonexistent.go")
	require.Error(t, err)
	require.Nil(t, functions)
}

func TestCountFunctionsInFile_WithFunctions(t *testing.T) {
	src := "package p\nfunc A() {}\nfunc B(x int) int { return x }\n"
	path := writeTempGoFile(t, src)

	fns, err := compute.CountFunctionsInFile(path)
	require.NoError(t, err)
	require.Len(t, fns, 2)
	require.Equal(t, "A", fns[0].Name)
	require.Equal(t, "B", fns[1].Name)
	require.False(t, fns[0].Covered)
	require.GreaterOrEqual(t, fns[0].EndLine, fns[0].StartLine)
}

func TestCountFunctionsInFile_WithMethods(t *testing.T) {
	src := "package p\ntype T struct{}\ntype P struct{}\nfunc (v T) Value() {}\nfunc (p *P) Ptr() {}\n"
	path := writeTempGoFile(t, src)

	fns, err := compute.CountFunctionsInFile(path)
	require.NoError(t, err)
	require.Len(t, fns, 2)

	names := make(map[string]bool)
	for _, fn := range fns {
		names[fn.Name] = true
	}
	// Value receiver: *ast.Ident → "T.Value"
	require.True(t, names["T.Value"])
	// Pointer receiver: *ast.StarExpr → "*P.Ptr"
	require.True(t, names["*P.Ptr"])
}

func TestMatchFunctionsWithCoverage(t *testing.T) {
	functions := []compute.FunctionInfo{
		{Name: "func1", StartLine: 1, EndLine: 5, Covered: false},
		{Name: "func2", StartLine: 10, EndLine: 15, Covered: false},
	}

	blocks := []cover.ProfileBlock{
		{StartLine: 2, StartCol: 1, EndLine: 3, EndCol: 10, Count: 1},
		{StartLine: 12, StartCol: 1, EndLine: 13, EndCol: 10, Count: 0},
	}

	result := compute.MatchFunctionsWithCoverage(functions, blocks)
	require.Len(t, result, 2)
	require.True(t, result[0].Covered)  // func1 should be covered (block with Count: 1)
	require.False(t, result[1].Covered) // func2 should not be covered (block with Count: 0)
}

func TestMatchFunctionsWithCoverage_Empty(t *testing.T) {
	// Empty functions list
	result := compute.MatchFunctionsWithCoverage(nil, nil)
	require.Empty(t, result)

	// Block outside function range should not mark anything covered
	fns := []compute.FunctionInfo{
		{Name: "f", StartLine: 10, EndLine: 20, Covered: false},
	}
	blocks := []cover.ProfileBlock{
		{StartLine: 5, Count: 1}, // before the function
	}
	result = compute.MatchFunctionsWithCoverage(fns, blocks)
	require.Len(t, result, 1)
	require.False(t, result[0].Covered)
}