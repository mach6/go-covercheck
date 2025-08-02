package output

import (
	"fmt"
	"os"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

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
	var headers table.Row
	var columnConfigs []table.ColumnConfig
	fixedWidth := 15

	if !cfg.HideUncoveredLines {
		headers = table.Row{"", "Statements", "Blocks", "Statement %", "Block %", "Uncovered Lines"}
		columnConfigs = []table.ColumnConfig{
			{Name: "", Align: text.AlignLeft, AlignFooter: text.AlignLeft},
			{Name: "Statements", Align: text.AlignRight, AlignFooter: text.AlignRight, WidthMin: fixedWidth},
			{Name: "Blocks", Align: text.AlignRight, AlignFooter: text.AlignRight, WidthMin: fixedWidth},
			{Name: "Statement %", Align: text.AlignRight, AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
			{Name: "Block %", Align: text.AlignRight, AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
			{Name: "Uncovered Lines", Align: text.AlignLeft, AlignFooter: text.AlignLeft, WidthMax: 20},
		}
	} else {
		headers = table.Row{"", "Statements", "Blocks", "Statement %", "Block %"}
		columnConfigs = []table.ColumnConfig{
			{Name: "", Align: text.AlignLeft, AlignFooter: text.AlignLeft},
			{Name: "Statements", Align: text.AlignRight, AlignFooter: text.AlignRight, WidthMin: fixedWidth},
			{Name: "Blocks", Align: text.AlignRight, AlignFooter: text.AlignRight, WidthMin: fixedWidth},
			{Name: "Statement %", Align: text.AlignRight, AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
			{Name: "Block %", Align: text.AlignRight, AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
		}
	}

	t.AppendHeader(headers)
	t.SetColumnConfigs(columnConfigs)

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY FILE")})
	t.AppendSeparator()

	for _, r := range results.ByFile {
		stmtColor := severityColor(r.StatementPercentage, r.StatementThreshold)
		blockColor := severityColor(r.BlockPercentage, r.BlockThreshold)

		var row table.Row
		if !cfg.HideUncoveredLines {
			uncoveredLinesFormatted := r.UncoveredLines
			if uncoveredLinesFormatted != "" {
				// Apply color to uncovered lines - use red to indicate missing coverage
				uncoveredLinesFormatted = redColor(uncoveredLinesFormatted)
			}
			row = table.Row{
				r.File,
				r.Statements,
				r.Blocks,
				stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
				blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
				uncoveredLinesFormatted,
			}
		} else {
			row = table.Row{
				r.File,
				r.Statements,
				r.Blocks,
				stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
				blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
			}
		}
		t.AppendRow(row)
	}

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY PACKAGE")})
	t.AppendSeparator()

	for _, r := range results.ByPackage {
		stmtColor := severityColor(r.StatementPercentage, r.StatementThreshold)
		blockColor := severityColor(r.BlockPercentage, r.BlockThreshold)

		var row table.Row
		if !cfg.HideUncoveredLines {
			// Packages don't have individual file uncovered lines, so show empty string
			row = table.Row{
				r.Package,
				r.Statements,
				r.Blocks,
				stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
				blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
				"",
			}
		} else {
			row = table.Row{
				r.Package,
				r.Statements,
				r.Blocks,
				stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
				blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
			}
		}
		t.AppendRow(row)
	}

	stmtColor := severityColor(results.ByTotal.Statements.Percentage, results.ByTotal.Statements.Threshold)
	blockColor := severityColor(results.ByTotal.Blocks.Percentage, results.ByTotal.Blocks.Threshold)

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY TOTAL")})
	t.AppendSeparator()

	var footer table.Row
	if !cfg.HideUncoveredLines {
		footer = table.Row{
			"",
			text.Bold.Sprint(results.ByTotal.Statements.Coverage),
			text.Bold.Sprint(results.ByTotal.Blocks.Coverage),
			stmtColor(text.Bold.Sprintf("%.1f", results.ByTotal.Statements.Percentage)),
			blockColor(text.Bold.Sprintf("%.1f", results.ByTotal.Blocks.Percentage)),
			"",
		}
	} else {
		footer = table.Row{
			"",
			text.Bold.Sprint(results.ByTotal.Statements.Coverage),
			text.Bold.Sprint(results.ByTotal.Blocks.Coverage),
			stmtColor(text.Bold.Sprintf("%.1f", results.ByTotal.Statements.Percentage)),
			blockColor(text.Bold.Sprintf("%.1f", results.ByTotal.Blocks.Percentage)),
		}
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
