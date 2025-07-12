package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/formatter"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

	TerminalWidthFlag      = "term-width"
	TerminalWidthFlagUsage = "force output to specified column width - autodetect with 0"
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
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:gocritic
			return run(cmd, args)
		},
	}
)

func run(cmd *cobra.Command, args []string) error {
	cfg, err := getConfig(cmd)
	if err != nil {
		return err
	}

	profiles, err := cover.ParseProfiles(args[0])
	if err != nil {
		return err
	}

	// Filter profiles
	filtered := make([]*cover.Profile, 0)
	for _, p := range profiles {
		if shouldSkip(p.FileName, cfg.Skip) {
			continue
		}
		filtered = append(filtered, p)
	}

	setupColor(cfg)
	setupTerminalWidth(cfg)

	failed := formatter.FormatAndReport(filtered, cfg)
	if failed {
		os.Exit(1)
	}
	return nil
}

func isCIEnvWithColor(env []string) bool {
	// likely not an exhaustive list
	supportedCIs := []string{
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"BITBUCKET_BUILD_NUMBER",
		"BUILDKITE",
		"APPVEYOR",
		"TEAMCITY_VERSION",
		"DRONE",
		"GITEA_ACTIONS",
	}

	// Detect CI systems that support color
	for _, e := range env {
		for _, ciVar := range supportedCIs {
			if strings.HasPrefix(e, ciVar+"=") {
				return true
			}
		}
	}

	// Detect 'act' runner
	if os.Getenv("ACT") == "true" || os.Getenv("ACT") == "1" {
		return true
	}
	return false
}

func setupColor(cfg *config.Config) {
	_, noColor := os.LookupEnv("NO_COLOR")
	if cfg.NoColor || noColor {
		color.NoColor = true
		return
	}

	if isCIEnvWithColor(os.Environ()) || os.Getenv("FORCE_COLOR") != "0" {
		color.NoColor = false
		return
	}
}

func isAnyCIEnv(env []string) bool {
	knownCIEnvVars := []string{
		"CI",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
		"DRONE",
		"BITBUCKET_BUILD_NUMBER",
		"BUILDKITE",
		"APPVEYOR",
		"TEAMCITY_VERSION",
		"JENKINS_URL",
		"TF_BUILD", // Azure Pipelines
		"HEROKU_TEST_RUN_ID",
		"CIRRUS_CI",
		"CODEBUILD_BUILD_ID",
		"GOCD_SERVER_HOST",
		"SEMAPHORE",
		"WERCKER",
		"HUDSON_URL",
		"ACT",
	}

	for _, e := range env {
		for _, ciVar := range knownCIEnvVars {
			if strings.HasPrefix(e, ciVar+"=") {
				return true
			}
		}
	}

	return false
}

func getTerminalWidth() int {
	// respect common env var
	if col := os.Getenv("COLUMNS"); col != "" {
		if w, err := strconv.Atoi(col); err == nil && w > 0 {
			return w
		}
	}
	if isAnyCIEnv(os.Environ()) {
		return 120 //nolint:mnd
	}
	// real tty
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			return w
		}
	}
	return 80 //nolint:mnd
}

func setupTerminalWidth(cfg *config.Config) {
	if cfg.TerminalWidth <= 0 {
		cfg.TerminalWidth = getTerminalWidth()
	}
}

func getConfig(cmd *cobra.Command) (*config.Config, error) {
	cfgPath, _ := cmd.Flags().GetString(ConfigFlag)
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	noConfigFile := true

	// Load the config if the file exists.
	if _, err := os.Stat(cfgPath); err == nil {
		cfg, err = config.Load(cfgPath)
		if err != nil {
			return cfg, err
		}
		noConfigFile = false
	}

	applyConfigOverrides(cfg, cmd, noConfigFile)

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyConfigOverrides(cfg *config.Config, cmd *cobra.Command, noConfigFile bool) { //nolint:cyclop
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
	if v, _ := cmd.Flags().GetInt(TerminalWidthFlag); cmd.Flags().Changed(TerminalWidthFlag) ||
		noConfigFile {
		cfg.TerminalWidth = v
	}
}

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

	rootCmd.Flags().Int(
		TerminalWidthFlag,
		0,
		TerminalWidthFlagUsage,
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
