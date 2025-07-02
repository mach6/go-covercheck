package formatter

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/config"
)

func renderTable(results Results, cfg *config.Config) {
	if cfg.NoTable {
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"File", "Statements", "Blocks", "Statement %", "Block %"})

	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "File", Align: text.AlignLeft, AlignFooter: text.AlignLeft},
		{Name: "Statements", Align: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Blocks", Align: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Statement %", Align: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Block %", Align: text.AlignRight, AlignFooter: text.AlignRight},
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
