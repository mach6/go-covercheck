package formatter

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/config"
)

func renderSummary(hasFailure bool, results Results, cfg *config.Config) {
	if cfg.NoSummary {
		return
	}

	if !hasFailure {
		fmt.Println(color.New(color.FgGreen).Sprint("✔"), "All good")
		return
	}

	fmt.Fprintln(os.Stderr, color.New(color.FgRed).Sprint("✘"), "Coverage check failed")
	msgF := " [%s] %s [improvement of %s required to meet %s threshold]\n"

	for _, r := range results.Results {
		if !r.Failed {
			continue
		}

		if r.StatementPercentage < r.StatementThreshold {
			gap := r.StatementThreshold - r.StatementPercentage
			fmt.Fprintf(os.Stderr, msgF,
				color.New(color.FgCyan).Sprint("S"),
				r.File,
				severityColor(r.StatementPercentage, r.StatementThreshold)(fmt.Sprintf("%.1f%%", gap)),
				color.New(color.FgCyan).Sprintf("%.1f%%", r.StatementThreshold),
			)
		}

		if r.BlockPercentage < cfg.BlockThreshold {
			gap := r.BlockThreshold - r.BlockPercentage
			fmt.Fprintf(os.Stderr, msgF,
				color.New(color.FgHiMagenta).Sprint("B"),
				r.File,
				severityColor(r.BlockPercentage, r.BlockThreshold)(fmt.Sprintf("%.1f%%", gap)),
				color.New(color.FgHiMagenta).Sprintf("%.1f%%", r.BlockThreshold),
			)
		}
	}

	percentTotalStatements := percent(results.TotalCoveredStatements, results.TotalStatements)
	if percentTotalStatements < cfg.StatementThreshold {
		gap := cfg.StatementThreshold - percentTotalStatements
		fmt.Fprintf(os.Stderr, msgF,
			color.New(color.FgCyan).Sprint("S"),
			"total statements",
			severityColor(percentTotalStatements, cfg.StatementThreshold)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgCyan).Sprintf("%.1f%%", cfg.StatementThreshold),
		)
	}

	percentTotalBlocks := percent(results.TotalCoveredBlocks, results.TotalBlocks)
	if percentTotalBlocks < cfg.BlockThreshold {
		gap := cfg.BlockThreshold - percentTotalBlocks
		fmt.Fprintf(os.Stderr, msgF,
			color.New(color.FgHiMagenta).Sprint("B"),
			"total blocks",
			severityColor(percentTotalBlocks, cfg.BlockThreshold)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgHiMagenta).Sprintf("%.1f%%", cfg.BlockThreshold),
		)
	}
}
