package compute //nolint:testpackage

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
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
