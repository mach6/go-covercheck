package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/gitdiff"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/samples"
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
	StatementThresholdFlagUsage = "global statement threshold to enforce [0=disabled]"

	BlockThresholdFlag      = "block-threshold"
	BlockThresholdFlagShort = "b"
	BlockThresholdFlagUsage = "global block threshold to enforce [0=disabled]"

	TotalStatementThresholdFlag      = "total-statement-threshold"
	TotalStatementThresholdFlagShort = "S"
	TotalStatementThresholdFlagUsage = "total statement threshold to enforce [0=disabled]"

	TotalBlockThresholdFlag      = "total-block-threshold"
	TotalBlockThresholdFlagShort = "B"
	TotalBlockThresholdFlagUsage = "total block threshold to enforce [0=disabled]"

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
	TerminalWidthFlagUsage = "force output to specified column width [0=autodetect]"

	SaveHistoryFlag      = "save-history"
	SaveHistoryFlagShort = "H"
	SaveHistoryFlagUsage = "add coverage result to history"

	HistoryFileFlag = "history-file"

	HistoryLabelFlag      = "label"
	HistoryLabelFlagShort = "l"
	HistoryLabelFlagUsage = "optional label name for history entry"

	CompareHistoryFlag      = "compare-history"
	CompareHistoryFlagShort = "C"
	CompareHistoryFlagUsage = "compare current coverage against historical ref [commit|branch|tag|label]"

	ShowHistoryFlag      = "show-history"
	ShowHistoryFlagShort = "I"
	ShowHistoryFlagUsage = "show historical entries in tabular format"

	DeleteHistoryFlag      = "delete-history"
	DeleteHistoryFlagShort = "D"
	DeleteHistoryFlagUsage = "delete historical entry by ref [commit|branch|tag|label]"

	HistoryLimitFlag      = "limit-history"
	HistoryLimitFlagShort = "L"
	HistoryLimitFlagUsage = "limit number of historical entries to save or display [0=no limit]"

	ModuleNameFlag      = "module-name"
	ModuleNameFlagShort = "m"
	ModuleNameFlagUsage = "explicitly set module name for path normalization (overrides module inference)"

	InitFlag      = "init"
	InitFlagUsage = "create a sample .go-covercheck.yml config file in the current directory"

	DiffOnlyFlag      = "diff-only"
	DiffOnlyFlagUsage = "enforce thresholds only on files changed in git diff"

	AgainstFlag      = "against"
	AgainstFlagUsage = "git reference (commit/branch/tag) to diff against [default: HEAD~1]"

	// ConfigFilePermissions permissions.
	ConfigFilePermissions = 0600
)

// Execute the CLI application.
func Execute() {
	initFlags(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		// the error message is printed by default -- just exit.
		os.Exit(1)
	}
}

// Variables for this application.
var (
	ConfigFlagDefault = "." + config.AppName + ".yml"

	HistoryFileFlagUsage   = "path to " + config.AppName + " history file"
	HistoryFileFlagDefault = "." + config.AppName + ".history.json"

	NoTableFlagUsage = fmt.Sprintf(
		"suppress tabular output and only show failure summary [disabled for %s|%s]",
		config.FormatJSON, config.FormatYAML,
	)

	NoSummaryFlagUsage = fmt.Sprintf(
		"suppress failure summary and only show tabular output [disabled for %s|%s]",
		config.FormatJSON, config.FormatYAML,
	)

	SortByFlagUsage = fmt.Sprintf(
		"sort-by [%s|%s|%s|%s|%s]",
		config.SortByFile,
		config.SortByBlocks,
		config.SortByStatements,
		config.SortByStatementPercent,
		config.SortByBlockPercent,
	)

	SortOrderFlagUsage = fmt.Sprintf("sort order [%s|%s]",
		config.SortOrderAsc,
		config.SortOrderDesc,
	)

	FormatFlagUsage = fmt.Sprintf("output format [%s|%s|%s|%s|%s|%s|%s]",
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
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:gocritic
			return run(cmd, args)
		},
		SilenceUsage: true,
	}
)

