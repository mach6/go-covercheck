package output_test

import (
	"errors"
	"testing"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
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
		output.FormatAndReport(results, cfg, failed)
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
		output.FormatAndReport(results, cfg, failed)
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

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		output.FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, color.New(color.FgGreen, color.Bold).Sprint("\"main.go\""))
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

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.False(t, failed)
		output.FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, text.FgHiGreen.Sprintf(" main.go"))
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
		output.FormatAndReport(results, cfg, failed)
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
		output.FormatAndReport(results, cfg, failed)
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
		output.FormatAndReport(results, cfg, failed)
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
		output.FormatAndReport(results, cfg, failed)
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
		output.FormatAndReport(results, cfg, failed)
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
		output.FormatAndReport(results, cfg, failed)
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
		output.FormatAndReport(results, cfg, failed)
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
		output.FormatAndReport(results, cfg, failed)
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
			output.PrintDiffWarning(err, cfg)
		})
		require.Empty(t, stderr)
		require.Contains(t, stdout, "Warning: Failed to get changed files for diff mode: test error")
		require.Contains(t, stdout, "Falling back to checking all files.")
	})

	t.Run("JSON format", func(t *testing.T) {
		cfg.Format = config.FormatJSON
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.PrintDiffWarning(err, cfg)
		})
		require.Empty(t, stdout)
		require.Empty(t, stderr)
	})
}

func TestPrintNoDiffChanges(t *testing.T) {
	cfg := &config.Config{}

	t.Run("table format", func(t *testing.T) {
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.PrintNoDiffChanges(cfg)
		})
		require.Empty(t, stderr)
		require.Contains(t, stdout, "No files changed in diff. No coverage to check.")
	})

	t.Run("JSON format", func(t *testing.T) {
		cfg.Format = config.FormatJSON
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.PrintNoDiffChanges(cfg)
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
			output.PrintDiffModeInfo(changedCount, totalCount, cfg)
		})
		require.Empty(t, stderr)
		require.Contains(t, stdout, "Diff mode: Checking coverage for 5 changed files (out of 10 total files)")
	})

	t.Run("JSON format", func(t *testing.T) {
		cfg.Format = config.FormatJSON
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.PrintDiffModeInfo(changedCount, totalCount, cfg)
		})
		require.Empty(t, stdout)
		require.Empty(t, stderr)
	})
}
