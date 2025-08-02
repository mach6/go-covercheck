package compute_test

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func TestGetFunctionCoverageForFile(t *testing.T) {
	// Test with a Go file that doesn't exist
	total, covered, err := compute.GetFunctionCoverageForFile("nonexistent.go", nil)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Equal(t, 0, covered)
	
	// Test with a non-Go file
	total, covered, err = compute.GetFunctionCoverageForFile("readme.txt", nil)
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Equal(t, 0, covered)
}

func TestCountFunctionsInFile(t *testing.T) {
	// Test with a non-existent file
	functions, err := compute.CountFunctionsInFile("nonexistent.go")
	require.Error(t, err)
	require.Nil(t, functions)
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