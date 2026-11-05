package output

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
)

// heatmapReport renders the profile from uncoveredFunctionsProfile as a heat map that fails its statement threshold.
func heatmapReport(t *testing.T, configure func(*config.Config)) string {
	t.Helper()
	return renderUncoveredFunctions(t, config.FormatHeatmap, func(cfg *config.Config) {
		cfg.StatementThreshold = 100 // the profile covers half the statements
		cfg.NoSummary = false
		configure(cfg)
	})
}

func TestHeatmap_NoTableHidesTheGridAndKeepsTheSummary(t *testing.T) {
	out := heatmapReport(t, func(cfg *config.Config) { cfg.NoTable = true })

	require.NotContains(t, out, "threshold met", "the legend is part of the grid")
	require.NotContains(t, out, "\nBy File\n", "the grid has its own heading; the summary's is ' → By File'")
	require.Contains(t, out, "Coverage check failed")
}

func TestHeatmap_NoSummaryHidesTheSummaryAndKeepsTheGrid(t *testing.T) {
	out := heatmapReport(t, func(cfg *config.Config) { cfg.NoSummary = true })

	require.Contains(t, out, "threshold met")
	require.Contains(t, out, "\nBy File\n")
	require.NotContains(t, out, "Coverage check failed")
}

func TestHeatmap_GridAndSummaryAreShownByDefault(t *testing.T) {
	out := heatmapReport(t, func(*config.Config) {})

	require.Contains(t, out, "\nBy File\n")
	require.Contains(t, out, "Coverage check failed")
}
