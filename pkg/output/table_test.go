package output_test

import (
	"testing"

	"github.com/jedib0t/go-pretty/v6/table"
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
