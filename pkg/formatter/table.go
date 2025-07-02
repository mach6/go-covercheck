package formatter

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/config"
	"golang.org/x/term"
)

func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80 //nolint:mnd
	}
	return width
}

func renderTable(results Results, cfg *config.Config) {
	if cfg.NoTable {
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetAllowedRowLength(getTerminalWidth())
	t.SetStyle(
		table.Style{
			Name:    "Custom",
			Box:     table.StyleBoxLight,
			Color:   table.ColorOptionsDefault,
			Format:  table.FormatOptionsDefault,
			HTML:    table.DefaultHTMLOptions,
			Options: table.OptionsDefault,
			Size: table.SizeOptions{
				WidthMax: getTerminalWidth(),
				WidthMin: 0,
			},
			Title: table.TitleOptionsDefault,
		},
	)

	t.AppendHeader(table.Row{"File", "Statements", "Blocks", "Statement %", "Block %"})

	fixedWidth := 15
	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "File", Align: text.AlignLeft,
			AlignFooter: text.AlignLeft},
		{Name: "Statements", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMin: fixedWidth},
		{Name: "Blocks", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMin: fixedWidth},
		{Name: "Statement %", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
		{Name: "Block %", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
	})

	for _, r := range results.Results {
		stmtColor := severityColor(r.StatementPercentage, r.StatementThreshold)
		blockColor := severityColor(r.BlockPercentage, r.BlockThreshold)

		t.AppendRow(table.Row{
			r.File,
			r.Statements,
			r.Blocks,
			stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
			blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
		})
	}

	stmtTotalPct := percent(results.TotalCoveredStatements, results.TotalStatements)
	blockTotalPct := percent(results.TotalCoveredBlocks, results.TotalBlocks)
	stmtColor := severityColor(stmtTotalPct, cfg.StatementThreshold)
	blockColor := severityColor(blockTotalPct, cfg.BlockThreshold)

	t.AppendFooter(table.Row{
		text.Bold.Sprint("TOTAL"),
		text.Bold.Sprintf("%d/%d", results.TotalCoveredStatements, results.TotalStatements),
		text.Bold.Sprintf("%d/%d", results.TotalCoveredBlocks, results.TotalBlocks),
		stmtColor(text.Bold.Sprintf("%.1f", stmtTotalPct)),
		blockColor(text.Bold.Sprintf("%.1f", blockTotalPct)),
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
