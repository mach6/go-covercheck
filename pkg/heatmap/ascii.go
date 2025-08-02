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

func (h *ASCIIHeatmap) writeHeader() {
	fmt.Fprintln(h.writer, "")
	fmt.Fprintln(h.writer, "═══════════════════════════════════════════")
	fmt.Fprintln(h.writer, "           COVERAGE GRID HEAT MAP")
	fmt.Fprintln(h.writer, "═══════════════════════════════════════════")
	fmt.Fprintln(h.writer, "")
	
	// Legend - focus on highlighting opportunities (low coverage)
	fmt.Fprintln(h.writer, "Coverage Legend (highlighting improvement opportunities):")
	if !h.config.NoColor {
		fmt.Fprintf(h.writer, "  %s Critical  0-29%%   (High Priority)\n", color.New(color.FgHiRed, color.Bold).Sprint("██"))
		fmt.Fprintf(h.writer, "  %s Poor     30-49%%  (Medium Priority)\n", color.RedString("██"))
		fmt.Fprintf(h.writer, "  %s Fair     50-69%%  (Low Priority)\n", color.New(color.FgHiYellow).Sprint("██"))
		fmt.Fprintf(h.writer, "  %s Good     70-89%%  (Maintain)\n", color.YellowString("██"))
		fmt.Fprintf(h.writer, "  %s Excellent 90-100%% (Well Covered)\n", color.GreenString("██"))
	} else {
		fmt.Fprintln(h.writer, "  ██ Critical  0-29%   (High Priority)")
		fmt.Fprintln(h.writer, "  ██ Poor     30-49%  (Medium Priority)")
		fmt.Fprintln(h.writer, "  ██ Fair     50-69%  (Low Priority)")
		fmt.Fprintln(h.writer, "  ██ Good     70-89%  (Maintain)")
		fmt.Fprintln(h.writer, "  ██ Excellent 90-100% (Well Covered)")
	}
	fmt.Fprintln(h.writer, "")
}

func (h *ASCIIHeatmap) generateFileGridHeatmap(files []compute.ByFile) error {
	if len(files) == 0 {
		return nil
	}

	fmt.Fprintln(h.writer, "FILES BY COVERAGE (highlighting improvement opportunities):")
	fmt.Fprintln(h.writer, strings.Repeat("─", 80))

	// Sort files by coverage percentage (ascending to highlight worst first)
	sortedFiles := make([]compute.ByFile, len(files))
	copy(sortedFiles, files)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].StatementPercentage < sortedFiles[j].StatementPercentage
	})

	// Calculate grid dimensions - aim for roughly square grid
	cols := h.calculateGridCols(len(sortedFiles))
	rows := (len(sortedFiles) + cols - 1) / cols

	for row := 0; row < rows; row++ {
		// Print file cells
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedFiles) {
				break
			}
			file := sortedFiles[idx]
			cell := h.generateCoverageCell(file.StatementPercentage)
			fmt.Fprint(h.writer, cell)
		}
		fmt.Fprintln(h.writer)

		// Print file names below the cells (abbreviated)
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedFiles) {
				break
			}
			file := sortedFiles[idx]
			// Extract just the filename without path and package
			filename := h.extractFilename(file.File)
			abbreviated := h.abbreviateText(filename, 8)
			fmt.Fprintf(h.writer, "%-8s ", abbreviated)
		}
		fmt.Fprintln(h.writer)

		// Print coverage percentages
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedFiles) {
				break
			}
			file := sortedFiles[idx]
			fmt.Fprintf(h.writer, "%6.1f%% ", file.StatementPercentage)
		}
		fmt.Fprintln(h.writer)
		fmt.Fprintln(h.writer) // Extra spacing between rows
	}

	fmt.Fprintln(h.writer)
	return nil
}

