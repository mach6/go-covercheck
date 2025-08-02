package output

import (
	"fmt"
	"os"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func trimWithEllipsis(str string, maxLen int) string {
	if maxLen < 0 {
		return ""
	}

	sLen := text.StringWidthWithoutEscSequences(str)
	if sLen <= maxLen {
		return str
	}

	ellipsis := "..."
	trimLen := maxLen - len(ellipsis)
	val := text.StripEscape(str)
	return ellipsis + val[len(val)-trimLen:]
}

func renderTable(results compute.Results, cfg *config.Config) {
	if cfg.NoTable {
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetAllowedRowLength(cfg.TerminalWidth)
	t.SetStyle(
		table.Style{
			Name:  "Custom",
			Box:   table.StyleBoxLight,
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
		},
	)

	// Set up headers based on whether uncovered lines should be shown
	var columnConfigs []table.ColumnConfig
	fixedWidth := 15
	headers := table.Row{"", "Statements", "Blocks", "Lines", "Statement %", "Block %", "Line %"}
	columnConfigs = []table.ColumnConfig{
		{Name: "", Align: text.AlignLeft, AlignFooter: text.AlignLeft, WidthMax: fixedWidth * 2,
			WidthMaxEnforcer: trimWithEllipsis},
		{Name: "Statements", Align: text.AlignRight, AlignHeader: text.AlignLeft,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth},
		{Name: "Blocks", Align: text.AlignRight, AlignHeader: text.AlignLeft,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth},
		{Name: "Lines", Align: text.AlignRight, AlignHeader: text.AlignLeft,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth},
		{Name: "Statement %", Align: text.AlignRight, AlignHeader: text.AlignLeft,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth},
		{Name: "Block %", Align: text.AlignRight, AlignHeader: text.AlignLeft,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth},
		{Name: "Line %", Align: text.AlignRight, AlignHeader: text.AlignLeft,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth},
	}
	if !cfg.HideUncoveredLines {
		usedWidth := 0
		for _, cc := range columnConfigs {
			if cc.WidthMax > 0 {
				usedWidth += cc.WidthMax
			} else if cc.WidthMin > 0 {
				usedWidth += cc.WidthMin
			}
		}
		remainingWidth := cfg.TerminalWidth - usedWidth
		if remainingWidth < fixedWidth {
			remainingWidth = fixedWidth
		}

		headers = append(headers, "Uncovered Lines")
		columnConfigs = append(columnConfigs,
			table.ColumnConfig{
				Name:             "Uncovered Lines",
				Align:            text.AlignLeft,
				AlignHeader:      text.AlignLeft,
				AlignFooter:      text.AlignLeft,
				WidthMax:         remainingWidth,
				WidthMaxEnforcer: trimWithEllipsis,
			},
		)
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

		row := table.Row{
			r.File,
			r.Statements,
			r.Blocks,
			r.Lines,
			stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
			blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
			lineColor(fmt.Sprintf("%.1f", r.LinePercentage)),
		}

		if !cfg.HideUncoveredLines {
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

		row := table.Row{
			r.Package,
			r.Statements,
			r.Blocks,
			r.Lines,
			stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
			blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
			lineColor(fmt.Sprintf("%.1f", r.LinePercentage)),
		}
		if !cfg.HideUncoveredLines {
			// Packages don't have uncovered lines, so show empty string
			row = append(row, "")
		}
		t.AppendRow(row)
	}

	stmtColor := severityColor(results.ByTotal.Statements.Percentage, results.ByTotal.Statements.Threshold)
	blockColor := severityColor(results.ByTotal.Blocks.Percentage, results.ByTotal.Blocks.Threshold)
	lineColor := severityColor(results.ByTotal.Lines.Percentage, results.ByTotal.Lines.Threshold)

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY TOTAL")})
	t.AppendSeparator()

	footer := table.Row{
		"",
		text.Bold.Sprint(results.ByTotal.Statements.Coverage),
		text.Bold.Sprint(results.ByTotal.Blocks.Coverage),
		text.Bold.Sprint(results.ByTotal.Lines.Coverage),
		stmtColor(text.Bold.Sprintf("%.1f", results.ByTotal.Statements.Percentage)),
		blockColor(text.Bold.Sprintf("%.1f", results.ByTotal.Blocks.Percentage)),
		lineColor(text.Bold.Sprintf("%.1f", results.ByTotal.Lines.Percentage)),
	}
	if !cfg.HideUncoveredLines {
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
