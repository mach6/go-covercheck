// Package config implements configuration concerns for the application.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/chroma/v2/styles"
	"gopkg.in/yaml.v3"
)

// Application config variables that are updated by the build.
var (
	AppName        = "go-covercheck"
	AppVersion     = "1.0.0-dev"
	AppRevision    = "HEAD"
	BuiltBy        = ""
	BuildTimeStamp = ""
)

// Application config constants.
const (
	SortByFile             = "file"
	SortByStatements       = "statements"
	SortByBlocks           = "blocks"
	SortByLines            = "lines"
	SortByFunctions        = "functions"
	SortByStatementPercent = "statement-percent"
	SortByBlockPercent     = "block-percent"
	SortByLinePercent      = "line-percent"
	SortByFunctionPercent  = "function-percent"
	SortByDefault          = SortByFile

	SortOrderAsc     = "asc"
	SortOrderDesc    = "desc"
	SortOrderDefault = SortOrderAsc

	FormatJSON    = "json"
	FormatYAML    = "yaml"
	FormatTable   = "table"
	FormatCSV     = "csv"
	FormatHTML    = "html"
	FormatTSV     = "tsv"
	FormatMD      = "md"
	FormatDefault = FormatTable

	TableStyleDefault  = "default"
	TableStyleLight    = "light"
	TableStyleBold     = "bold"
	TableStyleRounded  = "rounded"
	TableStyleDouble   = "double"
	TableStyleDefValue = TableStyleLight

	thresholdOff = 0
	thresholdMax = 100

	StatementThresholdDefault = 70
	StatementThresholdOff     = thresholdOff
	StatementThresholdMax     = thresholdMax

	BlockThresholdDefault = 50
	BlockThresholdOff     = thresholdOff
	BlockThresholdMax     = thresholdMax

	LineThresholdDefault = 50
	LineThresholdOff     = thresholdOff
	LineThresholdMax     = thresholdMax

	FunctionThresholdDefault = 60
	FunctionThresholdOff     = thresholdOff
	FunctionThresholdMax     = thresholdMax

	// InspectContextDefault is the default number of context lines shown around
	// uncovered blocks by --inspect.
	InspectContextDefault = 2

	// SyntaxStyleDefault is the default chroma syntax style used by --inspect.
	// The value "auto" defers the choice to runtime terminal-background
	// detection so dark terminals get github-dark and light terminals get
	// github. Any concrete chroma style name (e.g. "monokai") short-circuits
	// detection and is used as-is.
	SyntaxStyleDefault = SyntaxStyleAuto

	// SyntaxStyleAuto selects github or github-dark based on detected
	// terminal background.
	SyntaxStyleAuto = "auto"

	StatementsSection = "statements"
	BlocksSection     = "blocks"
	LinesSection      = "lines"
	FunctionsSection  = "functions"
)

// PerOverride holds override per thresholds.
type PerOverride map[string]float64

// PerThresholdOverride holds PerOverride's for Statements, Blocks, Lines, and Functions.
type PerThresholdOverride struct {
	Statements PerOverride `yaml:"statements"`
	Blocks     PerOverride `yaml:"blocks"`
	Lines      PerOverride `yaml:"lines"`
	Functions  PerOverride `yaml:"functions"`
}

// Config for application.
type Config struct {
	StatementThreshold float64              `yaml:"statementThreshold,omitempty"`
	BlockThreshold     float64              `yaml:"blockThreshold,omitempty"`
	LineThreshold      float64              `yaml:"lineThreshold,omitempty"`
	FunctionThreshold  float64              `yaml:"functionThreshold,omitempty"`
	SortBy             string               `yaml:"sortBy,omitempty"`
	SortOrder          string               `yaml:"sortOrder,omitempty"`
	Skip               []string             `yaml:"skip,omitempty"`
	PerFile            PerThresholdOverride `yaml:"perFile,omitempty"`
	PerPackage         PerThresholdOverride `yaml:"perPackage,omitempty"`
	Total              PerOverride          `yaml:"total,omitempty"`
	NoTable            bool                 `yaml:"noTable,omitempty"`
	NoSummary          bool                 `yaml:"noSummary,omitempty"`
	NoColor            bool                 `yaml:"noColor,omitempty"`
	Format             string               `yaml:"format,omitempty"`
	TableStyle         string               `yaml:"tableStyle,omitempty"`
	TerminalWidth      int                  `yaml:"terminalWidth,omitempty"`
	ModuleName         string               `yaml:"moduleName,omitempty"`
	DiffFrom           string               `yaml:"diffFrom,omitempty"`
	NoUncoveredLines   bool                 `yaml:"noUncoveredLines,omitempty"`
	InspectContext     int                  `yaml:"inspectContext,omitempty"`
	SyntaxStyle        string               `yaml:"syntaxStyle,omitempty"`
	// not configurable via YAML
	InspectFiles []string `yaml:"-"`
	Inspect      bool     `yaml:"-"`
}

// Load a Config from a path or produce an error.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, err
	}
	cfg := new(Config)
	cfg.ApplyDefaults()
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ApplyDefaults to the Config.
func (c *Config) ApplyDefaults() {
	c.StatementThreshold = StatementThresholdDefault
	c.BlockThreshold = BlockThresholdDefault
	c.LineThreshold = LineThresholdDefault
	c.FunctionThreshold = FunctionThresholdDefault
	c.SortBy = SortByDefault
	c.SortOrder = SortOrderDefault
	c.Skip = []string{}
	c.InspectFiles = []string{}
	c.Format = FormatDefault
	c.TableStyle = TableStyleDefValue
	c.SyntaxStyle = SyntaxStyleDefault
	c.InspectContext = InspectContextDefault

	c.initPerFileWhenNil()
	c.initPerPackageWhenNil()
	c.setTotalThresholds(StatementThresholdDefault, BlockThresholdDefault, LineThresholdDefault, FunctionThresholdDefault)
}

