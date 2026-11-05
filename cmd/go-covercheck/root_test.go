package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/filters"
	"github.com/mach6/go-covercheck/pkg/history"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
	"gopkg.in/yaml.v3"
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

// requireOnlySourceWarnings fails unless every line of stderr is a
// missing-source warning (stderr may be empty).
func requireOnlySourceWarnings(t *testing.T, stdErr string) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(stdErr), "\n") {
		if line != "" {
			require.True(t, strings.HasPrefix(line, "warning: "), "unexpected stderr line: %s", line)
		}
	}
}

func coverageFileFor(t *testing.T, fileName string) string {
	t.Helper()
	return test.CreateTempCoverageFile(t, "mode: set\n"+fileName+":3.16,5.2 1 1\n")
}

func Test_run_WarnsOnMissingSource(t *testing.T) {
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"-w", "-f", "json", coverageFileFor(t, "example.com/mod/nope/missing.go")})

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)

	require.Equal(t, 1, strings.Count(stdErr, "warning: "))
	require.Contains(t, stdErr, "warning: example.com/mod/nope/missing.go: source file not found")
	require.Contains(t, stdErr, "function coverage unavailable")

	// stdout is still one valid JSON document, free of the warning.
	require.NotContains(t, stdOut, "warning")
	r := new(compute.Results)
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdOut)), r))
	require.Equal(t, "0/0", r.ByFile[0].Functions)
}

func Test_run_NoWarningWhenSourceFound(t *testing.T) {
	cmd := setupTestCmd()
	cmd.SetArgs([]string{"-w", "-f", "json", coverageFileFor(t, "../../pkg/compute/testdata/alpha/alpha.go")})

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)
	r := new(compute.Results)
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdOut)), r))
	require.Equal(t, "1/2", r.ByFile[0].Functions)
}

func TestFilterBySkipped_MatchesPrefix(t *testing.T) {
	profiles := []*cover.Profile{
		{FileName: "vendor/foo.go"},
		{FileName: "src/main.go"},
	}
	cfg := &config.Config{}
	cfg.ApplyDefaults()            // Apply defaults first
	cfg.Skip = []string{"vendor/"} // Then set skip patterns
	filtered := filters.FilterProfiles(profiles, cfg)

	require.Len(t, filtered, 1)
	require.Equal(t, "src/main.go", filtered[0].FileName)
}

func TestFilterBySkipped_MatchesExact(t *testing.T) {
	profiles := []*cover.Profile{
		{FileName: "internal/tmp_test.go"},
		{FileName: "src/main.go"},
	}
	cfg := &config.Config{}
	cfg.ApplyDefaults()             // Apply defaults first
	cfg.Skip = []string{"_test.go"} // Then set skip patterns
	filtered := filters.FilterProfiles(profiles, cfg)
	require.Len(t, filtered, 1)
	require.Equal(t, "src/main.go", filtered[0].FileName)
}

func TestFilterBySkipped_NoMatches(t *testing.T) {
	profiles := []*cover.Profile{
		{FileName: "main.go"},
		{FileName: "src/foo/bar.go"},
	}
	cfg := &config.Config{}
	cfg.ApplyDefaults()                 // Apply defaults first
	cfg.Skip = []string{"generated.go"} // Then set skip patterns
	filtered := filters.FilterProfiles(profiles, cfg)
	require.Len(t, filtered, 2)
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

func extractJSONFromOutput(output string) string {
	// The output format might have warning messages before JSON
	// Find the first line that starts with { (JSON start)
	lines := strings.Split(output, "\n")
	jsonLines := make([]string, 0)
	startFound := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !startFound && strings.HasPrefix(trimmed, "{") {
			startFound = true
		}
		if startFound {
			// Stop if we hit a success message marker
			if strings.HasPrefix(trimmed, "✓") {
				break
			}
			jsonLines = append(jsonLines, line)
		}
	}

	return strings.Join(jsonLines, "\n")
}

