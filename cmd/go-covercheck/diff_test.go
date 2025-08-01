package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestExecute_DiffMode(t *testing.T) {
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--diff-only", "--help"})
	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)
	require.Contains(t, stdOut, "--diff-only")
	require.Contains(t, stdOut, "--against")
}

func TestExecute_DiffMode_InvalidRepo(t *testing.T) {
	// Create a temporary directory that's not a git repo
	tempDir := t.TempDir()
	
	// Create a fake coverage file
	coverageFile := test.CreateTempFile(t, "coverage.out", "mode: set")
	
	// Change to the temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	
	// Run diff mode in non-git directory
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--diff-only", "--no-color", coverageFile})
	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err) // Should not fail, just fall back
	require.Contains(t, stdErr, "Warning: Failed to get changed files")
	require.Contains(t, stdErr, "Falling back to checking all files")
	require.Contains(t, stdOut, "All good")
}

func TestExecute_DiffMode_WithJSON(t *testing.T) {
	// Create a fake coverage file
	tempDir := t.TempDir()
	coverageFile := test.CreateTempFile(t, "coverage.out", "mode: set")
	
	// Change to the temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(oldWd)
	}()
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	
	// Run diff mode with JSON output
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--diff-only", "--format", "json", "--no-color", coverageFile})
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