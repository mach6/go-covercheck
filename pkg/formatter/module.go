package formatter

import (
	"sort"
	"strings"

	"golang.org/x/tools/cover"
)

// longestCommonPrefix returns the longest common prefix shared by all strings in the input slice.
func longestCommonPrefix(strs []string) string {
	// Handle edge cases
	if len(strs) == 0 || len(strs) == 1 {
		return ""
	}

	// Sort the slice to easily find the common prefix
	// After sorting, we only need to compare first and last strings
	sort.Strings(strs)

	// Get first and last strings
	first := strs[0]
	last := strs[len(strs)-1]

	// Initialize variables for finding common prefix
	var commonPrefix string
	minLength := min(len(first), len(last))

	// Compare characters until mismatch or reaching the end of shorter string
	for i := range minLength {
		if first[i] != last[i] {
			break
		}
		commonPrefix += string(first[i])
	}

	return commonPrefix
}

func findModuleName(profiles []*cover.Profile) string {
	names := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		names = append(names, profile.FileName)
	}
	return longestCommonPrefix(names)
}

func normalizeNames(profiles []*cover.Profile) {
	moduleName := findModuleName(profiles)
	for _, profile := range profiles {
		profile.FileName = strings.Replace(profile.FileName, moduleName, "", 1)
	}
}
