package output

// Re-exports of internal helpers for black-box tests in package output_test.

var (
	BoxStyleFor          = boxStyleFor
	GetTableStyle        = getTableStyle
	GetHistoryTableStyle = getHistoryTableStyle
	TrimWithEllipsis     = trimWithEllipsis
	ApplyTableWidths     = applyTableWidths
	MatchesInspectFile   = matchesInspectFile
)

const FixedColumnWidth = fixedColumnWidth