func (h *ASCIIHeatmap) generatePackageGridHeatmap(packages []compute.ByPackage) error {
	if len(packages) == 0 {
		return nil
	}

	fmt.Fprintln(h.writer, "PACKAGES BY COVERAGE (highlighting improvement opportunities):")
	fmt.Fprintln(h.writer, strings.Repeat("─", 80))

	// Sort packages by coverage percentage (ascending to highlight worst first)  
	sortedPackages := make([]compute.ByPackage, len(packages))
	copy(sortedPackages, packages)
	sort.Slice(sortedPackages, func(i, j int) bool {
		return sortedPackages[i].StatementPercentage < sortedPackages[j].StatementPercentage
	})

	// Calculate grid dimensions - aim for roughly square grid
	cols := h.calculateGridCols(len(sortedPackages))
	rows := (len(sortedPackages) + cols - 1) / cols

	for row := 0; row < rows; row++ {
		// Print package cells
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedPackages) {
				break
			}
			pkg := sortedPackages[idx]
			cell := h.generateCoverageCell(pkg.StatementPercentage)
			fmt.Fprint(h.writer, cell)
		}
		fmt.Fprintln(h.writer)

		// Print package names below the cells (abbreviated)
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedPackages) {
				break
			}
			pkg := sortedPackages[idx]
			abbreviated := h.abbreviateText(pkg.Package, 12)
			fmt.Fprintf(h.writer, "%-12s ", abbreviated)
		}
		fmt.Fprintln(h.writer)

		// Print coverage percentages
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedPackages) {
				break
			}
			pkg := sortedPackages[idx]
			fmt.Fprintf(h.writer, "%11.1f%% ", pkg.StatementPercentage)
		}
		fmt.Fprintln(h.writer)
		fmt.Fprintln(h.writer) // Extra spacing between rows
	}

	fmt.Fprintln(h.writer)
	return nil
}

func (h *ASCIIHeatmap) calculateGridCols(itemCount int) int {
	if itemCount <= 0 {
		return 1
	}
	
	// Aim for roughly square grid, but consider terminal width
	maxCols := 6 // Conservative for readability
	if itemCount <= 4 {
		return itemCount
	}
	if itemCount <= 16 {
		return 4
	}
	if itemCount <= 36 {
		return 6
	}
	return maxCols
}

func (h *ASCIIHeatmap) generateCoverageCell(percentage float64) string {
	// Use block characters for grid cells
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
	cell := "██ "
	switch {
	case percentage >= 90:
		return color.GreenString(cell)     // Well covered - green
	case percentage >= 70:
		return color.YellowString(cell)    // Good - yellow
	case percentage >= 50:
		return color.New(color.FgHiYellow).Sprint(cell) // Fair - bright yellow
	case percentage >= 30:
		return color.RedString(cell)       // Poor - red (opportunity)
	default:
		return color.New(color.FgHiRed, color.Bold).Sprint(cell) // Critical - bright red (high opportunity)
	}
}

func (h *ASCIIHeatmap) extractFilename(fullPath string) string {
	// Extract just the filename from a full path
	lastSlash := strings.LastIndex(fullPath, "/")
	if lastSlash != -1 && lastSlash < len(fullPath)-1 {
		return fullPath[lastSlash+1:]
	}
	return fullPath
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
	fmt.Fprintln(h.writer, "OVERALL COVERAGE SUMMARY:")
	fmt.Fprintln(h.writer, strings.Repeat("─", 80))
	
	stmtCell := h.generateCoverageCell(totals.Statements.Percentage)
	blockCell := h.generateCoverageCell(totals.Blocks.Percentage)
	
	fmt.Fprintf(h.writer, "%s Statement Coverage %s (%.1f%%)\n", stmtCell, totals.Statements.Coverage, totals.Statements.Percentage)
	fmt.Fprintf(h.writer, "%s Block Coverage     %s (%.1f%%)\n", blockCell, totals.Blocks.Coverage, totals.Blocks.Percentage)
	fmt.Fprintln(h.writer, "")
}

