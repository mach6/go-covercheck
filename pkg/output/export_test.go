package output

import "image/color"

// Re-exports of internal helpers for black-box tests in package output_test.

var (
	BoxStyleFor          = boxStyleFor
	GetTableStyle        = getTableStyle
	GetHistoryTableStyle = getHistoryTableStyle
	TrimWithEllipsis     = trimWithEllipsis
	ApplyTableWidths     = applyTableWidths
	MatchesInspectFile   = matchesInspectFile
	RenderHeatmap        = renderHeatmap
	FitHeatmapText       = fit
)

const FixedColumnWidth = fixedColumnWidth

const (
	HeatNone = int(heatNone)
	HeatLow  = int(heatLow)
	HeatMid  = int(heatMid)
	HeatMet  = int(heatMet)

	PNGWidth   = pngWidth
	PNGRowStep = pngCellH + pngGap
)

func HeatColor(h int) color.RGBA { return heat(h).rgba() }
