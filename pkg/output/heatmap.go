package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/math"
)

// heat is a coverage level relative to a threshold goal. The levels mirror the
// color legend used by severityColor.
type heat int

const (
	heatNone heat = iota // no goal to compare against
	heatLow              // <= 50% of the goal
	heatMid              // > 50% and <= 99% of the goal
	heatMet              // goal met
)

const (
	heatLowCeiling = 50
	heatMidCeiling = 99

	defaultHeatmapWidth = 80
	heatmapCellInner    = 20 // target display columns for a cell's content; wide enough to tell similar paths apart
	heatmapMinInner     = 8  // fits "▒ 100.0%"
	heatmapCellPadding  = 2  // one column of padding on each side of the content
	heatmapCellGap      = 1
)

// heatFor classifies actual against goal. A failed item is never reported as
// met, so a file that fails on blocks or lines is not painted green because its
// statement coverage happens to reach the goal.
func heatFor(actual, goal float64, failed bool) heat {
	if goal <= 0 {
		return heatNone
	}
	pct := math.PercentFloat(actual, goal)
	switch {
	case pct <= heatLowCeiling:
		return heatLow
	case pct <= heatMidCeiling, failed:
		return heatMid
	default:
		return heatMet
	}
}

var heatLevels = []heat{heatLow, heatMid, heatMet, heatNone}

func (h heat) label() string {
	switch h {
	case heatLow:
		return "<= 50% of threshold"
	case heatMid:
		return "below threshold"
	case heatMet:
		return "threshold met"
	default:
		return "no threshold"
	}
}

// glyph distinguishes levels without relying on color.
func (h heat) glyph() string {
	switch h {
	case heatLow:
		return "░"
	case heatMid:
		return "▒"
	case heatMet:
		return "█"
	default:
		return "·"
	}
}

func (h heat) attrs() []color.Attribute {
	switch h {
	case heatLow:
		return []color.Attribute{color.BgRed, color.FgWhite}
	case heatMid:
		return []color.Attribute{color.BgYellow, color.FgBlack}
	case heatMet:
		return []color.Attribute{color.BgGreen, color.FgBlack}
	default:
		return []color.Attribute{color.BgHiBlack, color.FgWhite}
	}
}

type heatCell struct {
	name string
	pct  float64
	heat heat
}

type heatmapRenderer struct {
	w      io.Writer
	colors bool
	width  int // total terminal columns available
	inner  int // content columns per cell
}

func newHeatmapRenderer(w io.Writer, cfg *config.Config) *heatmapRenderer {
	width := cfg.TerminalWidth
	if width <= 0 {
		width = defaultHeatmapWidth
	}
	inner := heatmapCellInner
	if avail := width - heatmapCellPadding; avail < inner {
		inner = max(avail, heatmapMinInner)
	}
	return &heatmapRenderer{w: w, colors: !cfg.NoColor && !color.NoColor, width: width, inner: inner}
}

// paint pads s with one column on each side and applies the heat color, if enabled.
func (r *heatmapRenderer) paint(s string, h heat) string {
	s = " " + s + " "
	if !r.colors {
		return s
	}
	return color.New(h.attrs()...).Sprint(s)
}

// fit pads or truncates s to exactly width display columns. Truncation drops
// the head and keeps the tail, so a path keeps its file name.
func fit(s string, width int) string {
	if text.StringWidthWithoutEscSequences(s) > width {
		runes := []rune(text.StripEscape(s))
		cut := 0
		for cut < len(runes) && 1+text.StringWidthWithoutEscSequences(string(runes[cut:])) > width {
			cut++
		}
		s = "…" + string(runes[cut:])
	}
	return text.Pad(s, width, ' ')
}

func (r *heatmapRenderer) bold(s string) string {
	if !r.colors {
		return s
	}
	return text.Bold.Sprint(s)
}

func (r *heatmapRenderer) cell(c heatCell) [2]string {
	pct := fmt.Sprintf("%s %.1f%%", c.heat.glyph(), c.pct)
	if r.colors {
		pct = fmt.Sprintf("%.1f%%", c.pct)
	}
	return [2]string{
		r.paint(fit(c.name, r.inner), c.heat),
		r.paint(fit(pct, r.inner), c.heat),
	}
}

