package output_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/output"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func heatByFile(name string, pct, goal float64, failed bool) compute.ByFile {
	return compute.ByFile{
		File: name,
		By:   compute.By{StatementPercentage: pct, StatementThreshold: goal, Failed: failed},
	}
}

func renderPlain(t *testing.T, results compute.Results, width int) string {
	t.Helper()
	cfg := &config.Config{NoColor: true, TerminalWidth: width}
	var buf bytes.Buffer
	output.RenderHeatmap(&buf, results, cfg)
	return buf.String()
}

func TestHeatFor(t *testing.T) {
	tests := []struct {
		name   string
		actual float64
		goal   float64
		failed bool
		want   string
	}{
		{"no goal", 90, 0, false, "none"},
		{"no goal and no coverage", 0, 0, false, "none"},
		{"exactly half of goal", 35, 70, false, "low"},
		{"just over half of goal", 36, 70, false, "mid"},
		{"99 percent of goal", 69.3, 70, false, "mid"},
		{"goal met", 70, 70, false, "met"},
		{"above goal", 100, 70, false, "met"},
		{"met on statements but failed elsewhere", 80, 70, true, "mid"},
		{"low stays low when failed", 10, 70, true, "low"},
	}
	glyphs := map[string]string{"none": "·", "low": "░", "mid": "▒", "met": "█"}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			results := compute.Results{ByFile: []compute.ByFile{heatByFile("f.go", tc.actual, tc.goal, tc.failed)}}
			output.RenderHeatmap(&buf, results, &config.Config{NoColor: true})
			// the cell glyph precedes the percentage in the second line of the cell
			require.Contains(t, buf.String(), fmt.Sprintf(" %s %.1f%%", glyphs[tc.want], tc.actual))
		})
	}
}

func TestRenderHeatmap_PlainGolden(t *testing.T) {
	results := compute.Results{
		ByFile: []compute.ByFile{
			heatByFile("cmd/main.go", 100, 70, false),
			heatByFile("pkg/very/long/dir/name/handler.go", 50, 70, true),
			heatByFile("pkg/x.go", 10, 70, true),
		},
		ByPackage: []compute.ByPackage{
			{Package: "pkg", By: compute.By{StatementPercentage: 30}},
		},
	}
	results.ByTotal.Statements = compute.TotalStatements{Coverage: "5/10", Percentage: 50, Threshold: 70, Failed: true}
	results.ByTotal.Blocks = compute.TotalBlocks{Coverage: "9/10", Percentage: 90, Threshold: 70}

	want := strings.Join([]string{
		" ░  <= 50% of threshold   ▒  below threshold   █  threshold met",
		" ·  no threshold",
		"",
		"By File",
		" cmd/main.go            …dir/name/handler.go   pkg/x.go",
		" █ 100.0%               ▒ 50.0%                ░ 10.0%",
		"",
		"By Package",
		" pkg",
		" · 30.0%",
		"",
		"By Total",
		" ▒  Statements   50.0%  5/10",
		" █  Blocks       90.0%  9/10",
		" ·  Lines         0.0%  ",
		"",
		"",
	}, "\n")
	require.Equal(t, want, renderPlain(t, results, 80))
}

func TestRenderHeatmap_NarrowTerminalNeverOverflows(t *testing.T) {
	files := make([]compute.ByFile, 0, 25)
	for i := range 25 {
		files = append(files, heatByFile(fmt.Sprintf("pkg/dir%02d/file%02d.go", i, i), float64(i*4), 70, false))
	}
	for _, width := range []int{1, 8, 20, 33, 80, 200} {
		out := renderPlain(t, compute.Results{ByFile: files}, width)
		_, grid, _ := strings.Cut(out, "By File") // the legend and totals lines cannot shrink
		grid, _, _ = strings.Cut(grid, "By Total")
		for _, line := range strings.Split(grid, "\n") {
			// a cell cannot shrink below its minimum, so allow for it on tiny terminals
			limit := max(width, 10)
			require.LessOrEqual(t, text.StringWidthWithoutEscSequences(line), limit, "width %d: %q", width, line)
		}
		require.Contains(t, out, "24.go")
		require.Contains(t, out, " 96.0%")
	}
}

func TestRenderHeatmap_ZeroWidthFallsBackTo80(t *testing.T) {
	files := make([]compute.ByFile, 0, 10)
	for i := range 10 {
		files = append(files, heatByFile(fmt.Sprintf("f%d.go", i), 50, 70, false))
	}
	out := renderPlain(t, compute.Results{ByFile: files}, 0)
	// 80 columns fit three 23 wide cells per row; ten files make four rows.
	require.Equal(t, 10, strings.Count(out, " ▒ 50.0%"))
	require.Contains(t, out, " f0.go                  f1.go                  f2.go\n")
	require.Contains(t, out, " f9.go\n")
	for _, line := range strings.Split(out, "\n") {
		require.LessOrEqual(t, text.StringWidthWithoutEscSequences(line), 80)
	}
}

func TestRenderHeatmap_ColumnCount(t *testing.T) {
	files := make([]compute.ByFile, 0, 12)
	for i := range 12 {
		files = append(files, heatByFile(fmt.Sprintf("f%02d.go", i), 50, 70, false))
	}
	tests := []struct{ width, cols int }{
		{22, 1}, {44, 1}, {45, 2}, {100, 4}, {200, 8},
	}
	for _, tc := range tests {
		out := renderPlain(t, compute.Results{ByFile: files}, tc.width)
		_, grid, _ := strings.Cut(out, "By File\n")
		firstRow, _, _ := strings.Cut(grid, "\n")
		require.Equal(t, tc.cols, strings.Count(firstRow, ".go"), "width %d: %q", tc.width, firstRow)
	}
}

