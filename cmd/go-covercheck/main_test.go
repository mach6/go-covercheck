package main

import (
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
