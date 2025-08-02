package heatmap

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"sort"
	"strings"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// FlameGraphPNG represents a PNG flame graph generator.
type FlameGraphPNG struct {
	writer io.Writer
	config *config.Config
	width  int
	height int
}

// NewFlameGraphPNG creates a new PNG flame graph generator.
func NewFlameGraphPNG(writer io.Writer, cfg *config.Config) *FlameGraphPNG {
	return &FlameGraphPNG{
		writer: writer,
		config: cfg,
		width:  1200, // Wide for flame graph visualization
		height: 800,  // Will be calculated based on content
	}
}

// Generate creates a PNG flame graph visualization of the coverage results.
func (f *FlameGraphPNG) Generate(results compute.Results) error {
	// Calculate required height based on content
	f.height = f.calculateRequiredHeight(results)
	
	// Create the image
	img := image.NewRGBA(image.Rect(0, 0, f.width, f.height))
	
	// Fill background with dark theme for flame graph
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{25, 25, 25, 255}}, image.Point{}, draw.Src)
	
	// Draw title
	f.drawTitle(img, "Coverage Flame Graph")
	
	// Generate flame graph visualization
	if len(results.ByPackage) > 0 {
		f.drawFlameGraph(img, results.ByPackage)
	}
	
	// Draw legend
	f.drawLegend(img)
	
	// Encode to PNG
	return png.Encode(f.writer, img)
}

func (f *FlameGraphPNG) calculateRequiredHeight(results compute.Results) int {
	baseHeight := 150 // Title and margins
	
	// Calculate flame graph height based on package hierarchy depth
	maxDepth := f.calculateMaxPackageDepth(results.ByPackage)
	flameHeight := maxDepth * 25 + 100 // 25 pixels per level + padding
	
	legendHeight := 100
	
	totalHeight := baseHeight + flameHeight + legendHeight
	
	// Ensure minimum height
	if totalHeight < 400 {
		totalHeight = 400
	}
	
	return totalHeight
}

