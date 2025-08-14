package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"gopkg.in/yaml.v3"
)

func bailOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// PrintDiffWarning prints a warning message when git diff operations fail.
func PrintDiffWarning(err error, cfg *config.Config) {
	// Don't print warnings in JSON/YAML mode as they would contaminate the output
	if cfg.Format == config.FormatJSON || cfg.Format == config.FormatYAML {
		return
	}
	fmt.Printf("Warning: Failed to get changed files for diff mode: %v\n", err)
	fmt.Println("Falling back to checking all files.")
}

// PrintNoDiffChanges prints a message when no files have changed in diff mode.
func PrintNoDiffChanges(cfg *config.Config) {
	// Don't print messages in JSON/YAML mode as they would contaminate the output
	if cfg.Format == config.FormatJSON || cfg.Format == config.FormatYAML {
		return
	}
	fmt.Println("No files changed in diff. No coverage to check.")
}

// PrintDiffModeInfo prints information about how many files are being checked in diff mode.
func PrintDiffModeInfo(changedCount, totalCount int, cfg *config.Config) {
	// Don't print info messages in JSON/YAML mode as they would contaminate the output
	if cfg.Format == config.FormatJSON || cfg.Format == config.FormatYAML {
		return
	}
	fmt.Printf("Diff mode: Checking coverage for %d changed files (out of %d total files)\n",
		changedCount, totalCount)
}

// isEmptyResults checks if the results contain no coverage data.
func isEmptyResults(results compute.Results) bool {
	return len(results.ByFile) == 0 && len(results.ByPackage) == 0 &&
		results.ByTotal.Statements.Coverage == "0/0" &&
		results.ByTotal.Blocks.Coverage == "0/0"
}

// FormatAndReport writes out formatted profile results.
func FormatAndReport(results compute.Results, cfg *config.Config, hasFailure bool) {
	isEmpty := isEmptyResults(results)
	switch cfg.Format {
	case config.FormatTable, config.FormatMD, config.FormatHTML, config.FormatCSV, config.FormatTSV:
		if isEmpty {
			fmt.Println(color.New(color.FgYellow).Sprint("⚠"), "No coverage results to display")
		} else {
			renderTable(results, cfg)
			_ = os.Stdout.Sync()
			renderSummary(hasFailure, results, cfg)
		}
	case config.FormatJSON:
		if cfg.NoColor {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			err := enc.Encode(results)
			bailOnError(err)
		} else {
			s, err := prettyjson.Marshal(results)
			bailOnError(err)
			fmt.Println(string(s))
		}
	case config.FormatYAML:
		if cfg.NoColor {
			err := yaml.NewEncoder(os.Stdout).Encode(results)
			bailOnError(err)
		} else {
			y, err := yaml.Marshal(results)
			bailOnError(err)
			yamlColor(y)
		}
	default:
		bailOnError(errors.New(color.RedString("Unsupported format: %s", cfg.Format)))
	}
}
