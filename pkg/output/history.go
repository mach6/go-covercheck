package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/history"
)

// getHistoryTableStyle returns the appropriate table.Style for history tables.
func getHistoryTableStyle(cfg *config.Config) table.Style {
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
			SeparateRows:                   true,
		},
		Size: table.SizeOptions{
			WidthMax: cfg.TerminalWidth,
			WidthMin: 0,
		},
		Title: table.TitleOptionsDefault,
	}
}

// ShowHistory displays a summary table of the history entries.
func ShowHistory(h *history.History, limit int, cfg *config.Config) {
	count := limit
	if count <= 0 || count > len(h.Entries) {
		count = len(h.Entries)
	}

	if len(h.Entries) == 0 || count == 0 {
		fmt.Printf("≡ No history entries to show\n")
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(getHistoryTableStyle(cfg))

	t.AppendHeader(table.Row{"Timestamp", "Commit", "Branch", "Tags", "Label", "Coverage"})

	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "Timestamp", Align: text.AlignLeft},
		{Name: "Commit", Align: text.AlignLeft},
		{Name: "Branch", Align: text.AlignLeft},
		{Name: "Tags", Align: text.AlignLeft},
		{Name: "Label", Align: text.AlignLeft},
		{Name: "Coverage", Align: text.AlignLeft},
	})

	for i := range count {
		entry := h.Entries[i]

		stmtColor := severityColor(entry.Results.ByTotal.Statements.Percentage,
			entry.Results.ByTotal.Statements.Threshold)
		blockColor := severityColor(entry.Results.ByTotal.Blocks.Percentage,
			entry.Results.ByTotal.Blocks.Threshold)
		lineColor := severityColor(entry.Results.ByTotal.Lines.Percentage,
			entry.Results.ByTotal.Lines.Threshold)

		wrapTextWidth := 20

		// Build coverage display string - show line coverage only if data exists
		coverageDisplay := stmtColor(fmt.Sprintf("%-7s", entry.Results.ByTotal.Statements.Coverage)) + " [S]\n" +
			blockColor(fmt.Sprintf("%-7s", entry.Results.ByTotal.Blocks.Coverage)) + " [B]"
		if entry.Results.ByTotal.Lines.Coverage != "" {
			coverageDisplay += "\n" + lineColor(fmt.Sprintf("%-7s", entry.Results.ByTotal.Lines.Coverage)) + " [L]"
		}

		t.AppendRow(table.Row{
			fmt.Sprintf("%-10s", entry.Timestamp.Format("2006-01-02")),
			fmt.Sprintf("%-7s", history.ShortCommit(entry.Commit)),
			fmt.Sprintf("%-15s", entry.Branch),
			fmt.Sprintf("%-15s", wrapText(strings.Join(entry.Tags, ", "), wrapTextWidth)),
			wrapText(fmt.Sprintf("%-15s", entry.Label), wrapTextWidth),
			coverageDisplay,
		})
	}

	t.Render()

	// clever conditional in the absence of a ternary operator?
	fmt.Printf("≡ Showing last %d history entr%s\n", count,
		map[bool]string{true: "y", false: "ies"}[count == 1])
}

func formatDelta(delta float64) (string, bool) {
	if delta == 0 {
		return "", false
	}
	if delta < 0 {
		return fmt.Sprintf("−%-4.1f%%", -delta), true
	}
	return fmt.Sprintf("+%-4.1f%%", delta), true
}

func wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}
	var wrapped []string
	words := strings.Fields(text)
	line := ""
	for _, word := range words {
		if len(line)+len(word)+1 > width {
			wrapped = append(wrapped, line)
			line = word
		} else {
			if line != "" {
				line += " "
			}
			line += word
		}
	}
	if line != "" {
		wrapped = append(wrapped, line)
	}
	return strings.Join(wrapped, "\n")
}

func compareShowS() {
	fmt.Printf("    [%s] ", color.New(color.FgCyan).Sprint("S"))
}

func compareShowB() {
	fmt.Printf("    [%s] ", color.New(color.FgHiMagenta).Sprint("B"))
}

func compareShowL() {
	fmt.Printf("    [%s] ", color.New(color.FgYellow).Sprint("L"))
}
