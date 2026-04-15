package output

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/lines"
	"golang.org/x/term"
	"golang.org/x/tools/cover"
)

// InspectUncoveredLines displays uncovered lines for the specified files.
func InspectUncoveredLines(profiles []*cover.Profile, cfg *config.Config) error {
	var profilesToShow []*cover.Profile

	// Filter profiles to show
	if len(cfg.InspectFiles) > 0 {
		for _, profile := range profiles {
			for _, uncoveredFile := range cfg.InspectFiles {
				if matchesInspectFile(profile.FileName, uncoveredFile) {
					profilesToShow = append(profilesToShow, profile)
					break
				}
			}
		}
		if len(profilesToShow) == 0 {
			return fmt.Errorf("file(s) %q not found in coverage data", strings.Join(cfg.InspectFiles, ", "))
		}
	} else {
		profilesToShow = profiles
	}

	entries := make([]fileUncovered, 0, len(profilesToShow))
	for _, p := range profilesToShow {
		// Read the source file once per profile; both CollectBlocks and every
		// merged hunk's SourceLinesInRange would otherwise re-read it.
		sourceLines, _ := lines.ReadSourceFile(p.FileName)
		blocks := lines.CollectBlocksFromSource(p, sourceLines)
		uncovered := uncoveredHunks(p, sourceLines, blocks, cfg.InspectContext)
		if len(uncovered) > 0 {
			entries = append(entries, fileUncovered{filename: p.FileName, blocks: uncovered})
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].filename < entries[j].filename })

	return displayUncoveredLines(entries, cfg)
}

// uncoveredHunks walks the collected blocks and returns one hunk per
// contiguous run of uncovered line ranges. Blocks whose ranges touch or
// overlap (including after the requested context expansion would make them
// visually adjacent) are merged into a single hunk so the rendered output
// shows 200-205 instead of 200-203 / 201-205. Each hunk is then populated
// via lines.SourceLinesInRangeFromSource so every source line in the merged
// range is present — even lines that weren't attributed to any profile
// block — without re-reading the source file per hunk.
func uncoveredHunks(p *cover.Profile, sourceLines []string, blocks []lines.Block, contextSize int) []uncoveredBlock {
	type span struct{ start, end int }
	var spans []span
	for _, block := range blocks {
		if block.IsCovered() || !blockHasUnfilteredLine(block) {
			continue
		}
		spans = append(spans, span{start: block.StartLine, end: block.EndLine})
	}
	if len(spans) == 0 {
		return nil
	}
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start < spans[j].start
		}
		return spans[i].end < spans[j].end
	})

	// Merge when two spans touch, overlap, or fall within one another's
	// context window. The gap is both sides of context plus one line of
	// slack so the rendered hunks don't show the same context line twice.
	mergeGap := contextSize + contextSize + 1
	merged := []span{spans[0]}
	for _, s := range spans[1:] {
		last := &merged[len(merged)-1]
		if s.start <= last.end+mergeGap {
			if s.end > last.end {
				last.end = s.end
			}
			continue
		}
		merged = append(merged, s)
	}

	out := make([]uncoveredBlock, 0, len(merged))
	for _, s := range merged {
		out = append(out, uncoveredBlock{
			start:   s.start,
			end:     s.end,
			context: lines.SourceLinesInRangeFromSource(p, sourceLines, s.start, s.end, contextSize),
		})
	}
	return out
}

