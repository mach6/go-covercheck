// Package filters provides filtering functionality for coverage profiles.
package filters

import (
	"regexp"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/gitdiff"
	"github.com/mach6/go-covercheck/pkg/output"
	"golang.org/x/tools/cover"
)

var (
	// defaultRepoPath is the default path to the git repository.
	// It is used when no specific path is provided.
	// Typically, it is the current working directory.
	//
	// Defined as a package variable to allow testing with different paths.
	defaultRepoPath = "."
)

// FilterProfiles applies all filtering logic to the given profiles.
func FilterProfiles(profiles []*cover.Profile, cfg *config.Config) []*cover.Profile {
	filtered := filterBySkipped(profiles, cfg.Skip)

	if cfg.DiffFrom != "" {
		return filterByGitDiff(filtered, cfg)
	}

	return filtered
}

// filterBySkipped filters profiles based on skip patterns.
func filterBySkipped(profiles []*cover.Profile, skip []string) []*cover.Profile {
	filtered := make([]*cover.Profile, 0)
	for _, p := range profiles {
		if shouldSkip(p.FileName, skip) {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

// filterByGitDiff filters profiles to only include files that have changed in git diff.
func filterByGitDiff(profiles []*cover.Profile, cfg *config.Config) []*cover.Profile {
	changedFiles, err := gitdiff.GetChangedFiles(defaultRepoPath, cfg.DiffFrom)
	if err != nil {
		// Log error but don't fail - fall back to normal behavior
		output.PrintDiffWarning(err, cfg)
		return profiles
	}

	// If no files changed, return empty result
	if len(changedFiles) == 0 {
		output.PrintNoDiffChanges(cfg)
		return []*cover.Profile{}
	}

	diffFiltered := gitdiff.FilterProfilesByChangedFiles(profiles, changedFiles, cfg.ModuleName)
	output.PrintDiffModeInfo(len(diffFiltered), len(profiles), cfg)
	return diffFiltered
}

// shouldSkip checks if a filename should be skipped based on regex patterns.
func shouldSkip(filename string, skip []string) bool {
	for _, s := range skip {
		regex := regexp.MustCompile(s)
		if regex.MatchString(filename) {
			return true
		}
	}
	return false
}