func run(cmd *cobra.Command, args []string) error {
	cfg, err := getConfig(cmd)
	if err != nil {
		return err
	}

	setupColor(cfg)
	setupTerminalWidth(cfg)

	if handled, err := handleNonCoverageOperationsWhichShouldExit(cmd, cfg); handled || err != nil {
		return err
	}

	// showCoverage and get the results.
	results, failed, err := showCoverage(args, cfg)
	if err != nil {
		return err
	}

	// handle history operations (compare and save)
	if err := handleHistoryOperations(cmd, results, cfg); err != nil {
		return err
	}

	if failed {
		os.Exit(1)
	}
	return nil
}

func handleNonCoverageOperationsWhichShouldExit(cmd *cobra.Command, cfg *config.Config) (bool, error) {
	// check if --init flag is specified and handle it
	bInit, _ := cmd.Flags().GetBool(InitFlag)
	if bInit {
		return true, initConfigFile(cmd)
	}

	historyLimit, _ := cmd.Flags().GetInt(HistoryLimitFlag)

	// delete history entry and exit when requested.
	deleteRef, _ := cmd.Flags().GetString(DeleteHistoryFlag)
	if deleteRef != "" {
		return true, deleteHistory(cmd, deleteRef, historyLimit)
	}

	// show history and exit when requested.
	bShowHistory, _ := cmd.Flags().GetBool(ShowHistoryFlag)
	if bShowHistory {
		return true, showHistory(cmd, historyLimit, cfg)
	}
	return false, nil
}

func showCoverage(args []string, cfg *config.Config) (compute.Results, bool, error) {
	// we need coverage profile input from here on.
	profiles, err := getCoverProfileData(args)
	if err != nil {
		return compute.Results{}, false, err
	}
	filtered := filter(profiles, cfg)

	results, failed := compute.CollectResults(filtered, cfg)
	output.FormatAndReport(results, cfg, failed)
	return results, failed, nil
}

// filter profiles.
func filter(profiles []*cover.Profile, cfg *config.Config) []*cover.Profile {
	// First apply skip patterns
	filtered := make([]*cover.Profile, 0)
	for _, p := range profiles {
		if shouldSkip(p.FileName, cfg.Skip) {
			continue
		}
		filtered = append(filtered, p)
	}

	// Then apply diff filtering if enabled
	if cfg.DiffOnly {
		changedFiles, err := gitdiff.GetChangedFiles(".", cfg.Against)
		if err != nil {
			// Log error but don't fail - fall back to normal behavior
			fmt.Fprintf(os.Stderr, "Warning: Failed to get changed files for diff mode: %v\n", err)
			fmt.Fprintf(os.Stderr, "Falling back to checking all files.\n")
			return filtered
		}

		// If no files changed, return empty result
		if len(changedFiles) == 0 {
			fmt.Fprintf(os.Stderr, "No files changed in diff. No coverage to check.\n")
			return []*cover.Profile{}
		}

		diffFiltered := gitdiff.FilterProfilesByChangedFiles(filtered, changedFiles, cfg.ModuleName)
		fmt.Fprintf(os.Stderr, "Diff mode: Checking coverage for %d changed files (out of %d total files)\n",
			len(diffFiltered), len(filtered))
		return diffFiltered
	}

	return filtered
}

func getCoverProfileData(args []string) ([]*cover.Profile, error) {
	var coveragePath string
	if len(args) > 0 {
		coveragePath = args[0]
	}

	var profiles []*cover.Profile
	var err error

	if coveragePath != "" && coveragePath != "-" {
		// use positional file argument
		profiles, err = cover.ParseProfiles(coveragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse coverage file %q: %w", coveragePath, err)
		}
		return profiles, nil
	}

	// check if stdin is available
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// data is being piped
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read coverage from stdin: %w", err)
		}
		profiles, err = cover.ParseProfilesFromReader(strings.NewReader(string(data)))
		if err != nil {
			return nil, fmt.Errorf("failed to parse coverage from stdin: %w", err)
		}
		return profiles, nil
	}

	// fallback to coverage.out file
	if _, err := os.Stat("coverage.out"); err == nil {
		profiles, err = cover.ParseProfiles("coverage.out")
		if err != nil {
			return nil, fmt.Errorf("failed to parse default coverage.out: %w", err)
		}
	} else {
		return nil, errors.New("no coverprofile input provided (pass a filename, pipe via stdin, " +
			"or include it via a 'coverage.out' file in the present working directory)")
	}

	return profiles, nil
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
		text.DisableColors()
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

