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
