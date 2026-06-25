package compute

import (
	"fmt"
	"path"
	"sort"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/lines"
	"github.com/mach6/go-covercheck/pkg/math"

	"golang.org/x/tools/cover"
)

// CollectResults collects all the details from a []*cover.Profile and returns
// Results. Call NormalizeNames(profiles, cfg) beforehand if module-relative
// filenames are desired in the output; CollectResults does not mutate
// profile.FileName itself.
func CollectResults(profiles []*cover.Profile, cfg *config.Config) (Results, bool) {
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
			Functions:  TotalFunctions{},
		},
		ByPackage: make([]ByPackage, 0),
	}

	for _, p := range profiles {
		stmts, stmtHits, blocks, blockHits := 0, 0, 0, 0
		// Collect blocks once per profile and reuse for line coverage and
		// uncovered-line formatting; both walks read/parse the source file.
		collectedBlocks := lines.CollectBlocks(p)
		linesCount, lineHits := lines.CoverageFromBlocks(collectedBlocks)

		for _, b := range p.Blocks {
			stmts += b.NumStmt
			if b.Count > 0 {
				stmtHits += b.NumStmt
				blockHits++
			}
			blocks++
		}

		// Calculate function coverage
		functions, functionHits, err := GetFunctionCoverageForFile(p.FileName, p.Blocks)
		if err != nil {
			// If we can't get function coverage, default to 0
			functions, functionHits = 0, 0
		}

		stmtPct := math.Percent(stmtHits, stmts)
		blockPct := math.Percent(blockHits, blocks)
		linePct := math.Percent(lineHits, linesCount)
		functionPct := math.Percent(functionHits, functions)

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

		functionThreshold := cfg.FunctionThreshold
		if t, ok := cfg.PerFile.Functions[p.FileName]; ok {
			functionThreshold = t
		}
		failed = failed || functionPct < functionThreshold

		if failed {
			hasFailure = true
		}

		byFile := ByFile{
			File: p.FileName,
			By: By{
				Statements:          fmt.Sprintf("%d/%d", stmtHits, stmts),
				Blocks:              fmt.Sprintf("%d/%d", blockHits, blocks),
				Lines:               fmt.Sprintf("%d/%d", lineHits, linesCount),
				Functions:           fmt.Sprintf("%d/%d", functionHits, functions),
				StatementPercentage: stmtPct,
				StatementThreshold:  stmtThreshold,
				BlockPercentage:     blockPct,
				BlockThreshold:      blockThreshold,
				LinePercentage:      linePct,
				LineThreshold:       lineThreshold,
				FunctionPercentage:  functionPct,
				FunctionThreshold:   functionThreshold,
				Failed:              failed,
				stmtHits:            stmtHits,
				blockHits:           blockHits,
				lineHits:            lineHits,
				stmts:               stmts,
				blocks:              blocks,
				lines:               linesCount,
				functions:           functions,
				functionHits:        functionHits,
			},
		}

		if !cfg.NoUncoveredLines {
			byFile.UncoveredLines = lines.FormatUncoveredFromBlocks(collectedBlocks)
		}

		results.ByFile = append(results.ByFile, byFile)

		results.ByTotal.Statements.totalStatements += stmts
		results.ByTotal.Statements.totalCoveredStatements += stmtHits
		results.ByTotal.Blocks.totalBlocks += blocks
		results.ByTotal.Blocks.totalCoveredBlocks += blockHits
		results.ByTotal.Lines.totalLines += linesCount
		results.ByTotal.Lines.totalCoveredLines += lineHits
		results.ByTotal.Functions.totalFunctions += functions
		results.ByTotal.Functions.totalCoveredFunctions += functionHits
	}

	sortFileResults(results.ByFile, cfg)
	hasPackageFailure := collectPackageResults(&results, cfg)
	sortPackageResults(results.ByPackage, cfg)
	setTotals(&results, cfg)

	return results, hasFailure || hasPackageFailure ||
		results.ByTotal.Statements.Failed ||
		results.ByTotal.Blocks.Failed ||
		results.ByTotal.Lines.Failed ||
		results.ByTotal.Functions.Failed
}

