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

// Generate creates an ASCII art heat map of the coverage results.
func (h *ASCIIHeatmap) Generate(results compute.Results) error {
	// Generate header
	h.writeHeader()

	// Generate file coverage heat map
	if err := h.generateFileHeatmap(results.ByFile); err != nil {
		return err
	}

	// Generate package coverage heat map
	if err := h.generatePackageHeatmap(results.ByPackage); err != nil {
		return err
	}

	// Generate overall coverage summary
	h.generateSummary(results.ByTotal)

	return nil
}

func (h *ASCIIHeatmap) writeHeader() {
	fmt.Fprintln(h.writer, "")
	fmt.Fprintln(h.writer, "═══════════════════════════════════════════")
	fmt.Fprintln(h.writer, "           COVERAGE HEAT MAP")
	fmt.Fprintln(h.writer, "═══════════════════════════════════════════")
	fmt.Fprintln(h.writer, "")
	
	// Legend
	fmt.Fprintln(h.writer, "Coverage Legend:")
	if !h.config.NoColor {
		fmt.Fprintf(h.writer, "  %s ========== %s  90-100%%\n", color.GreenString(""), "Excellent")
		fmt.Fprintf(h.writer, "  %s ========== %s  70-89%%\n", color.YellowString(""), "Good")
		fmt.Fprintf(h.writer, "  %s ========== %s  50-69%%\n", color.New(color.FgHiYellow).Sprint(""), "Fair")
		fmt.Fprintf(h.writer, "  %s ========== %s  30-49%%\n", color.New(color.FgRed).Sprint(""), "Poor")
		fmt.Fprintf(h.writer, "  %s ========== %s  0-29%%\n", color.New(color.FgHiRed).Sprint(""), "Critical")
	} else {
		fmt.Fprintln(h.writer, "  ========== Excellent  90-100%")
		fmt.Fprintln(h.writer, "  ========== Good       70-89%")
		fmt.Fprintln(h.writer, "  ========== Fair       50-69%")
		fmt.Fprintln(h.writer, "  ========== Poor       30-49%")
		fmt.Fprintln(h.writer, "  ========== Critical   0-29%")
	}
	fmt.Fprintln(h.writer, "")
}

func (h *ASCIIHeatmap) generateFileHeatmap(files []compute.ByFile) error {
	if len(files) == 0 {
		return nil
	}

	fmt.Fprintln(h.writer, "FILES BY COVERAGE:")
	fmt.Fprintln(h.writer, strings.Repeat("─", 60))

	// Sort files by coverage percentage (descending)
	sortedFiles := make([]compute.ByFile, len(files))
	copy(sortedFiles, files)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].StatementPercentage > sortedFiles[j].StatementPercentage
	})

	for _, file := range sortedFiles {
		bar := h.generateCoverageBar(file.StatementPercentage)
		fileName := h.truncateFileName(file.File, 35)
		fmt.Fprintf(h.writer, "%s %s %6.1f%%\n", bar, fileName, file.StatementPercentage)
	}
	fmt.Fprintln(h.writer, "")
	return nil
}

func (h *ASCIIHeatmap) generatePackageHeatmap(packages []compute.ByPackage) error {
	if len(packages) == 0 {
		return nil
	}

	fmt.Fprintln(h.writer, "PACKAGES BY COVERAGE:")
	fmt.Fprintln(h.writer, strings.Repeat("─", 60))

	// Sort packages by coverage percentage (descending)
	sortedPackages := make([]compute.ByPackage, len(packages))
	copy(sortedPackages, packages)
	sort.Slice(sortedPackages, func(i, j int) bool {
		return sortedPackages[i].StatementPercentage > sortedPackages[j].StatementPercentage
	})

	for _, pkg := range sortedPackages {
		bar := h.generateCoverageBar(pkg.StatementPercentage)
		packageName := h.truncateFileName(pkg.Package, 35)
		fmt.Fprintf(h.writer, "%s %s %6.1f%%\n", bar, packageName, pkg.StatementPercentage)
	}
	fmt.Fprintln(h.writer, "")
	return nil
}

func (h *ASCIIHeatmap) generateSummary(totals compute.Totals) {
	fmt.Fprintln(h.writer, "OVERALL COVERAGE:")
	fmt.Fprintln(h.writer, strings.Repeat("─", 60))
	
	stmtBar := h.generateCoverageBar(totals.Statements.Percentage)
	blockBar := h.generateCoverageBar(totals.Blocks.Percentage)
	
	fmt.Fprintf(h.writer, "%s Statements %s %6.1f%%\n", stmtBar, totals.Statements.Coverage, totals.Statements.Percentage)
	fmt.Fprintf(h.writer, "%s Blocks     %s %6.1f%%\n", blockBar, totals.Blocks.Coverage, totals.Blocks.Percentage)
	fmt.Fprintln(h.writer, "")
}

func (h *ASCIIHeatmap) generateCoverageBar(percentage float64) string {
	var chars string

	if h.config.NoColor {
		// Use different characters for different coverage levels when color is disabled
		switch {
		case percentage >= 90:
			chars = "=========="
		case percentage >= 70:
			chars = "=========="
		case percentage >= 50:
			chars = "=========="
		case percentage >= 30:
			chars = "=========="
		default:
			chars = "=========="
		}
	} else {
		chars = "=========="
	}

	// Scale the bar to represent the percentage
	filledLength := int(percentage / 10)
	if filledLength > 10 {
		filledLength = 10
	}

	bar := chars[:filledLength] + strings.Repeat(" ", 10-filledLength)
	
	if h.config.NoColor {
		return fmt.Sprintf("[%s]", bar)
	}

	// Color the bar based on coverage level
	switch {
	case percentage >= 90:
		return fmt.Sprintf("[%s]", color.GreenString(bar))
	case percentage >= 70:
		return fmt.Sprintf("[%s]", color.YellowString(bar))
	case percentage >= 50:
		return fmt.Sprintf("[%s]", color.New(color.FgHiYellow).Sprint(bar))
	case percentage >= 30:
		return fmt.Sprintf("[%s]", color.RedString(bar))
	default:
		return fmt.Sprintf("[%s]", color.New(color.FgHiRed).Sprint(bar))
	}
}

func (h *ASCIIHeatmap) truncateFileName(filename string, maxLen int) string {
	if len(filename) <= maxLen {
		return fmt.Sprintf("%-*s", maxLen, filename)
	}
	return filename[:maxLen-3] + "..."
}