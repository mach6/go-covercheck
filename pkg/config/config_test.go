package config_test

import (
	"os"
	"path"
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidYAML_WithDefaults(t *testing.T) {
	yaml := `
statementThreshold: 75.0
perFile:
  statements:
    foo/bar.go: 90.0
`
	tmpFile := path.Join(t.TempDir(), "test_config_valid.yaml")
	err := os.WriteFile(tmpFile, []byte(yaml), 0600)
	require.NoError(t, err)
	defer os.Remove(tmpFile) //nolint:errcheck

	cfg, err := config.Load(tmpFile)
	require.NoError(t, err)
	require.InEpsilon(t, 75.0, cfg.StatementThreshold, 1)
	require.InEpsilon(t, 90.0, cfg.PerFile.Statements["foo/bar.go"], 1)
	require.Equal(t, "file", cfg.SortBy)
	require.Equal(t, "asc", cfg.SortOrder)
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := config.Load("does_not_exist.yaml")
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpFile := path.Join(t.TempDir(), "test_config_invalid.yaml")
	_ = os.WriteFile(tmpFile, []byte("statementThreshold: [not-a-number]"), 0600)
	defer os.Remove(tmpFile) //nolint:errcheck

	cfg, err := config.Load(tmpFile)
	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestLoad_SortByConfig(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		expectedSortBy string
		expectedOrder  string
		shouldError    bool
	}{
		{
			name: "sortBy statement-percent with desc order",
			yamlContent: `
statementThreshold: 70.0
sortBy: statement-percent
sortOrder: desc
`,
			expectedSortBy: "statement-percent",
			expectedOrder:  "desc",
			shouldError:    false,
		},
		{
			name: "sortBy blocks with asc order",
			yamlContent: `
statementThreshold: 70.0
sortBy: blocks
sortOrder: asc
`,
			expectedSortBy: "blocks",
			expectedOrder:  "asc",
			shouldError:    false,
		},
		{
			name: "sortBy file (default order)",
			yamlContent: `
statementThreshold: 70.0
sortBy: file
`,
			expectedSortBy: "file",
			expectedOrder:  "asc", // should use default
			shouldError:    false,
		},
		{
			name: "sortBy statements",
			yamlContent: `
statementThreshold: 70.0
sortBy: statements
sortOrder: desc
`,
			expectedSortBy: "statements",
			expectedOrder:  "desc",
			shouldError:    false,
		},
		{
			name: "sortBy block-percent",
			yamlContent: `
statementThreshold: 70.0
sortBy: block-percent
sortOrder: asc
`,
			expectedSortBy: "block-percent",
			expectedOrder:  "asc",
			shouldError:    false,
		},
		{
			name: "sortBy functions",
			yamlContent: `
sortBy: functions
`,
			expectedSortBy: "functions",
			expectedOrder:  "asc",
		},
		{
			name: "sortBy function-percent",
			yamlContent: `
sortBy: function-percent
sortOrder: desc
`,
			expectedSortBy: "function-percent",
			expectedOrder:  "desc",
		},
		{
			name: "invalid sortBy option",
			yamlContent: `
statementThreshold: 70.0
sortBy: invalid-option
`,
			expectedSortBy: "",
			expectedOrder:  "",
			shouldError:    true,
		},
		{
			name: "invalid sortOrder option",
			yamlContent: `
statementThreshold: 70.0
sortBy: file
sortOrder: invalid-order
`,
			expectedSortBy: "",
			expectedOrder:  "",
			shouldError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := path.Join(t.TempDir(), "test_config_sortby.yaml")
			err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0600)
			require.NoError(t, err)
			defer os.Remove(tmpFile) //nolint:errcheck

			cfg, err := config.Load(tmpFile)

			if tt.shouldError {
				require.Error(t, err)
				require.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				require.Equal(t, tt.expectedSortBy, cfg.SortBy)
				require.Equal(t, tt.expectedOrder, cfg.SortOrder)
			}
		})
	}
}

func TestApplyDefaults_TableStyle(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()
	require.Equal(t, config.TableStyleDefValue, cfg.TableStyle)
	require.Equal(t, config.TableStyleLight, cfg.TableStyle)
}

