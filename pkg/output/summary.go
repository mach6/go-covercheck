package output

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

var (
	msgF = "    [%s] %s [+%s required for %s threshold]\n"
)

func renderSummary(hasFailure bool, results compute.Results, cfg *config.Config, showedEmptyMessage bool) {
	if cfg.NoSummary {
		return
	}

	if !hasFailure && !showedEmptyMessage {
		fmt.Println(color.New(color.FgGreen).Sprint("✔"), "All good")
		return
	}
	
	if !hasFailure && showedEmptyMessage {
		// No summary message when we already showed empty results message
		return
	}

	_, _ = fmt.Println(color.New(color.FgRed).Sprint("✘"), "Coverage check failed")
	renderByFile(results)
	renderByPackage(results)
	renderTotal(results)
}

func renderTotal(results compute.Results) {
	if !results.ByTotal.Statements.Failed && !results.ByTotal.Blocks.Failed {
		return
	}

	_, _ = fmt.Println(" → By Total")
	totalExpect := results.ByTotal.Statements.Threshold
	percentTotalStatements := results.ByTotal.Statements.Percentage
	if percentTotalStatements < totalExpect {
		gap := totalExpect - percentTotalStatements
		_, _ = fmt.Printf(msgF,
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
		_, _ = fmt.Printf(msgF,
			color.New(color.FgHiMagenta).Sprint("B"),
			"total",
			severityColor(percentTotalBlocks, totalExpect)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgHiMagenta).Sprintf("%.1f%%", totalExpect),
		)
	}
}

func renderByPackage(results compute.Results) {
	bPrinted := false
	for _, r := range results.ByPackage {
		if !r.Failed {
			continue
		}

		if !bPrinted {
			_, _ = fmt.Println(" → By Package")
			bPrinted = true
		}

		renderBy(r, r.Package)
	}
}

func renderByFile(results compute.Results) {
	bPrinted := false
	for _, r := range results.ByFile {
		if !r.Failed {
			continue
		}

		if !bPrinted {
			_, _ = fmt.Println(" → By File")
			bPrinted = true
		}

		renderBy(r, r.File)
	}
}

func renderBy[T compute.HasBy](by T, item string) {
	r := by.GetBy()

	if r.StatementPercentage < r.StatementThreshold {
		gap := r.StatementThreshold - r.StatementPercentage
		_, _ = fmt.Printf(msgF,
			color.New(color.FgCyan).Sprint("S"),
			item,
			severityColor(r.StatementPercentage, r.StatementThreshold)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgCyan).Sprintf("%.1f%%", r.StatementThreshold),
		)
	}

	if r.BlockPercentage < r.BlockThreshold {
		gap := r.BlockThreshold - r.BlockPercentage
		_, _ = fmt.Printf(msgF,
			color.New(color.FgHiMagenta).Sprint("B"),
			item,
			severityColor(r.BlockPercentage, r.BlockThreshold)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgHiMagenta).Sprintf("%.1f%%", r.BlockThreshold),
		)
	}
}
