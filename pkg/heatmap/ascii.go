package heatmap

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

// ASCIIHeatmap represents an ASCII art coverage heat map.
type ASCIIHeatmap struct {
	writer io.Writer
	config *config.Config
}

// NewASCIIHeatmap creates a new ASCII heat map generator.
func NewASCIIHeatmap(writer io.Writer, cfg *config.Config) *ASCIIHeatmap {
	return &ASCIIHeatmap{
		writer: writer,
		config: cfg,
	}
}

// Generate creates an ASCII art grid heat map of the coverage results.
func (h *ASCIIHeatmap) Generate(results compute.Results) error {
	// Generate legend at the top
	h.writeLegend()

	// Generate header
	h.writeHeader()

	// Generate file-level grid heat map
	if err := h.generateFileGridHeatmap(results.ByFile); err != nil {
		return err
	}

	// Generate package-level grid heat map
	if err := h.generatePackageGridHeatmap(results.ByPackage); err != nil {
		return err
	}

	// Generate overall coverage summary
	h.generateSummary(results.ByTotal)

	return nil
}

func (h *ASCIIHeatmap) writeLegend() {
	fmt.Fprintln(h.writer, "")
	fmt.Fprintln(h.writer, "Coverage Legend:")
	if !h.config.NoColor {
		fmt.Fprintf(h.writer, "  %s Critical  0-29%%   (High Priority)\n", color.New(color.FgHiRed, color.Bold).Sprint("██████"))
		fmt.Fprintf(h.writer, "  %s Poor     30-49%%  (Medium Priority)\n", color.RedString("██████"))
		fmt.Fprintf(h.writer, "  %s Fair     50-69%%  (Low Priority)\n", color.New(color.FgHiYellow).Sprint("██████"))
		fmt.Fprintf(h.writer, "  %s Good     70-89%%  (Maintain)\n", color.YellowString("██████"))
		fmt.Fprintf(h.writer, "  %s Excellent 90-100%% (Well Covered)\n", color.GreenString("██████"))
	} else {
		fmt.Fprintln(h.writer, "  ██████ Critical  0-29%   (High Priority)")
		fmt.Fprintln(h.writer, "  ▓▓▓▓▓▓ Poor     30-49%  (Medium Priority)")
		fmt.Fprintln(h.writer, "  ░░░░░░ Fair     50-69%  (Low Priority)")
		fmt.Fprintln(h.writer, "  ▒▒▒▒▒▒ Good     70-89%  (Maintain)")
		fmt.Fprintln(h.writer, "  ▓▓▓▓▓▓ Excellent 90-100% (Well Covered)")
	}
	fmt.Fprintln(h.writer, "")
}

func (h *ASCIIHeatmap) writeHeader() {
	fmt.Fprintln(h.writer, "═══════════════════════════════════════════")
	fmt.Fprintln(h.writer, "           COVERAGE GRID HEAT MAP")
	fmt.Fprintln(h.writer, "═══════════════════════════════════════════")
	fmt.Fprintln(h.writer, "")
}

func (h *ASCIIHeatmap) generateFileGridHeatmap(files []compute.ByFile) error {
	if len(files) == 0 {
		return nil
	}

	fmt.Fprintln(h.writer, "By Files:")
	fmt.Fprintln(h.writer, strings.Repeat("─", 80))

	// Sort files by coverage percentage (ascending to highlight worst first)
	sortedFiles := make([]compute.ByFile, len(files))
	copy(sortedFiles, files)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].StatementPercentage < sortedFiles[j].StatementPercentage
	})

	cols := h.calculateGridCols(len(sortedFiles))
	rows := (len(sortedFiles) + cols - 1) / cols

	for row := range make([]int, rows) {
		for cellLine := range make([]int, 3) {
			for col := range make([]int, cols) {
				idx := row*cols + col
				if idx >= len(sortedFiles) {
					break
				}
				file := sortedFiles[idx]
				cellLines := strings.Split(h.generateCoverageCell7x3(file.StatementPercentage), "\n")
				fmt.Fprint(h.writer, cellLines[cellLine])
			}
			// Add file names to the right of the grid on the middle line only
			if cellLine == 1 {
				fmt.Fprint(h.writer, "  ")
				for col := range make([]int, cols) {
					idx := row*cols + col
					if idx >= len(sortedFiles) {
						break
					}
					file := sortedFiles[idx]
					abbreviated := h.truncateFilename(file.File)
					if col > 0 {
						fmt.Fprint(h.writer, ", ")
					}
					fmt.Fprint(h.writer, abbreviated)
				}
			}
			fmt.Fprintln(h.writer)
		}
	}
	fmt.Fprintln(h.writer)
	return nil
}

