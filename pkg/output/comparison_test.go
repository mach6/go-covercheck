package output_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/history"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func ptr(f float64) *float64 { return &f }

func buildComparison(entry *history.Entry, results compute.Results) *compute.Comparison {
	return compute.BuildComparison("main", history.ShortCommit(entry.Commit), entry.Results, results)
}

func testEntry() *history.Entry {
	by := func(s, b, l float64) compute.By {
		return compute.By{StatementPercentage: s, BlockPercentage: b, LinePercentage: l, Lines: "1/2"}
	}
	totals := compute.Totals{}
	totals.Statements.Percentage = 50
	totals.Blocks.Percentage = 50
	totals.Lines.Percentage = 50
	totals.Lines.Coverage = "1/2"
	return &history.Entry{
		Commit: "0123456789abcdef",
		Results: compute.Results{
			ByFile: []compute.ByFile{
				{File: "a.go", By: by(50, 50, 50)},
				{File: "same.go", By: by(50, 50, 50)},
				{File: "gone.go", By: by(10, 10, 10)},
			},
			ByPackage: []compute.ByPackage{{Package: "p", By: by(50, 50, 50)}},
			ByTotal:   totals,
		},
	}
}

func testResults() compute.Results {
	by := func(s, b, l float64) compute.By {
		return compute.By{StatementPercentage: s, BlockPercentage: b, LinePercentage: l}
	}
	totals := compute.Totals{}
	totals.Statements.Percentage = 75
	totals.Blocks.Percentage = 40
	totals.Lines.Percentage = 60
	return compute.Results{
		ByFile: []compute.ByFile{
			{File: "a.go", By: by(75, 40, 60)},
			{File: "same.go", By: by(50, 50, 50)},
			{File: "new.go", By: by(90, 90, 90)},
		},
		ByPackage: []compute.ByPackage{{Package: "p", By: by(75, 40, 60)}},
		ByTotal:   totals,
	}
}

func TestPrintComparison_NoChange(t *testing.T) {
	entry := testEntry()
	c := buildComparison(entry, entry.Results)

	// lists are empty rather than null, so JSON consumers can always iterate.
	require.Equal(t, []compute.FileDelta{}, c.ByFile)
	require.Equal(t, []compute.PackageDelta{}, c.ByPackage)
	require.Equal(t, compute.Changes{Files: []compute.FileCoverage{}, Packages: []compute.PackageCoverage{}}, c.Added)
	require.Equal(t, compute.Changes{Files: []compute.FileCoverage{}, Packages: []compute.PackageCoverage{}}, c.Removed)
	require.Nil(t, c.ByTotal)

	stdout, _ := test.RepipeStdOutAndErrForTest(func() { output.PrintComparison(c) })
	require.Contains(t, stdout, " → No change")
}

func reportWithComparison(t *testing.T, format string) string {
	t.Helper()
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = format
	cfg.NoColor = true
	current := color.NoColor
	color.NoColor = true
	t.Cleanup(func() { color.NoColor = current })

	results := testResults()
	c := buildComparison(testEntry(), results)
	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		output.FormatAndReportWithComparison(results, c, cfg, false)
	})
	require.Empty(t, stderr)
	return stdout
}

func TestFormatAndReportWithComparison_JSON(t *testing.T) {
	var doc map[string]any
	require.NoError(t, json.Unmarshal([]byte(reportWithComparison(t, config.FormatJSON)), &doc))

	// existing keys are untouched
	require.Contains(t, doc, "byFile")
	require.Contains(t, doc, "byPackage")
	require.Contains(t, doc, "byTotal")

	require.Equal(t, map[string]any{
		"ref":    "main",
		"commit": "0123456",
		"byFile": []any{
			map[string]any{"file": "a.go", "statements": 25.0, "blocks": -10.0, "lines": 10.0},
		},
		"byPackage": []any{
			map[string]any{"package": "p", "statements": 25.0, "blocks": -10.0, "lines": 10.0},
		},
		"added": map[string]any{
			"files":    []any{map[string]any{"file": "new.go", "statements": 90.0, "blocks": 90.0}},
			"packages": []any{},
		},
		"removed": map[string]any{
			"files":    []any{map[string]any{"file": "gone.go", "statements": 10.0, "blocks": 10.0, "lines": 10.0}},
			"packages": []any{},
		},
		"byTotal": map[string]any{"statements": 25.0, "blocks": -10.0, "lines": 10.0},
	}, doc["comparison"])
}

