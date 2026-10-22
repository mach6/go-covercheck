package output

import (
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

const uncoveredFunctionsSource = `package p

func Hit() {
	println()
}

func (s *S[K]) Miss() {
	println()
}
`

const uncoveredFunctionEntry = "(*S[K]).Miss:7"

// uncoveredFunctionsProfile profiles a temporary source file in which Hit ran
// and the generic pointer method Miss did not.
func uncoveredFunctionsProfile(t *testing.T) *cover.Profile {
	t.Helper()
	return &cover.Profile{
		FileName: test.CreateTempFile(t, "uncovered_functions.go", uncoveredFunctionsSource),
		Mode:     "set",
		Blocks: []cover.ProfileBlock{
			{StartLine: 3, StartCol: 13, EndLine: 5, EndCol: 2, NumStmt: 1, Count: 1},
			{StartLine: 7, StartCol: 21, EndLine: 9, EndCol: 2, NumStmt: 1, Count: 0},
		},
	}
}

func renderUncoveredFunctions(t *testing.T, format string, configure func(cfg *config.Config)) string {
	t.Helper()
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = format
	cfg.NoColor = true
	cfg.NoSummary = true
	prev := color.NoColor
	t.Cleanup(func() { color.NoColor = prev })
	color.NoColor = true
	text.DisableColors()
	configure(cfg)

	profiles := []*cover.Profile{uncoveredFunctionsProfile(t)}
	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		FormatAndReport(results, cfg, failed)
	})
	return stdout
}

func withFunctionThreshold(cfg *config.Config) { cfg.FunctionThreshold = 50 }

func TestRenderTable_UncoveredFunctionsColumn(t *testing.T) {
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
		{"table with threshold", config.FormatTable, withFunctionThreshold, true},
		{"md with threshold", config.FormatMD, withFunctionThreshold, true},
		{"html with threshold", config.FormatHTML, withFunctionThreshold, true},
		{"csv always", config.FormatCSV, none, true},
		{"tsv always", config.FormatTSV, none, true},
		{"table with threshold, suppressed", config.FormatTable, func(c *config.Config) {
			withFunctionThreshold(c)
			c.NoUncoveredFunctions = true
		}, false},
		{"md with threshold, suppressed", config.FormatMD, func(c *config.Config) {
			withFunctionThreshold(c)
			c.NoUncoveredFunctions = true
		}, false},
		{"html with threshold, suppressed", config.FormatHTML, func(c *config.Config) {
			withFunctionThreshold(c)
			c.NoUncoveredFunctions = true
		}, false},
		{"csv suppressed", config.FormatCSV, func(c *config.Config) { c.NoUncoveredFunctions = true }, false},
		{"tsv suppressed", config.FormatTSV, func(c *config.Config) { c.NoUncoveredFunctions = true }, false},
		{"table keeps column when only lines are suppressed", config.FormatTable, func(c *config.Config) {
			withFunctionThreshold(c)
			c.NoUncoveredLines = true
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := renderUncoveredFunctions(t, tt.format, tt.configure)
			require.Equal(t, tt.want, strings.Contains(strings.ToLower(headerRow(t, out)), "uncovered functions"), out)
			require.Equal(t, tt.want, strings.Contains(out, uncoveredFunctionEntry), out)
		})
	}
}

func TestRenderTable_UncoveredFunctionsBeforeUncoveredLines(t *testing.T) {
	out := renderUncoveredFunctions(t, config.FormatCSV, func(*config.Config) {})
	header := headerRow(t, out)
	require.Contains(t, header, "function %,uncovered functions,uncovered lines")
	require.Contains(t, out, ","+uncoveredFunctionEntry+",")
}

func TestRenderTable_RowsMatchHeaderWidthWithUncoveredFunctions(t *testing.T) {
	for _, format := range []string{config.FormatTable, config.FormatCSV} {
		out := renderUncoveredFunctions(t, format, withFunctionThreshold)
		sep := "│"
		if format == config.FormatCSV {
			sep = ","
		}
		var rows []string
		for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
			if format == config.FormatCSV || strings.HasPrefix(l, sep) {
				rows = append(rows, l)
			}
		}
		require.Greater(t, len(rows), 2)
		want := strings.Count(rows[0], sep)
		for _, l := range rows {
			require.Equal(t, want, strings.Count(l, sep), "%s: %s", format, l)
		}
	}
}

func TestFormatAndReport_StructuredUncoveredFunctions(t *testing.T) {
	none := func(*config.Config) {}
	suppress := func(c *config.Config) { c.NoUncoveredFunctions = true }

	j := renderUncoveredFunctions(t, config.FormatJSON, none)
	require.Contains(t, j, `"uncoveredFunctions": "`+uncoveredFunctionEntry+`"`)
	require.NotContains(t, renderUncoveredFunctions(t, config.FormatJSON, suppress), "uncoveredFunctions")

	y := renderUncoveredFunctions(t, config.FormatYAML, none)
	require.Contains(t, y, "uncoveredFunctions: "+uncoveredFunctionEntry)
	require.NotContains(t, renderUncoveredFunctions(t, config.FormatYAML, suppress), "uncoveredFunctions")

	// Structured output carries the functions without any function threshold,
	// and --no-uncovered-lines leaves them alone.
	onlyLinesOff := renderUncoveredFunctions(t, config.FormatJSON, func(c *config.Config) { c.NoUncoveredLines = true })
	require.Contains(t, onlyLinesOff, uncoveredFunctionEntry)
	require.NotContains(t, onlyLinesOff, "uncoveredLines")
}