func TestLoad_TableStyleFromYAML(t *testing.T) {
	yaml := `
statementThreshold: 70.0
tableStyle: bold
`
	tmpFile := path.Join(t.TempDir(), "test_table_style.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(yaml), 0600))
	defer os.Remove(tmpFile) //nolint:errcheck

	cfg, err := config.Load(tmpFile)
	require.NoError(t, err)
	require.Equal(t, config.TableStyleBold, cfg.TableStyle)
}

func TestValidate_TableStyle(t *testing.T) {
	valid := []string{
		config.TableStyleDefault,
		config.TableStyleLight,
		config.TableStyleBold,
		config.TableStyleRounded,
		config.TableStyleDouble,
	}
	for _, style := range valid {
		t.Run("valid/"+style, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.ApplyDefaults()
			cfg.TableStyle = style
			require.NoError(t, cfg.Validate())
		})
	}

	t.Run("invalid style", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		cfg.TableStyle = "plaid"
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "table-style")
	})
}

func TestValidate_SyntaxStyle(t *testing.T) {
	valid := []string{
		config.SyntaxStyleAuto,
		"github",
		"github-dark",
		"monokai",
		"dracula",
	}
	for _, style := range valid {
		t.Run("valid/"+style, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.ApplyDefaults()
			cfg.SyntaxStyle = style
			require.NoError(t, cfg.Validate())
		})
	}

	t.Run("unknown chroma style is rejected", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		cfg.SyntaxStyle = "definitely-not-a-chroma-style"
		err := cfg.Validate()
		require.Error(t, err)
		require.Contains(t, err.Error(), "syntax-style")
	})
}

func TestLoad_FunctionThresholds(t *testing.T) {
	yaml := `
functionThreshold: 65.0
perFile:
  functions:
    foo/bar.go: 90.0
perPackage:
  functions:
    foo: 80.0
total:
  functions: 75.0
`
	tmpFile := path.Join(t.TempDir(), "test_config_functions.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(yaml), 0600))

	cfg, err := config.Load(tmpFile)
	require.NoError(t, err)
	require.InDelta(t, 65.0, cfg.FunctionThreshold, 0.001)
	require.InDelta(t, 90.0, cfg.PerFile.Functions["foo/bar.go"], 0.001)
	require.InDelta(t, 80.0, cfg.PerPackage.Functions["foo"], 0.001)
	require.InDelta(t, 75.0, cfg.Total[config.FunctionsSection], 0.001)
}

func TestApplyDefaults_FunctionThresholdDisabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()
	require.Zero(t, cfg.FunctionThreshold)
	require.NotNil(t, cfg.PerFile.Functions)
	require.NotNil(t, cfg.PerPackage.Functions)
	v, ok := cfg.Total[config.FunctionsSection]
	require.True(t, ok)
	require.Zero(t, v)
}

func TestValidate_FunctionThreshold(t *testing.T) {
	for _, tc := range []struct {
		threshold float64
		wantErr   bool
	}{
		{threshold: -0.1, wantErr: true},
		{threshold: 0},
		{threshold: 100},
		{threshold: 100.1, wantErr: true},
	} {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		cfg.FunctionThreshold = tc.threshold
		err := cfg.Validate()
		if tc.wantErr {
			require.EqualError(t, err, "function threshold must be between 0 and 100")
		} else {
			require.NoError(t, err)
		}
	}
}

func TestLoad_Comment(t *testing.T) {
	yaml := `
comment:
  enabled: true
  platform:
    type: GitLab
    repository: group/project
    pullRequestId: 4
    updateExisting: true
    author: ci-bot
`
	tmpFile := path.Join(t.TempDir(), "comment.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(yaml), 0600))

	cfg, err := config.Load(tmpFile)
	require.NoError(t, err)
	require.True(t, cfg.Comment.Enabled)
	require.Equal(t, "gitlab", cfg.Comment.Platform.Type, "platform type is normalized to lower case")
	require.Equal(t, "group/project", cfg.Comment.Platform.Repository)
	require.Equal(t, 4, cfg.Comment.Platform.PullRequestID)
	require.True(t, cfg.Comment.Platform.UpdateExisting)
	require.Equal(t, "ci-bot", cfg.Comment.Platform.Author)
	require.True(t, cfg.Comment.Platform.IncludeColors, "includeColors defaults to true")
}

func TestLoad_CommentIncludeColorsDisabled(t *testing.T) {
	tmpFile := path.Join(t.TempDir(), "comment.yaml")
	require.NoError(t, os.WriteFile(tmpFile, []byte("comment:\n  platform:\n    includeColors: false\n"), 0600))

	cfg, err := config.Load(tmpFile)
	require.NoError(t, err)
	require.False(t, cfg.Comment.Platform.IncludeColors)
}

func TestValidate_CommentPlatform(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()
	require.NoError(t, cfg.Validate(), "no platform is valid until comments are enabled")

	cfg.Comment.Platform.Type = "bitbucket"
	require.ErrorContains(t, cfg.Validate(), "comment platform must be one of github|gitlab|gitea")
}

func TestValidate_Format(t *testing.T) {
	for _, format := range []string{
		config.FormatTable, config.FormatJSON, config.FormatYAML, config.FormatMD,
		config.FormatHTML, config.FormatCSV, config.FormatTSV, config.FormatHeatmap,
	} {
		t.Run("valid/"+format, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.ApplyDefaults()
			cfg.Format = format
			require.NoError(t, cfg.Validate())
		})
	}

	cfg := &config.Config{}
	cfg.ApplyDefaults()
	cfg.Format = "png"
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), config.FormatHeatmap)
}

