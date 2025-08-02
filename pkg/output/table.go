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

	t.AppendHeader(table.Row{"", "Statements", "Blocks", "Functions", "Statement %", "Block %", "Function %"})

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY FILE")})
	t.AppendSeparator()

	fixedWidth := 15
	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "", Align: text.AlignLeft,
			AlignFooter: text.AlignLeft},
		{Name: "Statements", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMin: fixedWidth},
		{Name: "Blocks", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMin: fixedWidth},
		{Name: "Functions", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMin: fixedWidth},
		{Name: "Statement %", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
		{Name: "Block %", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
		{Name: "Function %", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
	})

	for _, r := range results.ByFile {
		stmtColor := severityColor(r.StatementPercentage, r.StatementThreshold)
		blockColor := severityColor(r.BlockPercentage, r.BlockThreshold)
		functionColor := severityColor(r.FunctionPercentage, r.FunctionThreshold)

		t.AppendRow(table.Row{
			r.File,
			r.Statements,
			r.Blocks,
			r.Functions,
			stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
			blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
			functionColor(fmt.Sprintf("%.1f", r.FunctionPercentage)),
		})
	}

	stmtColor := severityColor(results.ByTotal.Statements.Percentage, results.ByTotal.Statements.Threshold)
	blockColor := severityColor(results.ByTotal.Blocks.Percentage, results.ByTotal.Blocks.Threshold)
	functionColor := severityColor(results.ByTotal.Functions.Percentage, results.ByTotal.Functions.Threshold)

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY PACKAGE")})
	t.AppendSeparator()

	for _, r := range results.ByPackage {
		stmtColor := severityColor(r.StatementPercentage, r.StatementThreshold)
		blockColor := severityColor(r.BlockPercentage, r.BlockThreshold)
		functionColor := severityColor(r.FunctionPercentage, r.FunctionThreshold)

		t.AppendRow(table.Row{
			r.Package,
			r.Statements,
			r.Blocks,
			r.Functions,
			stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
			blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
			functionColor(fmt.Sprintf("%.1f", r.FunctionPercentage)),
		})
	}

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY TOTAL")})
	t.AppendSeparator()

	t.AppendFooter(table.Row{
		"",
		text.Bold.Sprint(results.ByTotal.Statements.Coverage),
		text.Bold.Sprint(results.ByTotal.Blocks.Coverage),
		text.Bold.Sprint(results.ByTotal.Functions.Coverage),
		stmtColor(text.Bold.Sprintf("%.1f", results.ByTotal.Statements.Percentage)),
		blockColor(text.Bold.Sprintf("%.1f", results.ByTotal.Blocks.Percentage)),
		functionColor(text.Bold.Sprintf("%.1f", results.ByTotal.Functions.Percentage)),
	})

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
