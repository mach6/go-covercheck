package output

import (
	"encoding/csv"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func uncoveredFunctionsProfiles(t *testing.T) []*cover.Profile {
	t.Helper()
	return []*cover.Profile{uncoveredFunctionsProfile(t)}
}

// allHeaders are the headers of the table, in order, when every column is shown.
var allHeaders = []string{
	colName, colStatements, colBlocks, colLines, colFunctions, colStatementPct, colBlockPct, colLinePct,
	colFunctionPct, colUncoveredFuncs, colUncoveredLines,
}

// hiddenHeader is the header each --hide-columns name hides.
var hiddenHeader = map[string]string{
	config.ColumnStatements:         colStatements,
	config.ColumnBlocks:             colBlocks,
	config.ColumnLines:              colLines,
	config.ColumnFunctions:          colFunctions,
	config.ColumnStatementPercent:   colStatementPct,
	config.ColumnBlockPercent:       colBlockPct,
	config.ColumnLinePercent:        colLinePct,
	config.ColumnFunctionPercent:    colFunctionPct,
	config.ColumnUncoveredFunctions: colUncoveredFuncs,
	config.ColumnUncoveredLines:     colUncoveredLines,
}

// tempIndex matches the numbered temp directory each render of the test profile is written to.
var tempIndex = regexp.MustCompile(`/00\d+`)

// sameRender renders twice and returns both outputs with the temp directory numbers removed, so
// that they can be compared. Every render writes its profile to a new temp directory.
func sameRender(t *testing.T, format string, a, b func(*config.Config)) (string, string) {
	t.Helper()
	first := tempIndex.ReplaceAllString(renderUncoveredFunctions(t, format, a), "/NNN")
	second := tempIndex.ReplaceAllString(renderUncoveredFunctions(t, format, b), "/NNN")
	return first, second
}

func hide(names ...string) func(*config.Config) {
	return func(cfg *config.Config) { cfg.HideColumns = names }
}

// csvRecords parses csv output. Every record must have as many fields as the header.
func csvRecords(t *testing.T, out string) [][]string {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(out)).ReadAll()
	require.NoError(t, err, "every row must have a cell for every shown column")
	require.NotEmpty(t, records)
	return records
}

func without(headers []string, hidden ...string) []string {
	return slices.DeleteFunc(slices.Clone(headers), func(h string) bool { return slices.Contains(hidden, h) })
}

func TestHideColumns_EachColumnCanBeHidden(t *testing.T) {
	require.Len(t, hiddenHeader, len(config.HideableColumns), "every hideable column needs a header")

	for _, name := range config.HideableColumns {
		t.Run(name, func(t *testing.T) {
			records := csvRecords(t, renderUncoveredFunctions(t, config.FormatCSV, hide(name)))

			require.Equal(t, without(allHeaders, hiddenHeader[name]), records[0])
		})
	}
}

func TestHideColumns_NothingHiddenByDefault(t *testing.T) {
	records := csvRecords(t, renderUncoveredFunctions(t, config.FormatCSV, func(*config.Config) {}))

	require.Equal(t, allHeaders, records[0])
}

func TestHideColumns_SeveralAtOnce(t *testing.T) {
	records := csvRecords(t, renderUncoveredFunctions(t, config.FormatCSV,
		hide(config.ColumnStatements, config.ColumnBlocks, config.ColumnStatementPercent, config.ColumnBlockPercent)))

	require.Equal(t, without(allHeaders, colStatements, colBlocks, colStatementPct, colBlockPct), records[0])
}

func TestHideColumns_AllOfThem(t *testing.T) {
	out := renderUncoveredFunctions(t, config.FormatCSV, hide(config.HideableColumns...))

	for _, record := range csvRecords(t, out) {
		require.Len(t, record, 1, "only the name column is left")
	}
}

func TestHideColumns_TableMarkdownAndHTML(t *testing.T) {
	for _, format := range []string{config.FormatTable, config.FormatMD, config.FormatHTML} {
		t.Run(format, func(t *testing.T) {
			out := strings.ToUpper(renderUncoveredFunctions(t, format, hide(config.ColumnStatements)))

			require.NotContains(t, out, "STATEMENTS")
			require.Contains(t, out, "BLOCKS")
			require.Contains(t, out, "STATEMENT %", "only the named column is hidden")
		})
	}
}