func TestValidate_HeatmapMetric(t *testing.T) {
	for _, m := range []string{
		config.HeatmapMetricWorst, config.HeatmapMetricStatements, config.HeatmapMetricBlocks,
		config.HeatmapMetricLines, config.HeatmapMetricFunctions,
	} {
		cfg := &config.Config{}
		cfg.ApplyDefaults()
		cfg.HeatmapMetric = m
		require.NoError(t, cfg.Validate(), m)
	}

	cfg := &config.Config{}
	cfg.ApplyDefaults()
	require.Equal(t, config.HeatmapMetricWorst, cfg.HeatmapMetric)
	cfg.HeatmapMetric = "branches"
	require.ErrorContains(t, cfg.Validate(), "heatmap-metric must be one of")
}

func loadYAML(t *testing.T, content string) *config.Config {
	t.Helper()
	p := path.Join(t.TempDir(), ".go-covercheck.yml")
	require.NoError(t, os.WriteFile(p, []byte(content), 0600))
	cfg, err := config.Load(p)
	require.NoError(t, err)
	return cfg
}

func TestLoad_TotalDefaultsToGlobalThresholds(t *testing.T) {
	cfg := loadYAML(t, `
statementThreshold: 91
blockThreshold: 92
lineThreshold: 93
functionThreshold: 94
`)

	require.InDelta(t, 91, cfg.Total[config.StatementsSection], 0.001)
	require.InDelta(t, 92, cfg.Total[config.BlocksSection], 0.001)
	require.InDelta(t, 93, cfg.Total[config.LinesSection], 0.001)
	require.InDelta(t, 94, cfg.Total[config.FunctionsSection], 0.001)
}

func TestLoad_TotalDefaultsWhenNothingIsSet(t *testing.T) {
	cfg := loadYAML(t, "format: table\n")

	require.InDelta(t, config.StatementThresholdDefault, cfg.Total[config.StatementsSection], 0.001)
	require.InDelta(t, config.BlockThresholdDefault, cfg.Total[config.BlocksSection], 0.001)
	require.InDelta(t, config.LineThresholdDefault, cfg.Total[config.LinesSection], 0.001)
	require.InDelta(t, config.FunctionThresholdDefault, cfg.Total[config.FunctionsSection], 0.001)
}

func TestLoad_TotalIsDisabledWithGlobalZero(t *testing.T) {
	cfg := loadYAML(t, "lineThreshold: 0\n")

	require.InDelta(t, 0, cfg.Total[config.LinesSection], 0.001)
}

func TestLoad_TotalOverridesAreIndependentOfGlobals(t *testing.T) {
	cfg := loadYAML(t, `
statementThreshold: 91
lineThreshold: 0
total:
  lines: 40
  blocks: 0
`)

	require.InDelta(t, 40, cfg.Total[config.LinesSection], 0.001, "an explicit total wins over the global threshold")
	require.InDelta(t, 0, cfg.Total[config.BlocksSection], 0.001, "an explicit 0 disables the total")
	require.InDelta(t, 91, cfg.Total[config.StatementsSection], 0.001, "totals not set follow their global threshold")
}

