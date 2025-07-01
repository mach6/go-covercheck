package formatter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/config"
	"golang.org/x/tools/cover"
	"gopkg.in/yaml.v3"
)

// Result holds information for a single cover.Profile result.
type Result struct {
	File                string  `json:"file"                yaml:"file"`
	Statements          string  `json:"statements"          yaml:"statements"`
	Blocks              string  `json:"blocks"              yaml:"blocks"`
	StatementPercentage float64 `json:"statementPercentage" yaml:"statementPercentage"`
	BlockPercentage     float64 `json:"blockPercentage"     yaml:"blockPercentage"`
	StatementThreshold  float64 `json:"statementThreshold"  yaml:"statementThreshold"`
	BlockThreshold      float64 `json:"blockThreshold"      yaml:"blockThreshold"`
	Failed              bool    `json:"failed"              yaml:"failed"`
}

// Results holds information for all cover.Profile results.
type Results struct {
	Results                []Result `json:"results"                yaml:"results"`
	TotalStatements        int      `json:"totalStatements"        yaml:"totalStatements"`
	TotalBlocks            int      `json:"totalBlocks"            yaml:"totalBlocks"`
	TotalCoveredStatements int      `json:"totalCoveredStatements" yaml:"totalCoveredStatements"`
	TotalCoveredBlocks     int      `json:"totalCoveredBlocks"     yaml:"totalCoveredBlocks"`
}

// FormatAndReport creates and writes out formatted profile results and return true or false if there is/are coverage
// failure(s).
func FormatAndReport(
	profiles []*cover.Profile,
	cfg *config.Config,
) bool {
	normalizeNames(profiles)
	results, hasFailure := collect(profiles, cfg)

	switch cfg.Format {
	case config.FormatTable, config.FormatMD, config.FormatHTML, config.FormatCSV, config.FormatTSV:
		renderTable(results, cfg)
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

	return hasFailure
}

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

func collect(profiles []*cover.Profile, cfg *config.Config) (Results, bool) {
	hasFailure := false
	results := Results{
		Results: make([]Result, 0),
	}

	for _, p := range profiles {
		stmts, stmtHits, blocks, blockHits := 0, 0, 0, 0
		for _, b := range p.Blocks {
			stmts += b.NumStmt
			if b.Count > 0 {
				stmtHits += b.NumStmt
				blockHits++
			}
			blocks++
		}

		stmtPct := percent(stmtHits, stmts)
		blockPct := percent(blockHits, blocks)

		stmtThreshold := cfg.StatementThreshold
		if t, ok := cfg.PerFile[config.PerFileStatementThresholdSection][p.FileName]; ok {
			stmtThreshold = t
		}
		failed := stmtPct < stmtThreshold

		blockThreshold := cfg.BlockThreshold
		if t, ok := cfg.PerFile[config.PerFileBlockThresholdSection][p.FileName]; ok {
			blockThreshold = t
		}
		failed = failed || blockPct < blockThreshold

		if failed {
			hasFailure = true
		}

		results.Results = append(results.Results, Result{
			File:                p.FileName,
			Statements:          fmt.Sprintf("%d/%d", stmtHits, stmts),
			Blocks:              fmt.Sprintf("%d/%d", blockHits, blocks),
			StatementPercentage: stmtPct,
			StatementThreshold:  stmtThreshold,
			BlockPercentage:     blockPct,
			BlockThreshold:      blockThreshold,
			Failed:              failed,
		})

		results.TotalStatements += stmts
		results.TotalCoveredStatements += stmtHits
		results.TotalBlocks += blocks
		results.TotalCoveredBlocks += blockHits
	}

	sortResults(results.Results, cfg)

	return results, hasFailure ||
		percent(results.TotalCoveredStatements, results.TotalStatements) < cfg.StatementThreshold ||
		percent(results.TotalCoveredBlocks, results.TotalBlocks) < cfg.BlockThreshold
}

func renderTable(results Results, cfg *config.Config) {
	if cfg.NoTable {
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"File", "Statements", "Blocks", "Statement %", "Block %"})

	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "File", Align: text.AlignLeft, AlignFooter: text.AlignLeft},
		{Name: "Statements", Align: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Blocks", Align: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Statement %", Align: text.AlignRight, AlignFooter: text.AlignRight},
		{Name: "Block %", Align: text.AlignRight, AlignFooter: text.AlignRight},
	})

	for _, r := range results.Results {
		stmtColor := severityColor(r.StatementPercentage, r.StatementThreshold)
		blockColor := severityColor(r.BlockPercentage, r.BlockThreshold)

		t.AppendRow(table.Row{
			r.File,
			r.Statements,
			r.Blocks,
			stmtColor(fmt.Sprintf("%.1f", r.StatementPercentage)),
			blockColor(fmt.Sprintf("%.1f", r.BlockPercentage)),
		})
	}

	stmtTotalPct := percent(results.TotalCoveredStatements, results.TotalStatements)
	blockTotalPct := percent(results.TotalCoveredBlocks, results.TotalBlocks)
	stmtColor := severityColor(stmtTotalPct, cfg.StatementThreshold)
	blockColor := severityColor(blockTotalPct, cfg.BlockThreshold)

	t.AppendFooter(table.Row{
		text.Bold.Sprint("TOTAL"),
		text.Bold.Sprintf("%d/%d", results.TotalCoveredStatements, results.TotalStatements),
		text.Bold.Sprintf("%d/%d", results.TotalCoveredBlocks, results.TotalBlocks),
		stmtColor(text.Bold.Sprintf("%.1f", stmtTotalPct)),
		blockColor(text.Bold.Sprintf("%.1f", blockTotalPct)),
	})

	if cfg.Format == config.FormatTable {
		t.Render()
	}
	if cfg.Format == config.FormatMD {
		t.RenderMarkdown()
	}
	if cfg.Format == config.FormatHTML {
		t.RenderHTML()
	}
	if cfg.Format == config.FormatTSV {
		t.RenderTSV()
	}
	if cfg.Format == config.FormatCSV {
		t.RenderCSV()
	}
}

