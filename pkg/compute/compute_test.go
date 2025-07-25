package compute //nolint:testpackage

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

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

func TestCollectResults(t *testing.T) {
	profiles := make([]*cover.Profile, 0)
	profiles = append(profiles, &cover.Profile{
		FileName: "foo",
		Mode:     "set",
		Blocks: []cover.ProfileBlock{
			{
				StartLine: 0,
				StartCol:  0,
				EndLine:   10,
				EndCol:    120,
				NumStmt:   150,
				Count:     1,
			},
		},
	})
	r, failed := CollectResults(profiles, new(config.Config))
	require.False(t, failed)
	require.NotNil(t, r)
	require.InEpsilon(t, 100.0, r.ByTotal.Blocks.Percentage, 0)
	require.InEpsilon(t, 100.0, r.ByTotal.Statements.Percentage, 0)
}
