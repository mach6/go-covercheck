package output_test

import (
	"fmt"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/history"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func TestCompareHistory(t *testing.T) {
	cPath := test.CreateTempCoverageFile(t, test.TestCoverageOut)
	profiles, err := cover.ParseProfiles(cPath)
	require.NoError(t, err)
	require.NotEmpty(t, profiles)

	results, _ := compute.CollectResults(profiles, new(config.Config))
	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		hPath := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
		h, err := history.Load(hPath)
		require.NoError(t, err)
		entry := h.FindByRef("main")
		output.CompareHistory("main", entry, results)
	})

	require.Empty(t, stderr)
	require.Equal(t, `
≡ Comparing against ref: main [commit e402629]
 → By File
    [S] github.com/mach6/go-covercheck/pkg/math/math.go [−25.0%]
    [B] github.com/mach6/go-covercheck/pkg/math/math.go [−25.0%]
 → By Package
    [S] github.com/mach6/go-covercheck/pkg/math [−25.0%]
    [B] github.com/mach6/go-covercheck/pkg/math [−25.0%]
 → By Total
    [S] total [+22.2%]
    [B] total [+26.8%]
`, stdout)
}

func TestShowHistory(t *testing.T) {
	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		hPath := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
		h, err := history.Load(hPath)
		require.NoError(t, err)
		output.ShowHistory(h, 1, new(config.Config))
	})
	require.Empty(t, stderr)
	require.Equal(t, `┌────────────┬─────────┬─────────────────┬─────────────────┬─────────────────┬─────────────┐
│  TIMESTAMP │  COMMIT │      BRANCH     │       TAGS      │      LABEL      │   COVERAGE  │
├────────────┼─────────┼─────────────────┼─────────────────┼─────────────────┼─────────────┤
│ 2025-07-18 │ e402629 │ main            │                 │                 │ 180/648 [S] │
│            │         │                 │                 │                 │ 95/409  [B] │
└────────────┴─────────┴─────────────────┴─────────────────┴─────────────────┴─────────────┘
≡ Showing last 1 history entry
`, stdout)
}

func shortCommitCases() map[string]string {
	return map[string]string{
		"":    "",
		"abc": "abc",
		"e402629a1b2c3d4e5f60718293a4b5c6d7e8f901": "e402629",
	}
}

func TestCompareHistory_ShortCommits(t *testing.T) {
	cPath := test.CreateTempCoverageFile(t, test.TestCoverageOut)
	profiles, err := cover.ParseProfiles(cPath)
	require.NoError(t, err)
	results, _ := compute.CollectResults(profiles, new(config.Config))

	for commit, want := range shortCommitCases() {
		hPath := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
		h, err := history.Load(hPath)
		require.NoError(t, err)
		entry := h.FindByRef("main")
		require.NotNil(t, entry)
		entry.Commit = commit

		var out string
		require.NotPanics(t, func() {
			out, _ = test.RepipeStdOutAndErrForTest(func() {
				output.CompareHistory("main", entry, results)
			})
		})
		require.Contains(t, out, "[commit "+want+"]")
	}
}

func TestShowHistory_ShortCommits(t *testing.T) {
	for commit, want := range shortCommitCases() {
		hPath := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
		h, err := history.Load(hPath)
		require.NoError(t, err)
		require.NotEmpty(t, h.Entries)
		h.Entries[0].Commit = commit

		var out string
		require.NotPanics(t, func() {
			out, _ = test.RepipeStdOutAndErrForTest(func() {
				output.ShowHistory(h, 1, new(config.Config))
			})
		})
		require.Contains(t, out, "│ "+fmt.Sprintf("%-7s", want)+" │")
	}
}
