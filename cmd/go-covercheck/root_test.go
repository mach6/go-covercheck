package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/history"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const (
	testCoverageOut = `mode: set
github.com/mach6/go-covercheck/pkg/math/math.go:5.49,6.16 1 1
github.com/mach6/go-covercheck/pkg/math/math.go:6.16,8.3 1 0`

	testCoverageHistory = `{
  "entries": [
    {
      "commit": "e40262964cc463a18753e2834c04230c2a356f20",
      "branch": "main",
      "timestamp": "2025-07-18T08:41:38.076764523Z",
      "results": {
        "byFile": [
          {
            "statementCoverage": "3/4",
            "blockCoverage": "3/4",
            "statementPercentage": 75,
            "blockPercentage": 75,
            "statementThreshold": 65,
            "blockThreshold": 50,
            "failed": false,
            "file": "github.com/mach6/go-covercheck/pkg/math/math.go"
          }
        ],
        "byPackage": [
          {
            "statementCoverage": "3/4",
            "blockCoverage": "3/4",
            "statementPercentage": 75,
            "blockPercentage": 75,
            "statementThreshold": 65,
            "blockThreshold": 50,
            "failed": false,
            "package": "github.com/mach6/go-covercheck/pkg/math"
          }
        ],
        "byTotal": {
          "statements": {
            "coverage": "180/648",
            "threshold": 70,
            "percentage": 27.77777777777778,
            "failed": true
          },
          "blocks": {
            "coverage": "95/409",
            "threshold": 50,
            "percentage": 23.227383863080682,
            "failed": true
          }
        }
      }
    }
	]
}`
)

func createTempCoverageFile(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/coverage.out"
	_ = os.WriteFile(path, []byte(testCoverageOut), 0600)

	t.Cleanup(func() {
		_ = os.Remove(path)
	})

	return path
}

func createTempHistoryFile(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/.go-covercheck.history.json"
	_ = os.WriteFile(path, []byte(testCoverageHistory), 0600)

	t.Cleanup(func() {
		_ = os.Remove(path)
	})

	return path
}

func setupTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:gocritic
			return run(cmd, args)
		},
	}
	initFlags(cmd)
	return cmd
}

func runCmdForTest(cmd *cobra.Command) (string, string, error) {
	// repipe all stdout and stderr
	oOut := os.Stdout
	rOut, wOut, _ := os.Pipe()

	oErr := os.Stderr
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	// exec the cmd, capture the error
	err := cmd.Execute()

	// restore standard stdout and stderr
	_ = wOut.Close()
	_ = wErr.Close()

	os.Stdout = oOut
	os.Stderr = oErr

	// return data from repiped stdout, stderr
	var stdOut bytes.Buffer
	_, _ = stdOut.ReadFrom(rOut)

	var stdErr bytes.Buffer
	_, _ = stdErr.ReadFrom(rErr)

	return stdOut.String(), stdErr.String(), err
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

func Test_run_ShowHistory(t *testing.T) {
	path := createTempHistoryFile(t)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--show-history", "-w",
		createTempCoverageFile(t)},
	)

	stdOut, stdErr, err := runCmdForTest(cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)

	require.Contains(t, stdOut, "e402629")
	require.Contains(t, stdOut, "main")
	require.Contains(t, stdOut, "Showing last 1 history entry")
}

func Test_run_SaveHistory(t *testing.T) {
	path := createTempHistoryFile(t)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--save-history", "-w",
		"-s", "1", "-b", "1", "-f", "json",
		createTempCoverageFile(t)},
	)

	stdOut, stdErr, err := runCmdForTest(cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)

	// unmarshal the json output and confirm it used the block and statement coverage thresholds specified in the command
	// flags.
	r := new(compute.Results)
	err = json.Unmarshal([]byte(stdOut), &r)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, r.ByTotal.Blocks.Threshold, 0)
	require.InEpsilon(t, 1.0, r.ByTotal.Statements.Threshold, 0)

	// open the history file and confirm it has new content
	h, err := history.Load(path)
	require.NoError(t, err)
	require.Len(t, h.Entries, 2)
}

func Test_run_CompareHistory(t *testing.T) {
	path := createTempHistoryFile(t)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--compare-history", "main", "-w",
		"-s", "1", "-b", "1",
		createTempCoverageFile(t)},
	)

	stdOut, stdErr, err := runCmdForTest(cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)

	require.Contains(t, stdOut, "[S] github.com/mach6/go-covercheck/pkg/math/math.go [−25.0%]")
	require.Contains(t, stdOut, "Comparing against ref: main")
}
