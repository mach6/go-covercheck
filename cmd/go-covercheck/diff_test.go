package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestExecute_DiffMode(t *testing.T) {
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--help"})
	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)
	require.Contains(t, stdOut, "--diff-against")
}

func TestExecute_DiffMode_InvalidRepo(t *testing.T) {
	// Create a fake coverage file
	coverageFile := test.CreateTempCoverageFile(t, "mode: set")

	// Change to the temp directory
	t.Chdir(filepath.Dir(coverageFile))

	// Run diff mode in non-git directory
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--diff-against=HEAD~1", "--no-color", coverageFile})
	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err) // Should not fail, just fall back
	require.Contains(t, stdErr, "Warning: Failed to get changed files")
	require.Contains(t, stdErr, "Falling back to checking all files")
	require.Equal(t, "⚠ No coverage results to display\n", stdOut)
}

func TestExecute_DiffMode_WithJSON(t *testing.T) {
	// Create a fake coverage file
	coverageFile := test.CreateTempCoverageFile(t, "mode: set")

	// Change to the temp directory
	t.Chdir(filepath.Dir(coverageFile))

	// Run diff mode with JSON output
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--diff-against=HEAD~1", "--format", "json", "--no-color", coverageFile})
	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Contains(t, stdErr, "Warning: Failed to get changed files")

	// Should output valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(stdOut), &result)
	require.NoError(t, err)

	// Verify structure
	require.Contains(t, result, "byFile")
	require.Contains(t, result, "byPackage")
	require.Contains(t, result, "byTotal")
}

func TestExecute_DiffMode_DefaultValue(t *testing.T) {
	// Create a fake coverage file
	coverageFile := test.CreateTempCoverageFile(t, "mode: set")

	// Change to the temp directory
	t.Chdir(filepath.Dir(coverageFile))

	// Run diff mode without specifying value (should default to HEAD~1)
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--diff-against", "--no-color", coverageFile})
	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err) // Should not fail, just fall back
	require.Contains(t, stdErr, "Warning: Failed to get changed files")
	require.Contains(t, stdErr, "Falling back to checking all files")
	// Use Contains instead of Equal to avoid color code issues
	require.Contains(t, stdOut, "No coverage results to display")
}
