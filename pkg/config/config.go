// Package config implements configuration concerns for the application.
package config

import (
	"errors"
	"fmt"
	"os"

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
	SortByStatementPercent = "statement-percent"
	SortByBlockPercent     = "block-percent"
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

	thresholdOff = 0
	thresholdMax = 100

	StatementThresholdDefault = 70
	StatementThresholdOff     = thresholdOff
	StatementThresholdMax     = thresholdMax

	BlockThresholdDefault = 50
	BlockThresholdOff     = thresholdOff
	BlockThresholdMax     = thresholdMax

	StatementsSection = "statements"
	BlocksSection     = "blocks"
)

// PerOverride holds override per thresholds.
type PerOverride map[string]float64

// PerThresholdOverride holds PerOverride's for Statements and Blocks.
type PerThresholdOverride struct {
	Statements PerOverride `yaml:"statements"`
	Blocks     PerOverride `yaml:"blocks"`
}

// Config for application.
type Config struct {
	StatementThreshold float64              `yaml:"statementThreshold,omitempty"`
	BlockThreshold     float64              `yaml:"blockThreshold,omitempty"`
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
	TerminalWidth      int                  `yaml:"terminalWidth,omitempty"`
	ModuleName         string               `yaml:"moduleName,omitempty"`
	DiffFrom           string               `yaml:"diffFrom,omitempty"`
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
	c.SortBy = SortByDefault
	c.SortOrder = SortOrderDefault
	c.Skip = []string{}
	c.Format = FormatDefault

	c.initPerFileWhenNil()
	c.initPerPackageWhenNil()
	c.setTotalThresholds(StatementThresholdDefault, BlockThresholdDefault)
}

// Validate the config or return an error if it is not valid.
func (c *Config) Validate() error { //nolint:cyclop
	if c.StatementThreshold < StatementThresholdOff || c.StatementThreshold > StatementThresholdMax {
		return errors.New("statement threshold must be between 0 and 100")
	}
	if c.BlockThreshold < BlockThresholdOff || c.BlockThreshold > BlockThresholdMax {
		return errors.New("block threshold must be between 0 and 100")
	}

	switch c.SortBy {
	case SortByFile, SortByStatements, SortByBlocks, SortByStatementPercent, SortByBlockPercent:
		break
	default:
		return fmt.Errorf("sort-by must be one of %s|%s|%s|%s|%s",
			SortByFile, SortByStatements, SortByBlocks, SortByStatementPercent, SortByBlockPercent)
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

	if c.NoSummary && c.NoTable && c.Format != FormatJSON && c.Format != FormatYAML {
		return fmt.Errorf("cannot specify both no-summary and no-table with format %s", c.Format)
	}

	if c.Skip == nil {
		c.Skip = []string{}
	}

	c.initPerFileWhenNil()
	c.initPerPackageWhenNil()
	c.setTotalThresholds(c.StatementThreshold, c.BlockThreshold)

	return nil
}

func (c *Config) initPerFileWhenNil() {
	if c.PerFile.Blocks == nil {
		c.PerFile.Blocks = PerOverride{}
	}
	if c.PerFile.Statements == nil {
		c.PerFile.Statements = PerOverride{}
	}
}

func (c *Config) initPerPackageWhenNil() {
	if c.PerPackage.Blocks == nil {
		c.PerPackage.Blocks = PerOverride{}
	}
	if c.PerPackage.Statements == nil {
		c.PerPackage.Statements = PerOverride{}
	}
}

func (c *Config) setTotalThresholds(totalStatement, totalBlock float64) {
	if c.Total == nil {
		c.Total = PerOverride{}
	}
	if _, exists := c.Total[StatementsSection]; !exists {
		c.Total[StatementsSection] = totalStatement
	}
	if _, exists := c.Total[BlocksSection]; !exists {
		c.Total[BlocksSection] = totalBlock
	}
}
