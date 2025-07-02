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
	for _, r := range results.Results {
		if !r.Failed {
			continue
		}

		if r.StatementPercentage < r.StatementThreshold {
			gap := r.StatementThreshold - r.StatementPercentage
			fmt.Fprintf(os.Stderr, " - %s [statement %% needs improvement of %s to meet threshold %s]\n",
				r.File,
				severityColor(r.StatementPercentage, r.StatementThreshold)(fmt.Sprintf("%.1f%%", gap)),
				color.New(color.FgBlue).Sprintf("%.1f%%", r.StatementThreshold),
			)
		}
		if r.BlockPercentage < cfg.BlockThreshold {
			gap := r.BlockThreshold - r.BlockPercentage
			fmt.Fprintf(os.Stderr, "   %s [block %% needs improvement of %s to meet threshold %s]\n",
				r.File,
				severityColor(r.BlockPercentage, r.BlockThreshold)(fmt.Sprintf("%.1f%%", gap)),
				color.New(color.FgBlue).Sprintf("%.1f%%", r.BlockThreshold),
			)
		}
	}

	percentTotalStatements := percent(results.TotalCoveredStatements, results.TotalStatements)
	if percentTotalStatements < cfg.StatementThreshold {
		gap := cfg.StatementThreshold - percentTotalStatements
		fmt.Fprintf(os.Stderr, " - total statement %% needs improvement of %s to meet threshold %s\n",
			severityColor(percentTotalStatements, cfg.StatementThreshold)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgBlue).Sprintf("%.1f%%", cfg.StatementThreshold),
		)
	}

	percentTotalBlocks := percent(results.TotalCoveredBlocks, results.TotalBlocks)
	if percentTotalBlocks < cfg.BlockThreshold {
		gap := cfg.BlockThreshold - percentTotalBlocks
		fmt.Fprintf(os.Stderr, " - total block %% needs improvement of %s to meet threshold %s\n",
			severityColor(percentTotalBlocks, cfg.BlockThreshold)(fmt.Sprintf("%.1f%%", gap)),
			color.New(color.FgBlue).Sprintf("%.1f%%", cfg.BlockThreshold),
		)
	}
}