func (f *FlameGraphPNG) calculateMaxPackageDepth(packages []compute.ByPackage) int {
	maxDepth := 0
	for _, pkg := range packages {
		depth := len(strings.Split(pkg.Package, "/"))
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func (f *FlameGraphPNG) drawTitle(img *image.RGBA, title string) {
	x := 50
	y := 40
	c := color.RGBA{255, 255, 255, 255} // White text on dark background
	
	f.drawText(img, title, x, y, c, true)
}

func (f *FlameGraphPNG) drawText(img *image.RGBA, text string, x, y int, c color.RGBA, bold bool) {
	face := basicfont.Face7x13
	if bold {
		// For bold, draw text twice with slight offset
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

func (f *FlameGraphPNG) drawFlameGraph(img *image.RGBA, packages []compute.ByPackage) {
	if len(packages) == 0 {
		return
	}
	
	// Sort packages by coverage percentage (descending) for better visualization
	sortedPackages := make([]compute.ByPackage, len(packages))
	copy(sortedPackages, packages)
	sort.Slice(sortedPackages, func(i, j int) bool {
		return sortedPackages[i].StatementPercentage > sortedPackages[j].StatementPercentage
	})
	
	// Group packages by hierarchy level
	hierarchyLevels := f.groupByHierarchyLevel(sortedPackages)
	
	startY := 80
	levelHeight := 25
	
	// Draw each hierarchy level
	for level := 0; level < len(hierarchyLevels); level++ {
		if len(hierarchyLevels[level]) == 0 {
			continue
		}
		
		y := startY + level*levelHeight
		f.drawHierarchyLevel(img, hierarchyLevels[level], y, levelHeight-2)
	}
}

func (f *FlameGraphPNG) groupByHierarchyLevel(packages []compute.ByPackage) [][]compute.ByPackage {
	maxDepth := f.calculateMaxPackageDepth(packages)
	levels := make([][]compute.ByPackage, maxDepth)
	
	for _, pkg := range packages {
		depth := len(strings.Split(pkg.Package, "/")) - 1
		if depth < 0 {
			depth = 0
		}
		if depth < maxDepth {
			levels[depth] = append(levels[depth], pkg)
		}
	}
	
	return levels
}

func (f *FlameGraphPNG) drawHierarchyLevel(img *image.RGBA, packages []compute.ByPackage, y, height int) {
	if len(packages) == 0 {
		return
	}
	
	// Calculate total weight for proportional sizing
	totalWeight := 0.0
	for _, pkg := range packages {
		totalWeight += pkg.StatementPercentage
	}
	
	if totalWeight == 0 {
		return
	}
	
	// Available width for flame graph (minus margins)
	availableWidth := f.width - 100
	startX := 50
	currentX := startX
	
	for _, pkg := range packages {
		// Calculate width proportional to coverage percentage
		proportion := pkg.StatementPercentage / totalWeight
		width := int(float64(availableWidth) * proportion)
		
		if width < 1 {
			width = 1 // Minimum width
		}
		
		// Get color based on coverage percentage
		rectColor := f.getFlameColor(pkg.StatementPercentage)
		
		// Draw flame rectangle
		f.drawRect(img, currentX, y, width, height, rectColor)
		
		// Draw package name if there's enough space
		packageName := f.getShortPackageName(pkg.Package)
		if width > len(packageName)*7 { // Rough character width check
			textColor := f.getTextColor(rectColor)
			f.drawText(img, packageName, currentX+2, y+15, textColor, false)
		}
		
		// Add coverage percentage if there's space
		percentText := fmt.Sprintf("%.1f%%", pkg.StatementPercentage)
		if width > 40 && height > 20 {
			textColor := f.getTextColor(rectColor)
			f.drawText(img, percentText, currentX+2, y+height-5, textColor, false)
		}
		
		currentX += width
	}
}

func (f *FlameGraphPNG) getShortPackageName(packagePath string) string {
	parts := strings.Split(packagePath, "/")
	if len(parts) == 0 {
		return packagePath
	}
	
	// Return the last part of the package path
	lastName := parts[len(parts)-1]
	
	// Truncate if too long
	if len(lastName) > 15 {
		return lastName[:12] + "..."
	}
	
	return lastName
}

func (f *FlameGraphPNG) getFlameColor(percentage float64) color.RGBA {
	// Use a heat map color scheme typical for flame graphs
	switch {
	case percentage >= 90:
		return color.RGBA{255, 99, 71, 255}   // Hot red
	case percentage >= 70:
		return color.RGBA{255, 165, 0, 255}   // Orange
	case percentage >= 50:
		return color.RGBA{255, 215, 0, 255}   // Gold
	case percentage >= 30:
		return color.RGBA{173, 216, 230, 255} // Light blue
	default:
		return color.RGBA{70, 130, 180, 255}  // Steel blue
	}
}

func (f *FlameGraphPNG) getTextColor(backgroundColor color.RGBA) color.RGBA {
	// Calculate brightness of background color
	brightness := float64(backgroundColor.R)*0.299 + float64(backgroundColor.G)*0.587 + float64(backgroundColor.B)*0.114
	
	// Use white text on dark background, black text on light background
	if brightness > 128 {
		return color.RGBA{0, 0, 0, 255} // Black
	}
	return color.RGBA{255, 255, 255, 255} // White
}

func (f *FlameGraphPNG) drawLegend(img *image.RGBA) {
	legendX := f.width - 300
	legendY := 100
	
	f.drawText(img, "Coverage Legend:", legendX, legendY, color.RGBA{255, 255, 255, 255}, false)
	legendY += 20
	
	legend := []struct {
		label string
		color color.RGBA
	}{
		{"90-100% Excellent", color.RGBA{255, 99, 71, 255}},
		{"70-89%  Good", color.RGBA{255, 165, 0, 255}},
		{"50-69%  Fair", color.RGBA{255, 215, 0, 255}},
		{"30-49%  Poor", color.RGBA{173, 216, 230, 255}},
		{"0-29%   Critical", color.RGBA{70, 130, 180, 255}},
	}
	
	for _, item := range legend {
		f.drawRect(img, legendX, legendY-8, 20, 15, item.color)
		f.drawText(img, item.label, legendX+30, legendY, color.RGBA{255, 255, 255, 255}, false)
		legendY += 20
	}
}

func (f *FlameGraphPNG) drawRect(img *image.RGBA, x, y, width, height int, c color.RGBA) {
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			if x+dx < f.width && y+dy < f.height && x+dx >= 0 && y+dy >= 0 {
				img.Set(x+dx, y+dy, c)
			}
		}
	}
}