// matchesInspectFile reports whether profileName matches the user-supplied
// --inspect-file pattern. The pattern matches when it is equal to the
// profile name, equal to the profile's trailing path segment, or appears as
// a path-boundary-aligned suffix (e.g. "pkg/foo.go" matches
// "github.com/acme/mod/pkg/foo.go"). Both sides are normalized to forward
// slashes regardless of runtime OS so Windows-style backslash paths behave
// identically; plain substring matching caused false positives like
// "foo.go" matching "barfoo.go".
func matchesInspectFile(profileName, pattern string) bool {
	if pattern == "" {
		return false
	}
	pn := strings.ReplaceAll(profileName, `\`, "/")
	pat := strings.ReplaceAll(pattern, `\`, "/")
	if pn == pat {
		return true
	}
	if path.Base(pn) == pat {
		return true
	}
	return strings.HasSuffix(pn, "/"+pat)
}

// blockHasUnfilteredLine reports whether block contains at least one line that
// survives filtering (blank / comment / bare brace). Blocks that are entirely
// filtered would otherwise render as an empty hunk header with no visible
// uncovered code.
func blockHasUnfilteredLine(block lines.Block) bool {
	for _, l := range block.Lines {
		if !l.IsFiltered {
			return true
		}
	}
	return false
}

type fileUncovered struct {
	filename string
	blocks   []uncoveredBlock
}

type uncoveredBlock struct {
	start, end int
	context    []lines.Line
}

// displayUncoveredLines displays the uncovered lines with formatting.
func displayUncoveredLines(entries []fileUncovered, cfg *config.Config) error {
	if len(entries) == 0 {
		fmt.Println("No uncovered lines found!")
		return nil
	}

	output := buildUncoveredOutput(entries, cfg)

	// Use pager if the output is large and we're in a terminal
	if shouldUsePager(output) {
		return displayWithPager(output)
	}

	fmt.Print(output)
	return nil
}

// buildUncoveredOutput builds the formatted output string.
func buildUncoveredOutput(entries []fileUncovered, cfg *config.Config) string {
	var sb strings.Builder
	highlighter := newSyntaxHighlighter("go", cfg)

	for _, entry := range entries {
		// File header
		fileHeader := "--- " + entry.filename
		if !cfg.NoColor {
			fileHeader = color.New(color.FgCyan, color.Bold).Sprint(fileHeader)
		}
		sb.WriteString(fileHeader + "\n")

		for _, block := range entry.blocks {
			header := fmt.Sprintf("@@ Lines %d-%d @@", block.start, block.end)
			if !cfg.NoColor {
				header = color.New(color.FgYellow).Sprint(header)
			}
			sb.WriteString(header + "\n")

			for _, line := range block.context {
				sb.WriteString(formatSourceLine(line, cfg, highlighter) + "\n")
			}

			sb.WriteString("\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// formatSourceLine formats a source line with proper coloring and indicators.
// Three visual classes: covered (green line number, space gutter), uncovered
// and executable (red line number, "-" gutter), and filtered structural lines
// (default-color line number, space gutter) which are rendered so merged
// hunks remain readable across bare braces / comments / blank lines.
func formatSourceLine(line lines.Line, cfg *config.Config, highlighter *syntaxHighlighter) string {
	lineNumStr := fmt.Sprintf("%4d", line.LineNumber)
	hitsStr := fmt.Sprintf("(%d)", line.Hits)
	gutter, colorize := lineGutterAndColor(line)

	if cfg.NoColor {
		return fmt.Sprintf("%s %s%5s: %s", gutter, lineNumStr, hitsStr, line.Content)
	}
	content := highlighter.highlight(line.Content)
	if colorize != nil {
		lineNumStr = colorize(lineNumStr)
	}
	return fmt.Sprintf("%s %s%5s: %s", gutter, lineNumStr, hitsStr, content)
}

func lineGutterAndColor(line lines.Line) (string, func(string) string) {
	switch {
	case line.IsFiltered:
		return " ", nil
	case line.Hits > 0:
		return " ", func(s string) string { return color.New(color.FgGreen).Sprint(s) }
	default:
		return "-", func(s string) string { return color.New(color.FgRed).Sprint(s) }
	}
}

// shouldUsePager determines if we should use a pager for output.
func shouldUsePager(output string) bool {
	lineCount := strings.Count(output, "\n")
	return lineCount > 30 && isTerminal() // Use pager for output with more than 30 lines
}

// isTerminal checks if we're running in a terminal.
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) //nolint:gosec // fd fits in int on supported platforms
}

// displayWithPager displays output in an in-process Bubble Tea pager.
// Falls back to plain stdout when not a TTY, NO_PAGER is set, or TERM=dumb.
func displayWithPager(output string) error {
	if !isTerminal() || os.Getenv("NO_PAGER") != "" || os.Getenv("TERM") == "dumb" {
		fmt.Print(output)
		return nil
	}
	return runBubblePager(output)
}
