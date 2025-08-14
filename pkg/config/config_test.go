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
