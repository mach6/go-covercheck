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

func TestLoad_ValidYAML_WithTableStyle(t *testing.T) {
	yaml := `
statementThreshold: 75.0
tableStyle: bold
`
	tmpFile := path.Join(t.TempDir(), "test_config_table_style.yaml")
	err := os.WriteFile(tmpFile, []byte(yaml), 0600)
	require.NoError(t, err)
	defer os.Remove(tmpFile) //nolint:errcheck

	cfg, err := config.Load(tmpFile)
	require.NoError(t, err)
	require.Equal(t, "bold", cfg.TableStyle)
}

func TestApplyDefaults_TableStyle(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()
	require.Equal(t, config.TableStyleDefValue, cfg.TableStyle)
	require.Equal(t, "light", cfg.TableStyle) // Should be light by default
}

func TestValidate_InvalidTableStyle(t *testing.T) {
	cfg := &config.Config{
		StatementThreshold: 70,
		BlockThreshold:     50,
		SortBy:             "file",
		SortOrder:          "asc",
		Format:             "table",
		TableStyle:         "invalid",
	}
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "table-style must be one of")
}

func TestValidate_ValidTableStyles(t *testing.T) {
	validStyles := []string{
		config.TableStyleDefault,
		config.TableStyleLight,
		config.TableStyleBold,
		config.TableStyleRounded,
		config.TableStyleDouble,
	}

	for _, style := range validStyles {
		t.Run(style, func(t *testing.T) {
			cfg := &config.Config{
				StatementThreshold: 70,
				BlockThreshold:     50,
				SortBy:             "file",
				SortOrder:          "asc",
				Format:             "table",
				TableStyle:         style,
			}
			err := cfg.Validate()
			require.NoError(t, err)
		})
	}
}
