package main

import (
	"os"
	"testing"

	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func setupTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		RunE:         rootCmd.RunE,
		SilenceUsage: rootCmd.SilenceUsage,
	}
	initFlags(cmd)
	return cmd
}

func TestExecute_Help(t *testing.T) {
	rootCmd.SetArgs([]string{
		"-h",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)
	stdOut, stdErr := test.RepipeStdOutAndErrForTest(func() {
		main()
	})
	require.Empty(t, stdErr)
	require.Contains(t, stdOut, rootCmd.Short)
}

func TestExecute_Init(t *testing.T) {
	// Use a temporary directory for this test
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--init"})
	
	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)
	require.Contains(t, stdOut, "Created sample config file: .go-covercheck.yml")
	
	// Verify the file was created
	configPath := ".go-covercheck.yml"
	_, err = os.Stat(configPath)
	require.NoError(t, err)
	
	// Verify the file contains expected content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "statementThreshold:")
	require.Contains(t, string(content), "blockThreshold:")
}

func TestExecute_Init_FileExists(t *testing.T) {
	// Use a temporary directory for this test
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	
	// Create an existing config file
	configPath := ".go-covercheck.yml"
	err := os.WriteFile(configPath, []byte("existing content"), ConfigFilePermissions)
	require.NoError(t, err)
	
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"--init"})
	
	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.Error(t, err)
	require.Empty(t, stdOut)
	require.Contains(t, stdErr, "config file .go-covercheck.yml already exists")
}