func applyConfigOverrides(cfg *config.Config, cmd *cobra.Command, noConfigFile bool) {
	applyFloat64FlagOverride(cmd, StatementThresholdFlag, &cfg.StatementThreshold, noConfigFile)
	applyFloat64FlagOverride(cmd, BlockThresholdFlag, &cfg.BlockThreshold, noConfigFile)
	applyFloat64TotalFlagOverride(cmd, TotalStatementThresholdFlag, config.StatementsSection, cfg.Total)
	applyFloat64TotalFlagOverride(cmd, TotalBlockThresholdFlag, config.BlocksSection, cfg.Total)
	applyStringFlagOverride(cmd, SortByFlag, &cfg.SortBy, noConfigFile)
	applyStringFlagOverride(cmd, SortOrderFlag, &cfg.SortOrder, noConfigFile)
	applyStringArrayFlagOverride(cmd, SkipFlag, &cfg.Skip, noConfigFile)
	applyStringFlagOverride(cmd, FormatFlag, &cfg.Format, noConfigFile)
	applyBoolFlagOverride(cmd, NoTableFlag, &cfg.NoTable, noConfigFile)
	applyBoolFlagOverride(cmd, NoSummaryFlag, &cfg.NoSummary, noConfigFile)
	applyBoolFlagOverride(cmd, NoColorFlag, &cfg.NoColor, noConfigFile)
	applyIntFlagOverride(cmd, TerminalWidthFlag, &cfg.TerminalWidth, noConfigFile)
	applyStringFlagOverride(cmd, ModuleNameFlag, &cfg.ModuleName, noConfigFile)
	applyBoolFlagOverride(cmd, DiffOnlyFlag, &cfg.DiffOnly, noConfigFile)
	applyStringFlagOverride(cmd, AgainstFlag, &cfg.Against, noConfigFile)

	// set cfg.Total thresholds to the global values, iff no override was specified for each.
	if v, _ := cmd.Flags().GetFloat64(StatementThresholdFlag); !cmd.Flags().Changed(TotalStatementThresholdFlag) &&
		cfg.Total[config.StatementsSection] == config.StatementThresholdDefault {
		cfg.Total[config.StatementsSection] = v
	}
	if v, _ := cmd.Flags().GetFloat64(BlockThresholdFlag); !cmd.Flags().Changed(TotalBlockThresholdFlag) &&
		cfg.Total[config.BlocksSection] == config.BlockThresholdDefault {
		cfg.Total[config.BlocksSection] = v
	}
}

func applyFloat64FlagOverride(cmd *cobra.Command, flagName string, target *float64, noConfigFile bool) {
	if v, _ := cmd.Flags().GetFloat64(flagName); cmd.Flags().Changed(flagName) || noConfigFile {
		*target = v
	}
}

func applyFloat64TotalFlagOverride(cmd *cobra.Command, flagName string, section string, target map[string]float64) {
	if v, _ := cmd.Flags().GetFloat64(flagName); cmd.Flags().Changed(flagName) {
		target[section] = v
	}
}

func applyStringFlagOverride(cmd *cobra.Command, flagName string, target *string, noConfigFile bool) {
	if v, _ := cmd.Flags().GetString(flagName); cmd.Flags().Changed(flagName) || noConfigFile {
		*target = v
	}
}

func applyStringArrayFlagOverride(cmd *cobra.Command, flagName string, target *[]string, noConfigFile bool) {
	if v, _ := cmd.Flags().GetStringArray(flagName); cmd.Flags().Changed(flagName) || noConfigFile {
		*target = v
	}
}

func applyBoolFlagOverride(cmd *cobra.Command, flagName string, target *bool, noConfigFile bool) {
	if v, _ := cmd.Flags().GetBool(flagName); cmd.Flags().Changed(flagName) || noConfigFile {
		*target = v
	}
}

