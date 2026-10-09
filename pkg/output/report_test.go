package output

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
	"gopkg.in/yaml.v3"
)

func TestFormatAndReport_FailsWhenUnderThreshold(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.StatementThreshold = 80.0
	cfg.BlockThreshold = 70.0
	cfg.SortBy = "file"
	cfg.SortOrder = "asc"

	cfg.NoColor = true
	color.NoColor = cfg.NoColor
	text.DisableColors()

	profiles := []*cover.Profile{
		{
			FileName: "example/foo.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 10, Count: 1},
				{NumStmt: 10, Count: 0},
			},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.True(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "TOTAL")
	require.Contains(t, stdout, "[S] total [+20.0% required for 70.0% threshold]")
}

func TestFormatAndReport_JSONOutput_NoColor(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatJSON
	cfg.StatementThreshold = 0
	cfg.BlockThreshold = 0
	current := color.NoColor
	cfg.NoColor = true
	defer func() {
		color.NoColor = current
	}()

	profiles := []*cover.Profile{
		{
			FileName: "main.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 5, Count: 1},
			},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, `"file":`)
	require.Contains(t, stdout, `"main.go"`)
}

func TestFormatAndReport_JSONOutput(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatJSON
	cfg.StatementThreshold = 0
	cfg.BlockThreshold = 0
	current := color.NoColor
	color.NoColor = false
	defer func() {
		color.NoColor = current
	}()

	profiles := []*cover.Profile{
		{
			FileName: "main.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 5, Count: 1},
			},
		},
	}

	jsonString := make([]byte, 0)
	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)

		var err error
		jsonString, err = json.MarshalIndent(results, "", "  ")
		require.NoError(t, err)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, highlightJSONSyntax(string(jsonString), cfg)+"\n")
}

func TestFormatAndReport_YAMLOutput(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatYAML
	cfg.StatementThreshold = 0
	cfg.BlockThreshold = 0
	current := color.NoColor
	cfg.NoColor = false
	color.NoColor = false
	defer func() {
		color.NoColor = current
	}()
	profiles := []*cover.Profile{
		{
			FileName: "main.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 5, Count: 1},
			},
		},
	}

	yamlData := make([]byte, 0)
	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)

		var err error
		yamlData, err = yaml.Marshal(results)
		require.NoError(t, err)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, highlightYAMLSyntax(string(yamlData), cfg)+"\n")
}

func TestFormatAndReport_YAMLOutput_NoColor(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatYAML
	cfg.StatementThreshold = 0
	cfg.BlockThreshold = 0
	current := color.NoColor
	cfg.NoColor = true
	color.NoColor = true
	defer func() {
		color.NoColor = current
	}()

	profiles := []*cover.Profile{
		{
			FileName: "main.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 5, Count: 1},
			},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "file:")
	require.Contains(t, stdout, "main.go")
}

func TestFormatAndReport_EmptyResults_Table(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatTable
	cfg.NoColor = true

	// Empty profiles should result in empty results
	var profiles []*cover.Profile

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "⚠ No coverage results to display")
	require.NotContains(t, stdout, "✔ All good")
	require.NotContains(t, stdout, "STATEMENTS")
	require.NotContains(t, stdout, "BLOCKS")
}

func TestFormatAndReport_EmptyResults_Markdown(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatMD
	cfg.NoColor = true

	var profiles []*cover.Profile

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "No coverage results to display")
}

func TestFormatAndReport_EmptyResults_CSV(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatCSV
	cfg.NoColor = true

	var profiles []*cover.Profile

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "No coverage results to display")
}

func TestFormatAndReport_EmptyResults_JSON(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatJSON
	cfg.NoColor = true

	var profiles []*cover.Profile

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, `"byFile": []`)
	require.Contains(t, stdout, `"byPackage": []`)
	require.Contains(t, stdout, `"coverage": "0/0"`)
	require.NotContains(t, stdout, "No coverage results to display")
}