func TestRenderHeatmap_SimilarPathsStayDistinct(t *testing.T) {
	results := compute.Results{ByFile: []compute.ByFile{
		heatByFile("cmd/go-covercheck/history.go", 50, 70, false),
		heatByFile("pkg/history/history.go", 50, 70, false),
		heatByFile("pkg/output/history.go", 50, 70, false),
	}}
	out := renderPlain(t, results, 100)
	row, _, _ := strings.Cut(strings.SplitN(out, "By File\n", 2)[1], "\n")
	require.Equal(t, 3, strings.Count(row, "history.go"))
	require.Contains(t, row, " …vercheck/history.go ")
	require.Contains(t, row, " …/history/history.go ")
	require.Contains(t, row, " …g/output/history.go")
}

func TestRenderHeatmap_PreservesResultOrder(t *testing.T) {
	results := compute.Results{ByFile: []compute.ByFile{
		heatByFile("z.go", 90, 70, false),
		heatByFile("a.go", 10, 70, false),
	}}
	out := renderPlain(t, results, 80)
	require.Less(t, strings.Index(out, "z.go"), strings.Index(out, "a.go"))
}

func TestRenderHeatmap_Single(t *testing.T) {
	out := renderPlain(t, compute.Results{ByFile: []compute.ByFile{heatByFile("only.go", 100, 70, false)}}, 80)
	require.Contains(t, out, "only.go")
	require.NotContains(t, out, "By Package")
}

func TestRenderHeatmap_Unicode(t *testing.T) {
	out := renderPlain(t, compute.Results{ByFile: []compute.ByFile{
		heatByFile("pkg/\u65e5\u672c\u8a9e\u306e\u30d5\u30a1\u30a4\u30eb\u540d\u524d.go", 100, 70, false),
	}}, 80)
	for _, line := range strings.Split(out, "\n") {
		require.LessOrEqual(t, text.StringWidthWithoutEscSequences(line), 80)
	}
	require.Contains(t, out, "\u540d\u524d.go")
	require.NotContains(t, out, "\ufffd")
}

func TestFitHeatmapText(t *testing.T) {
	require.Equal(t, "ab    ", output.FitHeatmapText("ab", 6))
	require.Equal(t, "…cdef", output.FitHeatmapText("abcdef", 5))
	require.Equal(t, "…\u8a9e\u540d", output.FitHeatmapText("\u65e5\u672c\u8a9e\u540d", 5))
	require.Equal(t, "…", output.FitHeatmapText("abc", 1))
}

func TestRenderHeatmap_Color(t *testing.T) {
	prev := color.NoColor
	defer func() { color.NoColor = prev }()
	color.NoColor = false
	results := compute.Results{ByFile: []compute.ByFile{
		heatByFile("hi.go", 100, 70, false),
		heatByFile("lo.go", 5, 70, false),
	}}

	var buf bytes.Buffer
	output.RenderHeatmap(&buf, results, &config.Config{TerminalWidth: 80})
	out := buf.String()
	require.Contains(t, out, "\x1b[42;30m") // green background: threshold met
	require.Contains(t, out, "\x1b[41;37m") // red background: far below threshold
	require.NotContains(t, out, "█ 100.0%") // color replaces the in-cell glyph

	buf.Reset()
	output.RenderHeatmap(&buf, results, &config.Config{TerminalWidth: 80, NoColor: true})
	require.NotContains(t, buf.String(), "\x1b[")
}

func TestRenderHeatmap_NoColorOnNonTTY(t *testing.T) {
	prev := color.NoColor
	defer func() { color.NoColor = prev }()
	color.NoColor = true // what fatih/color sets when stdout is not a terminal or NO_COLOR is set
	var buf bytes.Buffer
	output.RenderHeatmap(&buf, compute.Results{ByFile: []compute.ByFile{heatByFile("a.go", 80, 70, false)}},
		&config.Config{TerminalWidth: 80})
	require.NotContains(t, buf.String(), "\x1b[")
	require.Contains(t, buf.String(), "█ 80.0%")
}

func TestFormatAndReport_Heatmap(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatHeatmap
	cfg.StatementThreshold = 80
	cfg.NoColor = true
	cfg.TerminalWidth = 80
	prev := color.NoColor
	defer func() { color.NoColor = prev }()
	color.NoColor = true

	profiles := []*cover.Profile{{
		FileName: "example/foo.go",
		Blocks:   []cover.ProfileBlock{{NumStmt: 10, Count: 1}, {NumStmt: 10, Count: 0}},
	}}

	stdout, stderr := test.RepipeStdOutAndErrForTest(func() {
		results, failed := compute.CollectResults(profiles, cfg)
		require.True(t, failed)
		output.FormatAndReport(results, cfg, failed)
	})

	require.Empty(t, stderr)
	require.Contains(t, stdout, "By File")
	require.Contains(t, stdout, "example/foo.go")
	require.Contains(t, stdout, "▒ 50.0%")
	require.Contains(t, stdout, "Coverage check failed")
}

func TestFormatAndReport_HeatmapEmpty(t *testing.T) {
	cfg := new(config.Config)
	cfg.ApplyDefaults()
	cfg.Format = config.FormatHeatmap

	stdout, _ := test.RepipeStdOutAndErrForTest(func() {
		empty := compute.Results{}
		empty.ByTotal.Statements.Coverage = "0/0"
		empty.ByTotal.Blocks.Coverage = "0/0"
		output.FormatAndReport(empty, cfg, false)
	})
	require.Contains(t, stdout, "No coverage results to display")
}
