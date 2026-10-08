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

func loadYAMLForTest(t *testing.T, yaml string) *config.Config {
	t.Helper()

	p := path.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(p, []byte(yaml), 0600))
	cfg, err := config.Load(p)
	require.NoError(t, err)
	return cfg
}

func TestLoad_TotalDefaultsToGlobalThresholds(t *testing.T) {
	cfg := loadYAMLForTest(t, "statementThreshold: 90\nblockThreshold: 80\nlineThreshold: 85\n")
	require.InDelta(t, 90.0, cfg.Total[config.StatementsSection], 0)
	require.InDelta(t, 80.0, cfg.Total[config.BlocksSection], 0)
	require.InDelta(t, 85.0, cfg.Total[config.LinesSection], 0)
}

func TestLoad_ExplicitTotalWinsOverGlobal(t *testing.T) {
	cfg := loadYAMLForTest(t, "statementThreshold: 90\nblockThreshold: 80\nlineThreshold: 85\n"+
		"total:\n  statements: 10\n  blocks: 0\n")
	require.InDelta(t, 10.0, cfg.Total[config.StatementsSection], 0)
	require.InDelta(t, 0.0, cfg.Total[config.BlocksSection], 0) // explicit 0 means disabled, not unset
	require.InDelta(t, 85.0, cfg.Total[config.LinesSection], 0) // unset total follows its global
}

func TestLoad_NoThresholdsUsesBuiltInDefaults(t *testing.T) {
	cfg := loadYAMLForTest(t, "sortBy: lines\n")
	require.InDelta(t, config.StatementThresholdDefault, cfg.Total[config.StatementsSection], 0)
	require.InDelta(t, config.BlockThresholdDefault, cfg.Total[config.BlocksSection], 0)
	require.InDelta(t, config.LineThresholdDefault, cfg.Total[config.LinesSection], 0)
}

func TestValidate_DerivedTotalFollowsGlobalButExplicitDoesNot(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.SetTotalThreshold(config.BlocksSection, 20)
	cfg.StatementThreshold = 95
	cfg.BlockThreshold = 95
	require.NoError(t, cfg.Validate())
	require.InDelta(t, 95.0, cfg.Total[config.StatementsSection], 0)
	require.InDelta(t, 20.0, cfg.Total[config.BlocksSection], 0)
}
