package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
	"gopkg.in/yaml.v3"
)

// FormatAndReport writes out formatted profile results.
func FormatAndReport(results compute.Results, cfg *config.Config, hasFailure bool) {
	switch cfg.Format {
	case config.FormatTable, config.FormatMD, config.FormatHTML, config.FormatCSV, config.FormatTSV:
		renderTable(results, cfg)
		_ = os.Stdout.Sync()
		renderSummary(hasFailure, results, cfg)
	case config.FormatJSON:
		if cfg.NoColor {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(results); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			s, err := prettyjson.Marshal(results)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println(string(s))
		}
	case config.FormatYAML:
		if cfg.NoColor {
			_ = yaml.NewEncoder(os.Stdout).Encode(results)
		} else {
			y, _ := yaml.Marshal(results)
			yamlColor(y)
		}
	default:
		fmt.Fprintln(os.Stderr, color.RedString("Unsupported format: %s", cfg.Format))
	}
}
