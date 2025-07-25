package main

import (
	"encoding/json"
	"gopkg.in/yaml.v3"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/history"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func runCmdForTest(t *testing.T, cmd *cobra.Command) (string, string, error) {
	t.Helper()

	var err error
	stdOut, stdErr := test.RepipeStdOutAndErrForTest(func() {
		// exec the cmd, capture the error
		err = cmd.Execute()
	})

	t.Logf("--------[stdout]--------\n%s\n", stdOut)
	t.Logf("--------[stderr]--------\n%s\n", stdErr)
	t.Logf("--------[err]--------\n%v\n", err)
	return stdOut, stdErr, err
}

func TestShouldSkip_MatchesPrefix(t *testing.T) {
	require.True(t, shouldSkip("vendor/foo.go", []string{"vendor/"}))
	require.True(t, shouldSkip("gen/code.go", []string{"gen/"}))
}

func TestShouldSkip_MatchesExact(t *testing.T) {
	require.True(t, shouldSkip("internal/tmp_test.go", []string{"internal/tmp_test.go"}))
}

func TestShouldSkip_NoMatches(t *testing.T) {
	require.False(t, shouldSkip("main.go", []string{"generated.go"}))
	require.False(t, shouldSkip("src/foo/bar.go", nil))
}

func Test_run_ShowCoverageFails(t *testing.T) {
	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"-w",
		test.CreateTempCoverageFile(t, test.InvalidTestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.Empty(t, stdOut)
	require.NotEmpty(t, stdErr)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to parse coverage file")
	require.Contains(t, stdErr, " failed to parse coverage file")
}

func Test_run_ShowHistory(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--show-history", "-w",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)

	require.Contains(t, stdOut, "e402629")
	require.Contains(t, stdOut, "main")
	require.Contains(t, stdOut, "Showing last 1 history entry")
}

func Test_run_ShowHistoryFails(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.InvalidTestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--show-history", "-w",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.Empty(t, stdOut)
	require.NotEmpty(t, stdErr)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to load history")
	require.Contains(t, stdErr, "failed to load history")
}

func Test_run_SaveHistory(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--save-history", "-w",
		"-s", "1", "-b", "1", "-B", "2", "-f", "json",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)

	// unmarshal the json output and confirm it used the block and statement coverage thresholds specified in the command
	// flags.
	r := new(compute.Results)
	err = json.Unmarshal([]byte(stdOut), &r)
	require.NoError(t, err)
	require.InEpsilon(t, 2.0, r.ByTotal.Blocks.Threshold, 0)
	require.InEpsilon(t, 1.0, r.ByTotal.Statements.Threshold, 0)

	// open the history file and confirm it has new content
	h, err := history.Load(path)
	require.NoError(t, err)
	require.Len(t, h.Entries, 2)
}

func Test_run_SaveHistory_NoPreviousFile(t *testing.T) {
	path := t.TempDir() + ".go-covercheck.history.json"

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--save-history", "-w",
		"-s", "1", "-b", "1", "-B", "2", "-f", "json",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)

	// unmarshal the json output and confirm it used the block and statement coverage thresholds specified in the command
	// flags.
	r := new(compute.Results)
	err = json.Unmarshal([]byte(stdOut), &r)
	require.NoError(t, err)
	require.InEpsilon(t, 2.0, r.ByTotal.Blocks.Threshold, 0)
	require.InEpsilon(t, 1.0, r.ByTotal.Statements.Threshold, 0)

	// open the history file and confirm it has new content
	h, err := history.Load(path)
	require.NoError(t, err)
	require.Len(t, h.Entries, 1)
}

func Test_run_SaveHistoryFails(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.InvalidTestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--save-history", "-w",
		"-s", "1", "-b", "1",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.Error(t, err)
	require.NotEmpty(t, stdOut)
	require.NotEmpty(t, stdErr)
	require.ErrorContains(t, err, "failed to load history")
	require.Contains(t, stdErr, "failed to load history")
}

func Test_run_CompareHistory(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--compare-history", "main", "-w",
		"-s", "1", "-b", "1", "-S", "2", "-B", "2",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)

	require.Contains(t, stdOut, "[S] github.com/mach6/go-covercheck/pkg/math/math.go [−25.0%]")
	require.Contains(t, stdOut, "Comparing against ref: main")
}

func Test_run_CompareHistoryFails(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.InvalidTestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--compare-history", "main", "-w",
		"-s", "1", "-b", "1", "-S", "2", "-B", "2",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NotEmpty(t, stdOut)
	require.NotEmpty(t, stdErr)
	require.Error(t, err)
	require.Contains(t, stdOut, "✔ All good")
	require.Contains(t, stdErr, "failed to load history")
	require.ErrorContains(t, err, "failed to load history")
}

func Test_run_CompareHistoryFailsBadRef(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--compare-history", "unknown", "-w",
		"-s", "1", "-b", "1", "-S", "2", "-B", "2",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NotEmpty(t, stdOut)
	require.NotEmpty(t, stdErr)
	require.Error(t, err)
	require.Contains(t, stdOut, "✔ All good")
	require.Contains(t, stdErr, "no history entry found for ref: unknown")
	require.ErrorContains(t, err, "no history entry found for ref: unknown")
}

func Test_run_WithConfigFile(t *testing.T) {
	path := test.CreateTempConfigFile(t, test.TestConfig)
	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--config", path,
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)
	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.NotEmpty(t, stdOut)
	require.Empty(t, stdErr)

	// unmarshal the yaml output and confirm it used the block and statement coverage thresholds specified in the config
	// flags.
	r := new(compute.Results)
	err = yaml.Unmarshal([]byte(stdOut), &r)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, r.ByFile[0].StatementThreshold, 0)
	require.InEpsilon(t, 40.0, r.ByFile[0].BlockThreshold, 0)
	require.InEpsilon(t, 50.0, r.ByPackage[0].StatementThreshold, 0)
	require.InEpsilon(t, 1.0, r.ByPackage[0].BlockThreshold, 0)
	require.InEpsilon(t, 3.0, r.ByTotal.Blocks.Threshold, 0)
	require.InEpsilon(t, 2.0, r.ByTotal.Statements.Threshold, 0)
}

func Test_run_WithConfigFileFailsToUnmarshal(t *testing.T) {
	path := test.CreateTempConfigFile(t, test.ErrorTestConfig)
	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--config", path,
		test.CreateTempCoverageFile(t, test.TestCoverageOut),
	})

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.Error(t, err)
	require.Empty(t, stdOut)
	require.NotEmpty(t, stdErr)
	require.ErrorContains(t, err, "cannot unmarshal")
	require.Contains(t, stdErr, "cannot unmarshal")
}

func Test_run_WithConfigFileFailsToValidate(t *testing.T) {
	path := test.CreateTempConfigFile(t, test.InvalidTestConfig)
	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--config", path,
		test.CreateTempCoverageFile(t, test.TestCoverageOut),
	})

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.Error(t, err)
	require.Empty(t, stdOut)
	require.NotEmpty(t, stdErr)
	require.ErrorContains(t, err, "must be between 0 and 100")
	require.Contains(t, stdErr, "must be between 0 and 100")
}

func Test_getVersion(t *testing.T) {
	require.Contains(t, getVersion(), config.AppVersion)
	require.Contains(t, getVersion(), config.AppRevision)

	config.BuiltBy = "test"
	config.BuildTimeStamp = "0400"
	require.Contains(t, getVersion(), config.AppVersion)
	require.Contains(t, getVersion(), config.AppRevision)
	require.Contains(t, getVersion(), "built by test on 0400")
}