func Test_run_SaveHistory(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--save-history", "-w",
		"-s", "50", "-b", "50", "-B", "2", "-S", "1", "-f", "json",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	requireOnlySourceWarnings(t, stdErr)

	// For JSON format, a success message should NOT be present
	require.NotContains(t, stdOut, "≡ Saved history entry")

	// Extract JSON part from the output for parsing
	jsonOutput := extractJSONFromOutput(stdOut)

	// unmarshal the json output and confirm it used the block and statement coverage thresholds specified in the command
	// flags.
	r := new(compute.Results)
	err = json.Unmarshal([]byte(jsonOutput), &r)
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
		"-s", "50", "-b", "50", "-B", "2", "-S", "1", "-f", "json",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	requireOnlySourceWarnings(t, stdErr)

	// For JSON format, a success message should NOT be present
	require.NotContains(t, stdOut, "≡ Saved history entry")

	// Extract JSON part from the output for parsing
	jsonOutput := extractJSONFromOutput(stdOut)

	// unmarshal the json output and confirm it used the block and statement coverage thresholds specified in the command
	// flags.
	r := new(compute.Results)
	err = json.Unmarshal([]byte(jsonOutput), &r)
	require.NoError(t, err)
	require.InEpsilon(t, 2.0, r.ByTotal.Blocks.Threshold, 0)
	require.InEpsilon(t, 1.0, r.ByTotal.Statements.Threshold, 0)

	// open the history file and confirm it has new content
	h, err := history.Load(path)
	require.NoError(t, err)
	require.Len(t, h.Entries, 1)
}

func Test_run_SaveHistory_TableFormat(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--save-history", "-w",
		"-s", "1", "-b", "1", "-B", "2", "-S", "2", "-f", "table",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	requireOnlySourceWarnings(t, stdErr)

	// For table format, success message SHOULD be present
	require.Contains(t, stdOut, "≡ Saved history entry")

	// open the history file and confirm it has new content
	h, err := history.Load(path)
	require.NoError(t, err)
	require.Len(t, h.Entries, 2)
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
	requireOnlySourceWarnings(t, stdErr)

	require.Contains(t, stdOut, "[S] github.com/mach6/go-covercheck/pkg/math/math.go [−25.0%]")
	require.Contains(t, stdOut, "Comparing against ref: main")
}

func Test_run_CompareHistory_JSON(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path, "-f", "json", "--no-color",
		"--compare-history", "main",
		"-s", "1", "-b", "1", "-S", "2", "-B", "2",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	requireOnlyWarnings(t, stdErr)

	// stdout must be a single JSON document, with no trailing comparison text.
	var doc struct {
		Comparison struct {
			Ref       string `json:"ref"`
			Commit    string `json:"commit"`
			ByPackage []struct {
				Package    string  `json:"package"`
				Statements float64 `json:"statements"`
			} `json:"byPackage"`
			ByTotal struct {
				Statements float64 `json:"statements"`
			} `json:"byTotal"`
		} `json:"comparison"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdOut), &doc))
	require.Equal(t, "main", doc.Comparison.Ref)
	require.Equal(t, "e402629", doc.Comparison.Commit)
	require.Len(t, doc.Comparison.ByPackage, 1)
	require.Equal(t, "github.com/mach6/go-covercheck/pkg/math", doc.Comparison.ByPackage[0].Package)
	require.InDelta(t, -25.0, doc.Comparison.ByPackage[0].Statements, 0.001)
	require.InDelta(t, 22.2, doc.Comparison.ByTotal.Statements, 0.1)
}

func Test_run_CompareHistory_JSON_AddedAndRemoved(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
	// history has pkg/math/math.go only, so this file is added and math.go is removed.
	coverage := "mode: set\ngithub.com/mach6/go-covercheck/pkg/other/other.go:5.49,6.16 1 1\n"

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path, "-f", "json", "--no-color",
		"--compare-history", "main",
		test.CreateTempCoverageFile(t, coverage)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	requireOnlyWarnings(t, stdErr)

	var doc struct {
		Comparison struct {
			Added struct {
				Files []struct {
					File       string  `json:"file"`
					Statements float64 `json:"statements"`
				} `json:"files"`
				Packages []struct {
					Package string `json:"package"`
				} `json:"packages"`
			} `json:"added"`
			Removed struct {
				Files []struct {
					File string `json:"file"`
				} `json:"files"`
				Packages []struct {
					Package string `json:"package"`
				} `json:"packages"`
			} `json:"removed"`
		} `json:"comparison"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdOut), &doc))
	require.Len(t, doc.Comparison.Added.Files, 1)
	require.Equal(t, "github.com/mach6/go-covercheck/pkg/other/other.go", doc.Comparison.Added.Files[0].File)
	require.InDelta(t, 100.0, doc.Comparison.Added.Files[0].Statements, 0.001)
	require.Len(t, doc.Comparison.Added.Packages, 1)
	require.Equal(t, "github.com/mach6/go-covercheck/pkg/other", doc.Comparison.Added.Packages[0].Package)
	require.Len(t, doc.Comparison.Removed.Files, 1)
	require.Equal(t, "github.com/mach6/go-covercheck/pkg/math/math.go", doc.Comparison.Removed.Files[0].File)
	require.Len(t, doc.Comparison.Removed.Packages, 1)
}

