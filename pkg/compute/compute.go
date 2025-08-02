package compute

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/math"

	"golang.org/x/tools/cover"
)

// CollectResults collects all the details from a []*cover.Profile and returns Results.
func CollectResults(profiles []*cover.Profile, cfg *config.Config) (Results, bool) {
	normalizeNames(profiles, cfg)
	return collect(profiles, cfg)
}

func collectUncoveredLines(p *cover.Profile) string {
	var uncoveredLines []int

	// Collect all uncovered line numbers
	for _, b := range p.Blocks {
		if b.Count == 0 {
			// Add all lines in this uncovered block
			for line := b.StartLine; line <= b.EndLine; line++ {
				uncoveredLines = append(uncoveredLines, line)
			}
		}
	}

	if len(uncoveredLines) == 0 {
		return ""
	}

	// Remove duplicates and sort
	lineMap := make(map[int]bool)
	for _, line := range uncoveredLines {
		lineMap[line] = true
	}

	uniqueLines := make([]int, 0, len(lineMap))
	for line := range lineMap {
		uniqueLines = append(uniqueLines, line)
	}
	sort.Ints(uniqueLines)

	// Filter out excluded lines (empty, comments, closing braces)
	filteredLines := filterExcludedLines(uniqueLines, p.FileName)

	// Convert to ranges where possible
	return formatLineRanges(filteredLines)
}

// filterExcludedLines removes lines that should not be considered as uncovered:
// - Empty lines (only whitespace)
// - Comment lines (starting with //)
// - Lines that only contain closing braces }
func filterExcludedLines(lines []int, fileName string) []int {
	// Try to read the source file
	fileLines, err := readFileLines(fileName)
	if err != nil {
		// If we can't read the file, return all lines (fallback to original behavior)
		return lines
	}

	var filteredLines []int
	inBlockComment := false

	for _, lineNum := range lines {
		// Convert to 0-based index for slice access
		index := lineNum - 1
		if index < 0 || index >= len(fileLines) {
			// Line number out of range, include it
			filteredLines = append(filteredLines, lineNum)
			continue
		}

		line := fileLines[index]
		trimmedLine := strings.TrimSpace(line)

		// Handle block comment start/end
		lineHasBlockStart := strings.Contains(line, "/*")
		lineHasBlockEnd := strings.Contains(line, "*/")

		// Handle single-line block comments /* ... */
		if lineHasBlockStart && lineHasBlockEnd {
			// Remove the block comment part and check if there's remaining code
			commentStart := strings.Index(line, "/*")
			commentEnd := strings.Index(line, "*/") + 2

			// Create a line with the block comment removed
			beforeComment := line[:commentStart]
			afterComment := ""
			if commentEnd < len(line) {
				afterComment = line[commentEnd:]
			}
			lineWithoutBlockComment := beforeComment + afterComment
			trimmedWithoutBlock := strings.TrimSpace(lineWithoutBlockComment)

			// If there's still meaningful code after removing block comment, keep it
			if trimmedWithoutBlock != "" && !strings.HasPrefix(trimmedWithoutBlock, "//") && trimmedWithoutBlock != "}" {
				filteredLines = append(filteredLines, lineNum)
			}
			continue
		}

		// If this line ends a block comment, filter it and stop being in block comment
		if lineHasBlockEnd {
			inBlockComment = false
			continue
		}

		// Skip if we're currently in a block comment
		if inBlockComment {
			continue
		}

		// If this line starts a block comment, filter it and start being in block comment
		if lineHasBlockStart {
			inBlockComment = true
			continue
		}

		// Skip empty lines
		if trimmedLine == "" {
			continue
		}

		// Skip single-line comments
		if strings.HasPrefix(trimmedLine, "//") {
			continue
		}

		// Skip lines that only contain closing braces
		if trimmedLine == "}" {
			continue
		}

		// Include this line
		filteredLines = append(filteredLines, lineNum)
	}

	return filteredLines
}

// readFileLines reads a file and returns its lines
func readFileLines(fileName string) ([]string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

func formatLineRanges(lines []int) string {
	if len(lines) == 0 {
		return ""
	}

	var ranges []string
	start := lines[0]
	end := lines[0]

	for i := 1; i < len(lines); i++ {
		if lines[i] == end+1 {
			// Consecutive line, extend the range
			end = lines[i]
		} else {
			// Non-consecutive, finalize the current range and start a new one
			ranges = append(ranges, formatRange(start, end))
			start = lines[i]
			end = lines[i]
		}
	}

	// Add the final range
	ranges = append(ranges, formatRange(start, end))

	return strings.Join(ranges, ",")
}

func formatRange(start, end int) string {
	if start == end {
		return strconv.Itoa(start)
	}
	return strconv.Itoa(start) + "-" + strconv.Itoa(end)
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

		byFile := ByFile{
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
		}

		if !cfg.HideUncoveredLines {
			byFile.By.UncoveredLines = collectUncoveredLines(p)
		}

		results.ByFile = append(results.ByFile, byFile)

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
