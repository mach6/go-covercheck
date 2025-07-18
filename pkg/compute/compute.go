package compute

import (
	"fmt"
	"path"
	"sort"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/math"

	"golang.org/x/tools/cover"
)

func CollectResults(profiles []*cover.Profile, cfg *config.Config) (Results, bool) {
	normalizeNames(profiles)
	return collect(profiles, cfg)
}

func collect(profiles []*cover.Profile, cfg *config.Config) (Results, bool) { //nolint:cyclop
	hasFailure := false
	results := Results{
		ByFile: make([]ByFile, 0),
		ByTotal: Totals{
			Statements: TotalStatements{},
			Blocks:     TotalBlocks{},
		},
		ByPackage: make([]ByPackage, 0),
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

		stmtPct := math.Percent(stmtHits, stmts)
		blockPct := math.Percent(blockHits, blocks)

		stmtThreshold := cfg.StatementThreshold
		if t, ok := cfg.PerFile.Statements[p.FileName]; ok {
			stmtThreshold = t
		}
		failed := stmtPct < stmtThreshold

		blockThreshold := cfg.BlockThreshold
		if t, ok := cfg.PerFile.Blocks[p.FileName]; ok {
			blockThreshold = t
		}
		failed = failed || blockPct < blockThreshold

		if failed {
			hasFailure = true
		}

		results.ByFile = append(results.ByFile, ByFile{
			File: p.FileName,
			By: By{
				Statements:          fmt.Sprintf("%d/%d", stmtHits, stmts),
				Blocks:              fmt.Sprintf("%d/%d", blockHits, blocks),
				StatementPercentage: stmtPct,
				StatementThreshold:  stmtThreshold,
				BlockPercentage:     blockPct,
				BlockThreshold:      blockThreshold,
				Failed:              failed,
				stmtHits:            stmtHits,
				blockHits:           blockHits,
				stmts:               stmts,
				blocks:              blocks,
			},
		})

		results.ByTotal.Statements.totalStatements += stmts
		results.ByTotal.Statements.totalCoveredStatements += stmtHits
		results.ByTotal.Blocks.totalBlocks += blocks
		results.ByTotal.Blocks.totalCoveredBlocks += blockHits
	}

	sortFileResults(results.ByFile, cfg)
	hasPackageFailure := collectPackageResults(&results, cfg)
	sortPackageResults(results.ByPackage, cfg)
	setTotals(&results, cfg)

	return results, hasFailure || hasPackageFailure ||
		results.ByTotal.Statements.Failed ||
		results.ByTotal.Blocks.Failed
}

func collectPackageResults(results *Results, cfg *config.Config) bool {
	working := make(map[string]ByPackage)
	for _, v := range results.ByFile {
		p := ByPackage{
			Package: path.Dir(v.File),
		}
		if w, exists := working[path.Dir(v.File)]; exists {
			p = w
		}
		p.stmtHits += v.stmtHits
		p.blockHits += v.blockHits
		p.blocks += v.blocks
		p.stmts += v.stmts
		working[path.Dir(v.File)] = p
	}

	hasFailed := false
	for _, v := range working {
		v.Statements = fmt.Sprintf("%d/%d", v.stmtHits, v.stmts)
		v.Blocks = fmt.Sprintf("%d/%d", v.blockHits, v.blocks)
		v.StatementPercentage = math.Percent(v.stmtHits, v.stmts)
		v.BlockPercentage = math.Percent(v.blockHits, v.blocks)

		v.StatementThreshold = cfg.StatementThreshold
		if t, ok := cfg.PerPackage.Statements[v.Package]; ok {
			v.StatementThreshold = t
		}
		v.Failed = v.StatementPercentage < v.StatementThreshold

		v.BlockThreshold = cfg.BlockThreshold
		if t, ok := cfg.PerPackage.Blocks[v.Package]; ok {
			v.BlockThreshold = t
		}
		v.Failed = v.Failed || v.BlockPercentage < v.BlockThreshold

		if v.Failed {
			hasFailed = true
		}
		results.ByPackage = append(results.ByPackage, v)
	}
	return hasFailed
}

func setTotals(results *Results, cfg *config.Config) {
	results.ByTotal.Statements.Threshold = cfg.Total[config.StatementsSection]
	results.ByTotal.Statements.Coverage =
		fmt.Sprintf("%d/%d", results.ByTotal.Statements.totalCoveredStatements,
			results.ByTotal.Statements.totalStatements)
	results.ByTotal.Statements.Percentage = math.Percent(results.ByTotal.Statements.totalCoveredStatements,
		results.ByTotal.Statements.totalStatements)
	results.ByTotal.Statements.Failed = results.ByTotal.Statements.Percentage < results.ByTotal.Statements.Threshold

	results.ByTotal.Blocks.Threshold = cfg.Total[config.BlocksSection]
	results.ByTotal.Blocks.Coverage =
		fmt.Sprintf("%d/%d", results.ByTotal.Blocks.totalCoveredBlocks,
			results.ByTotal.Blocks.totalBlocks)
	results.ByTotal.Blocks.Percentage = math.Percent(results.ByTotal.Blocks.totalCoveredBlocks,
		results.ByTotal.Blocks.totalBlocks)
	results.ByTotal.Blocks.Failed = results.ByTotal.Blocks.Percentage < results.ByTotal.Blocks.Threshold
}

func sortBy[T HasBy](results []T, cfg *config.Config) {
	sort.Slice(results, func(i, j int) bool {
		sortByDesc := cfg.SortOrder == config.SortOrderDesc

		byI := results[i].GetBy()
		byJ := results[j].GetBy()

		switch cfg.SortBy {
		case config.SortByStatementPercent:
			if sortByDesc {
				return byI.StatementPercentage > byJ.StatementPercentage
			}
			return byI.StatementPercentage < byJ.StatementPercentage
		case config.SortByBlockPercent:
			if sortByDesc {
				return byI.BlockPercentage > byJ.BlockPercentage
			}
			return byI.BlockPercentage < byJ.BlockPercentage
		case config.SortByStatements:
			if sortByDesc {
				return byI.stmtHits > byJ.stmtHits
			}
			return byI.stmtHits < byJ.stmtHits
		case config.SortByBlocks:
			if sortByDesc {
				return byI.blockHits > byJ.blockHits
			}
			return byI.blockHits < byJ.blockHits
		default:
			return false
		}
	})
}

func sortFileResults(results []ByFile, cfg *config.Config) {
	switch cfg.SortBy {
	case config.SortByStatementPercent, config.SortByBlockPercent, config.SortByStatements, config.SortByBlocks:
		sortBy(results, cfg)
		return
	default:
		// called when sort-by == file
		sort.Slice(results, func(i, j int) bool {
			sortByDesc := cfg.SortOrder == config.SortOrderDesc
			if sortByDesc {
				return results[i].File > results[j].File
			}
			return results[i].File < results[j].File
		})
	}
}

func sortPackageResults(results []ByPackage, cfg *config.Config) {
	switch cfg.SortBy {
	case config.SortByStatementPercent, config.SortByBlockPercent, config.SortByStatements, config.SortByBlocks:
		sortBy(results, cfg)
		return
	default:
		// called when sort-by == file
		sort.Slice(results, func(i, j int) bool {
			sortByDesc := cfg.SortOrder == config.SortOrderDesc
			if sortByDesc {
				return results[i].Package > results[j].Package
			}
			return results[i].Package < results[j].Package
		})
	}
}