func (h *ASCIIHeatmap) generatePackageGridHeatmap(packages []compute.ByPackage) error {
	if len(packages) == 0 {
		return nil
	}

	fmt.Fprintln(h.writer, "By Packages:")
	fmt.Fprintln(h.writer, strings.Repeat("─", 80))

	sortedPackages := make([]compute.ByPackage, len(packages))
	copy(sortedPackages, packages)
	sort.Slice(sortedPackages, func(i, j int) bool {
		return sortedPackages[i].StatementPercentage < sortedPackages[j].StatementPercentage
	})

	cols := h.calculateGridCols(len(sortedPackages))
	rows := (len(sortedPackages) + cols - 1) / cols

	for row := range make([]int, rows) {
		for cellLine := range make([]int, 3) {
			for col := range make([]int, cols) {
				idx := row*cols + col
				if idx >= len(sortedPackages) {
					break
				}
				pkg := sortedPackages[idx]
				cellLines := strings.Split(h.generateCoverageCell7x3(pkg.StatementPercentage), "\n")
				fmt.Fprint(h.writer, cellLines[cellLine])
			}
			if cellLine == 1 {
				fmt.Fprint(h.writer, "  ")
				for col := range make([]int, cols) {
					idx := row*cols + col
					if idx >= len(sortedPackages) {
						break
					}
					pkg := sortedPackages[idx]
					abbreviated := h.truncatePackageName(pkg.Package)
					if col > 0 {
						fmt.Fprint(h.writer, ", ")
					}
					fmt.Fprint(h.writer, abbreviated)
				}
			}
			fmt.Fprintln(h.writer)
		}
	}
	fmt.Fprintln(h.writer)
	return nil
}

func (h *ASCIIHeatmap) calculateGridCols(itemCount int) int {
	if itemCount <= 0 {
		return 1
	}

	// With 6-character wide cells, we need to be more conservative with columns
	// to fit in terminal width (80 chars typically)
	maxCols := 10 // Each cell is 6 chars + 1 space = 7 chars per cell, so ~10 fits in 80 chars
	if itemCount <= 2 {
		return itemCount
	}
	if itemCount <= 8 {
		return 4
	}
	if itemCount <= 20 {
		return 6
	}
	if itemCount <= 40 {
		return 8
	}
	return maxCols
}

