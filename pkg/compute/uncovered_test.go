package compute

import (
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