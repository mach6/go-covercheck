package output_test

import (
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
