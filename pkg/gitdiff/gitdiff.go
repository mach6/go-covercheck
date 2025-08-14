// Package gitdiff provides git diff functionality for coverage filtering.
package gitdiff

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/plumbing/format/diff"
	"golang.org/x/tools/cover"
)

// GetChangedFiles returns a list of files that have changed between the target reference and HEAD.
// Returns a map where keys are file paths and values are true.
func GetChangedFiles(repoPath, targetRef string) (map[string]bool, error) {
	if repoPath == "" {
		repoPath = "."
	}
	if targetRef == "" {
		return nil, errors.New("target reference cannot be empty")
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository at %s: %w", repoPath, err)
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	headCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD commit: %w", err)
	}

	targetHash, err := resolveReference(repo, targetRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target reference %q: %w", targetRef, err)
	}

	targetCommit, err := repo.CommitObject(targetHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get target commit: %w", err)
	}

	patch, err := targetCommit.Patch(headCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to get patch: %w", err)
	}

	changedFiles := make(map[string]bool)
	for _, filePatch := range patch.FilePatches() {
		addChangedFilesFromPatch(filePatch, changedFiles)
	}

	return changedFiles, nil
}

// addChangedFilesFromPatch handles the logic for extracting changed file paths from a diff.FilePatch.
func addChangedFilesFromPatch(filePatch diff.FilePatch, changedFiles map[string]bool) {
	from, to := filePatch.Files()
	switch {
	case from == nil && to != nil:
		changedFiles[to.Path()] = true
		return
	case from != nil && to != nil:
		changedFiles[to.Path()] = true
		if from.Path() != to.Path() {
			changedFiles[from.Path()] = true
		}
		return
	default:
		changedFiles[from.Path()] = true
		return
	}
}

// resolveReference resolves a git reference (branch, tag, commit) to a hash.
func resolveReference(repo *git.Repository, ref string) (plumbing.Hash, error) {
	// Try to resolve as a hash first
	if len(ref) >= 7 && len(ref) <= 40 {
		if hash, _ := plumbing.FromHex(ref); !hash.IsZero() {
			// Verify the hash exists
			if _, err := repo.CommitObject(hash); err == nil {
				return hash, nil
			}
		}
	}

	// Try as a branch reference
	if branchRef, err := repo.Reference(plumbing.NewBranchReferenceName(ref), true); err == nil {
		return branchRef.Hash(), nil
	}

	// Try as a tag reference
	if tagRef, err := repo.Reference(plumbing.NewTagReferenceName(ref), true); err == nil {
		return tagRef.Hash(), nil
	}

	// Try as a remote branch reference
	if remoteRef, err := repo.Reference(plumbing.NewRemoteReferenceName("origin", ref), true); err == nil {
		return remoteRef.Hash(), nil
	}

	// Try to resolve as a revision (like HEAD~1, HEAD^, etc.)
	if hash, err := repo.ResolveRevision(plumbing.Revision(ref)); err == nil {
		return *hash, nil
	}

	return plumbing.ZeroHash, fmt.Errorf("could not resolve reference %q", ref)
}

// FilterProfilesByChangedFiles filters coverage profiles to only include files that have changed.
func FilterProfilesByChangedFiles(profiles []*cover.Profile, changedFiles map[string]bool,
	moduleName string) []*cover.Profile {
	if len(changedFiles) == 0 {
		return []*cover.Profile{}
	}

	filtered := make([]*cover.Profile, 0)
	for _, profile := range profiles {
		fileName := profile.FileName

		// Try to match the file name directly
		if changedFiles[fileName] {
			filtered = append(filtered, profile)
			continue
		}

		// Try to match by removing the module prefix
		if moduleName != "" && strings.HasPrefix(fileName, moduleName) {
			relativePath := strings.TrimPrefix(fileName, moduleName)
			relativePath = strings.TrimPrefix(relativePath, "/")
			if changedFiles[relativePath] {
				filtered = append(filtered, profile)
				continue
			}
		}

		// Try to match using filepath operations for better path handling
		for changedFile := range changedFiles {
			if matchFilePaths(fileName, changedFile) {
				filtered = append(filtered, profile)
				break
			}
		}
	}

	return filtered
}

// matchFilePaths attempts to match file paths using different strategies.
func matchFilePaths(profilePath, changedPath string) bool {
	// Direct match
	if profilePath == changedPath {
		return true
	}

	// Try matching the base names
	if filepath.Base(profilePath) == filepath.Base(changedPath) {
		// Additional check: ensure the directory structure is similar
		profileDir := filepath.Dir(profilePath)
		changedDir := filepath.Dir(changedPath)
		if strings.HasSuffix(profileDir, changedDir) || strings.HasSuffix(changedDir, profileDir) {
			return true
		}
	}

	// Try matching with cleaned paths
	cleanProfile := filepath.Clean(profilePath)
	cleanChanged := filepath.Clean(changedPath)
	if cleanProfile == cleanChanged {
		return true
	}

	// Check if one path ends with the other (handling relative vs absolute paths)
	if strings.HasSuffix(profilePath, changedPath) || strings.HasSuffix(changedPath, profilePath) {
		return true
	}

	return false
}
