package compute

// HasBy required interface for all descendants of By.
type HasBy interface {
	GetBy() By
}

// By holds cover.Profile information.
type By struct {
	Statements                         string  `json:"statementCoverage"   yaml:"statementCoverage"`
	Blocks                             string  `json:"blockCoverage"       yaml:"blockCoverage"`
	StatementPercentage                float64 `json:"statementPercentage" yaml:"statementPercentage"`
	BlockPercentage                    float64 `json:"blockPercentage"     yaml:"blockPercentage"`
	StatementThreshold                 float64 `json:"statementThreshold"  yaml:"statementThreshold"`
	BlockThreshold                     float64 `json:"blockThreshold"      yaml:"blockThreshold"`
	Failed                             bool    `json:"failed"              yaml:"failed"`
	stmts, blocks, stmtHits, blockHits int
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

// ComparisonDelta represents the comparison between current and historical results.
type ComparisonDelta struct {
	StatementsDelta float64 `json:"statementsDelta" yaml:"statementsDelta"`
	BlocksDelta     float64 `json:"blocksDelta"     yaml:"blocksDelta"`
}

// ComparisonResult represents comparison results for a specific item.
type ComparisonResult struct {
	Name   string          `json:"name"   yaml:"name"`
	Delta  ComparisonDelta `json:"delta"  yaml:"delta"`
	Type   string          `json:"type"   yaml:"type"` // "file", "package", or "total"
}

// ComparisonData holds all comparison information.
type ComparisonData struct {
	Ref     string             `json:"ref"     yaml:"ref"`
	Commit  string             `json:"commit"  yaml:"commit"`
	Results []ComparisonResult `json:"results" yaml:"results"`
}

// Results holds information for all stats collected form the cover.Profile data.
type Results struct {
	ByFile     []ByFile        `json:"byFile"              yaml:"byFile"`
	ByPackage  []ByPackage     `json:"byPackage"           yaml:"byPackage"`
	ByTotal    Totals          `json:"byTotal"             yaml:"byTotal"`
	Comparison *ComparisonData `json:"comparison,omitempty" yaml:"comparison,omitempty"`
}
