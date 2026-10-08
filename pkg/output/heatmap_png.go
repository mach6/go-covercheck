package output

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"

	"github.com/mach6/go-covercheck/pkg/compute"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Pixel layout of the PNG heat map. The width is fixed so that images from
// different runs line up; only the height grows with the number of items.
const (
	pngCols      = 5
	pngCellW     = 168
	pngCellH     = 40
	pngGap       = 8
	pngMargin    = 16
	pngPad       = 8  // text inset inside a cell
	pngLineH     = 13 // basicfont.Face7x13 line height
	pngGlyphW    = 7
	pngSwatch    = 14
	pngMaxHeight = 16384
	pngWidth     = 2*pngMargin + pngCols*pngCellW + (pngCols-1)*pngGap
	pngNameChars = (pngCellW - 2*pngPad) / pngGlyphW
	pngDoubleGap = 2 * pngGap
	pngHalfGap   = pngGap / 2
	pngDetailY   = pngPad + pngLineH + 2
)

var (
	pngBackground = color.RGBA{R: 255, G: 255, B: 255, A: 255} //nolint:mnd // white
	pngInk        = color.RGBA{A: 255}                         //nolint:mnd // black
)

//nolint:mnd // palette
func (h heat) rgba() color.RGBA {
	switch h {
	case heatLow:
		return color.RGBA{R: 192, G: 57, B: 43, A: 255}
	case heatMid:
		return color.RGBA{R: 241, G: 196, B: 15, A: 255}
	case heatMet:
		return color.RGBA{R: 96, G: 190, B: 110, A: 255}
	default:
		return color.RGBA{R: 120, G: 120, B: 120, A: 255}
	}
}

//nolint:mnd // palette
func (h heat) textRGBA() color.RGBA {
	if h == heatLow || h == heatNone {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	return pngInk
}

type pngCanvas struct {
	img *image.RGBA
}

func (c *pngCanvas) text(s string, x, y int, col color.RGBA) {
	d := font.Drawer{
		Dst:  c.img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  fixed.P(x, y+basicfont.Face7x13.Ascent),
	}
	d.DrawString(s)
}

func (c *pngCanvas) rect(x, y, w, h int, col color.RGBA) {
	draw.Draw(c.img, image.Rect(x, y, x+w, y+h), image.NewUniform(col), image.Point{}, draw.Src)
}

func (c *pngCanvas) cell(x, y int, name, detail string, h heat) {
	c.rect(x, y, pngCellW, pngCellH, h.rgba())
	c.text(name, x+pngPad, y+pngPad-1, h.textRGBA())
	c.text(detail, x+pngPad, y+pngDetailY, h.textRGBA())
}

// pngText makes s drawable with basicfont, which only has Latin glyphs, and
// shortens it to max characters keeping the tail so a path keeps its file name.
func pngText(s string, limit int) string {
	r := []rune(s)
	for i, c := range r {
		if c < ' ' || c > '~' {
			r[i] = '?'
		}
	}
	if len(r) > limit {
		return "~" + string(r[len(r)-limit+1:])
	}
	return string(r)
}

func pngRows(n int) int {
	return (n + pngCols - 1) / pngCols
}

func pngSectionHeight(cells int) int {
	if cells == 0 {
		return 0
	}
	return pngLineH + pngGap + pngRows(cells)*(pngCellH+pngGap) + pngGap
}

func (c *pngCanvas) section(title string, y int, cells []heatCell) int {
	if len(cells) == 0 {
		return y
	}
	c.text(title, pngMargin, y, pngInk)
	y += pngLineH + pngGap
	for i, cell := range cells {
		x := pngMargin + (i%pngCols)*(pngCellW+pngGap)
		cy := y + (i/pngCols)*(pngCellH+pngGap)
		c.cell(x, cy, pngText(cell.name, pngNameChars), fmt.Sprintf("%.1f%%", cell.pct), cell.heat)
	}
	return y + pngRows(len(cells))*(pngCellH+pngGap) + pngGap
}

// renderHeatmapImage draws the heat map. The colors follow the text heat map.
func renderHeatmapImage(results compute.Results) (image.Image, error) {
	files, pkgs := heatCells(results)
	totals := heatTotals(results.ByTotal)
	empty := isEmptyResults(results)

	height := pngMargin + pngLineH + pngGap + pngSwatch + pngDoubleGap
	if empty {
		height += pngLineH
	} else {
		height += pngSectionHeight(len(files)) + pngSectionHeight(len(pkgs)) +
			pngSectionHeight(len(totals))
	}
	height += pngMargin
	if height > pngMaxHeight {
		return nil, fmt.Errorf("heat map for %d files and %d packages is %d pixels tall, over the %d pixel limit",
			len(files), len(pkgs), height, pngMaxHeight)
	}

	c := &pngCanvas{img: image.NewRGBA(image.Rect(0, 0, pngWidth, height))}
	c.rect(0, 0, pngWidth, height, pngBackground)

	y := pngMargin
	c.text("go-covercheck coverage heat map", pngMargin, y, pngInk)
	y += pngLineH + pngGap
	x := pngMargin
	for _, h := range heatLevels {
		label := h.label()
		c.rect(x, y, pngSwatch, pngSwatch, h.rgba())
		c.text(label, x+pngSwatch+pngHalfGap, y, pngInk)
		x += pngSwatch + pngHalfGap + len(label)*pngGlyphW + pngDoubleGap
	}
	y += pngSwatch + pngDoubleGap

	if empty {
		c.text("No coverage results to display", pngMargin, y, pngInk)
		return c.img, nil
	}

	y = c.section("By File", y, files)
	y = c.section("By Package", y, pkgs)
	c.text("By Total", pngMargin, y, pngInk)
	y += pngLineH + pngGap
	for i, t := range totals {
		c.cell(pngMargin+i*(pngCellW+pngGap), y, t.name,
			strings.TrimSpace(fmt.Sprintf("%.1f%%  %s", t.pct, t.coverage)), t.heat)
	}
	return c.img, nil
}

// WriteHeatmapPNG renders the coverage heat map and writes it to path.
// Nothing is written to path unless the image renders.
func WriteHeatmapPNG(path string, results compute.Results) error {
	img, err := renderHeatmapImage(results)
	if err != nil {
		return err
	}
	f, err := os.Create(path) //nolint:gosec // path is the user-chosen output location
	if err != nil {
		return fmt.Errorf("create heat map: %w", err)
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return fmt.Errorf("write heat map: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("write heat map: %w", err)
	}
	return nil
}
