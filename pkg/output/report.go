package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
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

// report is the JSON and YAML document: the results, plus the comparison when one was requested.
type report struct {
	compute.Results `yaml:",inline"`
	Comparison      *compute.Comparison `json:"comparison,omitempty" yaml:"comparison,omitempty"`
}

// FormatAndReport writes out formatted profile results.
func FormatAndReport(results compute.Results, cfg *config.Config, hasFailure bool) {
	FormatAndReportWithComparison(results, nil, cfg, hasFailure)
}

// FormatAndReportWithComparison writes out formatted profile results and, when comparison is not nil,
// the comparison against history. JSON and YAML include it as a "comparison" object.
// Other formats print it as text after the report.
func FormatAndReportWithComparison(results compute.Results, comparison *compute.Comparison, cfg *config.Config,
	hasFailure bool,
) {
	doc := report{Results: results, Comparison: comparison}
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
		if comparison != nil {
			printComparison(comparison)
		}
	case config.FormatJSON:
		if cfg.NoColor {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			err := enc.Encode(doc)
			bailOnError(err)
		} else {
			jsonString, err := json.MarshalIndent(doc, "", "  ")
			bailOnError(err)
			fmt.Println(highlightJSONSyntax(string(jsonString), cfg))
		}
	case config.FormatYAML:
		if cfg.NoColor {
			err := yaml.NewEncoder(os.Stdout).Encode(doc)
			bailOnError(err)
		} else {
			yamlData, err := yaml.Marshal(doc)
			bailOnError(err)
			fmt.Println(highlightYAMLSyntax(string(yamlData), cfg))
		}
	default:
		bailOnError(errors.New(color.RedString("Unsupported format: %s", cfg.Format)))
	}
}