func applyIntFlagOverride(cmd *cobra.Command, flagName string, target *int, noConfigFile bool) {
	if v, _ := cmd.Flags().GetInt(flagName); cmd.Flags().Changed(flagName) || noConfigFile {
		*target = v
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

func initFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(
		ConfigFlag,
		ConfigFlagShort,
		ConfigFlagDefault,
		ConfigFlagUsage,
	)

	cmd.Flags().BoolP(
		NoTableFlag,
		NoTableFlagShort,
		NoTableFlagDefault,
		NoTableFlagUsage,
	)

	cmd.Flags().BoolP(
		NoSummaryFlag,
		NoSummaryFlagShort,
		NoSummaryFlagDefault,
		NoSummaryFlagUsage,
	)

	cmd.Flags().BoolP(
		NoColorFlag,
		NoColorFlagShort,
		NoColorFlagDefault,
		NoColorFlagUsage,
	)

	cmd.Flags().StringP(
		FormatFlag,
		FormatFlagShort,
		config.FormatDefault,
		FormatFlagUsage,
	)

	cmd.Flags().Float64P(
		StatementThresholdFlag,
		StatementThresholdFlagShort,
		config.StatementThresholdDefault,
		StatementThresholdFlagUsage,
	)

	cmd.Flags().Float64P(
		BlockThresholdFlag,
		BlockThresholdFlagShort,
		config.BlockThresholdDefault,
		BlockThresholdFlagUsage,
	)

	cmd.Flags().Float64P(
		TotalStatementThresholdFlag,
		TotalStatementThresholdFlagShort,
		0,
		TotalStatementThresholdFlagUsage,
	)

	cmd.Flags().Float64P(
		TotalBlockThresholdFlag,
		TotalBlockThresholdFlagShort,
		0,
		TotalBlockThresholdFlagUsage,
	)

	cmd.Flags().String(
		SortByFlag,
		config.SortByDefault,
		SortByFlagUsage,
	)

	cmd.Flags().String(
		SortOrderFlag,
		config.SortOrderDefault,
		SortOrderFlagUsage,
	)

	cmd.Flags().StringArrayP(
		SkipFlag,
		SkipFlagShort,
		SkipFlagDefault,
		SkipFlagUsage,
	)

	cmd.Flags().Int(
		TerminalWidthFlag,
		0,
		TerminalWidthFlagUsage,
	)

	cmd.Flags().String(
		HistoryFileFlag,
		HistoryFileFlagDefault,
		HistoryFileFlagUsage,
	)

	cmd.Flags().BoolP(
		SaveHistoryFlag,
		SaveHistoryFlagShort,
		false,
		SaveHistoryFlagUsage,
	)

	cmd.Flags().StringP(
		HistoryLabelFlag,
		HistoryLabelFlagShort,
		"",
		HistoryLabelFlagUsage,
	)

	cmd.Flags().StringP(
		CompareHistoryFlag,
		CompareHistoryFlagShort,
		"",
		CompareHistoryFlagUsage,
	)

	cmd.Flags().BoolP(
		ShowHistoryFlag,
		ShowHistoryFlagShort,
		false,
		ShowHistoryFlagUsage,
	)

	cmd.Flags().StringP(
		DeleteHistoryFlag,
		DeleteHistoryFlagShort,
		"",
		DeleteHistoryFlagUsage,
	)

	cmd.Flags().IntP(
		HistoryLimitFlag,
		HistoryLimitFlagShort,
		0,
		HistoryLimitFlagUsage,
	)

	cmd.Flags().StringP(
		ModuleNameFlag,
		ModuleNameFlagShort,
		"",
		ModuleNameFlagUsage,
	)

	cmd.Flags().Bool(
		InitFlag,
		false,
		InitFlagUsage,
	)

	cmd.Flags().Bool(
		DiffOnlyFlag,
		false,
		DiffOnlyFlagUsage,
	)

	cmd.Flags().String(
		AgainstFlag,
		"",
		AgainstFlagUsage,
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

func initConfigFile(cmd *cobra.Command) error {
	configPath, _ := cmd.Flags().GetString(ConfigFlag)

	// check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file %s already exists", configPath)
	}

	// write the embedded sample config to the current directory
	err := os.WriteFile(configPath, []byte(samples.SampleConfigYAML), ConfigFilePermissions)
	if err != nil {
		return fmt.Errorf("failed to create config file %s: %w", configPath, err)
	}

	fmt.Printf("Created sample config file: %s\n", configPath)
	return nil
}
