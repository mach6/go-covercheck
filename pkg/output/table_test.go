package output_test

import (
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/stretchr/testify/require"
)

func TestBoxStyleFor(t *testing.T) {
	tests := []struct {
		name     string
		style    string
		expected table.BoxStyle
	}{
		{"default", config.TableStyleDefault, table.StyleBoxDefault},
		{"light", config.TableStyleLight, table.StyleBoxLight},
		{"bold", config.TableStyleBold, table.StyleBoxBold},
		{"rounded", config.TableStyleRounded, table.StyleBoxRounded},
		{"double", config.TableStyleDouble, table.StyleBoxDouble},
		{"unknown falls back to light", "plaid", table.StyleBoxLight},
		{"empty falls back to light", "", table.StyleBoxLight},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, output.BoxStyleFor(tt.style))
		})
	}
}

func TestGetTableStyle_UsesConfiguredBox(t *testing.T) {
	cfg := &config.Config{TableStyle: config.TableStyleDouble, TerminalWidth: 100}
	require.Equal(t, table.StyleBoxDouble, output.GetTableStyle(cfg).Box)
}

func TestGetHistoryTableStyle_UsesConfiguredBox(t *testing.T) {
	cfg := &config.Config{TableStyle: config.TableStyleRounded, TerminalWidth: 100}
	require.Equal(t, table.StyleBoxRounded, output.GetHistoryTableStyle(cfg).Box)
}

func TestTrimWithEllipsis(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"empty", "", 10, ""},
		{"negative maxLen", "anything", -1, ""},
		{"already short", "hi", 10, "hi"},
		{"exact fit", "hello", 5, "hello"},
		{"too long", "abcdefghij", 7, "...ghij"},
		{"maxLen equals ellipsis", "abcdef", 3, "abc"},
		{"maxLen one", "abcdef", 1, "a"},
		{"maxLen zero", "abcdef", 0, ""},
		{"ansi color preserved when short", "\x1b[31mred\x1b[0m", 10, "\x1b[31mred\x1b[0m"},
		{"ansi stripped when trimming", "\x1b[31mlong long line\x1b[0m", 7, "...line"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, output.TrimWithEllipsis(tt.input, tt.maxLen))
		})
	}
}

func TestApplyTableWidths_NameAndNumericColumnsCapped(t *testing.T) {
	configs := []table.ColumnConfig{
		{Name: ""},
		{Name: "Statements"},
		{Name: "Blocks"},
		{Name: "Lines"},
		{Name: "Statement %"},
		{Name: "Block %"},
		{Name: "Line %"},
	}
	cfg := &config.Config{TerminalWidth: 120}
	output.ApplyTableWidths(configs, cfg, compute.Results{})

	require.Equal(t, output.FixedColumnWidth*2, configs[0].WidthMax)
	require.NotNil(t, configs[0].WidthMaxEnforcer)
	for i := 1; i < len(configs); i++ {
		require.Equalf(t, output.FixedColumnWidth, configs[i].WidthMax, "column %q", configs[i].Name)
		require.NotNilf(t, configs[i].WidthMaxEnforcer, "column %q", configs[i].Name)
	}
}

func TestApplyTableWidths_UncoveredLinesAbsorbsRemaining(t *testing.T) {
	configs := []table.ColumnConfig{
		{Name: ""},
		{Name: "Statements"},
		{Name: "Blocks"},
		{Name: "Lines"},
		{Name: "Statement %"},
		{Name: "Block %"},
		{Name: "Line %"},
		{Name: "Uncovered Lines"},
	}
	cfg := &config.Config{TerminalWidth: 160}
	output.ApplyTableWidths(configs, cfg, compute.Results{})

	last := configs[len(configs)-1]
	require.NotNil(t, last.WidthMaxEnforcer,
		"Uncovered Lines column should be ellipsis-truncated, not row-clipped")
	require.GreaterOrEqual(t, last.WidthMax, output.FixedColumnWidth,
		"Uncovered Lines column should get at least fixedColumnWidth")
	require.Less(t, last.WidthMax, cfg.TerminalWidth,
		"Uncovered Lines column must leave room for the other columns")
}

func TestApplyTableWidths_UncoveredLinesFloorOnNarrowTerminal(t *testing.T) {
	configs := []table.ColumnConfig{
		{Name: ""},
		{Name: "Statements"},
		{Name: "Uncovered Lines"},
	}
	cfg := &config.Config{TerminalWidth: 40}
	output.ApplyTableWidths(configs, cfg, compute.Results{})

	last := configs[len(configs)-1]
	require.GreaterOrEqual(t, last.WidthMax, output.FixedColumnWidth,
		"Uncovered Lines column should floor at fixedColumnWidth when the terminal is narrow")
}
