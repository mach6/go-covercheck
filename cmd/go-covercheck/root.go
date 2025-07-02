package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/formatter"
	"github.com/spf13/cobra"
	"golang.org/x/tools/cover"
)

// Constants for this application.
const (
	ConfigFlag      = "config"
	ConfigFlagShort = "c"
	ConfigFlagUsage = "path to YAML config file"

	NoTableFlag        = "no-table"
	NoTableFlagShort   = "t"
	NoTableFlagDefault = false

	NoSummaryFlag        = "no-summary"
	NoSummaryFlagShort   = "u"
	NoSummaryFlagDefault = false

	FormatFlag      = "format"
	FormatFlagShort = "f"

	StatementThresholdFlag      = "statement-threshold"
	StatementThresholdFlagShort = "s"
	StatementThresholdFlagUsage = "statement threshold to enforce - disabled with 0"

	BlockThresholdFlag      = "block-threshold"
	BlockThresholdFlagShort = "b"
	BlockThresholdFlagUsage = "block threshold to enforce - disabled with 0"

	SortByFlag    = "sort-by"
	SortOrderFlag = "sort-order"

	SkipFlag      = "skip"
	SkipFlagShort = "k"
	SkipFlagUsage = "regex string of file(s) and/or package(s) to skip"

	NoColorFlag        = "no-color"
	NoColorFlagShort   = "w"
	NoColorFlagDefault = false
	NoColorFlagUsage   = "disable color output"
)

// Execute the CLI application.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// the error message is printed by default -- just exit.
		os.Exit(1)
	}
}

// Variables for this application.
var (
	ConfigFlagDefault = "." + config.AppName + ".yml"

	NoTableFlagUsage = fmt.Sprintf(
		"suppress table output and only show failure summary - disabled by default for %s|%s",
		config.FormatJSON, config.FormatYAML,
	)

	NoSummaryFlagUsage = fmt.Sprintf(
		"suppress failure summary and only show table output - disabled by default for %s|%s",
		config.FormatJSON, config.FormatYAML,
	)

	SortByFlagUsage = fmt.Sprintf(
		"sort-by: %s|%s|%s|%s|%s",
		config.SortByFile,
		config.SortByBlocks,
		config.SortByStatements,
		config.SortByStatementPercent,
		config.SortByBlockPercent,
	)

	SortOrderFlagUsage = fmt.Sprintf("sort order: %s|%s",
		config.SortOrderAsc,
		config.SortOrderDesc,
	)

	FormatFlagUsage = fmt.Sprintf("output format: %s|%s|%s|%s|%s|%s|%s",
		config.FormatTable,
		config.FormatJSON,
		config.FormatYAML,
		config.FormatMD,
		config.FormatHTML,
		config.FormatCSV,
		config.FormatTSV,
	)

	SkipFlagDefault []string

	rootCmd = &cobra.Command{
		Version: getVersion(),
		Use:     config.AppName + " [coverage.out]",
		Short:   config.AppName + ": Coverage gatekeeper for enforcing test thresholds in Go",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString(ConfigFlag)
			cfg := new(config.Config)
			cfg.ApplyDefaults()
			noConfigFile := true

			// Load the config if the file exists.
			if _, err := os.Stat(cfgPath); err == nil {
				cfg, err = config.Load(cfgPath)
				if err != nil {
					return err
				}
				noConfigFile = false
			}

			// CLI overrides when a config file exists or values to use when it does not exist.
			if v, _ := cmd.Flags().GetFloat64(StatementThresholdFlag); cmd.Flags().Changed(StatementThresholdFlag) ||
				noConfigFile {
				cfg.StatementThreshold = v
			}
			if v, _ := cmd.Flags().GetFloat64(BlockThresholdFlag); cmd.Flags().Changed(BlockThresholdFlag) ||
				noConfigFile {
				cfg.BlockThreshold = v
			}
			if v, _ := cmd.Flags().GetString(SortByFlag); cmd.Flags().Changed(SortByFlag) ||
				noConfigFile {
				cfg.SortBy = v
			}
			if v, _ := cmd.Flags().GetString(SortOrderFlag); cmd.Flags().Changed(SortOrderFlag) ||
				noConfigFile {
				cfg.SortOrder = v
			}
			if v, _ := cmd.Flags().GetStringArray(SkipFlag); cmd.Flags().Changed(SkipFlag) ||
				noConfigFile {
				cfg.Skip = v
			}
			if v, _ := cmd.Flags().GetString(FormatFlag); cmd.Flags().Changed(FormatFlag) ||
				noConfigFile {
				cfg.Format = v
			}
			if v, _ := cmd.Flags().GetBool(NoTableFlag); cmd.Flags().Changed(NoTableFlag) ||
				noConfigFile {
				cfg.NoTable = v
			}
			if v, _ := cmd.Flags().GetBool(NoSummaryFlag); cmd.Flags().Changed(NoSummaryFlag) ||
				noConfigFile {
				cfg.NoSummary = v
			}
			if v, _ := cmd.Flags().GetBool(NoColorFlag); cmd.Flags().Changed(NoColorFlag) ||
				noConfigFile {
				cfg.NoColor = v
			}

			if err := cfg.Validate(); err != nil {
				return err
			}

			profiles, err := cover.ParseProfiles(args[0])
			if err != nil {
				return err
			}

			// Filter profiles
			var filtered []*cover.Profile
			for _, p := range profiles {
				if shouldSkip(p.FileName, cfg.Skip) {
					continue
				}
				filtered = append(filtered, p)
			}

			if cfg.NoColor {
				color.NoColor = true // disables color globally via fatih/color, in theory
			}

			failed := formatter.FormatAndReport(filtered, cfg)
			if failed {
				os.Exit(1)
			}
			return nil
		},
	}
)

