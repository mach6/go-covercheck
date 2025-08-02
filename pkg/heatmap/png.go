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
		width:  800,
		height: 600,
	}
}

// Generate creates a PNG heat map image of the coverage results.
func (h *PNGHeatmap) Generate(results compute.Results) error {
	// Create the image
	img := image.NewRGBA(image.Rect(0, 0, h.width, h.height))
	
	// Fill background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{245, 245, 245, 255}}, image.Point{}, draw.Src)
	
	// Draw title
	h.drawTitle(img, "Coverage Heat Map")
	
	// Calculate layout
	startY := 80
	fileHeight := h.calculateFileHeight(len(results.ByFile))
	packageHeight := h.calculatePackageHeight(len(results.ByPackage))
	
	// Draw file heat map
	if len(results.ByFile) > 0 {
		h.drawSectionTitle(img, "Files by Coverage", 60, startY)
		startY += 30
		startY = h.drawFileHeatmap(img, results.ByFile, startY, fileHeight)
		startY += 20
	}
	
	// Draw package heat map
	if len(results.ByPackage) > 0 {
		h.drawSectionTitle(img, "Packages by Coverage", 60, startY)
		startY += 30
		startY = h.drawPackageHeatmap(img, results.ByPackage, startY, packageHeight)
		startY += 20
	}
	
	// Draw summary
	h.drawSummary(img, results.ByTotal, startY)
	
	// Draw legend
	h.drawLegend(img)
	
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

func (h *PNGHeatmap) calculateFileHeight(fileCount int) int {
	maxHeight := 200
	itemHeight := 20
	totalHeight := fileCount * itemHeight
	if totalHeight > maxHeight {
		return maxHeight
	}
	return totalHeight
}

func (h *PNGHeatmap) calculatePackageHeight(packageCount int) int {
	maxHeight := 150
	itemHeight := 25
	totalHeight := packageCount * itemHeight
	if totalHeight > maxHeight {
		return maxHeight
	}
	return totalHeight
}

func (h *PNGHeatmap) drawFileHeatmap(img *image.RGBA, files []compute.ByFile, startY, maxHeight int) int {
	if len(files) == 0 {
		return startY
	}
	
	// Sort files by coverage percentage (descending)
	sortedFiles := make([]compute.ByFile, len(files))
	copy(sortedFiles, files)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].StatementPercentage > sortedFiles[j].StatementPercentage
	})
	
	itemHeight := 18
	x := 80
	y := startY
	barWidth := 200
	
	visibleFiles := len(sortedFiles)
	if maxHeight/itemHeight < visibleFiles {
		visibleFiles = maxHeight / itemHeight
	}
	
	for i := 0; i < visibleFiles; i++ {
		file := sortedFiles[i]
		
		// Draw coverage bar
		barColor := h.getCoverageColor(file.StatementPercentage)
		barLength := int(float64(barWidth) * file.StatementPercentage / 100.0)
		
		// Background bar
		h.drawRect(img, x, y-8, barWidth, 12, color.RGBA{220, 220, 220, 255})
		// Coverage bar
		if barLength > 0 {
			h.drawRect(img, x, y-8, barLength, 12, barColor)
		}
		
		// File name
		fileName := h.truncateText(file.File, 35)
		h.drawText(img, fileName, x+barWidth+10, y, color.RGBA{0, 0, 0, 255}, false)
		
		// Percentage
		percentText := fmt.Sprintf("%.1f%%", file.StatementPercentage)
		h.drawText(img, percentText, x+barWidth+300, y, color.RGBA{0, 0, 0, 255}, false)
		
		y += itemHeight
	}
	
	// Show truncation message if needed
	if len(sortedFiles) > visibleFiles {
		truncMsg := fmt.Sprintf("... and %d more files", len(sortedFiles)-visibleFiles)
		h.drawText(img, truncMsg, x, y, color.RGBA{128, 128, 128, 255}, false)
		y += itemHeight
	}
	
	return y
}

