package formatter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
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
