package compute

import (
	"fmt"
	"path"
	"sort"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/math"

	"golang.org/x/tools/cover"
)

// calculateLineCoverage calculates line coverage for a profile.
// It returns the total lines and covered lines count.
func calculateLineCoverage(p *cover.Profile) (int, int) {
	allLines := make(map[int]bool)     // track all lines
	coveredLines := make(map[int]bool) // track covered lines
	
	for _, b := range p.Blocks {
		// Add all lines in this block to allLines
		for line := b.StartLine; line <= b.EndLine; line++ {
			allLines[line] = true
			// If this block is covered (count > 0), mark lines as covered
			if b.Count > 0 {
				coveredLines[line] = true
			}
		}
	}
	
	return len(allLines), len(coveredLines)
}

// CollectResults collects all the details from a []*cover.Profile and returns Results.
func CollectResults(profiles []*cover.Profile, cfg *config.Config) (Results, bool) {
	normalizeNames(profiles, cfg)
	return collect(profiles, cfg)
}

func collect(profiles []*cover.Profile, cfg *config.Config) (Results, bool) { //nolint:cyclop
	hasFailure := false
	results := Results{
		ByFile: make([]ByFile, 0),
		ByTotal: Totals{
			Statements: TotalStatements{},
			Blocks:     TotalBlocks{},
			Lines:      TotalLines{},
		},
		ByPackage: make([]ByPackage, 0),
	}

	for _, p := range profiles {
		stmts, stmtHits, blocks, blockHits := 0, 0, 0, 0
		lines, lineHits := calculateLineCoverage(p)
		
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
		linePct := math.Percent(lineHits, lines)

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

		lineThreshold := cfg.LineThreshold
		if t, ok := cfg.PerFile.Lines[p.FileName]; ok {
			lineThreshold = t
		}
		failed = failed || linePct < lineThreshold

		if failed {
			hasFailure = true
		}

		results.ByFile = append(results.ByFile, ByFile{
			File: p.FileName,
			By: By{
				Statements:          fmt.Sprintf("%d/%d", stmtHits, stmts),
				Blocks:              fmt.Sprintf("%d/%d", blockHits, blocks),
				Lines:               fmt.Sprintf("%d/%d", lineHits, lines),
				StatementPercentage: stmtPct,
				StatementThreshold:  stmtThreshold,
				BlockPercentage:     blockPct,
				BlockThreshold:      blockThreshold,
				LinePercentage:      linePct,
				LineThreshold:       lineThreshold,
				Failed:              failed,
				stmtHits:            stmtHits,
				blockHits:           blockHits,
				lineHits:            lineHits,
				stmts:               stmts,
				blocks:              blocks,
				lines:               lines,
			},
		})

		results.ByTotal.Statements.totalStatements += stmts
		results.ByTotal.Statements.totalCoveredStatements += stmtHits
		results.ByTotal.Blocks.totalBlocks += blocks
		results.ByTotal.Blocks.totalCoveredBlocks += blockHits
		results.ByTotal.Lines.totalLines += lines
		results.ByTotal.Lines.totalCoveredLines += lineHits
	}

	sortFileResults(results.ByFile, cfg)
	hasPackageFailure := collectPackageResults(&results, cfg)
	sortPackageResults(results.ByPackage, cfg)
	setTotals(&results, cfg)

	return results, hasFailure || hasPackageFailure ||
		results.ByTotal.Statements.Failed ||
		results.ByTotal.Blocks.Failed ||
		results.ByTotal.Lines.Failed
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
		p.lineHits += v.lineHits
		p.blocks += v.blocks
		p.stmts += v.stmts
		p.lines += v.lines
		working[path.Dir(v.File)] = p
	}

	hasFailed := false
	for _, v := range working {
		v.Statements = fmt.Sprintf("%d/%d", v.stmtHits, v.stmts)
		v.Blocks = fmt.Sprintf("%d/%d", v.blockHits, v.blocks)
		v.Lines = fmt.Sprintf("%d/%d", v.lineHits, v.lines)
		v.StatementPercentage = math.Percent(v.stmtHits, v.stmts)
		v.BlockPercentage = math.Percent(v.blockHits, v.blocks)
		v.LinePercentage = math.Percent(v.lineHits, v.lines)

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

		v.LineThreshold = cfg.LineThreshold
		if t, ok := cfg.PerPackage.Lines[v.Package]; ok {
			v.LineThreshold = t
		}
		v.Failed = v.Failed || v.LinePercentage < v.LineThreshold

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

	results.ByTotal.Lines.Threshold = cfg.Total[config.LinesSection]
	results.ByTotal.Lines.Coverage =
		fmt.Sprintf("%d/%d", results.ByTotal.Lines.totalCoveredLines,
			results.ByTotal.Lines.totalLines)
	results.ByTotal.Lines.Percentage = math.Percent(results.ByTotal.Lines.totalCoveredLines,
		results.ByTotal.Lines.totalLines)
	results.ByTotal.Lines.Failed = results.ByTotal.Lines.Percentage < results.ByTotal.Lines.Threshold
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
		case config.SortByLinePercent:
			if sortByDesc {
				return byI.LinePercentage > byJ.LinePercentage
			}
			return byI.LinePercentage < byJ.LinePercentage
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
		case config.SortByLines:
			if sortByDesc {
				return byI.lineHits > byJ.lineHits
			}
			return byI.lineHits < byJ.lineHits
		default:
			return false
		}
	})
}

func sortFileResults(results []ByFile, cfg *config.Config) {
	switch cfg.SortBy {
	case config.SortByStatementPercent, config.SortByBlockPercent, config.SortByLinePercent, config.SortByStatements, config.SortByBlocks, config.SortByLines:
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
	case config.SortByStatementPercent, config.SortByBlockPercent, config.SortByLinePercent, config.SortByStatements, config.SortByBlocks, config.SortByLines:
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
