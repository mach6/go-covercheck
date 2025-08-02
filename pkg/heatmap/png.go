package heatmap

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"sort"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// PNGHeatmap represents a PNG image coverage heat map.
type PNGHeatmap struct {
	writer io.Writer
	config *config.Config
	width  int
	height int
}

// NewPNGHeatmap creates a new PNG heat map generator.
func NewPNGHeatmap(writer io.Writer, cfg *config.Config) *PNGHeatmap {
	return &PNGHeatmap{
		writer: writer,
		config: cfg,
		width:  1000, // Will be set dynamically
		height: 600,  // Will be calculated based on content
	}
}

// Generate creates a PNG grid heat map image of the coverage results.
func (h *PNGHeatmap) Generate(results compute.Results) error {
	// Calculate required height based on content
	requiredHeight := h.calculateRequiredHeight(results)
	h.height = requiredHeight
	h.width = 1600 // Make it even wider to fill page better

	// Create the image
	img := image.NewRGBA(image.Rect(0, 0, h.width, h.height))

	// Fill background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{245, 245, 245, 255}}, image.Point{}, draw.Src)

	// Draw title
	h.drawTitle(img, "Coverage Grid Heat Map")

	// Draw legend at top
	h.drawLegend(img)

	// Calculate layout
	startY := 250 // Start lower to accommodate legend at top

	// Draw file grid heat map
	if len(results.ByFile) > 0 {
		h.drawSectionTitle(img, "By Files", 60, startY)
		startY += 30
		startY = h.drawFileGridHeatmap(img, results.ByFile, startY)
		startY += 40
	}

	// Draw package grid heat map
	if len(results.ByPackage) > 0 {
		h.drawSectionTitle(img, "By Packages", 60, startY)
		startY += 30
		startY = h.drawPackageGridHeatmap(img, results.ByPackage, startY)
		startY += 40
	}

	// Draw summary
	h.drawSummary(img, results.ByTotal, startY)

	// Encode to PNG
	return png.Encode(h.writer, img)
}

func (h *PNGHeatmap) drawTitle(img *image.RGBA, title string) {
	x := 50
	y := 40
	c := color.RGBA{0, 0, 0, 255}

	// Draw title text (using basic font since we don't want external dependencies)
	h.drawText(img, title, x, y, c, true)
}

func (h *PNGHeatmap) drawSectionTitle(img *image.RGBA, title string, x, y int) {
	c := color.RGBA{60, 60, 60, 255}
	h.drawText(img, title, x, y, c, false)
}

func (h *PNGHeatmap) drawText(img *image.RGBA, text string, x, y int, c color.RGBA, bold bool) {
	face := basicfont.Face7x13
	if bold {
		// For bold, we'll draw the text twice with slight offset
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(c),
			Face: face,
			Dot:  fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)},
		}
		d.DrawString(text)
		d.Dot = fixed.Point26_6{fixed.Int26_6((x + 1) * 64), fixed.Int26_6(y * 64)}
		d.DrawString(text)
	} else {
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(c),
			Face: face,
			Dot:  fixed.Point26_6{fixed.Int26_6(x * 64), fixed.Int26_6(y * 64)},
		}
		d.DrawString(text)
	}
}

func (h *PNGHeatmap) calculateRequiredHeight(results compute.Results) int {
	baseHeight := 350 // Title, legend (now vertical), margins space

	// File section
	fileHeight := 0
	if len(results.ByFile) > 0 {
		cols := h.calculateGridCols(len(results.ByFile))
		rows := (len(results.ByFile) + cols - 1) / cols
		fileHeight = 70 + rows*120 // Section title + grid cells with labels (larger spacing)
	}

	// Package section
	packageHeight := 0
	if len(results.ByPackage) > 0 {
		cols := h.calculateGridCols(len(results.ByPackage))
		rows := (len(results.ByPackage) + cols - 1) / cols
		packageHeight = 70 + rows*140 // Section title + grid cells with labels (larger spacing)
	}

	// Summary section
	summaryHeight := 120 // Overall coverage section

	totalHeight := baseHeight + fileHeight + packageHeight + summaryHeight

	// Ensure minimum height
	if totalHeight < 600 {
		totalHeight = 600
	}

	return totalHeight
}