func getVersion() string {
	if config.BuiltBy != "" && config.BuildTimeStamp != "" {
		return fmt.Sprintf(
			"%s [%s] built by %s on %s",
			config.AppVersion,
			config.AppRevision,
			config.BuiltBy,
			config.BuildTimeStamp,
		)
	}
	return fmt.Sprintf(
		"%s [%s]",
		config.AppVersion,
		config.AppRevision,
	)
}

func init() {
	rootCmd.Flags().StringP(
		ConfigFlag,
		ConfigFlagShort,
		ConfigFlagDefault,
		ConfigFlagUsage,
	)

	rootCmd.Flags().BoolP(
		NoTableFlag,
		NoTableFlagShort,
		NoTableFlagDefault,
		NoTableFlagUsage,
	)

	rootCmd.Flags().BoolP(
		NoSummaryFlag,
		NoSummaryFlagShort,
		NoSummaryFlagDefault,
		NoSummaryFlagUsage,
	)

	rootCmd.Flags().BoolP(
		NoColorFlag,
		NoColorFlagShort,
		NoColorFlagDefault,
		NoColorFlagUsage,
	)

	rootCmd.Flags().StringP(
		FormatFlag,
		FormatFlagShort,
		config.FormatDefault,
		FormatFlagUsage,
	)

	rootCmd.Flags().Float64P(
		StatementThresholdFlag,
		StatementThresholdFlagShort,
		config.StatementThresholdDefault,
		StatementThresholdFlagUsage,
	)

	rootCmd.Flags().Float64P(
		BlockThresholdFlag,
		BlockThresholdFlagShort,
		config.BlockThresholdDefault,
		BlockThresholdFlagUsage,
	)

	rootCmd.Flags().String(
		SortByFlag,
		config.SortByDefault,
		SortByFlagUsage,
	)

	rootCmd.Flags().String(
		SortOrderFlag,
		config.SortOrderDefault,
		SortOrderFlagUsage,
	)

	rootCmd.Flags().StringArrayP(
		SkipFlag,
		SkipFlagShort,
		SkipFlagDefault,
		SkipFlagUsage,
	)
}

func shouldSkip(filename string, skip []string) bool {
	for _, s := range skip {
		regex := regexp.MustCompile(s)
		if regex.MatchString(filename) {
			return true
		}
	}
	return false
}
