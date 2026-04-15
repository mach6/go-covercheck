package output

import (
	"fmt"
	"os"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// boxStyleFor maps a cfg.TableStyle value to the corresponding go-pretty BoxStyle.
// Unknown values fall back to the light box to match cfg.ApplyDefaults.
func boxStyleFor(style string) table.BoxStyle {
	switch style {
	case config.TableStyleDefault:
		return table.StyleBoxDefault
	case config.TableStyleBold:
		return table.StyleBoxBold
	case config.TableStyleRounded:
		return table.StyleBoxRounded
	case config.TableStyleDouble:
		return table.StyleBoxDouble
	default: // config.TableStyleLight or any other value
		return table.StyleBoxLight
	}
}

// getTableStyle returns the appropriate table.Style based on the config.
func getTableStyle(cfg *config.Config) table.Style {
	return table.Style{
		Name:  "Custom",
		Box:   boxStyleFor(cfg.TableStyle),
		Color: table.ColorOptionsDefault,
		Format: table.FormatOptions{
			Footer:       text.FormatDefault,
			FooterAlign:  text.AlignRight,
			FooterVAlign: text.VAlignDefault,
			Header:       text.FormatUpper,
			HeaderAlign:  text.AlignCenter,
			HeaderVAlign: text.VAlignDefault,
			Row:          text.FormatDefault,
			RowAlign:     text.AlignRight,
			RowVAlign:    text.VAlignDefault,
		},
		HTML: table.DefaultHTMLOptions,
		Options: table.Options{
			DoNotColorBordersAndSeparators: false,
			DrawBorder:                     true,
			SeparateColumns:                true,
			SeparateFooter:                 true,
			SeparateHeader:                 true,
			SeparateRows:                   false,
		},
		Size: table.SizeOptions{
			WidthMax: cfg.TerminalWidth,
			WidthMin: 0,
		},
		Title: table.TitleOptionsDefault,
	}
}

// nameColumnWidthFactor is how much wider the leading name column is relative
// to the other fixed-width columns.
const nameColumnWidthFactor = 2

func trimWithEllipsis(str string, maxLen int) string {
	if maxLen < 0 {
		return ""
	}

	sLen := text.StringWidthWithoutEscSequences(str)
	if sLen <= maxLen {
		return str
	}

	const ellipsis = "..."
	runes := []rune(text.StripEscape(str))
	if maxLen <= len(ellipsis) {
		if maxLen >= len(runes) {
			return string(runes)
		}
		return string(runes[:maxLen])
	}
	trimLen := maxLen - len(ellipsis)
	if trimLen >= len(runes) {
		return string(runes)
	}
	return ellipsis + string(runes[len(runes)-trimLen:])
}

// fixedColumnWidth is the width cap (in characters) applied to each numeric
// column in the human-facing table output.
const fixedColumnWidth = 15

// perColumnOverhead is the rendered overhead (padding + one border) that
// go-pretty adds per column; there is also a single leading border on the row.
// These values line up with the default/light/bold/rounded/double box styles
// used by this tool.
const perColumnOverhead = 3

// Column header labels, referenced from multiple helpers.
const (
	colName           = ""
	colStatements     = "Statements"
	colBlocks         = "Blocks"
	colLines          = "Lines"
	colFunctions      = "Functions"
	colStatementPct   = "Statement %"
	colBlockPct       = "Block %"
	colLinePct        = "Line %"
	colFunctionPct    = "Function %"
	colUncoveredLines = "Uncovered Lines"
)

// applyTableWidths fills in WidthMax / WidthMaxEnforcer on the supplied column
// configs for human-facing table output. The leading name column gets a wider
// cap and the numeric columns are bounded to fixedColumnWidth. The trailing
// "Uncovered Lines" column absorbs whatever terminal width is left over after
// accounting for the actual rendered widths of the other columns — it is
// ellipsis-truncated via trimWithEllipsis rather than letting go-pretty clip
// the whole row with its "≈" marker.
//nolint:cyclop // split into per-column + remaining-width passes; merging hurts readability
func applyTableWidths(columnConfigs []table.ColumnConfig, cfg *config.Config, results compute.Results) {
	uncoveredIdx := -1
	for i := range columnConfigs {
		if columnConfigs[i].Name == colUncoveredLines {
			uncoveredIdx = i
			continue
		}
		width := fixedColumnWidth
		if columnConfigs[i].Name == colName {
			width = fixedColumnWidth * nameColumnWidthFactor
		}
		columnConfigs[i].WidthMax = width
		columnConfigs[i].WidthMaxEnforcer = trimWithEllipsis
	}

	if uncoveredIdx < 0 {
		return
	}

	// Without a positive terminal width we can't meaningfully compute how much
	// room is left. SetAllowedRowLength(0) already leaves the row unconstrained
	// in that case, so leave "Uncovered Lines" unconstrained too instead of
	// clamping it to fixedColumnWidth and producing surprisingly narrow output.
	if cfg.TerminalWidth <= 0 {
		return
	}

	usedContent := 0
	for i, cc := range columnConfigs {
		if i == uncoveredIdx {
			continue
		}
		content := maxRenderedWidth(cc.Name, results)
		if cc.WidthMax > 0 && content > cc.WidthMax {
			content = cc.WidthMax
		}
		usedContent += content
	}
	overhead := len(columnConfigs)*perColumnOverhead + 1
	remaining := cfg.TerminalWidth - usedContent - overhead
	if remaining < fixedColumnWidth {
		remaining = fixedColumnWidth
	}
	columnConfigs[uncoveredIdx].WidthMax = remaining
	columnConfigs[uncoveredIdx].WidthMaxEnforcer = trimWithEllipsis
}

// displayWidth measures terminal display width, skipping ANSI escape sequences
// and counting multi-byte runes correctly. go-pretty's text.StringWidth…
// helper does the same thing, and we use it here so remaining-width math for
// the "Uncovered Lines" column reflects the terminal columns a cell actually
// consumes rather than raw byte length.
func displayWidth(s string) int {
	return text.StringWidthWithoutEscSequences(s)
}

// maxRenderedWidth returns the widest string that will be rendered in the
// given column across the provided results, bounded below by the header label
// length (go-pretty won't shrink a column below its header).
func maxRenderedWidth(column string, results compute.Results) int {
	width := displayWidth(column)
	switch column {
	case colName:
		for _, r := range results.ByFile {
			width = maxInt(width, displayWidth(r.File))
		}
		for _, r := range results.ByPackage {
			width = maxInt(width, displayWidth(r.Package))
		}
	case colStatements, colBlocks, colLines, colFunctions:
		width = maxInt(width, maxCoverageWidth(column, results))
	case colStatementPct, colBlockPct, colLinePct, colFunctionPct:
		// Percentages render as "%.1f" — 5 cells max ("100.0").
		width = maxInt(width, displayWidth("100.0"))
	}
	return width
}

func maxCoverageWidth(column string, results compute.Results) int {
	width := 0
	for _, r := range results.ByFile {
		width = maxInt(width, displayWidth(coverageCell(column, r.By)))
	}
	for _, r := range results.ByPackage {
		width = maxInt(width, displayWidth(coverageCell(column, r.By)))
	}
	switch column {
	case colStatements:
		width = maxInt(width, displayWidth(results.ByTotal.Statements.Coverage))
	case colBlocks:
		width = maxInt(width, displayWidth(results.ByTotal.Blocks.Coverage))
	case colLines:
		width = maxInt(width, displayWidth(results.ByTotal.Lines.Coverage))
	case colFunctions:
		width = maxInt(width, displayWidth(results.ByTotal.Functions.Coverage))
	}
	return width
}

func coverageCell(column string, by compute.By) string {
	switch column {
	case colStatements:
		return by.Statements
	case colBlocks:
		return by.Blocks
	case colLines:
		return by.Lines
	case colFunctions:
		return by.Functions
	}
	return ""
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

//nolint:cyclop // sequential table setup; splitting hurts readability
func renderTable(results compute.Results, cfg *config.Config) {
	if cfg.NoTable {
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetAllowedRowLength(cfg.TerminalWidth)
	t.SetStyle(getTableStyle(cfg))

	headers := table.Row{"", "Statements", "Blocks", "Lines", "Functions", "Statement %", "Block %", "Line %", "Function %"}
	columnConfigs := []table.ColumnConfig{
		{Name: "", Align: text.AlignLeft, AlignFooter: text.AlignLeft},
		{Name: "Statements", Align: text.AlignRight, AlignHeader: text.AlignLeft, AlignFooter: text.AlignRight},
		{Name: "Blocks", Align: text.AlignRight, AlignHeader: text.AlignLeft, AlignFooter: text.AlignRight},
		{Name: "Lines", Align: text.AlignRight, AlignHeader: text.AlignLeft, AlignFooter: text.AlignRight},
		{Name: "Functions", Align: text.AlignRight, AlignHeader: text.AlignLeft, AlignFooter: text.AlignRight},
		{Name: "Statement %", Align: text.AlignRight, AlignHeader: text.AlignLeft, AlignFooter: text.AlignRight},
		{Name: "Block %", Align: text.AlignRight, AlignHeader: text.AlignLeft, AlignFooter: text.AlignRight},
		{Name: "Line %", Align: text.AlignRight, AlignHeader: text.AlignLeft, AlignFooter: text.AlignRight},
		{Name: "Function %", Align: text.AlignRight, AlignHeader: text.AlignLeft, AlignFooter: text.AlignRight},
	}
	if !cfg.NoUncoveredLines {
		headers = append(headers, "Uncovered Lines")
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Name:        "Uncovered Lines",
			Align:       text.AlignLeft,
			AlignHeader: text.AlignLeft,
			AlignFooter: text.AlignLeft,
		})
	}

	// Width-based truncation only makes sense for the human-facing table format;
	// md/html/csv/tsv should preserve full cell values.
	if cfg.Format == config.FormatTable {
		applyTableWidths(columnConfigs, cfg, results)
	}

	t.AppendHeader(headers)
	t.SetColumnConfigs(columnConfigs)

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY FILE")})
	t.AppendSeparator()

	for _, r := range results.ByFile {
		stmtColor := severityColor(r.StatementPercentage, r.StatementThreshold)
		blockColor := severityColor(r.BlockPercentage, r.BlockThreshold)
		lineColor := severityColor(r.LinePercentage, r.LineThreshold)
		functionColor := severityColor(r.FunctionPercentage, r.FunctionThreshold)

		row := table.Row{
			r.File,
			r.Statements,
			r.Blocks,
			r.Lines,
			r.Functions,
			stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
			blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
			lineColor(fmt.Sprintf("%.1f", r.LinePercentage)),
			functionColor(fmt.Sprintf("%.1f", r.FunctionPercentage)),
		}

		if !cfg.NoUncoveredLines {
			row = append(row, r.UncoveredLines)
		}
		t.AppendRow(row)
	}

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY PACKAGE")})
	t.AppendSeparator()

	for _, r := range results.ByPackage {
		stmtColor := severityColor(r.StatementPercentage, r.StatementThreshold)
		blockColor := severityColor(r.BlockPercentage, r.BlockThreshold)
		lineColor := severityColor(r.LinePercentage, r.LineThreshold)
		functionColor := severityColor(r.FunctionPercentage, r.FunctionThreshold)

		row := table.Row{
			r.Package,
			r.Statements,
			r.Blocks,
			r.Lines,
			r.Functions,
			stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
			blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
			lineColor(fmt.Sprintf("%.1f", r.LinePercentage)),
			functionColor(fmt.Sprintf("%.1f", r.FunctionPercentage)),
		}
		if !cfg.NoUncoveredLines {
			// Packages don't have uncovered lines, so show empty string
			row = append(row, "")
		}
		t.AppendRow(row)
	}

	stmtColor := severityColor(results.ByTotal.Statements.Percentage, results.ByTotal.Statements.Threshold)
	blockColor := severityColor(results.ByTotal.Blocks.Percentage, results.ByTotal.Blocks.Threshold)
	lineColor := severityColor(results.ByTotal.Lines.Percentage, results.ByTotal.Lines.Threshold)
	functionColor := severityColor(results.ByTotal.Functions.Percentage, results.ByTotal.Functions.Threshold)

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY TOTAL")})
	t.AppendSeparator()

	footer := table.Row{
		"",
		text.Bold.Sprint(results.ByTotal.Statements.Coverage),
		text.Bold.Sprint(results.ByTotal.Blocks.Coverage),
		text.Bold.Sprint(results.ByTotal.Lines.Coverage),
		text.Bold.Sprint(results.ByTotal.Functions.Coverage),
		stmtColor(text.Bold.Sprintf("%.1f", results.ByTotal.Statements.Percentage)),
		blockColor(text.Bold.Sprintf("%.1f", results.ByTotal.Blocks.Percentage)),
		lineColor(text.Bold.Sprintf("%.1f", results.ByTotal.Lines.Percentage)),
		functionColor(text.Bold.Sprintf("%.1f", results.ByTotal.Functions.Percentage)),
	}
	if !cfg.NoUncoveredLines {
		footer = append(footer, "")
	}
	t.AppendFooter(footer)

	switch cfg.Format {
	case config.FormatMD:
		t.RenderMarkdown()
	case config.FormatHTML:
		t.RenderHTML()
	case config.FormatTSV:
		t.RenderTSV()
	case config.FormatCSV:
		t.RenderCSV()
	default:
		t.Render()
	}
}