func TestFollowGlobalThresholds(t *testing.T) {
	t.Run("totals filled in from the globals follow a changed global", func(t *testing.T) {
		cfg := loadYAML(t, "statementThreshold: 91\nlineThreshold: 0\n")

		cfg.StatementThreshold = 0
		cfg.LineThreshold = 80
		cfg.FollowGlobalThresholds()

		require.InDelta(t, 0, cfg.Total[config.StatementsSection], 0.001)
		require.InDelta(t, 80, cfg.Total[config.LinesSection], 0.001)
	})

	t.Run("totals set in the file are left alone", func(t *testing.T) {
		cfg := loadYAML(t, "statementThreshold: 91\ntotal:\n  statements: 60\n  lines: 0\n")

		cfg.StatementThreshold = 10
		cfg.LineThreshold = 80
		cfg.FollowGlobalThresholds()

		require.InDelta(t, 60, cfg.Total[config.StatementsSection], 0.001)
		require.InDelta(t, 0, cfg.Total[config.LinesSection], 0.001)
	})

	t.Run("a total changed after it was filled in is left alone", func(t *testing.T) {
		cfg := new(config.Config)
		cfg.ApplyDefaults()
		cfg.Total[config.StatementsSection] = 65

		cfg.StatementThreshold = 90
		cfg.FollowGlobalThresholds()

		require.InDelta(t, 65, cfg.Total[config.StatementsSection], 0.001)
	})

	t.Run("defaults follow the globals too", func(t *testing.T) {
		cfg := new(config.Config)
		cfg.ApplyDefaults()

		cfg.BlockThreshold = 30
		cfg.FollowGlobalThresholds()

		require.InDelta(t, 30, cfg.Total[config.BlocksSection], 0.001)
		require.InDelta(t, config.StatementThresholdDefault, cfg.Total[config.StatementsSection], 0.001)
	})

	t.Run("repeated calls keep following", func(t *testing.T) {
		cfg := loadYAML(t, "statementThreshold: 91\n")

		cfg.StatementThreshold = 50
		cfg.FollowGlobalThresholds()
		cfg.StatementThreshold = 20
		cfg.FollowGlobalThresholds()

		require.InDelta(t, 20, cfg.Total[config.StatementsSection], 0.001)
	})
}

func TestLoad_HideColumns(t *testing.T) {
	cfg := loadYAML(t, "hideColumns: [Statements, ' blocks ', statements, '', line-percent]\n")

	// names are lower-cased and trimmed, and repeats and blanks are dropped.
	require.Equal(t, []string{"statements", "blocks", "line-percent"}, cfg.HideColumns)
	require.True(t, cfg.ColumnHidden(config.ColumnStatements))
	require.True(t, cfg.ColumnHidden(config.ColumnBlocks))
	require.True(t, cfg.ColumnHidden(config.ColumnLinePercent))
	require.False(t, cfg.ColumnHidden(config.ColumnLines))
}

func TestLoad_HideColumnsNothingHiddenByDefault(t *testing.T) {
	cfg := loadYAML(t, "format: table\n")

	require.Empty(t, cfg.HideColumns)
	for _, name := range config.HideableColumns {
		require.False(t, cfg.ColumnHidden(name), name)
	}
}

func TestValidate_HideColumns(t *testing.T) {
	for _, name := range config.HideableColumns {
		cfg := new(config.Config)
		cfg.ApplyDefaults()
		cfg.HideColumns = []string{name}
		require.NoError(t, cfg.Validate(), name)
	}

	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.HideColumns = []string{"lines", "nope"}
	err := cfg.Validate()
	require.Error(t, err)
	require.ErrorContains(t, err, `"nope"`)
	require.ErrorContains(t, err, "statement-percent", "the error lists the valid names")
}

func TestLoad_HideColumnsRejectsUnknownName(t *testing.T) {
	p := path.Join(t.TempDir(), ".go-covercheck.yml")
	require.NoError(t, os.WriteFile(p, []byte("hideColumns: [sorting]\n"), 0600))

	cfg, err := config.Load(p)

	require.Error(t, err)
	require.Nil(t, cfg)
	require.ErrorContains(t, err, `"sorting"`)
}

func TestValidate_NoTableAndNoSummaryTogether(t *testing.T) {
	for _, format := range []string{config.FormatTable, config.FormatMD, config.FormatCSV, config.FormatHeatmap} {
		cfg := new(config.Config)
		cfg.ApplyDefaults()
		cfg.Format = format
		cfg.NoTable = true
		cfg.NoSummary = true
		require.ErrorContains(t, cfg.Validate(), "cannot specify both no-summary and no-table", format)
	}

	// json and yaml ignore both flags, so the pair is allowed there.
	for _, format := range []string{config.FormatJSON, config.FormatYAML} {
		cfg := new(config.Config)
		cfg.ApplyDefaults()
		cfg.Format = format
		cfg.NoTable = true
		cfg.NoSummary = true
		require.NoError(t, cfg.Validate(), format)
	}
}