// Validate the config or return an error if it is not valid.
func (c *Config) Validate() error { //nolint:cyclop
	if c.StatementThreshold < StatementThresholdOff || c.StatementThreshold > StatementThresholdMax {
		return errors.New("statement threshold must be between 0 and 100")
	}
	if c.BlockThreshold < BlockThresholdOff || c.BlockThreshold > BlockThresholdMax {
		return errors.New("block threshold must be between 0 and 100")
	}
	if c.LineThreshold < LineThresholdOff || c.LineThreshold > LineThresholdMax {
		return errors.New("line threshold must be between 0 and 100")
	}
	if c.FunctionThreshold < FunctionThresholdOff || c.FunctionThreshold > FunctionThresholdMax {
		return errors.New("function threshold must be between 0 and 100")
	}
	if c.InspectContext < 0 {
		return errors.New("inspect-context must be greater than or equal to 0")
	}
	// styles.Get returns styles.Fallback (not nil) for unknown names, so compare
	// against Fallback to reject typos at config-load time rather than silently
	// falling back at render time.
	if c.SyntaxStyle != "" && c.SyntaxStyle != SyntaxStyleAuto && styles.Get(c.SyntaxStyle) == styles.Fallback {
		return fmt.Errorf("syntax-style %q is not a known chroma style; use %q or a bundled style name",
			c.SyntaxStyle, SyntaxStyleAuto)
	}

	switch c.SortBy {
	case SortByFile, SortByStatements, SortByBlocks, SortByLines, SortByFunctions,
		SortByStatementPercent, SortByBlockPercent, SortByLinePercent, SortByFunctionPercent:
		break
	default:
		return fmt.Errorf("sort-by must be one of %s|%s|%s|%s|%s|%s|%s|%s|%s",
			SortByFile, SortByStatements, SortByBlocks, SortByLines, SortByFunctions,
			SortByStatementPercent, SortByBlockPercent, SortByLinePercent, SortByFunctionPercent)
	}

	switch c.SortOrder {
	case SortOrderAsc, SortOrderDesc:
		break
	default:
		return fmt.Errorf("sort-order must be one of %s|%s", SortOrderAsc, SortOrderDesc)
	}

	switch c.Format {
	case FormatJSON, FormatYAML, FormatTable, FormatMD, FormatCSV, FormatHTML, FormatTSV:
		break
	default:
		return fmt.Errorf("format must be one of %s|%s|%s|%s|%s|%s|%s",
			FormatJSON, FormatYAML, FormatTable, FormatCSV, FormatHTML, FormatTSV, FormatMD)
	}

	switch c.TableStyle {
	case TableStyleDefault, TableStyleLight, TableStyleBold, TableStyleRounded, TableStyleDouble:
		break
	default:
		return fmt.Errorf("table-style must be one of %s|%s|%s|%s|%s",
			TableStyleDefault, TableStyleLight, TableStyleBold, TableStyleRounded, TableStyleDouble)
	}

	if c.NoSummary && c.NoTable && c.Format != FormatJSON && c.Format != FormatYAML {
		return fmt.Errorf("cannot specify both no-summary and no-table with format %s", c.Format)
	}

	if c.Skip == nil {
		c.Skip = []string{}
	}

	c.initPerFileWhenNil()
	c.initPerPackageWhenNil()
	c.setTotalThresholds(c.StatementThreshold, c.BlockThreshold, c.LineThreshold, c.FunctionThreshold)

	return nil
}

func (c *Config) initPerFileWhenNil() {
	if c.PerFile.Blocks == nil {
		c.PerFile.Blocks = PerOverride{}
	}
	if c.PerFile.Statements == nil {
		c.PerFile.Statements = PerOverride{}
	}
	if c.PerFile.Lines == nil {
		c.PerFile.Lines = PerOverride{}
	}
	if c.PerFile.Functions == nil {
		c.PerFile.Functions = PerOverride{}
	}
}

func (c *Config) initPerPackageWhenNil() {
	if c.PerPackage.Blocks == nil {
		c.PerPackage.Blocks = PerOverride{}
	}
	if c.PerPackage.Statements == nil {
		c.PerPackage.Statements = PerOverride{}
	}
	if c.PerPackage.Lines == nil {
		c.PerPackage.Lines = PerOverride{}
	}
	if c.PerPackage.Functions == nil {
		c.PerPackage.Functions = PerOverride{}
	}
}

func (c *Config) setTotalThresholds(totalStatement, totalBlock, totalLine, totalFunction float64) {
	if c.Total == nil {
		c.Total = PerOverride{}
	}
	if _, exists := c.Total[StatementsSection]; !exists {
		c.Total[StatementsSection] = totalStatement
	}
	if _, exists := c.Total[BlocksSection]; !exists {
		c.Total[BlocksSection] = totalBlock
	}
	if _, exists := c.Total[LinesSection]; !exists {
		c.Total[LinesSection] = totalLine
	}
	if _, exists := c.Total[FunctionsSection]; !exists {
		c.Total[FunctionsSection] = totalFunction
	}
}