func Test_run_CompareHistory_YAML(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path, "-f", "yaml", "--no-color",
		"--compare-history", "main",
		"-s", "1", "-b", "1", "-S", "2", "-B", "2",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	requireOnlyWarnings(t, stdErr)

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(stdOut), &doc))
	comparison, ok := doc["comparison"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "main", comparison["ref"])
	require.Equal(t, "e402629", comparison["commit"])
}

func Test_run_CompareHistoryBadRef_JSONStillReportsResults(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path, "-f", "json", "--no-color",
		"--compare-history", "unknown",
		"-s", "1", "-b", "1", "-S", "2", "-B", "2",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, _, err := runCmdForTest(t, cmd)
	require.ErrorContains(t, err, "no history entry found for ref: unknown")

	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdOut), &doc))
	require.Contains(t, doc, "byTotal")
	require.NotContains(t, doc, "comparison")
}

func Test_run_CompareAndSaveHistory_DoesNotStoreComparison(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path, "-f", "json", "--no-color",
		"--compare-history", "main", "--save-history",
		"-s", "1", "-b", "1", "-S", "2", "-B", "2",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	_, _, err := runCmdForTest(t, cmd)
	require.NoError(t, err)

	saved, err := os.ReadFile(path) //nolint:gosec // test temp file
	require.NoError(t, err)
	require.NotContains(t, string(saved), "comparison")
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
	requireOnlySourceWarnings(t, stdErr)

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

func Test_run_TotalThresholdsFollowGlobalsInConfigFile(t *testing.T) {
	tests := []struct {
		name                       string
		config                     string
		wantStatements, wantBlocks float64
		wantLines                  float64
	}{
		{
			name:           "totals not set follow the global thresholds",
			config:         "statementThreshold: 1\nblockThreshold: 2\nlineThreshold: 3\n",
			wantStatements: 1, wantBlocks: 2, wantLines: 3,
		},
		{
			name:           "a global of 0 turns the total off too",
			config:         "statementThreshold: 1\nblockThreshold: 2\nlineThreshold: 0\n",
			wantStatements: 1, wantBlocks: 2, wantLines: 0,
		},
		{
			name:           "an explicit total is used as is",
			config:         "statementThreshold: 1\nblockThreshold: 2\nlineThreshold: 0\ntotal:\n  lines: 4\n",
			wantStatements: 1, wantBlocks: 2, wantLines: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := setupTestCmd()
			cmd.SetArgs([]string{
				"--config", test.CreateTempConfigFile(t, tt.config), "-f", "yaml", "--no-color",
				test.CreateTempCoverageFile(t, test.TestCoverageOut)},
			)
			stdOut, stdErr, err := runCmdForTest(t, cmd)
			require.NoError(t, err)
			requireOnlySourceWarnings(t, stdErr)

			r := new(compute.Results)
			require.NoError(t, yaml.Unmarshal([]byte(stdOut), &r))
			require.InDelta(t, tt.wantStatements, r.ByTotal.Statements.Threshold, 0.001)
			require.InDelta(t, tt.wantBlocks, r.ByTotal.Blocks.Threshold, 0.001)
			require.InDelta(t, tt.wantLines, r.ByTotal.Lines.Threshold, 0.001)
		})
	}
}