func (h *PNGHeatmap) drawFileGridHeatmap(img *image.RGBA, files []compute.ByFile, startY int) int {
	if len(files) == 0 {
		return startY
	}

	// Sort files by coverage percentage (ascending to highlight worst first)
	sortedFiles := make([]compute.ByFile, len(files))
	copy(sortedFiles, files)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].StatementPercentage < sortedFiles[j].StatementPercentage
	})

	cols := h.calculateGridCols(len(sortedFiles))
	rows := (len(sortedFiles) + cols - 1) / cols

	cellSize := 80
	cellSpacing := 80 // No spacing between cells
	gridStartX := 80
	gridStartY := startY

	for row := 0; row < rows; row++ {
		maxNamesX := gridStartX + cols*cellSpacing + 20

		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedFiles) {
				break
			}

			file := sortedFiles[idx]
			x := gridStartX + col*cellSize // No extra spacing
			y := gridStartY + row*cellSize // No extra spacing

			// Draw coverage cell
			cellColor := h.getCoverageColor(file.StatementPercentage)
			h.drawRect(img, x, y, cellSize, cellSize, cellColor)

			// Draw percentage in the middle of the cell
			percentText := fmt.Sprintf("%.0f", file.StatementPercentage)
			textX := x + cellSize/2 - len(percentText)*3
			textY := y + cellSize/2 + 3
			h.drawText(img, percentText, textX, textY, color.RGBA{255, 255, 255, 255}, true)
		}

		// Draw file names to the right of the grid
		nameY := gridStartY + row*cellSize + cellSize/2 + 3
		namesList := ""
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedFiles) {
				break
			}
			file := sortedFiles[idx]
			filename := h.extractFilename(file.File)
			abbreviated := h.truncateFilename(filename)
			if col > 0 {
				namesList += ", "
			}
			namesList += abbreviated
		}
		h.drawText(img, namesList, maxNamesX, nameY, color.RGBA{0, 0, 0, 255}, false)
	}

	return gridStartY + rows*cellSize
}

func (h *PNGHeatmap) drawPackageGridHeatmap(img *image.RGBA, packages []compute.ByPackage, startY int) int {
	if len(packages) == 0 {
		return startY
	}

	// Sort packages by coverage percentage (ascending to highlight worst first)
	sortedPackages := make([]compute.ByPackage, len(packages))
	copy(sortedPackages, packages)
	sort.Slice(sortedPackages, func(i, j int) bool {
		return sortedPackages[i].StatementPercentage < sortedPackages[j].StatementPercentage
	})

	cols := h.calculateGridCols(len(sortedPackages))
	rows := (len(sortedPackages) + cols - 1) / cols

	cellSize := 100
	cellSpacing := 100 // No spacing between cells
	gridStartX := 80
	gridStartY := startY

	for row := 0; row < rows; row++ {
		maxNamesX := gridStartX + cols*cellSpacing + 20

		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedPackages) {
				break
			}

			pkg := sortedPackages[idx]
			x := gridStartX + col*cellSize // No extra spacing
			y := gridStartY + row*cellSize // No extra spacing

			// Draw coverage cell
			cellColor := h.getCoverageColor(pkg.StatementPercentage)
			h.drawRect(img, x, y, cellSize, cellSize, cellColor)

			// Draw percentage in the middle of the cell
			percentText := fmt.Sprintf("%.0f", pkg.StatementPercentage)
			textX := x + cellSize/2 - len(percentText)*4
			textY := y + cellSize/2 + 3
			h.drawText(img, percentText, textX, textY, color.RGBA{255, 255, 255, 255}, true)
		}

		// Draw package names to the right of the grid
		nameY := gridStartY + row*cellSize + cellSize/2 + 3
		namesList := ""
		for col := 0; col < cols; col++ {
			idx := row*cols + col
			if idx >= len(sortedPackages) {
				break
			}
			pkg := sortedPackages[idx]
			abbreviated := h.truncatePackageName(pkg.Package)
			if col > 0 {
				namesList += ", "
			}
			namesList += abbreviated
		}
		h.drawText(img, namesList, maxNamesX, nameY, color.RGBA{0, 0, 0, 255}, false)
	}

	return gridStartY + rows*cellSize
}

