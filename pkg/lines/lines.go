// Package lines provides functionality for analyzing and manipulating source code lines,
// particularly for identifying uncovered lines in Go code coverage reports. It offers
// features to filter out irrelevant lines like comments and empty lines, and to format
// line ranges for concise representation of uncovered code sections.
package lines

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/cover"
)

// Line represents a single line of source code.
type Line struct {
	// LineNumber is the line number in the source file.
	LineNumber int
	// Content is the actual source code on that line.
	Content string
	// IsFiltered indicates whether the line should be skipped in the output.
	IsFiltered bool
	// Hits is the number of times this line was executed.
	Hits int
}

// Block pairs a coverage profile block with the per-line detail for that block.
type Block struct {
	cover.ProfileBlock
	Lines []Line
}

// IsCovered reports whether the block was hit at least once.
func (b *Block) IsCovered() bool {
	return b.Count > 0
}

// ContextLines returns the block's lines plus contextSize lines of surrounding context
// drawn from allBlocks. The result is sorted by line number with duplicates merged
// (cover profile blocks can overlap on the same line); when a line appears in
// multiple blocks the highest hit count wins so a covered line isn't rendered as
// uncovered just because an overlapping uncovered block reached it first.
func (b *Block) ContextLines(allBlocks []Block, contextSize int) []Line {
	return ContextLinesForRange(allBlocks, b.StartLine, b.EndLine, contextSize)
}

// SourceLinesInRange returns one Line per source line in
// [start-contextSize, end+contextSize] (clamped to the file's extent),
// pulling content directly from the source file instead of the profile
// blocks. Line.Hits carries the highest Count of any profile block covering
// that line number (0 if no block references it), and Line.IsFiltered comes
// from shouldSkipLine plus the file-level block-comment scan. This is what
// --inspect uses to render merged hunks, where lines in between profile
// blocks (blanks, comments, bare braces) would otherwise be missing.
func SourceLinesInRange(p *cover.Profile, start, end, contextSize int) []Line {
	sourceLines, _ := ReadSourceFile(p.FileName)
	sourceAvailable := len(sourceLines) > 0
	rangeStart, rangeEnd := clampRange(start, end, contextSize, len(sourceLines))
	hits := hitsByLine(p.Blocks, rangeStart, rangeEnd)

	out := make([]Line, 0, rangeEnd-rangeStart+1)
	for ln := rangeStart; ln <= rangeEnd; ln++ {
		out = append(out, buildSourceLine(ln, sourceLines, sourceAvailable, hits))
	}
	return out
}

func clampRange(start, end, contextSize, sourceLen int) (int, int) {
	rangeStart := start - contextSize
	if rangeStart < 1 {
		rangeStart = 1
	}
	rangeEnd := end + contextSize
	if sourceLen > 0 && rangeEnd > sourceLen {
		rangeEnd = sourceLen
	}
	return rangeStart, rangeEnd
}

func hitsByLine(blocks []cover.ProfileBlock, rangeStart, rangeEnd int) map[int]int {
	hits := make(map[int]int)
	for _, b := range blocks {
		for ln := b.StartLine; ln <= b.EndLine; ln++ {
			if ln < rangeStart || ln > rangeEnd {
				continue
			}
			if existing, ok := hits[ln]; !ok || b.Count > existing {
				hits[ln] = b.Count
			}
		}
	}
	return hits
}

func buildSourceLine(ln int, sourceLines []string, sourceAvailable bool, hits map[int]int) Line {
	entry := Line{LineNumber: ln, Hits: hits[ln]}
	idx := ln - 1
	switch {
	case idx >= 0 && idx < len(sourceLines):
		entry.Content = sourceLines[idx]
		entry.IsFiltered = shouldSkipLine(sourceLines[idx])
	case sourceAvailable:
		// Source was readable but the block references a line past EOF;
		// filter so --inspect doesn't emit an empty hunk.
		entry.IsFiltered = true
	}
	// Source couldn't be read at all: leave IsFiltered false so the line
	// renders with its real gutter (red '-' for uncovered, green for
	// covered) instead of silently collapsing to the neutral gutter.
	return entry
}

// ContextLinesForRange is like Block.ContextLines but operates on an arbitrary
// [start, end] line range, so callers merging adjacent uncovered blocks can
// build one hunk from the combined range.
func ContextLinesForRange(allBlocks []Block, start, end, contextSize int) []Line {
	startLine := start - contextSize
	if startLine < 1 {
		startLine = 1
	}
	endLine := end + contextSize

	merged := make(map[int]Line)
	for _, block := range allBlocks {
		for _, l := range block.Lines {
			if l.LineNumber < startLine || l.LineNumber > endLine {
				continue
			}
			if existing, ok := merged[l.LineNumber]; ok && existing.Hits >= l.Hits {
				continue
			}
			merged[l.LineNumber] = l
		}
	}

	out := make([]Line, 0, len(merged))
	for _, l := range merged {
		out = append(out, l)
	}
	slices.SortFunc(out, func(a, b Line) int { return a.LineNumber - b.LineNumber })
	return out
}

// maxSourceLineBytes is the upper bound on a single source line read from disk.
// The default bufio.Scanner buffer tops out at 64 KiB, which trips ErrTooLong on
// generated or minified files. 10 MiB is plenty for any realistic source line.
const maxSourceLineBytes = 10 * 1024 * 1024

// readFileLines reads a file and returns its content as a slice of strings.
func readFileLines(fileName string) ([]string, error) {
	file, err := os.Open(fileName) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxSourceLineBytes) //nolint:mnd
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// shouldSkipLine filters structural lines — blanks, whole-line //
// comments, and bare closing braces — that Go coverage profile blocks
// routinely span through but that aren't themselves executable. Block
// comments (/* … */) are intentionally not handled here because the Go
// coverage tool does not extend profile blocks over them, so they never
// reach this check.
func shouldSkipLine(content string) bool {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return true
	}
	if strings.HasPrefix(trimmed, "//") {
		return true
	}
	return isBareClosingBrace(trimmed)
}