func Test_run_TotalThresholdsFollowGlobalFlags(t *testing.T) {
	tests := []struct {
		name           string
		config         string // empty means no config file
		args           []string
		wantStatements float64
	}{
		{
			name:           "a global flag of 0 turns off a total that followed the config",
			config:         "statementThreshold: 2\n",
			args:           []string{"--statement-threshold", "0"},
			wantStatements: 0,
		},
		{
			name:           "a global flag raises a total that followed a config global of 0",
			config:         "statementThreshold: 0\n",
			args:           []string{"--statement-threshold", "2"},
			wantStatements: 2,
		},
		{
			name:           "a total set in the config file is kept",
			config:         "statementThreshold: 2\ntotal:\n  statements: 1\n",
			args:           []string{"--statement-threshold", "0"},
			wantStatements: 1,
		},
		{
			name:           "a total flag wins over the global flag",
			config:         "statementThreshold: 2\n",
			args:           []string{"--statement-threshold", "0", "--total-statement-threshold", "1"},
			wantStatements: 1,
		},
		{
			name:           "the config global is used when no flag is given",
			config:         "statementThreshold: 2\n",
			wantStatements: 2,
		},
		{
			name:           "a global flag sets the total when there is no config file",
			args:           []string{"--statement-threshold", "2"},
			wantStatements: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgPath := path.Join(t.TempDir(), "missing.yml") // not created: no config file
			if tt.config != "" {
				cfgPath = test.CreateTempConfigFile(t, tt.config)
			}
			cmd := setupTestCmd()
			cmd.SetArgs(append([]string{"--config", cfgPath, "-f", "yaml", "--no-color"}, append(tt.args,
				test.CreateTempCoverageFile(t, test.TestCoverageOut))...))
			stdOut, stdErr, err := runCmdForTest(t, cmd)
			require.NoError(t, err)
			requireOnlySourceWarnings(t, stdErr)

			r := new(compute.Results)
			require.NoError(t, yaml.Unmarshal([]byte(stdOut), &r))
			require.InDelta(t, tt.wantStatements, r.ByTotal.Statements.Threshold, 0.001)
		})
	}
}

func Test_run_CompareHistory_FollowsSortFlags(t *testing.T) {
	// 10 single-statement blocks per file. z.go has 1 covered (10%), a.go 5 (50%), m.go 9 (90%),
	// so by statement percentage the order is z, a, m, which is not the name order a, m, z.
	var cov strings.Builder
	cov.WriteString("mode: set\n")
	for file, covered := range map[string]int{"z.go": 1, "a.go": 5, "m.go": 9} {
		for i := range 10 {
			count := 0
			if i < covered {
				count = 1
			}
			fmt.Fprintf(&cov, "example.com/m/%s:%d.1,%d.10 1 %d\n", file, i+1, i+1, count)
		}
	}
	// the saved run had every file at 20%, so every file changed.
	entry := func(file string) string {
		return `{"statementCoverage":"2/10","blockCoverage":"2/10","statementPercentage":20,"blockPercentage":20,` +
			`"statementThreshold":0,"blockThreshold":0,"failed":false,"file":"` + file + `"}`
	}
	historyJSON := `{"entries":[{"commit":"e40262964cc463a18753e2834c04230c2a356f20","branch":"main",` +
		`"timestamp":"2025-07-18T08:41:38Z","results":{"byFile":[` + entry("m.go") + `,` + entry("a.go") + `,` +
		entry("z.go") + `],"byPackage":[],"byTotal":{"statements":{"coverage":"6/30","threshold":0,` +
		`"percentage":20,"failed":false},"blocks":{"coverage":"6/30","threshold":0,"percentage":20,"failed":false}}}}]}`
	historyPath := test.CreateTempHistoryFile(t, historyJSON)

	tests := map[string][]string{
		"asc":  {"z.go", "a.go", "m.go"},
		"desc": {"m.go", "a.go", "z.go"},
	}
	for order, want := range tests {
		t.Run(order, func(t *testing.T) {
			cmd := setupTestCmd()
			cmd.SetArgs([]string{
				"--history-file", historyPath, "-f", "yaml", "--no-color",
				"--compare-history", "main", "--sort-by", "statement-percent", "--sort-order", order,
				"-s", "0", "-b", "0", "-n", "0", "-S", "0", "-B", "0", "-N", "0",
				test.CreateTempCoverageFile(t, cov.String())},
			)
			stdOut, _, err := runCmdForTest(t, cmd)
			require.NoError(t, err)

			var doc struct {
				ByFile []struct {
					File string `yaml:"file"`
				} `yaml:"byFile"`
				Comparison struct {
					ByFile []struct {
						File string `yaml:"file"`
					} `yaml:"byFile"`
				} `yaml:"comparison"`
			}
			require.NoError(t, yaml.Unmarshal([]byte(stdOut), &doc))

			report := make([]string, 0, len(doc.ByFile))
			comparison := make([]string, 0, len(doc.Comparison.ByFile))
			for _, f := range doc.ByFile {
				report = append(report, path.Base(f.File))
			}
			for _, f := range doc.Comparison.ByFile {
				comparison = append(comparison, path.Base(f.File))
			}
			// the report follows --sort-by and --sort-order, and so does the comparison.
			require.Equal(t, want, report)
			require.Equal(t, want, comparison)
		})
	}
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

func Test_run_DeleteHistory(t *testing.T) {
	// Create a history file with test data
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--delete-history", "main", "-w",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)
	require.Contains(t, stdOut, "≡ Deleted history entry for ref: main")

	// Verify the entry was actually deleted
	h, err := history.Load(path)
	require.NoError(t, err)
	require.Empty(t, h.Entries)
}

func Test_run_DeleteHistoryFails_BadRef(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--delete-history", "nonexistent", "-w",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.Error(t, err)
	require.Empty(t, stdOut)
	require.NotEmpty(t, stdErr)
	require.ErrorContains(t, err, "no history entry found for ref: nonexistent")
	require.Contains(t, stdErr, "no history entry found for ref: nonexistent")
}

func Test_run_DeleteHistoryFails_BadFile(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.InvalidTestCoverageHistory)

	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--history-file", path,
		"--delete-history", "main", "-w",
		test.CreateTempCoverageFile(t, test.TestCoverageOut)},
	)

	stdOut, stdErr, err := runCmdForTest(t, cmd)
	require.Error(t, err)
	require.Empty(t, stdOut)
	require.NotEmpty(t, stdErr)
	require.ErrorContains(t, err, "failed to load history")
	require.Contains(t, stdErr, "failed to load history")
}