func (r *heatmapRenderer) section(title string, cells []heatCell) {
	if len(cells) == 0 {
		return
	}
	cols := max((r.width+heatmapCellGap)/(r.inner+heatmapCellPadding+heatmapCellGap), 1)

	var b strings.Builder
	b.WriteString(r.bold(title) + "\n")
	for start := 0; start < len(cells); start += cols {
		if start > 0 {
			b.WriteString("\n")
		}
		end := min(start+cols, len(cells))
		var lines [2][]string
		for _, c := range cells[start:end] {
			l := r.cell(c)
			lines[0] = append(lines[0], l[0])
			lines[1] = append(lines[1], l[1])
		}
		for _, l := range lines {
			line := strings.Join(l, strings.Repeat(" ", heatmapCellGap))
			if !r.colors {
				line = strings.TrimRight(line, " ")
			}
			b.WriteString(line + "\n")
		}
	}
	_, _ = fmt.Fprintln(r.w, b.String())
}

// legend lists the levels, wrapping onto further lines to fit the width.
func (r *heatmapRenderer) legend() {
	var b strings.Builder
	lineWidth := 0
	for i, h := range heatLevels {
		item := r.paint(h.glyph(), h) + " " + h.label()
		itemWidth := text.StringWidthWithoutEscSequences(item)
		switch {
		case i == 0:
		case lineWidth+2+itemWidth > r.width:
			b.WriteString("\n")
			lineWidth = 0
		default:
			b.WriteString("  ")
			lineWidth += 2
		}
		b.WriteString(item)
		lineWidth += itemWidth
	}
	_, _ = fmt.Fprintln(r.w, b.String()+"\n")
}

func (r *heatmapRenderer) totals(t compute.Totals) {
	_, _ = fmt.Fprintln(r.w, r.bold("By Total"))
	for _, row := range heatTotals(t) {
		swatch := r.paint(row.heat.glyph(), row.heat)
		_, _ = fmt.Fprintf(r.w, "%s %-10s %6.1f%%  %s\n", swatch, row.name, row.pct, row.coverage)
	}
	_, _ = fmt.Fprintln(r.w)
}

// heatCells builds the file and package cells shared by the text and PNG heat maps.
func heatCells(results compute.Results) ([]heatCell, []heatCell) {
	files := make([]heatCell, 0, len(results.ByFile))
	for _, f := range results.ByFile {
		files = append(files, heatCell{f.File, f.StatementPercentage,
			heatFor(f.StatementPercentage, f.StatementThreshold, f.Failed)})
	}
	pkgs := make([]heatCell, 0, len(results.ByPackage))
	for _, p := range results.ByPackage {
		pkgs = append(pkgs, heatCell{p.Package, p.StatementPercentage,
			heatFor(p.StatementPercentage, p.StatementThreshold, p.Failed)})
	}
	return files, pkgs
}

type heatTotal struct {
	name     string
	coverage string
	pct      float64
	heat     heat
}

func heatTotals(t compute.Totals) []heatTotal {
	return []heatTotal{
		{colStatements, t.Statements.Coverage, t.Statements.Percentage,
			heatFor(t.Statements.Percentage, t.Statements.Threshold, t.Statements.Failed)},
		{colBlocks, t.Blocks.Coverage, t.Blocks.Percentage,
			heatFor(t.Blocks.Percentage, t.Blocks.Threshold, t.Blocks.Failed)},
		{colLines, t.Lines.Coverage, t.Lines.Percentage,
			heatFor(t.Lines.Percentage, t.Lines.Threshold, t.Lines.Failed)},
	}
}

// renderHeatmap writes a grid heat map of coverage results to w. Cells keep the
// order of the results so --sort-by and --sort-order apply.
func renderHeatmap(w io.Writer, results compute.Results, cfg *config.Config) {
	r := newHeatmapRenderer(w, cfg)

	files, pkgs := heatCells(results)

	r.legend()
	r.section("By File", files)
	r.section("By Package", pkgs)
	r.totals(results.ByTotal)
}

func renderHeatmapToStdout(results compute.Results, cfg *config.Config) {
	if cfg.NoTable {
		return
	}
	renderHeatmap(os.Stdout, results, cfg)
}