func (h *ASCIIHeatmap) generateCoverageCell7x3(percentage float64) string {
	// Use 7x3 block characters for grid cells with percentage in the middle
	percentStr := fmt.Sprintf("%3.0f", percentage) // Format as 3-character string without %

	// Center the percentage in the middle line (3 chars, 2 blocks on each side)
	line1 := "       "
	line2 := fmt.Sprintf("  %s  ", percentStr)
	line3 := "       "
	cellPattern := line1 + "\n" + line2 + "\n" + line3

	if h.config.NoColor {
		// Use different ASCII characters when color is disabled
		switch {
		case percentage >= 90:
			return strings.ReplaceAll(cellPattern, " ", "▓") // Well covered
		case percentage >= 70:
			return strings.ReplaceAll(cellPattern, " ", "▒") // Good
		case percentage >= 50:
			return strings.ReplaceAll(cellPattern, " ", "░") // Fair
		case percentage >= 30:
			return strings.ReplaceAll(cellPattern, " ", "▓") // Poor
		default:
			return strings.ReplaceAll(cellPattern, " ", "█") // Critical (needs attention)
		}
	}

	// Color the cell based on coverage level with consistent background colors
	// Color all three lines the same way for consistency
	var coloredLines [3]string
	switch {
	case percentage >= 90:
		coloredLines[0] = color.New(color.BgGreen, color.FgWhite).Sprint(line1)
		coloredLines[1] = color.New(color.BgGreen, color.FgWhite).Sprint(line2)
		coloredLines[2] = color.New(color.BgGreen, color.FgWhite).Sprint(line3)
	case percentage >= 70:
		coloredLines[0] = color.New(color.BgYellow, color.FgBlack).Sprint(line1)
		coloredLines[1] = color.New(color.BgYellow, color.FgBlack).Sprint(line2)
		coloredLines[2] = color.New(color.BgYellow, color.FgBlack).Sprint(line3)
	case percentage >= 50:
		coloredLines[0] = color.New(color.BgHiYellow, color.FgBlack).Sprint(line1)
		coloredLines[1] = color.New(color.BgHiYellow, color.FgBlack).Sprint(line2)
		coloredLines[2] = color.New(color.BgHiYellow, color.FgBlack).Sprint(line3)
	case percentage >= 30:
		coloredLines[0] = color.New(color.BgRed, color.FgWhite).Sprint(line1)
		coloredLines[1] = color.New(color.BgRed, color.FgWhite).Sprint(line2)
		coloredLines[2] = color.New(color.BgRed, color.FgWhite).Sprint(line3)
	default:
		coloredLines[0] = color.New(color.BgHiRed, color.FgWhite, color.Bold).Sprint(line1)
		coloredLines[1] = color.New(color.BgHiRed, color.FgWhite, color.Bold).Sprint(line2)
		coloredLines[2] = color.New(color.BgHiRed, color.FgWhite, color.Bold).Sprint(line3)
	}
	return strings.Join(coloredLines[:], "\n")
}

func (h *ASCIIHeatmap) truncateFilename(filename string) string {
	// Truncate file names to 6 characters
	if len(filename) <= 6 {
		return filename
	}
	return filename[:6]
}

func (h *ASCIIHeatmap) truncatePackageName(packageName string) string {
	// Prefix truncate to show last 6 characters for package names
	if len(packageName) <= 6 {
		return packageName
	}
	return packageName[len(packageName)-6:]
}

func (h *ASCIIHeatmap) abbreviateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-2] + ".."
}

func (h *ASCIIHeatmap) generateSummary(totals compute.Totals) {
	fmt.Fprintln(h.writer, "By Total:")
	fmt.Fprintln(h.writer, strings.Repeat("─", 80))

	// Create simpler single line cells for summary
	stmtCell := h.generateSimpleCoverageCell(totals.Statements.Percentage)
	blockCell := h.generateSimpleCoverageCell(totals.Blocks.Percentage)

	fmt.Fprintf(h.writer, "%s Statement Coverage %s (%.1f%%)\n", stmtCell, totals.Statements.Coverage, totals.Statements.Percentage)
	fmt.Fprintf(h.writer, "%s Block Coverage     %s (%.1f%%)\n", blockCell, totals.Blocks.Coverage, totals.Blocks.Percentage)
	fmt.Fprintln(h.writer, "")
}

func (h *ASCIIHeatmap) generateSimpleCoverageCell(percentage float64) string {
	// Use simple block characters for summary
	cell := "██ "

	if h.config.NoColor {
		// Use different ASCII characters when color is disabled
		switch {
		case percentage >= 90:
			return "▓▓ " // Well covered
		case percentage >= 70:
			return "▒▒ " // Good
		case percentage >= 50:
			return "░░ " // Fair
		case percentage >= 30:
			return "▓▓ " // Poor
		default:
			return "██ " // Critical (needs attention)
		}
	}

	// Color the cell based on coverage level - prioritize highlighting opportunities
	switch {
	case percentage >= 90:
		return color.GreenString(cell) // Well covered - green
	case percentage >= 70:
		return color.YellowString(cell) // Good - yellow
	case percentage >= 50:
		return color.New(color.FgHiYellow).Sprint(cell) // Fair - bright yellow
	case percentage >= 30:
		return color.RedString(cell) // Poor - red (opportunity)
	default:
		return color.New(color.FgHiRed, color.Bold).Sprint(cell) // Critical - bright red (high opportunity)
	}
}