func applyPackageThresholds(v *ByPackage, cfg *config.Config) {
	v.StatementThreshold = cfg.StatementThreshold
	if t, ok := cfg.PerPackage.Statements[v.Package]; ok {
		v.StatementThreshold = t
	}

	v.BlockThreshold = cfg.BlockThreshold
	if t, ok := cfg.PerPackage.Blocks[v.Package]; ok {
		v.BlockThreshold = t
	}

	v.LineThreshold = cfg.LineThreshold
	if t, ok := cfg.PerPackage.Lines[v.Package]; ok {
		v.LineThreshold = t
	}

	v.FunctionThreshold = cfg.FunctionThreshold
	if t, ok := cfg.PerPackage.Functions[v.Package]; ok {
		v.FunctionThreshold = t
	}

	v.Failed = v.StatementPercentage < v.StatementThreshold ||
		v.BlockPercentage < v.BlockThreshold ||
		v.LinePercentage < v.LineThreshold ||
		v.FunctionPercentage < v.FunctionThreshold
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
		p.functions += v.functions
		p.functionHits += v.functionHits
		working[path.Dir(v.File)] = p
	}

	hasFailed := false
	for _, v := range working {
		v.Statements = fmt.Sprintf("%d/%d", v.stmtHits, v.stmts)
		v.Blocks = fmt.Sprintf("%d/%d", v.blockHits, v.blocks)
		v.Lines = fmt.Sprintf("%d/%d", v.lineHits, v.lines)
		v.Functions = fmt.Sprintf("%d/%d", v.functionHits, v.functions)
		v.StatementPercentage = math.Percent(v.stmtHits, v.stmts)
		v.BlockPercentage = math.Percent(v.blockHits, v.blocks)
		v.LinePercentage = math.Percent(v.lineHits, v.lines)
		v.FunctionPercentage = math.Percent(v.functionHits, v.functions)

		applyPackageThresholds(&v, cfg)

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

	results.ByTotal.Functions.Threshold = cfg.Total[config.FunctionsSection]
	results.ByTotal.Functions.Coverage =
		fmt.Sprintf("%d/%d", results.ByTotal.Functions.totalCoveredFunctions,
			results.ByTotal.Functions.totalFunctions)
	results.ByTotal.Functions.Percentage = math.Percent(results.ByTotal.Functions.totalCoveredFunctions,
		results.ByTotal.Functions.totalFunctions)
	results.ByTotal.Functions.Failed = results.ByTotal.Functions.Percentage < results.ByTotal.Functions.Threshold
}

// sortKey returns the numeric value of the configured sort field for a row.
// The returned float covers both percentage and integer fields.
func sortKey(by *By, sortBy string) (float64, bool) {
	switch sortBy {
	case config.SortByStatementPercent:
		return by.StatementPercentage, true
	case config.SortByBlockPercent:
		return by.BlockPercentage, true
	case config.SortByLinePercent:
		return by.LinePercentage, true
	case config.SortByFunctionPercent:
		return by.FunctionPercentage, true
	case config.SortByStatements:
		return float64(by.stmtHits), true
	case config.SortByBlocks:
		return float64(by.blockHits), true
	case config.SortByLines:
		return float64(by.lineHits), true
	case config.SortByFunctions:
		return float64(by.functionHits), true
	default:
		return 0, false
	}
}

func sortBy[T HasBy](results []T, cfg *config.Config) {
	desc := cfg.SortOrder == config.SortOrderDesc
	sort.Slice(results, func(i, j int) bool {
		byI := results[i].GetBy()
		byJ := results[j].GetBy()
		vi, ok := sortKey(&byI, cfg.SortBy)
		if !ok {
			return false
		}
		vj, _ := sortKey(&byJ, cfg.SortBy)
		if desc {
			return vi > vj
		}
		return vi < vj
	})
}

func sortFileResults(results []ByFile, cfg *config.Config) {
	switch cfg.SortBy {
	case config.SortByStatementPercent, config.SortByBlockPercent, config.SortByLinePercent, config.SortByFunctionPercent,
		config.SortByStatements, config.SortByBlocks, config.SortByLines, config.SortByFunctions:
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
	case config.SortByStatementPercent, config.SortByBlockPercent, config.SortByLinePercent, config.SortByFunctionPercent,
		config.SortByStatements, config.SortByBlocks, config.SortByLines, config.SortByFunctions:
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