func (h *PNGHeatmap) calculateGridCols(itemCount int) int {
	if itemCount <= 0 {
		return 1
	}

	// With larger cells and wider image (1600px), we can fit more columns
	// Each cell is roughly 100-130px wide with spacing
	maxCols := 12 // Fits in 1600px width
	if itemCount <= 4 {
		return itemCount
	}
	if itemCount <= 16 {
		return 4
	}
	if itemCount <= 36 {
		return 6
	}
	if itemCount <= 64 {
		return 8
	}
	if itemCount <= 100 {
		return 10
	}
	return maxCols
}

func (h *PNGHeatmap) extractFilename(fullPath string) string {
	// Extract just the filename from a full path
	lastSlash := -1
	for i := len(fullPath) - 1; i >= 0; i-- {
		if fullPath[i] == '/' {
			lastSlash = i
			break
		}
	}
	if lastSlash != -1 && lastSlash < len(fullPath)-1 {
		return fullPath[lastSlash+1:]
	}
	return fullPath
}

func (h *PNGHeatmap) drawSummary(img *image.RGBA, totals compute.Totals, startY int) {
	h.drawSectionTitle(img, "By Total", 60, startY)

	x := 80
	y := startY + 30
	cellSize := 20

	// Statements
	stmtColor := h.getCoverageColor(totals.Statements.Percentage)
	h.drawRect(img, x, y, cellSize, cellSize, stmtColor)

	stmtText := fmt.Sprintf("Statement Coverage %s (%.1f%%)", totals.Statements.Coverage, totals.Statements.Percentage)
	h.drawText(img, stmtText, x+cellSize+10, y+15, color.RGBA{0, 0, 0, 255}, false)

	// Blocks
	y += 30
	blockColor := h.getCoverageColor(totals.Blocks.Percentage)
	h.drawRect(img, x, y, cellSize, cellSize, blockColor)

	blockText := fmt.Sprintf("Block Coverage     %s (%.1f%%)", totals.Blocks.Coverage, totals.Blocks.Percentage)
	h.drawText(img, blockText, x+cellSize+10, y+15, color.RGBA{0, 0, 0, 255}, false)
}

func (h *PNGHeatmap) drawLegend(img *image.RGBA) {
	legendX := 60
	legendY := 60

	h.drawText(img, "Coverage Legend:", legendX, legendY, color.RGBA{0, 0, 0, 255}, false)
	legendY += 25

	legend := []struct {
		label string
		color color.RGBA
	}{
		{"0-29%   Critical (High Priority)", color.RGBA{156, 39, 176, 255}},
		{"30-49%  Poor (Medium Priority)", color.RGBA{244, 67, 54, 255}},
		{"50-69%  Fair (Low Priority)", color.RGBA{255, 152, 0, 255}},
		{"70-89%  Good (Maintain)", color.RGBA{255, 193, 7, 255}},
		{"90-100% Excellent (Well Covered)", color.RGBA{76, 175, 80, 255}},
	}

	// Arrange legend vertically to avoid overlapping
	for _, item := range legend {
		h.drawRect(img, legendX, legendY-8, 15, 12, item.color)
		h.drawText(img, item.label, legendX+25, legendY, color.RGBA{0, 0, 0, 255}, false)
		legendY += 18 // Move to next line
	}
}

func (h *PNGHeatmap) getCoverageColor(percentage float64) color.RGBA {
	switch {
	case percentage >= 90:
		return color.RGBA{76, 175, 80, 255} // Green
	case percentage >= 70:
		return color.RGBA{255, 193, 7, 255} // Yellow
	case percentage >= 50:
		return color.RGBA{255, 152, 0, 255} // Orange
	case percentage >= 30:
		return color.RGBA{244, 67, 54, 255} // Red
	default:
		return color.RGBA{156, 39, 176, 255} // Purple
	}
}

func (h *PNGHeatmap) drawRect(img *image.RGBA, x, y, width, height int, c color.RGBA) {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			if x+dx < h.width && y+dy < h.height && x+dx >= 0 && y+dy >= 0 {
				img.Set(x+dx, y+dy, c)
			}
		}
	}
}

func (h *PNGHeatmap) truncateFilename(filename string) string {
	// Truncate file names to 6 characters
	if len(filename) <= 6 {
		return filename
	}
	return filename[:6]
}

func (h *PNGHeatmap) truncatePackageName(packageName string) string {
	// Prefix truncate to show last 6 characters for package names
	if len(packageName) <= 6 {
		return packageName
	}
	return packageName[len(packageName)-6:]
}

func (h *PNGHeatmap) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}