func TestHideColumns_FunctionColumnsNeedAThreshold(t *testing.T) {
	// without a function threshold the table does not show them, so there is nothing to hide.
	plain, hidden := sameRender(t, config.FormatTable, func(*config.Config) {},
		hide(config.ColumnFunctions, config.ColumnFunctionPercent))
	require.Equal(t, plain, hidden)

	// with one, they are shown and can be hidden. "Uncovered Functions" is a different column and stays.
	shown := strings.ToUpper(renderUncoveredFunctions(t, config.FormatTable, withFunctionThreshold))
	require.Contains(t, shown, "│ FUNCTIONS ")
	require.Contains(t, shown, "│ FUNCTION % ")
	hiddenToo := strings.ToUpper(renderUncoveredFunctions(t, config.FormatTable, func(cfg *config.Config) {
		withFunctionThreshold(cfg)
		cfg.HideColumns = []string{config.ColumnFunctions, config.ColumnFunctionPercent}
	}))
	require.NotContains(t, hiddenToo, "│ FUNCTIONS ")
	require.NotContains(t, hiddenToo, "│ FUNCTION % ")
	require.Contains(t, hiddenToo, "│ UNCOVERED FUNCTIONS ")
}

func TestHideColumns_UncoveredColumnsMatchTheNoUncoveredFlags(t *testing.T) {
	for _, format := range []string{config.FormatTable, config.FormatCSV, config.FormatMD} {
		t.Run(format, func(t *testing.T) {
			flag, column := sameRender(t, format, func(cfg *config.Config) { cfg.NoUncoveredLines = true },
				hide(config.ColumnUncoveredLines))
			require.Equal(t, flag, column)

			flag, column = sameRender(t, format, func(cfg *config.Config) { cfg.NoUncoveredFunctions = true },
				hide(config.ColumnUncoveredFunctions))
			require.Equal(t, flag, column)
		})
	}
}

// oldCSVColumns is the list in the changelog that brings back the csv and tsv columns of v0.6.1.
var oldCSVColumns = []string{
	config.ColumnLines, config.ColumnFunctions, config.ColumnLinePercent, config.ColumnFunctionPercent,
	config.ColumnUncoveredFunctions, config.ColumnUncoveredLines,
}

func TestHideColumns_BringsBackTheCSVAndTSVColumnsOfV061(t *testing.T) {
	csvOut := renderUncoveredFunctions(t, config.FormatCSV, hide(oldCSVColumns...))
	tsvOut := renderUncoveredFunctions(t, config.FormatTSV, hide(oldCSVColumns...))

	require.Equal(t, ",Statements,Blocks,Statement %,Block %", strings.SplitN(csvOut, "\n", 2)[0])
	require.Equal(t, "\tStatements\tBlocks\tStatement %\tBlock %", strings.SplitN(tsvOut, "\n", 2)[0])
}

func TestHideColumns_DisabledForJSONYAMLAndHeatmap(t *testing.T) {
	for _, format := range []string{config.FormatJSON, config.FormatYAML, config.FormatHeatmap} {
		t.Run(format, func(t *testing.T) {
			plain, hidden := sameRender(t, format, func(*config.Config) {}, hide(config.HideableColumns...))

			require.NotEmpty(t, plain)
			require.Equal(t, plain, hidden)
		})
	}
}

func TestHideColumns_HiddenColumnIsStillChecked(t *testing.T) {
	configure := func(cfg *config.Config) {
		cfg.HideColumns = []string{config.ColumnStatements, config.ColumnStatementPercent}
		cfg.StatementThreshold = 100 // the profile covers half the statements
		cfg.NoSummary = false
	}
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	configure(cfg)

	// the check does not know about hidden columns.
	_, failed := compute.CollectResults(uncoveredFunctionsProfiles(t), cfg)
	require.True(t, failed)

	// the failure is still listed in the summary, though the columns are not shown.
	out := renderUncoveredFunctions(t, config.FormatTable, configure)
	require.NotContains(t, strings.ToUpper(out), "STATEMENTS")
	require.Contains(t, out, "[S]")
	require.Contains(t, out, "required for 100.0% threshold")
}
