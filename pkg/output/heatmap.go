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
	heatmapLabelWidth   = 2  // "L " before the percentage
	heatmapCellPadding  = 2  // one column of padding on each side of the content
	heatmapCellGap      = 1
)

// heatFor classifies actual against goal. A failed item is never reported as
// met, so an item below its goal is not painted green when it rounds up to it.
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
	name  string
	pct   float64
	heat  heat
	label string // metric letter shown before the percentage, when the cell is not tied to one metric
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
	if r.inner < heatmapMinInner+heatmapLabelWidth {
		c.label = "" // no room for the metric letter; the percentage is never cut
	}
	pct := c.text(!r.colors)
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

func (r *heatmapRenderer) totals(t compute.Totals, showFunctions bool) {
	_, _ = fmt.Fprintln(r.w, r.bold("By Total"))
	for _, row := range heatTotals(t, showFunctions) {
		swatch := r.paint(row.heat.glyph(), row.heat)
		_, _ = fmt.Fprintf(r.w, "%s %-10s %6.1f%%  %s\n", swatch, row.name, row.pct, row.coverage)
	}
	_, _ = fmt.Fprintln(r.w)
}

// heatMetric picks one coverage metric out of a result.
type heatMetric struct {
	key   string // config.HeatmapMetric* value
	name  string
	short string // the letter used in the failure summary
	of    func(compute.By) (pct, goal float64)
}

var heatMetrics = []heatMetric{
	{config.HeatmapMetricStatements, colStatements, "S",
		func(b compute.By) (float64, float64) { return b.StatementPercentage, b.StatementThreshold }},
	{config.HeatmapMetricBlocks, colBlocks, "B",
		func(b compute.By) (float64, float64) { return b.BlockPercentage, b.BlockThreshold }},
	{config.HeatmapMetricLines, colLines, "L",
		func(b compute.By) (float64, float64) { return b.LinePercentage, b.LineThreshold }},
	{config.HeatmapMetricFunctions, colFunctions, "F",
		func(b compute.By) (float64, float64) { return b.FunctionPercentage, b.FunctionThreshold }},
}

// heatCellFor colors an item by one metric against that metric's own threshold, so
// an item below its goal is never painted green.
func heatCellFor(name string, by compute.By, m heatMetric, label bool) heatCell {
	pct, goal := m.of(by)
	c := heatCell{name: name, pct: pct, heat: heatFor(pct, goal, pct < goal)}
	if label {
		c.label = m.short
	}
	return c
}

// worstHeatCell colors an item by the metric furthest below its threshold,
// measured as a share of that threshold, and labels the cell with that metric.
// Metrics without a threshold are ignored; an item with none is grey and shows
// its statement coverage.
func worstHeatCell(name string, by compute.By) heatCell {
	worst, worstShare := -1, 0.0
	for i, m := range heatMetrics {
		pct, goal := m.of(by)
		if goal <= 0 {
			continue
		}
		if share := pct / goal; worst < 0 || share < worstShare {
			worst, worstShare = i, share
		}
	}
	if worst < 0 {
		return heatCellFor(name, by, heatMetrics[0], true)
	}
	return heatCellFor(name, by, heatMetrics[worst], true)
}

// heatSections builds the file and package grids shared by the text and PNG
// heat maps. metric is a config.HeatmapMetric* value; empty means worst.
func heatSections(results compute.Results, metric string) (heatSection, heatSection) {
	cell := worstHeatCell
	suffix := ""
	for _, m := range heatMetrics {
		if m.key == metric {
			cell = func(name string, by compute.By) heatCell { return heatCellFor(name, by, m, false) }
			suffix = " - " + m.name
		}
	}
	files := heatSection{title: "By File" + suffix, cells: make([]heatCell, 0, len(results.ByFile))}
	for _, f := range results.ByFile {
		files.cells = append(files.cells, cell(f.File, f.By))
	}
	pkgs := heatSection{title: "By Package" + suffix, cells: make([]heatCell, 0, len(results.ByPackage))}
	for _, p := range results.ByPackage {
		pkgs.cells = append(pkgs.cells, cell(p.Package, p.By))
	}
	return files, pkgs
}

// heatSection is one titled grid of cells.
type heatSection struct {
	title string
	cells []heatCell
}

// text is the second line of a cell: the metric letter when there is one, then the percentage.
func (c heatCell) text(withGlyph bool) string {
	s := fmt.Sprintf("%.1f%%", c.pct)
	if c.label != "" {
		s = c.label + " " + s
	}
	if withGlyph {
		s = c.heat.glyph() + " " + s
	}
	return s
}

type heatTotal struct {
	name     string
	coverage string
	pct      float64
	heat     heat
}

// heatTotals lists the total rows. Functions appears only when showFunctions is
// set (any function threshold is in effect, as in the table); it is grey when
// the total's own threshold is disabled.
func heatTotals(t compute.Totals, showFunctions bool) []heatTotal {
	rows := []heatTotal{
		{colStatements, t.Statements.Coverage, t.Statements.Percentage,
			heatFor(t.Statements.Percentage, t.Statements.Threshold, t.Statements.Failed)},
		{colBlocks, t.Blocks.Coverage, t.Blocks.Percentage,
			heatFor(t.Blocks.Percentage, t.Blocks.Threshold, t.Blocks.Failed)},
		{colLines, t.Lines.Coverage, t.Lines.Percentage,
			heatFor(t.Lines.Percentage, t.Lines.Threshold, t.Lines.Failed)},
	}
	if showFunctions {
		rows = append(rows, heatTotal{colFunctions, t.Functions.Coverage, t.Functions.Percentage,
			heatFor(t.Functions.Percentage, t.Functions.Threshold, t.Functions.Failed)})
	}
	return rows
}

// renderHeatmap writes a grid heat map of coverage results to w. Cells keep the
// order of the results so --sort-by and --sort-order apply.
func renderHeatmap(w io.Writer, results compute.Results, cfg *config.Config) {
	r := newHeatmapRenderer(w, cfg)

	files, pkgs := heatSections(results, cfg.HeatmapMetric)

	r.legend()
	r.section(files.title, files.cells)
	r.section(pkgs.title, pkgs.cells)
	r.totals(results.ByTotal, cfg.HasFunctionThreshold())
}

func renderHeatmapToStdout(results compute.Results, cfg *config.Config) {
	if cfg.NoTable {
		return
	}
	renderHeatmap(os.Stdout, results, cfg)
}