// isBareClosingBrace reports whether trimmed is a closing brace with no
// executable follow-up — e.g. `}`, `},`, `})`, `};`, `} // comment`. Lines
// such as `} else {` or `} else if cond {` return false because the Go
// compiler treats them as part of the if/else chain and they can be covered.
func isBareClosingBrace(trimmed string) bool {
	if !strings.HasPrefix(trimmed, "}") {
		return false
	}
	if idx := strings.Index(trimmed, "//"); idx >= 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}
	rest := strings.TrimLeft(trimmed, "})],;")
	return strings.TrimSpace(rest) == ""
}

// ReadSourceFile reads a source file, attempting to find it even with partial paths.
func ReadSourceFile(fileName string) ([]string, error) {
	// Try the filename as-is first.
	if lines, err := readFileLines(fileName); err == nil {
		return lines, nil
	}
	// Coverage data often has full package paths; try successively shorter
	// suffixes of the path relative to the current directory. Normalize to
	// forward slashes so Windows-style paths (\) are handled too.
	normalized := filepath.ToSlash(fileName)
	if !strings.Contains(normalized, "/") {
		return nil, os.ErrNotExist
	}
	parts := strings.Split(normalized, "/")
	for i := range parts {
		// Join with "/" (OSes accept it as a separator, including Windows) to
		// avoid filepath.Join's drive-relative surprise on Windows paths like
		// "C:/Users/..." where Join("C:", "Users", ...) produces "C:Users\..."
		// instead of "C:\Users\...".
		if lines, err := readFileLines(strings.Join(parts[i:], "/")); err == nil {
			return lines, nil
		}
	}
	return nil, os.ErrNotExist
}

// CollectBlocks builds Block entries for every ProfileBlock in p, attaching
// per-line source content and filter status when the source file can be
// read. When the source file is unreadable, blocks are still emitted with
// line numbers and hit counts only so callers driven purely by profile data
// (e.g. mocks in tests or stripped binaries) still see them.
func CollectBlocks(p *cover.Profile) []Block {
	blocksSlice := make([]Block, 0, len(p.Blocks))
	sourceLines, _ := ReadSourceFile(p.FileName)
	sourceAvailable := len(sourceLines) > 0

	for _, b := range p.Blocks {
		linesSlice := make([]Line, 0, b.EndLine-b.StartLine+1)
		for line := b.StartLine; line <= b.EndLine; line++ {
			entry := Line{LineNumber: line, Hits: b.Count}
			switch idx := line - 1; {
			case idx >= 0 && idx < len(sourceLines):
				entry.Content = sourceLines[idx]
				entry.IsFiltered = shouldSkipLine(sourceLines[idx])
			case sourceAvailable:
				// Source readable but block references a line past EOF;
				// filter it so --inspect doesn't emit an empty hunk.
				entry.IsFiltered = true
			}
			linesSlice = append(linesSlice, entry)
		}
		blocksSlice = append(blocksSlice, Block{
			ProfileBlock: b,
			Lines:        linesSlice,
		})
	}

	return blocksSlice
}

// FormatUncoveredLines collects all uncovered lines from a coverage profile,
// filters them, and formats them into a string of line ranges (e.g. "3-5,9").
// Go cover profiles can contain overlapping blocks on the same line, so a line
// is only considered uncovered when no covering block was executed.
func FormatUncoveredLines(p *cover.Profile) string {
	return FormatUncoveredFromBlocks(CollectBlocks(p))
}

// FormatUncoveredFromBlocks is like FormatUncoveredLines but operates on a
// pre-collected slice of blocks so callers that already did the work of
// reading/parsing the source file don't pay for it twice.
func FormatUncoveredFromBlocks(blocks []Block) string {
	covered := make(map[int]bool)
	uncovered := make(map[int]bool)

	for _, block := range blocks {
		for _, line := range block.Lines {
			if line.IsFiltered {
				continue
			}
			if block.IsCovered() {
				covered[line.LineNumber] = true
			} else {
				uncovered[line.LineNumber] = true
			}
		}
	}

	result := make([]int, 0, len(uncovered))
	for ln := range uncovered {
		if !covered[ln] {
			result = append(result, ln)
		}
	}

	if len(result) == 0 {
		return ""
	}

	slices.Sort(result)
	return formatLineRanges(result)
}

// CoverageFromBlocks returns the total and covered line counts for a
// pre-collected slice of blocks, mirroring the filtering used by
// FormatUncoveredFromBlocks so enforced Line % and reported uncovered lines
// share a single definition.
func CoverageFromBlocks(blocks []Block) (int, int) {
	allLines := make(map[int]bool)
	coveredLines := make(map[int]bool)

	for _, block := range blocks {
		for _, l := range block.Lines {
			if l.IsFiltered {
				continue
			}
			allLines[l.LineNumber] = true
			if block.IsCovered() {
				coveredLines[l.LineNumber] = true
			}
		}
	}

	return len(allLines), len(coveredLines)
}

// formatLineRanges formats a slice of line numbers into a string of ranges.
// For example, [1, 2, 3, 5, 6] becomes "1-3,5-6".
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

// formatRange formats a single range of line numbers into a string.
// If the start and end are the same, it returns a single number.
func formatRange(start, end int) string {
	if start == end {
		return strconv.Itoa(start)
	}
	return strconv.Itoa(start) + "-" + strconv.Itoa(end)
}