func TestFormatAndReportWithComparison_YAML(t *testing.T) {
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(reportWithComparison(t, config.FormatYAML)), &doc))

	require.Contains(t, doc, "byFile")
	require.Equal(t, map[string]any{
		"ref":    "main",
		"commit": "0123456",
		"byFile": []any{
			map[string]any{"file": "a.go", "statements": 25, "blocks": -10, "lines": 10},
		},
		"byPackage": []any{
			map[string]any{"package": "p", "statements": 25, "blocks": -10, "lines": 10},
		},
		"added": map[string]any{
			"files":    []any{map[string]any{"file": "new.go", "statements": 90, "blocks": 90}},
			"packages": []any{},
		},
		"removed": map[string]any{
			"files":    []any{map[string]any{"file": "gone.go", "statements": 10, "blocks": 10, "lines": 10}},
			"packages": []any{},
		},
		"byTotal": map[string]any{"statements": 25, "blocks": -10, "lines": 10},
	}, doc["comparison"])
}

func TestFormatAndReport_NoComparisonKeyWithoutComparison(t *testing.T) {
	for _, format := range []string{config.FormatJSON, config.FormatYAML} {
		cfg := new(config.Config)
		cfg.ApplyDefaults()
		cfg.Format = format
		cfg.NoColor = true
		stdout, _ := test.RepipeStdOutAndErrForTest(func() {
			output.FormatAndReport(testResults(), cfg, false)
		})
		require.NotContains(t, stdout, "comparison", format)
	}
}

func TestFormatAndReportWithComparison_TablePrintsText(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.NoColor = true
	results := testResults()
	c := buildComparison(testEntry(), results)

	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		output.FormatAndReportWithComparison(results, c, cfg, false)
	})
	require.Contains(t, stdout, "Comparing against ref: main [commit 0123456]")
	require.Contains(t, stdout, "[L] a.go [+10.0%]")
	require.Contains(t, stdout, "[B] total [−10.0%]")
}

func cov(s, b, l float64) compute.By {
	return compute.By{StatementPercentage: s, BlockPercentage: b, LinePercentage: l, Lines: "1/2"}
}

func entryWith(files []compute.ByFile, pkgs []compute.ByPackage) *history.Entry {
	return &history.Entry{Commit: "0123456789abcdef", Results: compute.Results{ByFile: files, ByPackage: pkgs}}
}

func TestPrintComparison_AddedOnly(t *testing.T) {
	entry := entryWith([]compute.ByFile{{File: "a.go", By: cov(50, 50, 50)}}, nil)
	results := compute.Results{ByFile: []compute.ByFile{
		{File: "b.go", By: cov(80, 70, 60)},
		{File: "a.go", By: cov(50, 50, 50)},
		{File: "0.go", By: cov(1, 2, 3)},
	}}
	c := buildComparison(entry, results)

	// sorted by name, not results order.
	require.Equal(t, []compute.FileCoverage{
		{File: "0.go", Coverage: compute.Coverage{Statements: 1, Blocks: 2, Lines: ptr(3)}},
		{File: "b.go", Coverage: compute.Coverage{Statements: 80, Blocks: 70, Lines: ptr(60)}},
	}, c.Added.Files)
	require.Empty(t, c.Removed.Files)
	require.Empty(t, c.ByFile)

	stdout, _ := test.RepipeStdOutAndErrForTest(func() { output.PrintComparison(c) })
	require.Contains(t, stdout, " → Added Files")
	require.Contains(t, stdout, "[S] b.go [80.0%]")
	require.Contains(t, stdout, "[L] b.go [60.0%]")
	require.NotContains(t, stdout, "No change")
	require.NotContains(t, stdout, "Removed")
}

