package compute

import (
	"bufio"
	"os"
	"sort"
	"strings"

	"github.com/mach6/go-covercheck/pkg/config"
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

// readModuleNameFromGoMod reads the module name from go.mod in the current working directory.
// Returns empty string if the go.mod file doesn't exist, or module name cannot be extracted.
func readModuleNameFromGoMod() string {
	readFile, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}

	file := strings.NewReader(string(readFile))
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 { //nolint:mnd
				return parts[1]
			}
		}
	}
	return ""
}

// validateModuleNameMatchesFilePaths checks if the given module name is a prefix
// for all file paths in the profiles.
func validateModuleNameMatchesFilePaths(moduleName string, profiles []*cover.Profile) bool {
	if moduleName == "" || len(profiles) == 0 {
		return false
	}

	// Ensure module name has trailing slash for proper prefix matching
	if !strings.HasSuffix(moduleName, "/") {
		moduleName += "/"
	}

	for _, profile := range profiles {
		if !strings.HasPrefix(profile.FileName, moduleName) {
			return false
		}
	}
	return true
}

func findModuleName(profiles []*cover.Profile, cfg *config.Config) string {
	// Use configured module name if provided
	if cfg != nil && cfg.ModuleName != "" {
		moduleName := cfg.ModuleName
		// Ensure module name ends with "/" for proper prefix replacement
		if !strings.HasSuffix(moduleName, "/") {
			moduleName += "/"
		}
		return moduleName
	}

	// Try to read module name from go.mod if it exists and matches all file paths
	if goModModuleName := readModuleNameFromGoMod(); goModModuleName != "" {
		if validateModuleNameMatchesFilePaths(goModModuleName, profiles) {
			// Ensure module name ends with "/" for proper prefix replacement
			if !strings.HasSuffix(goModModuleName, "/") {
				goModModuleName += "/"
			}
			return goModModuleName
		}
	}

	// Fallback to the longest common prefix logic
	names := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		names = append(names, profile.FileName)
	}
	return longestCommonPrefix(names)
}

func normalizeNames(profiles []*cover.Profile, cfg *config.Config) {
	moduleName := findModuleName(profiles, cfg)
	for _, profile := range profiles {
		profile.FileName = strings.Replace(profile.FileName, moduleName, "", 1)
	}
}
