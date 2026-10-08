package output_test

import (
	"fmt"
	"slices"
	"strings"
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
		output.PrintComparison(buildComparison(entry, results))
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
				output.PrintComparison(buildComparison(entry, results))
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

func historyEntry(commit, label string, tags []string, withLines bool) history.Entry {
	e := history.Entry{
		Commit:    commit,
		Branch:    "main",
		Tags:      tags,
		Label:     label,
		Timestamp: time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC),
	}
	e.Results.ByTotal.Statements.Coverage = "90/100"
	e.Results.ByTotal.Statements.Percentage = 90
	e.Results.ByTotal.Statements.Threshold = 80
	e.Results.ByTotal.Blocks.Coverage = "40/50"
	e.Results.ByTotal.Blocks.Percentage = 80
	e.Results.ByTotal.Blocks.Threshold = 75
	if withLines {
		e.Results.ByTotal.Lines.Coverage = "70/80"
		e.Results.ByTotal.Lines.Percentage = 87.5
		e.Results.ByTotal.Lines.Threshold = 85
	}
	return e
}

func TestShowHistoryEmpty(t *testing.T) {
	for _, limit := range []int{0, 5} {
		stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
			output.ShowHistory(&history.History{}, limit, new(config.Config))
		})
		require.Empty(t, stderr)
		require.Equal(t, "≡ No history entries to show\n", stdout)
	}
}

func TestShowHistoryLimit(t *testing.T) {
	h := &history.History{Entries: []history.Entry{
		historyEntry("aaaaaaa1111", "", nil, false),
		historyEntry("bbbbbbb2222", "", nil, false),
		historyEntry("ccccccc3333", "", nil, false),
	}}

	tests := []struct {
		name    string
		limit   int
		commits []string
		footer  string
	}{
		{"limit below count", 2, []string{"aaaaaaa", "bbbbbbb"}, "≡ Showing last 2 history entries\n"},
		{"limit one", 1, []string{"aaaaaaa"}, "≡ Showing last 1 history entry\n"},
		{"zero means all", 0, []string{"aaaaaaa", "bbbbbbb", "ccccccc"}, "≡ Showing last 3 history entries\n"},
		{"negative means all", -1, []string{"aaaaaaa", "bbbbbbb", "ccccccc"}, "≡ Showing last 3 history entries\n"},
		{"limit above count", 10, []string{"aaaaaaa", "bbbbbbb", "ccccccc"}, "≡ Showing last 3 history entries\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
				output.ShowHistory(h, tt.limit, new(config.Config))
			})
			require.Empty(t, stderr)
			for _, c := range []string{"aaaaaaa", "bbbbbbb", "ccccccc"} {
				require.Equal(t, slices.Contains(tt.commits, c), strings.Contains(stdout, c), c)
			}
			require.True(t, strings.HasSuffix(stdout, tt.footer), stdout)
		})
	}
}

func TestShowHistoryLinesTagsAndLabel(t *testing.T) {
	h := &history.History{Entries: []history.Entry{
		historyEntry("aaaaaaa1111", "release candidate number one", []string{"v1.0.0", "stable", "release"}, true),
		historyEntry("bbbbbbb2222", "short", []string{"v0.9"}, false),
	}}
	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		output.ShowHistory(h, 0, new(config.Config))
	})
	require.Empty(t, stderr)

	// line coverage only for the entry that has it
	require.Equal(t, 1, strings.Count(stdout, "[L]"))
	require.Contains(t, stdout, "70/80   [L]")
	require.Equal(t, 2, strings.Count(stdout, "[S]"))
	require.Equal(t, 2, strings.Count(stdout, "[B]"))

	// long tags and label are wrapped onto multiple lines
	require.Contains(t, stdout, "v1.0.0, stable,")
	require.Contains(t, stdout, "release")
	require.Contains(t, stdout, "release candidate")
	require.Contains(t, stdout, "number one")
	require.NotContains(t, stdout, "release candidate number one")
	require.Contains(t, stdout, "v0.9")
	require.Contains(t, stdout, "short")
	require.Contains(t, stdout, "2025-08-01")
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"empty", "", 10, ""},
		{"shorter than width", "alpha beta", 10, "alpha beta"},
		{"exactly width", "abcdefghij", 10, "abcdefghij"},
		{"wraps on words", "alpha beta gamma delta", 10, "alpha beta\ngamma\ndelta"},
		{"wraps tag list", "v1.0.0, stable, release", 20, "v1.0.0, stable,\nrelease"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, output.WrapText(tt.in, tt.width))
		})
	}
}

func TestFormatDelta(t *testing.T) {
	s, ok := output.FormatDelta(0)
	require.False(t, ok)
	require.Empty(t, s)

	s, ok = output.FormatDelta(2.5)
	require.True(t, ok)
	require.Equal(t, "+2.5 %", s)

	s, ok = output.FormatDelta(-12.5)
	require.True(t, ok)
	require.Equal(t, "−12.5%", s)
}
