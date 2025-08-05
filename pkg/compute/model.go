package compute

// HasBy required interface for all descendants of By.
type HasBy interface {
	GetBy() By
}

// By holds cover.Profile information.
type By struct {
	Statements                                   string  `json:"statementCoverage"   yaml:"statementCoverage"`
	Blocks                                       string  `json:"blockCoverage"       yaml:"blockCoverage"`
	Functions                                    string  `json:"functionCoverage"    yaml:"functionCoverage"`
	StatementPercentage                          float64 `json:"statementPercentage" yaml:"statementPercentage"`
	BlockPercentage                              float64 `json:"blockPercentage"     yaml:"blockPercentage"`
	FunctionPercentage                           float64 `json:"functionPercentage"  yaml:"functionPercentage"`
	StatementThreshold                           float64 `json:"statementThreshold"  yaml:"statementThreshold"`
	BlockThreshold                               float64 `json:"blockThreshold"      yaml:"blockThreshold"`
	FunctionThreshold                            float64 `json:"functionThreshold"   yaml:"functionThreshold"`
	Failed                                       bool    `json:"failed"              yaml:"failed"`
	stmts, blocks, stmtHits, blockHits           int
	functions, functionHits                      int
}

// ByFile holds information for a cover.Profile result of a file.
type ByFile struct {
	By   `yaml:",inline"`
	File string `json:"file"    yaml:"file"`
}

// GetBy returns the By struct for ByFile.
func (f ByFile) GetBy() By {
	return f.By
}

// ByPackage holds information for cover.Profile results by package.
type ByPackage struct {
	By      `yaml:",inline"`
	Package string `json:"package" yaml:"package"`
}

// GetBy returns the By struct for ByPackage.
func (f ByPackage) GetBy() By {
	return f.By
}

// Totals holds cover.Profile total results.
type Totals struct {
	Statements TotalStatements `json:"statements" yaml:"statements"`
	Blocks     TotalBlocks     `json:"blocks"     yaml:"blocks"`
	Functions  TotalFunctions  `json:"functions"  yaml:"functions"`
}

// TotalBlocks holds cover.Profile total block results.
type TotalBlocks struct {
	totalBlocks        int
	totalCoveredBlocks int
	Coverage           string  `json:"coverage"   yaml:"coverage"`
	Threshold          float64 `json:"threshold"  yaml:"threshold"`
	Percentage         float64 `json:"percentage" yaml:"percentage"`
	Failed             bool    `json:"failed"     yaml:"failed"`
}

// TotalStatements holds cover.Profile total statement results.
type TotalStatements struct {
	Coverage               string  `json:"coverage"   yaml:"coverage"`
	Threshold              float64 `json:"threshold"  yaml:"threshold"`
	Percentage             float64 `json:"percentage" yaml:"percentage"`
	Failed                 bool    `json:"failed"     yaml:"failed"`
	totalCoveredStatements int
	totalStatements        int
}

// TotalFunctions holds cover.Profile total function results.
type TotalFunctions struct {
	totalFunctions        int
	totalCoveredFunctions int
	Coverage              string  `json:"coverage"   yaml:"coverage"`
	Threshold             float64 `json:"threshold"  yaml:"threshold"`
	Percentage            float64 `json:"percentage" yaml:"percentage"`
	Failed                bool    `json:"failed"     yaml:"failed"`
}

// Results holds information for all stats collected form the cover.Profile data.
type Results struct {
	ByFile    []ByFile    `json:"byFile"    yaml:"byFile"`
	ByPackage []ByPackage `json:"byPackage" yaml:"byPackage"`
	ByTotal   Totals      `json:"byTotal"   yaml:"byTotal"`
}
