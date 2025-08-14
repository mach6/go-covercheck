package gitdiff

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func TestGetChangedFiles_InvalidRepo(t *testing.T) {
	_, err := GetChangedFiles("/nonexistent/path", "HEAD~1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to open git repository")
}

func TestFilterProfilesByChangedFiles(t *testing.T) {
	// Create test profiles
	profiles := []*cover.Profile{
		{FileName: "github.com/test/pkg1/file1.go"},
		{FileName: "github.com/test/pkg2/file2.go"},
		{FileName: "github.com/test/pkg3/file3.go"},
	}

	// Test with empty changed files - should return empty slice
	filtered := FilterProfilesByChangedFiles(profiles, map[string]bool{}, "github.com/test")
	require.Len(t, filtered, 0)

	// Test with some changed files
	changedFiles := map[string]bool{
		"pkg1/file1.go": true,
		"pkg2/file2.go": true,
	}
	filtered = FilterProfilesByChangedFiles(profiles, changedFiles, "github.com/test")
	require.Len(t, filtered, 2)
	require.Equal(t, "github.com/test/pkg1/file1.go", filtered[0].FileName)
	require.Equal(t, "github.com/test/pkg2/file2.go", filtered[1].FileName)

	// Test with direct match
	changedFiles = map[string]bool{
		"github.com/test/pkg1/file1.go": true,
	}
	filtered = FilterProfilesByChangedFiles(profiles, changedFiles, "")
	require.Len(t, filtered, 1)
	require.Equal(t, "github.com/test/pkg1/file1.go", filtered[0].FileName)
}

func TestMatchFilePaths(t *testing.T) {
	tests := []struct {
		name        string
		profilePath string
		changedPath string
		expected    bool
	}{
		{
			name:        "direct match",
			profilePath: "pkg/file.go",
			changedPath: "pkg/file.go",
			expected:    true,
		},
		{
			name:        "no match",
			profilePath: "pkg/file1.go",
			changedPath: "pkg/file2.go",
			expected:    false,
		},
		{
			name:        "suffix match",
			profilePath: "github.com/test/pkg/file.go",
			changedPath: "pkg/file.go",
			expected:    true,
		},
		{
			name:        "base name match with similar directory",
			profilePath: "github.com/test/pkg/file.go",
			changedPath: "test/pkg/file.go",
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchFilePaths(tt.profilePath, tt.changedPath)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetChangedFiles_WithGitRepo(t *testing.T) {
	// Create a temporary directory for the test repository
	repoDir := t.TempDir()

	// Initialize a git repository
	repo, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)

	// Get the worktree
	w, err := repo.Worktree()
	require.NoError(t, err)

	// Create and commit the first file
	test.CreateFile(t, filepath.Join(repoDir, "file1.go"), "package main\n\nfunc main() {}\n")
	_, err = w.Add("file1.go")
	require.NoError(t, err)

	commit1, err := w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Create and commit a second file
	test.CreateFile(t, filepath.Join(repoDir, "file2.go"), "package main\n\nfunc hello() {}\n")
	_, err = w.Add("file2.go")
	require.NoError(t, err)

	_, err = w.Commit("add file2", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Test getting changed files between first commit and HEAD
	changedFiles, err := GetChangedFiles(repoDir, commit1.String())
	require.NoError(t, err)
	require.Len(t, changedFiles, 1)
	require.True(t, changedFiles["file2.go"])

	// Test with HEAD~1 (should be the same result)
	changedFiles, err = GetChangedFiles(repoDir, "HEAD~1")
	require.NoError(t, err)
	require.Len(t, changedFiles, 1)
	require.True(t, changedFiles["file2.go"])
}

func TestResolveReference_InvalidRef(t *testing.T) {
	// Create a temporary directory for the test repository
	repoDir := t.TempDir()

	// Initialize a git repository
	repo, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)

	// Try to resolve a non-existent reference
	_, err = resolveReference(repo, "nonexistent-branch")
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not resolve reference")
}
