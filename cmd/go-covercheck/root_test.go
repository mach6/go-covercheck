package main

import (
	"os"
	"testing"

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
	_ = os.WriteFile(path, []byte(testCoverageOut), 0644)

	t.Cleanup(func() {
		_ = os.Remove(path)
	})

	return path
}

func createTempHistoryFile(t *testing.T) string {
	t.Helper()
	path := t.TempDir() + "/.go-covercheck.history.json"
	_ = os.WriteFile(path, []byte(testCoverageHistory), 0644)

	t.Cleanup(func() {
		_ = os.Remove(path)
	})

	return path
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
	_ = rootCmd.Flags().Set(HistoryFileFlag, path)
	_ = rootCmd.Flags().Set(ShowHistoryFlag, "true")
	err := run(rootCmd, []string{createTempCoverageFile(t)})
	require.NoError(t, err)
}

func Test_run_SaveHistory(t *testing.T) {
	path := createTempHistoryFile(t)
	_ = rootCmd.Flags().Set(HistoryFileFlag, path)
	_ = rootCmd.Flags().Set(SaveHistoryFlag, "true")
	_ = rootCmd.Flags().Set(StatementThresholdFlag, "0")
	_ = rootCmd.Flags().Set(BlockThresholdFlag, "0")
	err := run(rootCmd, []string{createTempCoverageFile(t)})
	require.NoError(t, err)
}

func Test_run_CompareHistory(t *testing.T) {
	path := createTempHistoryFile(t)
	_ = rootCmd.Flags().Set(TerminalWidthFlag, "180")
	_ = rootCmd.Flags().Set(CompareHistoryFlag, "main")
	_ = rootCmd.Flags().Set(HistoryFileFlag, path)
	_ = rootCmd.Flags().Set(SaveHistoryFlag, "true")
	_ = rootCmd.Flags().Set(StatementThresholdFlag, "0")
	_ = rootCmd.Flags().Set(BlockThresholdFlag, "0")
	err := run(rootCmd, []string{createTempCoverageFile(t)})
	require.NoError(t, err)
}