func TestApplyConfigOverrides_TableStyleFlag(t *testing.T) {
	cmd := &cobra.Command{}
	initFlags(cmd)

	cfg := &config.Config{}
	cfg.ApplyDefaults()
	require.Equal(t, config.TableStyleLight, cfg.TableStyle)

	require.NoError(t, cmd.Flags().Set(TableStyleFlag, config.TableStyleRounded))
	applyConfigOverrides(cfg, cmd, false)
	require.Equal(t, config.TableStyleRounded, cfg.TableStyle)
}

func TestApplyConfigOverrides_TableStyleNoConfigFile(t *testing.T) {
	cmd := &cobra.Command{}
	initFlags(cmd)

	cfg := &config.Config{TableStyle: config.TableStyleBold}
	// Flag not explicitly set; default ("light") should take over when there is no config file.
	applyConfigOverrides(cfg, cmd, true)
	require.Equal(t, config.TableStyleLight, cfg.TableStyle)
}

func TestApplyConfigOverrides_FunctionThresholdFlags(t *testing.T) {
	tests := []struct {
		name       string
		flags      map[string]string
		configured config.PerOverride
		wantGlobal float64
		wantTotal  float64
	}{
		{
			name: "disabled by default",
		},
		{
			name:       "global flag also sets total",
			flags:      map[string]string{FunctionThresholdFlag: "80"},
			wantGlobal: 80,
			wantTotal:  80,
		},
		{
			name:       "total flag overrides global",
			flags:      map[string]string{FunctionThresholdFlag: "80", TotalFunctionThresholdFlag: "90"},
			wantGlobal: 80,
			wantTotal:  90,
		},
		{
			name:       "configured total is kept",
			flags:      map[string]string{FunctionThresholdFlag: "80"},
			configured: config.PerOverride{config.FunctionsSection: 65},
			wantGlobal: 80,
			wantTotal:  65,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			initFlags(cmd)
			for k, v := range tt.flags {
				require.NoError(t, cmd.Flags().Set(k, v))
			}

			cfg := &config.Config{}
			cfg.ApplyDefaults()
			for k, v := range tt.configured {
				cfg.Total[k] = v
			}

			applyConfigOverrides(cfg, cmd, tt.configured == nil)
			require.NoError(t, cfg.Validate())
			require.InDelta(t, tt.wantGlobal, cfg.FunctionThreshold, 0.001)
			require.InDelta(t, tt.wantTotal, cfg.Total[config.FunctionsSection], 0.001)
		})
	}
}

// requireOnlyWarnings checks that stderr holds nothing but "warning:" lines, such as the ones
// reported for source files that cannot be found.
func requireOnlyWarnings(t *testing.T, stdErr string) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimSpace(stdErr), "\n") {
		if line != "" {
			require.True(t, strings.HasPrefix(line, "warning: "), "unexpected stderr line: %q", line)
		}
	}
}