func percent(count, total int) float64 {
	if total == 0 {
		return 100.0 //nolint:mnd
	}
	return (float64(count) / float64(total)) * 100 //nolint:mnd
}

func severityColor(actual, goal float64) func(a ...interface{}) string {
	switch {
	case goal <= 0 && actual <= 0:
		return color.New(color.Reset).SprintFunc()
	case goal <= 0:
		return color.New(color.Reset).SprintFunc()
	}

	pct := (actual / goal) * 100 //nolint:mnd
	switch {
	case pct <= 50: //nolint:mnd
		return color.New(color.FgRed).SprintFunc()
	case pct <= 99: //nolint:mnd
		return color.New(color.FgYellow).SprintFunc()
	default:
		return color.New(color.FgGreen).SprintFunc()
	}
}

func sortResults(results []Result, cfg *config.Config) { //nolint:cyclop
	sort.Slice(results, func(i, j int) bool {
		sortByDesc := cfg.SortOrder == config.SortOrderDesc

		switch cfg.SortBy {
		case config.SortByStatementPercent:
			if sortByDesc {
				return results[i].StatementPercentage > results[j].StatementPercentage
			}
			return results[i].StatementPercentage < results[j].StatementPercentage
		case config.SortByBlockPercent:
			if sortByDesc {
				return results[i].BlockPercentage > results[j].BlockPercentage
			}
			return results[i].BlockPercentage < results[j].BlockPercentage
		case config.SortByStatements:
			if sortByDesc {
				return results[i].Statements > results[j].Statements
			}
			return results[i].Statements < results[j].Statements
		case config.SortOrderDefault:
			if sortByDesc {
				return results[i].Blocks > results[j].Blocks
			}
			return results[i].Blocks < results[j].Blocks
		default:
			if sortByDesc {
				return results[i].File > results[j].File
			}
			return results[i].File < results[j].File
		}
	})
}