func TestFormatAndReport_EmptyResults_YAML(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatYAML
	cfg.NoColor = true

	var profiles []*cover.Profile

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "byFile: []")
	require.Contains(t, stdout, "byPackage: []")
	require.Contains(t, stdout, "coverage: 0/0")
	require.NotContains(t, stdout, "No coverage results to display")
}

func TestFormatAndReport_EmptyResults_NoTable(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatTable
	cfg.NoTable = true
	cfg.NoColor = true

	var profiles []*cover.Profile

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "No coverage results to display")
	require.NotContains(t, stdout, "✔ All good")
	require.NotContains(t, stdout, "STATEMENTS")
	require.NotContains(t, stdout, "BLOCKS")
}

func TestFormatAndReport_EmptyResults_NoSummary(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatTable
	cfg.NoSummary = true
	cfg.NoColor = true

	var profiles []*cover.Profile

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "No coverage results to display")
	require.NotContains(t, stdout, "✔ All good")
	require.NotContains(t, stdout, "STATEMENTS")
	require.NotContains(t, stdout, "BLOCKS")
}

func TestPrintDiffWarning(t *testing.T) {
	cfg := &config.Config{}
	err := errors.New("test error")

	t.Run("table format", func(t *testing.T) {
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			PrintDiffWarning(err, cfg)
		})
		require.Empty(t, stderr)
		require.Contains(t, stdout, "Warning: Failed to get changed files for diff mode: test error")
		require.Contains(t, stdout, "Falling back to checking all files.")
	})

	t.Run("JSON format", func(t *testing.T) {
		cfg.Format = config.FormatJSON
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			PrintDiffWarning(err, cfg)
		})
		require.Empty(t, stdout)
		require.Empty(t, stderr)
	})
}

func TestPrintNoDiffChanges(t *testing.T) {
	cfg := &config.Config{}

	t.Run("table format", func(t *testing.T) {
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			PrintNoDiffChanges(cfg)
		})
		require.Empty(t, stderr)
		require.Contains(t, stdout, "No files changed in diff. No coverage to check.")
	})

	t.Run("JSON format", func(t *testing.T) {
		cfg.Format = config.FormatJSON
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			PrintNoDiffChanges(cfg)
		})
		require.Empty(t, stdout)
		require.Empty(t, stderr)
	})
}

func TestPrintDiffModeInfo(t *testing.T) {
	cfg := &config.Config{}
	changedCount := 5
	totalCount := 10

	t.Run("table format", func(t *testing.T) {
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			PrintDiffModeInfo(changedCount, totalCount, cfg)
		})
		require.Empty(t, stderr)
		require.Contains(t, stdout, "Diff mode: Checking coverage for 5 changed files (out of 10 total files)")
	})

	t.Run("JSON format", func(t *testing.T) {
		cfg.Format = config.FormatJSON
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			PrintDiffModeInfo(changedCount, totalCount, cfg)
		})
		require.Empty(t, stdout)
		require.Empty(t, stderr)
	})
}

func TestFormatAndReport_CSVIncludesFunctionColumns(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatCSV
	cfg.NoColor = true
	cfg.NoSummary = true
	cfg.NoUncoveredLines = true
	color.NoColor = cfg.NoColor
	text.DisableColors()

	by := compute.By{
		Statements: "1/2", Blocks: "1/2", Lines: "1/2", Functions: "2/3",
		StatementPercentage: 50, BlockPercentage: 50, LinePercentage: 50, FunctionPercentage: 66.666,
	}
	results := compute.Results{
		ByFile:    []compute.ByFile{{File: "pkg/a.go", By: by}},
		ByPackage: []compute.ByPackage{{Package: "pkg", By: by}},
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "1/2", Percentage: 50},
			Blocks:     compute.TotalBlocks{Coverage: "1/2", Percentage: 50},
			Lines:      compute.TotalLines{Coverage: "1/2", Percentage: 50},
			Functions:  compute.TotalFunctions{Coverage: "2/3", Percentage: 66.666},
		},
	}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		FormatAndReport(results, cfg, false)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, ",Statements,Blocks,Lines,Functions,Statement %,Block %,Line %,Function %\n")
	require.Contains(t, stdout, "pkg/a.go,1/2,1/2,1/2,2/3,50.0,50.0,50.0,66.7\n")
	require.Contains(t, stdout, "\n,1/2,1/2,1/2,2/3,50.0,50.0,50.0,66.7")
}