func (h *PNGHeatmap) drawPackageHeatmap(img *image.RGBA, packages []compute.ByPackage, startY, maxHeight int) int {
	if len(packages) == 0 {
		return startY
	}
	
	// Sort packages by coverage percentage (descending)
	sortedPackages := make([]compute.ByPackage, len(packages))
	copy(sortedPackages, packages)
	sort.Slice(sortedPackages, func(i, j int) bool {
		return sortedPackages[i].StatementPercentage > sortedPackages[j].StatementPercentage
	})
	
	itemHeight := 22
	x := 80
	y := startY
	barWidth := 200
	
	visiblePackages := len(sortedPackages)
	if maxHeight/itemHeight < visiblePackages {
		visiblePackages = maxHeight / itemHeight
	}
	
	for i := 0; i < visiblePackages; i++ {
		pkg := sortedPackages[i]
		
		// Draw coverage bar
		barColor := h.getCoverageColor(pkg.StatementPercentage)
		barLength := int(float64(barWidth) * pkg.StatementPercentage / 100.0)
		
		// Background bar
		h.drawRect(img, x, y-8, barWidth, 12, color.RGBA{220, 220, 220, 255})
		// Coverage bar
		if barLength > 0 {
			h.drawRect(img, x, y-8, barLength, 12, barColor)
		}
		
		// Package name
		packageName := h.truncateText(pkg.Package, 35)
		h.drawText(img, packageName, x+barWidth+10, y, color.RGBA{0, 0, 0, 255}, false)
		
		// Percentage
		percentText := fmt.Sprintf("%.1f%%", pkg.StatementPercentage)
		h.drawText(img, percentText, x+barWidth+300, y, color.RGBA{0, 0, 0, 255}, false)
		
		y += itemHeight
	}
	
	// Show truncation message if needed
	if len(sortedPackages) > visiblePackages {
		truncMsg := fmt.Sprintf("... and %d more packages", len(sortedPackages)-visiblePackages)
		h.drawText(img, truncMsg, x, y, color.RGBA{128, 128, 128, 255}, false)
		y += itemHeight
	}
	
	return y
}

func (h *PNGHeatmap) drawSummary(img *image.RGBA, totals compute.Totals, startY int) {
	h.drawSectionTitle(img, "Overall Coverage", 60, startY)
	
	x := 80
	y := startY + 30
	barWidth := 200
	
	// Statements
	stmtColor := h.getCoverageColor(totals.Statements.Percentage)
	stmtLength := int(float64(barWidth) * totals.Statements.Percentage / 100.0)
	
	h.drawRect(img, x, y-8, barWidth, 12, color.RGBA{220, 220, 220, 255})
	if stmtLength > 0 {
		h.drawRect(img, x, y-8, stmtLength, 12, stmtColor)
	}
	
	stmtText := fmt.Sprintf("Statements %s (%.1f%%)", totals.Statements.Coverage, totals.Statements.Percentage)
	h.drawText(img, stmtText, x+barWidth+10, y, color.RGBA{0, 0, 0, 255}, false)
	
	// Blocks
	y += 25
	blockColor := h.getCoverageColor(totals.Blocks.Percentage)
	blockLength := int(float64(barWidth) * totals.Blocks.Percentage / 100.0)
	
	h.drawRect(img, x, y-8, barWidth, 12, color.RGBA{220, 220, 220, 255})
	if blockLength > 0 {
		h.drawRect(img, x, y-8, blockLength, 12, blockColor)
	}
	
	blockText := fmt.Sprintf("Blocks     %s (%.1f%%)", totals.Blocks.Coverage, totals.Blocks.Percentage)
	h.drawText(img, blockText, x+barWidth+10, y, color.RGBA{0, 0, 0, 255}, false)
}

func (h *PNGHeatmap) drawLegend(img *image.RGBA) {
	legendX := h.width - 200
	legendY := 100
	
	h.drawText(img, "Coverage Legend:", legendX, legendY, color.RGBA{0, 0, 0, 255}, false)
	legendY += 20
	
	legend := []struct {
		label string
		color color.RGBA
	}{
		{"90-100% Excellent", color.RGBA{76, 175, 80, 255}},
		{"70-89%  Good", color.RGBA{255, 193, 7, 255}},
		{"50-69%  Fair", color.RGBA{255, 152, 0, 255}},
		{"30-49%  Poor", color.RGBA{244, 67, 54, 255}},
		{"0-29%   Critical", color.RGBA{156, 39, 176, 255}},
	}
	
	for _, item := range legend {
		h.drawRect(img, legendX, legendY-8, 15, 12, item.color)
		h.drawText(img, item.label, legendX+25, legendY, color.RGBA{0, 0, 0, 255}, false)
		legendY += 18
	}
}

func (h *PNGHeatmap) getCoverageColor(percentage float64) color.RGBA {
	switch {
	case percentage >= 90:
		return color.RGBA{76, 175, 80, 255}  // Green
	case percentage >= 70:
		return color.RGBA{255, 193, 7, 255}  // Yellow
	case percentage >= 50:
		return color.RGBA{255, 152, 0, 255}  // Orange
	case percentage >= 30:
		return color.RGBA{244, 67, 54, 255}  // Red
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

func (h *PNGHeatmap) truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}