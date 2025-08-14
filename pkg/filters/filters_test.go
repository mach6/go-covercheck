package filters //nolint:testpackage

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func Test_filterBySkipped(t *testing.T) {
	profiles := []*cover.Profile{
		{FileName: "foo.go"},
		{FileName: "bar.go"},
		{FileName: "baz.go"},
	}
	skip := []string{"bar.go", "ba.*"}
	result := filterBySkipped(profiles, skip)
	if len(result) != 1 || result[0].FileName != "foo.go" {
		t.Errorf("expected only foo.go, got %v", result)
	}
}

func Test_shouldSkip(t *testing.T) {
	tests := []struct {
		filename string
		skip     []string
		want     bool
	}{
		{"foo.go", []string{"bar.go"}, false},
		{"bar.go", []string{"bar.go"}, true},
		{"baz.go", []string{"ba.*"}, true},
	}
	for _, tt := range tests {
		got := shouldSkip(tt.filename, tt.skip)
		if got != tt.want {
			t.Errorf("shouldSkip(%q, %v) = %v, want %v", tt.filename, tt.skip, got, tt.want)
		}
	}
}

func Test_filterByGitDiff(t *testing.T) {
	// Create a temporary directory for the test repository
	repoDir := t.TempDir()
	defaultRepoPath = repoDir

	// Initialize a git repository
	repo, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)

	// Get the worktree
	w, err := repo.Worktree()
	require.NoError(t, err)

	// Create and commit the first file
	test.CreateFile(t, filepath.Join(repoDir, "foo.go"), "package main\n\nfunc main() {}\n")
	_, err = w.Add("foo.go")
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
	test.CreateFile(t, filepath.Join(repoDir, "bar.go"), "package main\n\nfunc hello() {}\n")
	_, err = w.Add("bar.go")
	require.NoError(t, err)

	_, err = w.Commit("add bar.go", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	t.Run("with changed files", func(t *testing.T) {
		profiles := []*cover.Profile{{FileName: "foo.go"}, {FileName: "bar.go"}}
		cfg := &config.Config{DiffFrom: commit1.String()} // Use the first commit as DiffFrom
		result := filterByGitDiff(profiles, cfg)

		require.Len(t, result, 1)
		require.Equal(t, "bar.go", result[0].FileName)
	})

	t.Run("no changed files", func(t *testing.T) {
		profiles := []*cover.Profile{{FileName: "foo.go"}}
		cfg := &config.Config{DiffFrom: "HEAD~1"} // No changes relative to HEAD~1
		result := filterByGitDiff(profiles, cfg)

		require.Empty(t, result)
	})

	t.Run("GetChangedFiles returns error", func(t *testing.T) {
		// Create a config that will cause an error in GetChangedFiles (empty repoPath)
		defaultRepoPath = "" // Reset to empty to simulate error

		profiles := []*cover.Profile{{FileName: "foo.go"}}
		cfg := &config.Config{DiffFrom: "HEAD~1"}
		result := filterByGitDiff(profiles, cfg)

		require.Len(t, result, 1)
		require.Equal(t, "foo.go", result[0].FileName)
	})
}
