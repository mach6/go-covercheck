package output_test

import (
	"testing"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
)

//nolint:dupl // parallel subtests exercise distinct failure configurations
func TestRenderSummary(t *testing.T) {
	prevNoColor := color.NoColor
	t.Cleanup(func() { color.NoColor = prevNoColor })
	color.NoColor = true

	t.Run("NoSummary flag enabled", func(t *testing.T) {
		cfg := &config.Config{NoSummary: true, Format: config.FormatTable}
		results := compute.Results{}

		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.FormatAndReport(results, cfg, true)
		})

		t.Logf("--------[stdout]--------\n%s\n", stdout)
		t.Logf("--------[stderr]--------\n%s\n", stderr)

		require.NotEmpty(t, stdout)
		require.NotContains(t, stdout, "All good")
		require.NotContains(t, stdout, "Coverage check failed")
		require.Empty(t, stderr)
	})

	t.Run("No failure", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		results := compute.Results{}
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.FormatAndReport(results, cfg, false)
		})
		require.Contains(t, stdout, "All good")
		require.Empty(t, stderr)
	})

	t.Run("With all failures", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		cfg.StatementThreshold = 80
		cfg.BlockThreshold = 80
		cfg.LineThreshold = 80

		results := compute.Results{
			ByFile: []compute.ByFile{
				{
					By: compute.By{
						StatementPercentage: 70, BlockPercentage: 70, LinePercentage: 70,
						StatementThreshold: 80, BlockThreshold: 80, LineThreshold: 80,
						Failed: true,
					},
					File: "file1.go",
				},
			},
			ByPackage: []compute.ByPackage{
				{
					By: compute.By{
						StatementPercentage: 60, BlockPercentage: 60, LinePercentage: 60,
						StatementThreshold: 80, BlockThreshold: 80, LineThreshold: 80,
						Failed: true,
					},
					Package: "pkg1",
				},
			},
			ByTotal: compute.Totals{
				Statements: compute.TotalStatements{Percentage: 50, Threshold: 80, Failed: true},
				Blocks:     compute.TotalBlocks{Percentage: 50, Threshold: 80, Failed: true},
				Lines:      compute.TotalLines{Percentage: 50, Threshold: 80, Failed: true},
			},
		}

		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.FormatAndReport(results, cfg, true)
		})

		require.Empty(t, stderr)
		require.Contains(t, stdout, "Coverage check failed")

		// Check By File output
		require.Contains(t, stdout, "→ By File")
		require.Contains(t, stdout, "[S] file1.go [+10.0% required for 80.0% threshold]")
		require.Contains(t, stdout, "[B] file1.go [+10.0% required for 80.0% threshold]")
		require.Contains(t, stdout, "[L] file1.go [+10.0% required for 80.0% threshold]")

		// Check By Package output
		require.Contains(t, stdout, "→ By Package")
		require.Contains(t, stdout, "[S] pkg1 [+20.0% required for 80.0% threshold]")
		require.Contains(t, stdout, "[B] pkg1 [+20.0% required for 80.0% threshold]")
		require.Contains(t, stdout, "[L] pkg1 [+20.0% required for 80.0% threshold]")

		// Check By Total output
		require.Contains(t, stdout, "→ By Total")
		require.Contains(t, stdout, "[S] total [+30.0% required for 80.0% threshold]")
		require.Contains(t, stdout, "[B] total [+30.0% required for 80.0% threshold]")
		require.Contains(t, stdout, "[L] total [+30.0% required for 80.0% threshold]")
	})

	t.Run("Partial failures within categories", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		cfg.StatementThreshold = 80
		cfg.BlockThreshold = 80
		cfg.LineThreshold = 80

		results := compute.Results{
			ByFile: []compute.ByFile{
				{
					By: compute.By{
						StatementPercentage: 90, BlockPercentage: 70, LinePercentage: 90, // pass, fail, pass
						StatementThreshold: 80, BlockThreshold: 80, LineThreshold: 80,
						Failed: true,
					},
					File: "file1.go",
				},
			},
			ByPackage: []compute.ByPackage{
				{
					By: compute.By{
						StatementPercentage: 70, BlockPercentage: 90, LinePercentage: 90, // fail, pass, pass
						StatementThreshold: 80, BlockThreshold: 80, LineThreshold: 80,
						Failed: true,
					},
					Package: "pkg1",
				},
			},
			ByTotal: compute.Totals{
				Statements: compute.TotalStatements{Percentage: 90, Threshold: 80, Failed: false}, // pass
				Blocks:     compute.TotalBlocks{Percentage: 90, Threshold: 80, Failed: false},     // pass
				Lines:      compute.TotalLines{Percentage: 70, Threshold: 80, Failed: true},       // fail
			},
		}

		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.FormatAndReport(results, cfg, true)
		})

		require.Empty(t, stderr)
		require.Contains(t, stdout, "Coverage check failed")

		// Check By File output
		require.Contains(t, stdout, "→ By File")
		require.NotContains(t, stdout, "[S] file1.go")
		require.Contains(t, stdout, "[B] file1.go [+10.0% required for 80.0% threshold]")
		require.NotContains(t, stdout, "[L] file1.go")

		// Check By Package output
		require.Contains(t, stdout, "→ By Package")
		require.Contains(t, stdout, "[S] pkg1 [+10.0% required for 80.0% threshold]")
		require.NotContains(t, stdout, "[B] pkg1")
		require.NotContains(t, stdout, "[L] pkg1")

		// Check By Total output
		require.Contains(t, stdout, "→ By Total")
		require.NotContains(t, stdout, "[S] total")
		require.NotContains(t, stdout, "[B] total")
		require.Contains(t, stdout, "[L] total [+10.0% required for 80.0% threshold]")
	})

	t.Run("No failures in one category", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		cfg.StatementThreshold = 80

		results := compute.Results{
			ByFile: []compute.ByFile{
				{
					By: compute.By{
						StatementPercentage: 90, StatementThreshold: 80, Failed: false, // No failure for this file
					},
					File: "file1.go",
				},
			},
			ByPackage: []compute.ByPackage{
				{
					By: compute.By{
						StatementPercentage: 60, StatementThreshold: 80, Failed: true,
					},
					Package: "pkg1",
				},
			},
		}

		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.FormatAndReport(results, cfg, true)
		})

		require.Empty(t, stderr)
		require.Contains(t, stdout, "Coverage check failed")
		require.NotContains(t, stdout, "→ By File")
		require.Contains(t, stdout, "→ By Package")
	})

	t.Run("No total failures", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		cfg.StatementThreshold = 80

		results := compute.Results{
			ByFile: []compute.ByFile{
				{
					By: compute.By{
						StatementPercentage: 70, StatementThreshold: 80, Failed: true,
					},
					File: "file1.go",
				},
			},
			ByTotal: compute.Totals{
				Statements: compute.TotalStatements{Percentage: 90, Threshold: 80, Failed: false},
			},
		}

		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.FormatAndReport(results, cfg, true)
		})

		require.Empty(t, stderr)
		require.Contains(t, stdout, "Coverage check failed")
		require.Contains(t, stdout, "→ By File")
		require.NotContains(t, stdout, "→ By Total")
	})
}
