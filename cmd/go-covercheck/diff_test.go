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
	require.Contains(t, stdOut, "--diff-from")
}

func TestExecute_DiffMode_InvalidRepo(t *testing.T) {
	// Create a fake coverage file
	coverageFile := test.CreateTempCoverageFile(t, "mode: set")

	// Change to the temp directory
	t.Chdir(filepath.Dir(coverageFile))

	// Run diff mode in non-git directory
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--diff-from=HEAD~1", "--no-color", coverageFile})
	stdOut, _, err := runCmdForTest(t, cmd)
	require.NoError(t, err) // Should not fail, just fall back
	require.Contains(t, stdOut, "Warning: Failed to get changed files")
	require.Contains(t, stdOut, "Falling back to checking all files")
	require.Contains(t, stdOut, "No coverage results to display")
}

func TestExecute_DiffMode_WithJSON(t *testing.T) {
	// Create a fake coverage file
	coverageFile := test.CreateTempCoverageFile(t, "mode: set")

	// Change to the temp directory
	t.Chdir(filepath.Dir(coverageFile))

	// Run diff mode with JSON output
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--diff-from=HEAD~1", "--format", "json", "--no-color", coverageFile})
	stdOut, _, err := runCmdForTest(t, cmd)
	require.NoError(t, err)

	// Extract JSON part from the output (after any warning messages)
	jsonOutput := extractJSONFromOutput(stdOut)

	// Should output valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(jsonOutput), &result)
	require.NoError(t, err)

	// Verify structure
	require.Contains(t, result, "byFile")
	require.Contains(t, result, "byPackage")
	require.Contains(t, result, "byTotal")
}
