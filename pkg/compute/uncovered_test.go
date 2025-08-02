package compute

import (
	"os"
	"path/filepath"
	"testing"

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
			result := collectUncoveredLines(tt.profile)
			if result != tt.expected {
				t.Errorf("collectUncoveredLines() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCollectUncoveredLinesWithFiltering(t *testing.T) {
	// Create a temporary test file with various line types
	testFile := filepath.Join(t.TempDir(), "test_filtering.go")
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

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

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
			// Should filter out:
			// Line 2: empty line
			// Line 4: empty line
			// Line 5: comment line "// This is a comment line"
			// Line 8: empty line
			// Line 11: closing brace "}"
			// Line 12: empty line
			// Line 13-16: block comment lines
			// Line 17: empty line
			// Line 20: closing brace "}"
			// Line 21: closing brace "}"
			// Should keep lines: 1,3,6,7,9,10,18,19
			expected: "1,3,6-7,9-10,18-19",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectUncoveredLines(tt.profile)
			if result != tt.expected {
				t.Errorf("collectUncoveredLines() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterExcludedLines(t *testing.T) {
	// Create a temporary test file
	testFile := filepath.Join(os.TempDir(), "test_filter.go")
	content := `package main

import "fmt"

// Comment line
func test() {
	fmt.Println("hello")
	
	if true {
		fmt.Println("world")
	}
	
	/*
	 * Block comment
	 */
	
	for i := 0; i < 5; i++ {
		// Another comment
		fmt.Println(i)
	}
}`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	// Create another test file for edge cases
	edgeTestFile := filepath.Join(os.TempDir(), "test_edge.go")
	edgeContent := `package main
/* single line comment */ var x = 1
fmt.Println("test") /* inline */ fmt.Println("more")
/* comment */
}`

	err = os.WriteFile(edgeTestFile, []byte(edgeContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create edge test file: %v", err)
	}
	defer os.Remove(edgeTestFile)

	tests := []struct {
		name     string
		lines    []int
		fileName string
		expected []int
	}{
		{
			name:     "filters out empty lines, comments, and closing braces",
			lines:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			fileName: testFile,
			// Should keep: 1(package), 3(import), 6(func), 7(fmt.Println), 9(if), 10(fmt.Println), 17(for),
			//	19(fmt.Println)
			// Should filter: 2(empty), 5(comment), 8(empty), 11(}), 12(empty), 13-15(block comment),
			//	16(empty), 18(comment), 20(})
			expected: []int{1, 3, 6, 7, 9, 10, 17, 19},
		},
		{
			name:     "handles single-line block comments and inline comments",
			lines:    []int{1, 2, 3, 4, 5},
			fileName: edgeTestFile,
			// Line 1: package main - keep
			// Line 2: /* single line comment */ var x = 1 - keep (has code after comment)
			// Line 3: fmt.Println("test") /* inline */ fmt.Println("more") - keep (has code)
			// Line 4: /* comment */ - filter (only comment)
			// Line 5: } - filter (closing brace)
			expected: []int{1, 2, 3},
		},
		{
			name:     "handles non-existent file gracefully",
			lines:    []int{1, 2, 3},
			fileName: "non-existent-file.go",
			expected: []int{1, 2, 3}, // Should return all lines if file can't be read
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterExcludedLines(tt.lines, tt.fileName)
			if len(result) != len(tt.expected) {
				t.Errorf("filterExcludedLines() length = %d, want %d", len(result), len(tt.expected))
				t.Errorf("filterExcludedLines() = %v, want %v", result, tt.expected)
				return
			}
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("filterExcludedLines()[%d] = %d, want %d", i, line, tt.expected[i])
				}
			}
		})
	}
}

func TestReadFileLines(t *testing.T) {
	// Create a temporary test file
	testFile := filepath.Join(os.TempDir(), "test_read.go")
	content := "line1\nline2\nline3"

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFile)

	lines, err := readFileLines(testFile)
	if err != nil {
		t.Errorf("readFileLines() error = %v", err)
		return
	}

	expected := []string{"line1", "line2", "line3"}
	if len(lines) != len(expected) {
		t.Errorf("readFileLines() length = %d, want %d", len(lines), len(expected))
		return
	}

	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("readFileLines()[%d] = %q, want %q", i, line, expected[i])
		}
	}

	// Test non-existent file
	_, err = readFileLines("non-existent-file.go")
	if err == nil {
		t.Error("readFileLines() should return error for non-existent file")
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
			if result != tt.expected {
				t.Errorf("formatLineRanges() = %q, want %q", result, tt.expected)
			}
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
			if result != tt.expected {
				t.Errorf("formatRange() = %q, want %q", result, tt.expected)
			}
		})
	}
}