func renderForFormat(t *testing.T, format string, configure func(cfg *config.Config)) string {
	t.Helper()
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = format
	cfg.NoColor = true
	cfg.NoSummary = true
	color.NoColor = true
	text.DisableColors()
	configure(cfg)

	profiles := []*cover.Profile{{
		FileName: "example/foo.go",
		Blocks:   []cover.ProfileBlock{{StartLine: 1, EndLine: 1, NumStmt: 1, Count: 1}},
	}}
	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		FormatAndReport(results, cfg, failed)
	})
	return stdout
}

// headerRow returns the lowercased header of a rendered table: the thead of
// html output, otherwise the first line naming the columns.
func headerRow(t *testing.T, out string) string {
	t.Helper()
	out = strings.ToLower(out)
	if start := strings.Index(out, "<thead>"); start >= 0 {
		return out[start : strings.Index(out, "</thead>")+len("</thead>")]
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "statements") {
			return line
		}
	}
	require.Fail(t, "no header row", out)
	return ""
}

func TestRenderTable_FunctionColumns(t *testing.T) {
	none := func(*config.Config) {}
	tests := []struct {
		name      string
		format    string
		configure func(cfg *config.Config)
		want      bool
	}{
		{"table without threshold", config.FormatTable, none, false},
		{"md without threshold", config.FormatMD, none, false},
		{"html without threshold", config.FormatHTML, none, false},
		{"csv without threshold", config.FormatCSV, none, true},
		{"tsv without threshold", config.FormatTSV, none, true},
		{"table with global threshold", config.FormatTable, func(c *config.Config) { c.FunctionThreshold = 50 }, true},
		{"md with total threshold", config.FormatMD, func(c *config.Config) {
			c.Total[config.FunctionsSection] = 50
		}, true},
		{"html with per-file override", config.FormatHTML, func(c *config.Config) {
			c.PerFile.Functions = config.PerOverride{"foo.go": 50}
		}, true},
		{"table with per-package override", config.FormatTable, func(c *config.Config) {
			c.PerPackage.Functions = config.PerOverride{"example": 50}
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := headerRow(t, renderForFormat(t, tt.format, tt.configure))
			require.Equal(t, tt.want, strings.Contains(header, "functions"), header)
			require.Equal(t, tt.want, strings.Contains(header, "function %"), header)
			require.Contains(t, header, "line %")
		})
	}
}

func TestRenderTable_RowsMatchHeaderWidth(t *testing.T) {
	for _, withThreshold := range []bool{false, true} {
		for _, format := range []string{config.FormatTable, config.FormatCSV} {
			out := renderForFormat(t, format, func(c *config.Config) {
				if withThreshold {
					c.FunctionThreshold = 50
				}
			})
			lines := strings.Split(strings.TrimSpace(out), "\n")
			require.Greater(t, len(lines), 2)
			sep := "│"
			if format == config.FormatCSV {
				sep = ","
			}
			var rows []string
			for _, l := range lines {
				if format == config.FormatCSV || strings.HasPrefix(l, sep) { // skip box-drawing rules
					rows = append(rows, l)
				}
			}
			want := strings.Count(rows[0], sep)
			for _, l := range rows {
				require.Equal(t, want, strings.Count(l, sep), "%s: %s", format, l)
			}
		}
	}
}