func TestPrintComparison_RemovedOnly(t *testing.T) {
	entry := entryWith([]compute.ByFile{
		{File: "z.go", By: cov(10, 20, 30)},
		{File: "a.go", By: cov(50, 50, 50)},
		{File: "keep.go", By: cov(50, 50, 50)},
	}, nil)
	results := compute.Results{ByFile: []compute.ByFile{{File: "keep.go", By: cov(50, 50, 50)}}}
	c := buildComparison(entry, results)

	require.Empty(t, c.Added.Files)
	require.Equal(t, []compute.FileCoverage{
		{File: "a.go", Coverage: compute.Coverage{Statements: 50, Blocks: 50, Lines: ptr(50)}},
		{File: "z.go", Coverage: compute.Coverage{Statements: 10, Blocks: 20, Lines: ptr(30)}},
	}, c.Removed.Files)

	stdout, _ := test.RepipeStdOutAndErrForTest(func() { output.PrintComparison(c) })
	require.Contains(t, stdout, " → Removed Files")
	require.Contains(t, stdout, "[S] z.go [10.0%]")
	require.NotContains(t, stdout, "No change")
	require.NotContains(t, stdout, "Added")
}

func TestPrintComparison_AddedAndRemovedPrintOrder(t *testing.T) {
	entry := entryWith([]compute.ByFile{{File: "old.go", By: cov(1, 1, 1)}},
		[]compute.ByPackage{{Package: "oldpkg", By: cov(1, 1, 1)}})
	results := compute.Results{
		ByFile:    []compute.ByFile{{File: "new.go", By: cov(2, 2, 2)}},
		ByPackage: []compute.ByPackage{{Package: "newpkg", By: cov(2, 2, 2)}},
	}
	c := buildComparison(entry, results)
	stdout, _ := test.RepipeStdOutAndErrForTest(func() { output.PrintComparison(c) })

	order := []string{" → Added Files", " → Added Packages", " → Removed Files", " → Removed Packages"}
	last := -1
	for _, h := range order {
		i := strings.Index(stdout, h)
		require.Greater(t, i, last, h)
		last = i
	}
}

func TestPrintComparison_HistoryWithoutLinesRemoved(t *testing.T) {
	by := compute.By{StatementPercentage: 10, BlockPercentage: 20}
	entry := entryWith([]compute.ByFile{{File: "gone.go", By: by}}, []compute.ByPackage{{Package: "gone", By: by}})
	c := buildComparison(entry, compute.Results{})

	require.Equal(t, compute.Coverage{Statements: 10, Blocks: 20}, c.Removed.Files[0].Coverage)
	require.Nil(t, c.Removed.Files[0].Lines)
	require.Nil(t, c.Removed.Packages[0].Lines)

	stdout, _ := test.RepipeStdOutAndErrForTest(func() { output.PrintComparison(c) })
	require.Contains(t, stdout, "[B] gone.go [20.0%]")
	require.NotContains(t, stdout, "[L]")
}

func TestFormatAndReportWithComparison_EmptyAddedRemovedAreArrays(t *testing.T) {
	entry := testEntry()
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.NoColor = true
	cfg.Format = config.FormatJSON
	c := buildComparison(entry, entry.Results)
	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		output.FormatAndReportWithComparison(entry.Results, c, cfg, false)
	})
	require.Contains(t, stdout, `"added": {`)
	require.Contains(t, stdout, `"removed": {`)
	require.NotContains(t, stdout, "null")

	var doc struct {
		Comparison struct {
			Added   map[string][]any `json:"added"`
			Removed map[string][]any `json:"removed"`
		} `json:"comparison"`
	}
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	require.Equal(t, map[string][]any{"files": {}, "packages": {}}, doc.Comparison.Added)
	require.Equal(t, map[string][]any{"files": {}, "packages": {}}, doc.Comparison.Removed)
}
