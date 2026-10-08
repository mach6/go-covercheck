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
