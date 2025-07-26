package output

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/history"
)

// CompareHistory shows the comparison output for a ref: and the results.
func CompareHistory(ref string, refEntry *history.Entry, results compute.Results) {
	fmt.Printf("\n≡ Comparing against ref: %s [commit %s]\n",
		color.New(color.FgBlue).Sprint(ref),
		color.New(color.FgHiBlack).Sprint(refEntry.Commit[:7]),
	)

	bPrintedFile := compareByFile(results, refEntry)
	bPrintedPkg := compareByPackage(results, refEntry)
	bPrintedTotal := compareByTotal(results, refEntry)

	if !bPrintedTotal && !bPrintedPkg && !bPrintedFile {
		fmt.Println(" → No change")
	}
}

func compareByPackage(results compute.Results, refEntry *history.Entry) bool {
	// ByPackage
	bPrintedPkg := false
	for _, curr := range results.ByPackage {
		for _, prev := range refEntry.Results.ByPackage {
			if curr.Package == prev.Package { //nolint:nestif
				s, ss := formatDelta(curr.StatementPercentage - prev.StatementPercentage)
				b, sb := formatDelta(curr.BlockPercentage - prev.BlockPercentage)
				if ss || sb {
					if !bPrintedPkg {
						fmt.Printf(" → By Package\n")
						bPrintedPkg = true
					}

					if ss {
						compareShowS()
						fmt.Printf("%s [%s]\n", curr.Package, s)
					}
					if sb {
						compareShowB()
						fmt.Printf("%s [%s]\n", curr.Package, b)
					}
					fmt.Println()
				}
			}
		}
	}
	return bPrintedPkg
}

func compareByTotal(results compute.Results, refEntry *history.Entry) bool {
	// Totals
	bPrintedTotal := false
	deltaS, okS := formatDelta(results.ByTotal.Statements.Percentage - refEntry.Results.ByTotal.Statements.Percentage)
	deltaB, okB := formatDelta(results.ByTotal.Blocks.Percentage - refEntry.Results.ByTotal.Blocks.Percentage)

	if okS || okB {
		fmt.Printf(" → By Total\n")
		bPrintedTotal = true
		if okS {
			compareShowS()
			fmt.Printf("total [%s]\n", deltaS)
		}
		if okB {
			compareShowB()
			fmt.Printf("total [%s]\n", deltaB)
		}
	}
	return bPrintedTotal
}

func compareByFile(results compute.Results, refEntry *history.Entry) bool {
	// ByFile
	bPrintedFile := false
	for _, curr := range results.ByFile {
		for _, prev := range refEntry.Results.ByFile {
			if curr.File == prev.File { //nolint:nestif
				s, ss := formatDelta(curr.StatementPercentage - prev.StatementPercentage)
				b, sb := formatDelta(curr.BlockPercentage - prev.BlockPercentage)
				if ss || sb {
					if !bPrintedFile {
						fmt.Printf(" → By File\n")
						bPrintedFile = true
					}

					if ss {
						compareShowS()
						fmt.Printf("%s [%s]\n", curr.File, s)
					}
					if sb {
						compareShowB()
						fmt.Printf("%s [%s]\n", curr.File, b)
					}
					fmt.Println()
				}
			}
		}
	}
	return bPrintedFile
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
				SeparateRows:                   true,
			},
			Size: table.SizeOptions{
				WidthMax: cfg.TerminalWidth,
				WidthMin: 0,
			},
			Title: table.TitleOptionsDefault,
		},
	)

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

		wrapTextWidth := 20
		t.AppendRow(table.Row{
			fmt.Sprintf("%-10s", entry.Timestamp.Format("2006-01-02")),
			fmt.Sprintf("%-7s", entry.Commit[:7]),
			fmt.Sprintf("%-15s", entry.Branch),
			fmt.Sprintf("%-15s", wrapText(strings.Join(entry.Tags, ", "), wrapTextWidth)),
			wrapText(fmt.Sprintf("%-15s", entry.Label), wrapTextWidth),
			stmtColor(fmt.Sprintf("%-7s", entry.Results.ByTotal.Statements.Coverage)) + " [S]\n" +
				blockColor(fmt.Sprintf("%-7s", entry.Results.ByTotal.Blocks.Coverage)) + " [B]",
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

// displayComparisonFromData displays comparison results from structured comparison data.
func displayComparisonFromData(comparison *compute.ComparisonData) {
	fmt.Printf("\n≡ Comparing against ref: %s [commit %s]\n",
		color.New(color.FgBlue).Sprint(comparison.Ref),
		color.New(color.FgHiBlack).Sprint(comparison.Commit),
	)

	if len(comparison.Results) == 0 {
		fmt.Println(" → No change")
		return
	}

	// Group results by type for organized display
	fileResults := make([]compute.ComparisonResult, 0)
	packageResults := make([]compute.ComparisonResult, 0)
	totalResults := make([]compute.ComparisonResult, 0)

	for _, result := range comparison.Results {
		switch result.Type {
		case "file":
			fileResults = append(fileResults, result)
		case "package":
			packageResults = append(packageResults, result)
		case "total":
			totalResults = append(totalResults, result)
		}
	}

	// Display file comparisons
	if len(fileResults) > 0 {
		fmt.Printf(" → By File\n")
		for _, result := range fileResults {
			if result.Delta.StatementsDelta != 0 {
				compareShowS()
				delta, _ := formatDelta(result.Delta.StatementsDelta)
				fmt.Printf("%s [%s]\n", result.Name, delta)
			}
			if result.Delta.BlocksDelta != 0 {
				compareShowB()
				delta, _ := formatDelta(result.Delta.BlocksDelta)
				fmt.Printf("%s [%s]\n", result.Name, delta)
			}
			fmt.Println()
		}
	}

	// Display package comparisons
	if len(packageResults) > 0 {
		fmt.Printf(" → By Package\n")
		for _, result := range packageResults {
			if result.Delta.StatementsDelta != 0 {
				compareShowS()
				delta, _ := formatDelta(result.Delta.StatementsDelta)
				fmt.Printf("%s [%s]\n", result.Name, delta)
			}
			if result.Delta.BlocksDelta != 0 {
				compareShowB()
				delta, _ := formatDelta(result.Delta.BlocksDelta)
				fmt.Printf("%s [%s]\n", result.Name, delta)
			}
			fmt.Println()
		}
	}

	// Display total comparisons
	if len(totalResults) > 0 {
		fmt.Printf(" → By Total\n")
		for _, result := range totalResults {
			if result.Delta.StatementsDelta != 0 {
				compareShowS()
				delta, _ := formatDelta(result.Delta.StatementsDelta)
				fmt.Printf("%s [%s]\n", result.Name, delta)
			}
			if result.Delta.BlocksDelta != 0 {
				compareShowB()
				delta, _ := formatDelta(result.Delta.BlocksDelta)
				fmt.Printf("%s [%s]\n", result.Name, delta)
			}
		}
	}
}

func compareShowS() {
	fmt.Printf("    [%s] ", color.New(color.FgCyan).Sprint("S"))
}

func compareShowB() {
	fmt.Printf("    [%s] ", color.New(color.FgHiMagenta).Sprint("B"))
}
