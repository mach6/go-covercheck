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

// Generate creates a PNG heat map image of the coverage results.
func (h *PNGHeatmap) Generate(results compute.Results) error {
	// Calculate required height based on content
	requiredHeight := h.calculateRequiredHeight(results)
	h.height = requiredHeight
	h.width = 1000 // Make it wider to accommodate more content
	
	// Create the image
	img := image.NewRGBA(image.Rect(0, 0, h.width, h.height))
	
	// Fill background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{245, 245, 245, 255}}, image.Point{}, draw.Src)
	
	// Draw title
	h.drawTitle(img, "Coverage Heat Map")
	
	// Calculate layout
	startY := 80
	
	// Draw package heat map only (no files)
	if len(results.ByPackage) > 0 {
		h.drawSectionTitle(img, "Packages by Statement Coverage", 60, startY)
		startY += 30
		startY = h.drawPackageHeatmap(img, results.ByPackage, startY)
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

func (h *PNGHeatmap) calculateRequiredHeight(results compute.Results) int {
	baseHeight := 200 // Title, margins, legend space
	
	// Package section
	packageHeight := len(results.ByPackage) * 22 // 22 pixels per package
	if packageHeight > 0 {
		packageHeight += 60 // Section title and spacing
	}
	
	// Summary section
	summaryHeight := 100 // Overall coverage section
	
	// Legend
	legendHeight := 120
	
	totalHeight := baseHeight + packageHeight + summaryHeight + legendHeight
	
	// Ensure minimum height
	if totalHeight < 400 {
		totalHeight = 400
	}
	
	return totalHeight
}

func (h *PNGHeatmap) drawPackageHeatmap(img *image.RGBA, packages []compute.ByPackage, startY int) int {
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
	
	// Draw all packages without truncation
	for _, pkg := range sortedPackages {
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
		packageName := h.truncateText(pkg.Package, 50) // Longer since we have more width
		h.drawText(img, packageName, x+barWidth+10, y, color.RGBA{0, 0, 0, 255}, false)
		
		// Percentage
		percentText := fmt.Sprintf("%.1f%%", pkg.StatementPercentage)
		h.drawText(img, percentText, x+barWidth+400, y, color.RGBA{0, 0, 0, 255}, false)
		
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
	
	stmtText := fmt.Sprintf("Statement Coverage %s (%.1f%%)", totals.Statements.Coverage, totals.Statements.Percentage)
	h.drawText(img, stmtText, x+barWidth+10, y, color.RGBA{0, 0, 0, 255}, false)
	
	// Blocks
	y += 25
	blockColor := h.getCoverageColor(totals.Blocks.Percentage)
	blockLength := int(float64(barWidth) * totals.Blocks.Percentage / 100.0)
	
	h.drawRect(img, x, y-8, barWidth, 12, color.RGBA{220, 220, 220, 255})
	if blockLength > 0 {
		h.drawRect(img, x, y-8, blockLength, 12, blockColor)
	}
	
	blockText := fmt.Sprintf("Block Coverage     %s (%.1f%%)", totals.Blocks.Coverage, totals.Blocks.Percentage)
	h.drawText(img, blockText, x+barWidth+10, y, color.RGBA{0, 0, 0, 255}, false)
}

func (h *PNGHeatmap) drawLegend(img *image.RGBA) {
	legendX := h.width - 250 // Adjusted for wider image
	legendY := 100
	
	h.drawText(img, "Statement Coverage Legend:", legendX, legendY, color.RGBA{0, 0, 0, 255}, false)
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