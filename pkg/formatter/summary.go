package formatter

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/config"
)

var (
	msgF = " [%s] %s [improvement of %s required to meet %s threshold]\n"
)

func renderSummary(hasFailure bool, results Results, cfg *config.Config) {
	if cfg.NoSummary {
		return
	}

	if !hasFailure {
		fmt.Println(color.New(color.FgGreen).Sprint("✔"), "All good")
		return
	}

	_, _ = fmt.Fprintln(os.Stderr, color.New(color.FgRed).Sprint("✘"), "Coverage check failed")
	renderByFile(results)
	renderByPackage(results)
	renderTotal(results)
}

func renderTotal(results Results) {
	if !results.ByTotal.Statements.Failed && !results.ByTotal.Blocks.Failed {
		return
	}

	_, _ = fmt.Fprintln(os.Stderr, "\nBy Total")
	totalExpect := results.ByTotal.Statements.Threshold
	percentTotalStatements := results.ByTotal.Statements.Percentage
	if percentTotalStatements < totalExpect {
		gap := totalExpect - percentTotalStatements
		_, _ = fmt.Fprintf(os.Stderr, msgF,
			color.New(color.FgCyan).Sprint("S"),
			"total",
			severityColor(percentTotalStatements, totalExpect)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgCyan).Sprintf("%.1f%%", totalExpect),
		)
	}

	totalExpect = results.ByTotal.Blocks.Threshold
	percentTotalBlocks := results.ByTotal.Blocks.Percentage
	if percentTotalBlocks < totalExpect {
		gap := totalExpect - percentTotalBlocks
		_, _ = fmt.Fprintf(os.Stderr, msgF,
			color.New(color.FgHiMagenta).Sprint("B"),
			"total",
			severityColor(percentTotalBlocks, totalExpect)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgHiMagenta).Sprintf("%.1f%%", totalExpect),
		)
	}
}

func renderByPackage(results Results) {
	bPrinted := false
	for _, r := range results.ByPackage {
		if !r.Failed {
			continue
		}

		if !bPrinted {
			_, _ = fmt.Fprintln(os.Stderr, "\nBy Package")
			bPrinted = true
		}

		renderBy(r, r.Package)
	}
}

func renderByFile(results Results) {
	bPrinted := false
	for _, r := range results.ByFile {
		if !r.Failed {
			continue
		}

		if !bPrinted {
			_, _ = fmt.Fprintln(os.Stderr, "\nBy File")
			bPrinted = true
		}

		renderBy(r, r.File)
	}
}

func renderBy[T HasBy](by T, item string) {
	r := by.GetBy()

	if r.StatementPercentage < r.StatementThreshold {
		gap := r.StatementThreshold - r.StatementPercentage
		_, _ = fmt.Fprintf(os.Stderr, msgF,
			color.New(color.FgCyan).Sprint("S"),
			item,
			severityColor(r.StatementPercentage, r.StatementThreshold)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgCyan).Sprintf("%.1f%%", r.StatementThreshold),
		)
	}

	if r.BlockPercentage < r.BlockThreshold {
		gap := r.BlockThreshold - r.BlockPercentage
		_, _ = fmt.Fprintf(os.Stderr, msgF,
			color.New(color.FgHiMagenta).Sprint("B"),
			item,
			severityColor(r.BlockPercentage, r.BlockThreshold)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgHiMagenta).Sprintf("%.1f%%", r.BlockThreshold),
		)
	}
}
