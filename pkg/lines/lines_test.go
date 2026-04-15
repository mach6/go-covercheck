package lines

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func TestCollectUncoveredLines(t *testing.T) {
	tests := []struct {
		name     string
		profile  *cover.Profile
		expected string
	}{
		{
			name: "no uncovered lines",
			profile: &cover.Profile{
				FileName: "test.go",
				Blocks: []cover.ProfileBlock{
					{StartLine: 1, EndLine: 3, Count: 1},
					{StartLine: 5, EndLine: 7, Count: 2},
				},
			},
			expected: "",
		},
		{
			name: "single line uncovered",
			profile: &cover.Profile{
				FileName: "test.go",
				Blocks: []cover.ProfileBlock{
					{StartLine: 1, EndLine: 1, Count: 0},
					{StartLine: 2, EndLine: 2, Count: 1},
				},
			},
			expected: "1",
		},
		{
			name: "consecutive lines uncovered",
			profile: &cover.Profile{
				FileName: "test.go",
				Blocks: []cover.ProfileBlock{
					{StartLine: 1, EndLine: 3, Count: 0},
					{StartLine: 4, EndLine: 4, Count: 1},
				},
			},
			expected: "1-3",
		},
		{
			name: "multiple separate uncovered lines",
			profile: &cover.Profile{
				FileName: "test.go",
				Blocks: []cover.ProfileBlock{
					{StartLine: 1, EndLine: 1, Count: 0},
					{StartLine: 2, EndLine: 2, Count: 1},
					{StartLine: 3, EndLine: 3, Count: 0},
					{StartLine: 5, EndLine: 7, Count: 0},
				},
			},
			expected: "1,3,5-7",
		},
		{
			name: "overlapping blocks handled correctly",
			profile: &cover.Profile{
				FileName: "test.go",
				Blocks: []cover.ProfileBlock{
					{StartLine: 1, EndLine: 3, Count: 0},
					{StartLine: 2, EndLine: 4, Count: 0},
					{StartLine: 6, EndLine: 6, Count: 0},
				},
			},
			expected: "1-4,6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUncoveredLines(tt.profile)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipLine(t *testing.T) {
	content := `package main

import "fmt"

// This is a comment line
func main() {
	fmt.Println("Hello") // inline comment

	if true {
		fmt.Println("world")
	}

	/*
	 * Block comment
	 * Another line
	 */

	for i := 0; i < 5; i++ {
		fmt.Println(i)
	}
}`
	testFile := test.CreateTempFile(t, "test_filtering.go", content)

	tests := []struct {
		name     string
		profile  *cover.Profile
		expected string
	}{
		{
			name: "filters out comments and empty lines",
			profile: &cover.Profile{
				FileName: testFile,
				Blocks: []cover.ProfileBlock{
					// Lines 1-21 are all "uncovered"
					{StartLine: 1, EndLine: 21, Count: 0},
				},
			},
			// Filters: blank lines (2, 4, 8, 12, 17), // comments (5),
			// bare closing braces (11, 20, 21). Block-comment interiors
			// (13-16) are NOT filtered — Go's coverage tool doesn't
			// normally span profile blocks over them, but this synthetic
			// profile covers lines 1-21 as a single block so they appear.
			expected: "1,3,6-7,9-10,13-16,18-19",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatUncoveredLines(tt.profile)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatLineRanges(t *testing.T) {
	tests := []struct {
		name     string
		lines    []int
		expected string
	}{
		{
			name:     "empty",
			lines:    []int{},
			expected: "",
		},
		{
			name:     "single line",
			lines:    []int{5},
			expected: "5",
		},
		{
			name:     "consecutive lines",
			lines:    []int{1, 2, 3, 4},
			expected: "1-4",
		},
		{
			name:     "separate lines",
			lines:    []int{1, 3, 5},
			expected: "1,3,5",
		},
		{
			name:     "mixed ranges and singles",
			lines:    []int{1, 2, 3, 5, 7, 8, 10},
			expected: "1-3,5,7-8,10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLineRanges(tt.lines)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatRange(t *testing.T) {
	tests := []struct {
		name     string
		start    int
		end      int
		expected string
	}{
		{
			name:     "single line",
			start:    5,
			end:      5,
			expected: "5",
		},
		{
			name:     "range",
			start:    1,
			end:      3,
			expected: "1-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatRange(tt.start, tt.end)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSkipLine_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{"   ", true},
		{"// comment", true},
		// Block-comment shapes aren't filtered here — Go's coverage tool
		// doesn't extend profile blocks over them, so they never reach
		// shouldSkipLine in practice.
		{"/* block comment start", false},
		{"* block comment content", false},
		{"*/", false},
		{"some code */", false},
		{"}", true},
		{"random text", false},
		{"package main", false},
		{"func main() {}", false},
		{"var x = 1", false},
		{"fmt.Println(\"hello\")", false},
		{"go func() {}", false},
		{"select {}", false},
		{"default:", false},
		{"range x", false},
		{"x && y", false},
		{"x++", false},
		{"x--", false},
		{"x := 1", false},
		{"x = 1", false},
		{"x <- ch", false},
		{"import \"fmt\"", false},
	}
	for _, tt := range tests {
		require.Equal(t, tt.expected, shouldSkipLine(tt.input), tt.input)
	}
}

func TestReadSourceFile_NotFound(t *testing.T) {
	_, err := ReadSourceFile("does-not-exist.go")
	require.Error(t, err)
}

func TestReadSourceFile_RelativePath(t *testing.T) {
	tmpFile := test.CreateTempFile(t, "testsource.go", "package main\nfunc main() {}\n")

	linesRead, err := ReadSourceFile(tmpFile)
	require.NoError(t, err)
	require.Len(t, linesRead, 2)
	require.Equal(t, "package main", linesRead[0])
	require.Equal(t, "func main() {}", linesRead[1])
}

func TestCollect(t *testing.T) {
	// Create a temporary source file
	content := `package main
func main() {
 println("hello")
}`
	tmpFile := test.CreateTempFile(t, "test_collect.go", content)

	// Create a coverage profile
	profile := &cover.Profile{
		FileName: tmpFile,
		Blocks: []cover.ProfileBlock{
			{StartLine: 1, EndLine: 1, Count: 1},
			{StartLine: 2, EndLine: 2, Count: 0},
			{StartLine: 3, EndLine: 3, Count: 1},
		},
	}

	result := CollectBlocks(profile)
	require.Len(t, result, 3)

	require.Equal(t, 1, result[0].Lines[0].LineNumber)
	require.Equal(t, "package main", result[0].Lines[0].Content)

	require.Equal(t, 2, result[1].Lines[0].LineNumber)
	require.Equal(t, "func main() {", result[1].Lines[0].Content)

	require.Equal(t, 3, result[2].Lines[0].LineNumber)
	require.Equal(t, ` println("hello")`, result[2].Lines[0].Content)
}

func TestSourceLinesInRange(t *testing.T) {
	content := `package main

// an uncovered comment
func a() {
	x := 1
	_ = x
}

func b() {
	y := 2
	_ = y
}
`
	file := test.CreateTempFile(t, "srclines.go", content)

	profile := &cover.Profile{
		FileName: file,
		Blocks: []cover.ProfileBlock{
			{StartLine: 4, EndLine: 6, NumStmt: 2, Count: 0},
			{StartLine: 9, EndLine: 11, NumStmt: 2, Count: 3},
		},
	}

	got := SourceLinesInRange(profile, 4, 11, 0)
	require.Len(t, got, 8, "should include every line in [4, 11]")
	require.Equal(t, 4, got[0].LineNumber)
	require.Equal(t, 11, got[7].LineNumber)

	// Hits overlay: lines 4-6 have Hits=0, 9-11 have Hits=3.
	require.Equal(t, 0, got[0].Hits)
	require.Equal(t, 3, got[5].Hits)

	// Line 7 is a bare closing brace — filtered but still present.
	require.Equal(t, 7, got[3].LineNumber)
	require.True(t, got[3].IsFiltered)

	// Line 8 is blank — filtered but still present, so merged hunks keep the gap visible.
	require.Equal(t, 8, got[4].LineNumber)
	require.True(t, got[4].IsFiltered)
}

func TestSourceLinesInRange_ContextClamped(t *testing.T) {
	content := "a\nb\nc\nd\ne\n"
	file := test.CreateTempFile(t, "clamp.go", content)
	profile := &cover.Profile{FileName: file, Blocks: []cover.ProfileBlock{
		{StartLine: 3, EndLine: 3, Count: 0},
	}}

	got := SourceLinesInRange(profile, 3, 3, 10)
	require.Len(t, got, 5, "context expansion should clamp to file extent")
	require.Equal(t, 1, got[0].LineNumber)
	require.Equal(t, 5, got[4].LineNumber)
}

func TestClampRange(t *testing.T) {
	rs, re := clampRange(10, 20, 5, 30)
	require.Equal(t, 5, rs)
	require.Equal(t, 25, re)

	rs, re = clampRange(1, 5, 10, 8)
	require.Equal(t, 1, rs)
	require.Equal(t, 8, re)

	rs, re = clampRange(1, 5, 10, 0)
	require.Equal(t, 1, rs)
	require.Equal(t, 15, re, "no source: end stays at requested value")
}

func TestHitsByLine_PrefersHighest(t *testing.T) {
	blocks := []cover.ProfileBlock{
		{StartLine: 1, EndLine: 3, Count: 0},
		{StartLine: 2, EndLine: 4, Count: 5},
		{StartLine: 10, EndLine: 10, Count: 1},
	}
	hits := hitsByLine(blocks, 1, 5)
	require.Equal(t, 0, hits[1])
	require.Equal(t, 5, hits[2], "overlapping covered block should win")
	require.Equal(t, 5, hits[3])
	require.Equal(t, 5, hits[4])
	_, ok := hits[10]
	require.False(t, ok, "out-of-range block must not appear in map")
}

func TestBlock_ContextLines(t *testing.T) {
	blocks := []Block{
		{
			ProfileBlock: cover.ProfileBlock{StartLine: 1, EndLine: 1, Count: 1},
			Lines: []Line{
				{LineNumber: 1, Content: "package main", Hits: 1},
			},
		},
		{
			ProfileBlock: cover.ProfileBlock{StartLine: 2, EndLine: 2, Count: 0},
			Lines: []Line{
				{LineNumber: 2, Content: "func main() {", Hits: 0},
			},
		},
		{
			ProfileBlock: cover.ProfileBlock{StartLine: 3, EndLine: 3, Count: 2},
			Lines: []Line{
				{LineNumber: 3, Content: " println(\"hello\")", Hits: 2},
			},
		},
	}

	// Context size 1 for the second block
	contextLines := blocks[1].ContextLines(blocks, 1)
	require.Len(t, contextLines, 3)
	require.Equal(t, 1, contextLines[0].LineNumber)
	require.Equal(t, 1, contextLines[0].Hits)
	require.Equal(t, 2, contextLines[1].LineNumber)
	require.Equal(t, 0, contextLines[1].Hits)
	require.Equal(t, 3, contextLines[2].LineNumber)
	require.Equal(t, 2, contextLines[2].Hits)
}
