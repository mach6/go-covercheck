package formatter //nolint:testpackage

import (
	"bytes"
	"os"
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func TestFormatAndReport_FailsWhenUnderThreshold(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.StatementThreshold = 80.0
	cfg.BlockThreshold = 70.0
	cfg.SortBy = "file"
	cfg.SortOrder = "asc"

	profiles := []*cover.Profile{
		{
			FileName: "example/foo.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 10, Count: 1},
				{NumStmt: 10, Count: 0},
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	failed := FormatAndReport(profiles, cfg)

	_ = w.Close()
	os.Stdout = old

	var out bytes.Buffer
	_, _ = out.ReadFrom(r)

	require.True(t, failed)
	require.Contains(t, out.String(), "TOTAL")
}

func TestFormatAndReport_JSONOutput(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatJSON
	cfg.StatementThreshold = 0
	cfg.BlockThreshold = 0

	profiles := []*cover.Profile{
		{
			FileName: "main.go",
			Blocks: []cover.ProfileBlock{
				{NumStmt: 5, Count: 1},
			},
		},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	failed := FormatAndReport(profiles, cfg)

	_ = w.Close()
	os.Stdout = old

	var out bytes.Buffer
	_, _ = out.ReadFrom(r)

	require.False(t, failed)
	require.Contains(t, out.String(), `"file":`)
	require.Contains(t, out.String(), `"main.go"`)
}

func Test_sortResults_ByFile(t *testing.T) {
	results := []ByFile{
		{
			File: "b",
			By: By{
				Statements:          "2/2",
				Blocks:              "2/2",
				StatementPercentage: 100,
				BlockPercentage:     100,
				StatementThreshold:  0,
				BlockThreshold:      0,
				Failed:              false,
			},
		},
		{
			File: "a",
			By: By{
				Statements:          "1/2",
				Blocks:              "1/2",
				StatementPercentage: 50,
				BlockPercentage:     50,
				StatementThreshold:  0,
				BlockThreshold:      0,
				Failed:              false,
			},
		},
		{
			File: "c",
			By: By{
				Statements:          "0/3",
				Blocks:              "0/3",
				StatementPercentage: 0,
				BlockPercentage:     0,
				StatementThreshold:  0,
				BlockThreshold:      0,
				Failed:              true,
			},
		},
	}

	cfg := new(config.Config)

	// test ascending
	cfg.ApplyDefaults()
	sortFileResults(results, cfg)
	expect := []string{"a", "b", "c"}
	for i, v := range results {
		require.Equal(t, expect[i], v.File)
	}

	// test descending
	cfg.SortOrder = config.SortOrderDesc
	sortFileResults(results, cfg)
	expect = []string{"c", "b", "a"}
	for i, v := range results {
		require.Equal(t, expect[i], v.File)
	}
}
