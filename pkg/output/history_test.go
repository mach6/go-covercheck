package output_test

import (
	"fmt"
	"testing"
	"time"

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

func functionHistoryEntry(filePct, pkgPct, totalPct float64) *history.Entry {
	return &history.Entry{
		Commit:    "0123456789abcdef",
		Branch:    "main",
		Timestamp: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Results: compute.Results{
			ByFile: []compute.ByFile{{
				File: "pkg/a.go",
				By:   compute.By{Functions: "1/2", FunctionPercentage: filePct},
			}},
			ByPackage: []compute.ByPackage{{
				Package: "pkg",
				By:      compute.By{Functions: "1/2", FunctionPercentage: pkgPct},
			}},
			ByTotal: compute.Totals{
				Statements: compute.TotalStatements{Coverage: "1/2"},
				Blocks:     compute.TotalBlocks{Coverage: "1/2"},
				Functions:  compute.TotalFunctions{Coverage: "1/2", Percentage: totalPct},
			},
		},
	}
}

func TestCompareHistory_FunctionCoverage(t *testing.T) {
	ref := functionHistoryEntry(50, 50, 50)
	current := functionHistoryEntry(100, 75, 60).Results

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		output.CompareHistory("main", ref, current)
	})

	require.Empty(t, stderr)
	require.Equal(t, `
≡ Comparing against ref: main [commit 0123456]
 → By File
    [F] pkg/a.go [+50.0%]
 → By Package
    [F] pkg [+25.0%]
 → By Total
    [F] total [+10.0%]
`, stdout)
}

func TestCompareHistory_LegacyEntryWithoutFunctionCoverage(t *testing.T) {
	ref := functionHistoryEntry(0, 0, 0)
	ref.Results.ByFile[0].Functions = ""
	ref.Results.ByPackage[0].Functions = ""
	ref.Results.ByTotal.Functions.Coverage = ""
	current := functionHistoryEntry(100, 100, 100).Results

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		output.CompareHistory("main", ref, current)
	})

	require.Empty(t, stderr)
	require.NotContains(t, stdout, "[F]")
	require.Contains(t, stdout, "→ No change")
}

func TestShowHistory_FunctionCoverage(t *testing.T) {
	h := history.New("")
	h.Entries = append(h.Entries, *functionHistoryEntry(50, 50, 50))

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		output.ShowHistory(h, 1, new(config.Config))
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "1/2     [B]")
	require.Contains(t, stdout, "1/2     [F]")
	require.NotContains(t, stdout, "[L]")
}
