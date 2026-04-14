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

func renderTable(results compute.Results, cfg *config.Config) {
	if cfg.NoTable {
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetAllowedRowLength(cfg.TerminalWidth)
	t.SetStyle(getTableStyle(cfg))

	t.AppendHeader(table.Row{"", "Statements", "Blocks", "Statement %", "Block %"})

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
		{Name: "Statement %", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
		{Name: "Block %", Align: text.AlignRight,
			AlignFooter: text.AlignRight, WidthMax: fixedWidth, WidthMin: fixedWidth},
	})

	for _, r := range results.ByFile {
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

	stmtColor := severityColor(results.ByTotal.Statements.Percentage, results.ByTotal.Statements.Threshold)
	blockColor := severityColor(results.ByTotal.Blocks.Percentage, results.ByTotal.Blocks.Threshold)

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY PACKAGE")})
	t.AppendSeparator()

	for _, r := range results.ByPackage {
		stmtColor := severityColor(r.StatementPercentage, r.StatementThreshold)
		blockColor := severityColor(r.BlockPercentage, r.BlockThreshold)

		t.AppendRow(table.Row{
			r.Package,
			r.Statements,
			r.Blocks,
			stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
			blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
		})
	}

	t.AppendSeparator()
	t.AppendRow(table.Row{text.Bold.Sprint("BY TOTAL")})
	t.AppendSeparator()

	t.AppendFooter(table.Row{
		"",
		text.Bold.Sprint(results.ByTotal.Statements.Coverage),
		text.Bold.Sprint(results.ByTotal.Blocks.Coverage),
		stmtColor(text.Bold.Sprintf("%.1f", results.ByTotal.Statements.Percentage)),
		blockColor(text.Bold.Sprintf("%.1f", results.ByTotal.Blocks.Percentage)),
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
