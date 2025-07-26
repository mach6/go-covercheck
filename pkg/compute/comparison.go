package compute

// BuildComparisonData creates comparison data between current results and a historical entry.
// It takes the necessary parameters without directly importing the history package to avoid cycles.
func BuildComparisonData(ref string, commit string, historicalResults Results, currentResults Results) *ComparisonData {
	comparison := &ComparisonData{
		Ref:     ref,
		Commit:  commit,
		Results: make([]ComparisonResult, 0),
	}

	// Compare by file
	for _, curr := range currentResults.ByFile {
		for _, prev := range historicalResults.ByFile {
			if curr.File == prev.File {
				statementsDelta := curr.StatementPercentage - prev.StatementPercentage 
				blocksDelta := curr.BlockPercentage - prev.BlockPercentage
				if statementsDelta != 0 || blocksDelta != 0 {
					comparison.Results = append(comparison.Results, ComparisonResult{
						Name: curr.File,
						Type: "file",
						Delta: ComparisonDelta{
							StatementsDelta: statementsDelta,
							BlocksDelta:     blocksDelta,
						},
					})
				}
				break
			}
		}
	}

	// Compare by package
	for _, curr := range currentResults.ByPackage {
		for _, prev := range historicalResults.ByPackage {
			if curr.Package == prev.Package {
				statementsDelta := curr.StatementPercentage - prev.StatementPercentage
				blocksDelta := curr.BlockPercentage - prev.BlockPercentage
				if statementsDelta != 0 || blocksDelta != 0 {
					comparison.Results = append(comparison.Results, ComparisonResult{
						Name: curr.Package,
						Type: "package",
						Delta: ComparisonDelta{
							StatementsDelta: statementsDelta,
							BlocksDelta:     blocksDelta,
						},
					})
				}
				break
			}
		}
	}

	// Compare totals
	statementsDelta := currentResults.ByTotal.Statements.Percentage - historicalResults.ByTotal.Statements.Percentage
	blocksDelta := currentResults.ByTotal.Blocks.Percentage - historicalResults.ByTotal.Blocks.Percentage
	if statementsDelta != 0 || blocksDelta != 0 {
		comparison.Results = append(comparison.Results, ComparisonResult{
			Name: "total",
			Type: "total",
			Delta: ComparisonDelta{
				StatementsDelta: statementsDelta,
				BlocksDelta:     blocksDelta,
			},
		})
	}

	// Only return comparison data if there are actual differences
	if len(comparison.Results) == 0 {
		return nil
	}

	return comparison
}