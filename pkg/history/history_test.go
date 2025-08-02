package history //nolint:testpackage

import (
	"os"
	"testing"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/object"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.TestCoverageHistory)
	h, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, h)
	require.Len(t, h.Entries, 1)
}

func TestLoad_InvalidFile(t *testing.T) {
	path := test.CreateTempHistoryFile(t, test.InvalidTestCoverageHistory)
	_, err := Load(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot unmarshal string into Go struct field History.entries")
}

func TestLoad_NonExistentFile(t *testing.T) {
	path := t.TempDir() + "/nonexistent_history.json"
	_, err := Load(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such file or directory")
}

func TestNew(t *testing.T) {
	h := New("")
	require.NotNil(t, h)
	require.Empty(t, h.path)
}

func TestHistory_AddResults(t *testing.T) {
	type addCall struct {
		results compute.Results
		label   string
	}
	type testCase struct {
		name          string
		addCalls      []addCall
		expectEntries []Entry
	}

	r1 := compute.Results{
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "1/1"},
		},
	}
	r2 := compute.Results{
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "5/5"},
		},
	}

	tests := []testCase{
		{
			name: "AddResultsToEmptyHistory",
			addCalls: []addCall{
				{results: r1, label: "label1"},
			},
			expectEntries: []Entry{
				{Commit: "unknown", Branch: "unknown", Label: "label1", Results: r1},
			},
		},
		{
			name: "ChangesLabelOnly",
			addCalls: []addCall{
				{results: r1, label: "label1"},
				{results: r1, label: "label2"},
			},
			expectEntries: []Entry{
				{Commit: "unknown", Branch: "unknown", Label: "label2", Results: r1},
			},
		},
		{
			name: "ChangesLabelAndResults",
			addCalls: []addCall{
				{results: r1, label: "label1"},
				{results: r2, label: "label2"},
			},
			expectEntries: []Entry{
				{Commit: "unknown", Branch: "unknown", Label: "label2", Results: r2},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := New("")
			require.NotNil(t, h)
			for _, call := range tc.addCalls {
				h.AddResults(call.results, call.label)
			}
			require.Len(t, tc.expectEntries, len(h.Entries))
			for i, want := range tc.expectEntries {
				got := h.Entries[i]
				require.Equal(t, want.Commit, got.Commit)
				require.Equal(t, want.Branch, got.Branch)
				require.Equal(t, want.Label, got.Label)
				require.Equal(t, want.Results, got.Results)
			}
		})
	}
}

func TestHistory_AddResults_AreSorted(t *testing.T) {
	repoDir := t.TempDir()
	defaultRepoPath = repoDir
	defer func() {
		defaultRepoPath = "."
	}()

	h := New("")
	require.NotNil(t, h)

	repo, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)

	w, err := repo.Worktree()
	require.NoError(t, err)

	// First commit
	filePath := w.Filesystem.Join("file1.txt")
	f, err := w.Filesystem.Create(filePath)
	require.NoError(t, err)
	_, err = f.Write([]byte("content1"))
	require.NoError(t, err)
	require.NoError(t, f.Close())
	_, err = w.Add("file1.txt")
	require.NoError(t, err)
	commitHash1, err := w.Commit("first commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// add history entry for first commit
	h.AddResults(compute.Results{
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "1/1"},
		},
	}, "first")

	time.Sleep(1 * time.Second) // Ensure timestamp difference

	// Second commit (new file)
	filePath2 := w.Filesystem.Join("file2.txt")
	f2, err := w.Filesystem.Create(filePath2)
	require.NoError(t, err)
	_, err = f2.Write([]byte("content2"))
	require.NoError(t, err)
	require.NoError(t, f2.Close())
	_, err = w.Add("file2.txt")
	require.NoError(t, err)
	commitHash2, err := w.Commit("second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now().Add(1 * time.Second),
		},
	})
	require.NoError(t, err)

	// Add results for second commit
	h.AddResults(compute.Results{
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "2/2"},
		},
	}, "second")

	require.Len(t, h.Entries, 2)
	// Entries should be sorted by timestamp, newest first (second commit first)
	require.Equal(t, commitHash2.String(), h.Entries[0].Commit)
	require.Equal(t, "second", h.Entries[0].Label)
	require.Equal(t, commitHash1.String(), h.Entries[1].Commit)
	require.Equal(t, "first", h.Entries[1].Label)
}

func TestFindsEntryByCommit(t *testing.T) {
	h := New("")
	require.NotNil(t, h)

	h.AddResults(compute.Results{}, "label1")
	h.Entries[0].Commit = "commit123"

	entry := h.FindByRef("commit123")
	require.NotNil(t, entry)
	require.Equal(t, "commit123", entry.Commit)
}

func TestFindsEntryByBranch(t *testing.T) {
	h := New("")
	require.NotNil(t, h)

	h.AddResults(compute.Results{}, "label1")
	h.Entries[0].Branch = "main"

	entry := h.FindByRef("main")
	require.NotNil(t, entry)
	require.Equal(t, "main", entry.Branch)
}

func TestFindsEntryByLabel(t *testing.T) {
	h := New("")
	require.NotNil(t, h)

	h.AddResults(compute.Results{}, "label1")

	entry := h.FindByRef("label1")
	require.NotNil(t, entry)
	require.Equal(t, "label1", entry.Label)
}

func TestFindsEntryByTag(t *testing.T) {
	h := New("")
	require.NotNil(t, h)

	h.AddResults(compute.Results{}, "label1")
	h.Entries[0].Tags = []string{"v1.0.0"}

	entry := h.FindByRef("v1.0.0")
	require.NotNil(t, entry)
	require.Contains(t, entry.Tags, "v1.0.0")
}

func TestReturnsNilForNonexistentRef(t *testing.T) {
	h := New("")
	require.NotNil(t, h)

	h.AddResults(compute.Results{}, "label1")

	entry := h.FindByRef("nonexistent")
	require.Nil(t, entry)
}

func TestHistory_Save(t *testing.T) {
	path := t.TempDir() + "/history"
	defer t.Cleanup(func() {
		_ = os.RemoveAll(t.TempDir())
	})
	h := New(path)
	require.NotNil(t, h)
	err := h.Save(0)
	require.NoError(t, err)
}

func TestHistory_Save_WriteFileError(t *testing.T) {
	h := New(t.TempDir()) // path is a directory, not a file
	h.Entries = append(h.Entries, Entry{Commit: "test"})
	err := h.Save(0)
	require.Error(t, err)
}

func TestLoadEntryWithInvalidRepoPath(t *testing.T) {
	repoPath := "/invalid/path"
	_, err := git.PlainOpen(repoPath)
	require.Error(t, err)
}

func TestCreateEntryWithDetachedHead(t *testing.T) {
	repoDir := t.TempDir()
	repo, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)

	w, err := repo.Worktree()
	require.NoError(t, err)

	// Create a file to ensure there is something to commit
	filePath := w.Filesystem.Join("file.txt")
	f, err := w.Filesystem.Create(filePath)
	require.NoError(t, err)
	_, err = f.Write([]byte("content"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = w.Add("file.txt")
	require.NoError(t, err)

	commitHash, err := w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Detach HEAD by checking out the commit by hash
	err = w.Checkout(&git.CheckoutOptions{
		Hash: commitHash,
	})
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)
	require.False(t, head.Name().IsBranch())

	entry := startEntry("detached-head-test", repoDir)
	require.Equal(t, "detached", entry.Branch)
	require.Equal(t, head.Hash().String(), entry.Commit)
}

func TestCreateEntryWithTags(t *testing.T) {
	repoDir := t.TempDir()
	repo, err := git.PlainInit(repoDir, false)
	require.NoError(t, err)

	w, err := repo.Worktree()
	require.NoError(t, err)

	// Create a file to commit (go-git does not allow empty commits)
	filePath := w.Filesystem.Join("file.txt")
	f, err := w.Filesystem.Create(filePath)
	require.NoError(t, err)
	_, err = f.Write([]byte("content"))
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = w.Add("file.txt")
	require.NoError(t, err)

	_, err = w.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	_, err = repo.CreateTag("v1.0.0", head.Hash(), nil)
	require.NoError(t, err)

	entry := startEntry("tagged-commit-test", repoDir)
	require.Contains(t, entry.Tags, "v1.0.0")
	require.Equal(t, head.Hash().String(), entry.Commit)
}

func TestHistory_DeleteByRef(t *testing.T) {
	h := New("")
	require.NotNil(t, h)

	// Add a single test entry
	h.AddResults(compute.Results{
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "1/1"},
		},
	}, "label1")

	h.Entries[0].Commit = "commit123"
	h.Entries[0].Branch = "main"
	h.Entries[0].Tags = []string{"v1.0.0"}

	require.Len(t, h.Entries, 1)

	// Test delete by commit
	deleted := h.DeleteByRef("commit123")
	require.True(t, deleted)
	require.Empty(t, h.Entries)

	// Add entry back and test delete by branch
	h.AddResults(compute.Results{
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "2/2"},
		},
	}, "label2")
	h.Entries[0].Branch = "feature"

	deleted = h.DeleteByRef("feature")
	require.True(t, deleted)
	require.Empty(t, h.Entries)

	// Add entry back and test delete by tag
	h.AddResults(compute.Results{
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "3/3"},
		},
	}, "label3")
	h.Entries[0].Tags = []string{"v2.0.0"}

	deleted = h.DeleteByRef("v2.0.0")
	require.True(t, deleted)
	require.Empty(t, h.Entries)

	// Add entry back and test delete by label
	h.AddResults(compute.Results{
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "4/4"},
		},
	}, "label4")

	deleted = h.DeleteByRef("label4")
	require.True(t, deleted)
	require.Empty(t, h.Entries)
}

func TestHistory_DeleteByRef_ShortCommit(t *testing.T) {
	h := New("")
	require.NotNil(t, h)

	h.AddResults(compute.Results{}, "label1")
	h.Entries[0].Commit = "commit123456789"

	// Test delete by short commit (7 chars)
	deleted := h.DeleteByRef("commit1")
	require.True(t, deleted)
	require.Empty(t, h.Entries)
}

func TestHistory_DeleteByRef_NonExistent(t *testing.T) {
	h := New("")
	require.NotNil(t, h)

	h.AddResults(compute.Results{}, "label1")

	// Test delete by non-existent ref
	deleted := h.DeleteByRef("nonexistent")
	require.False(t, deleted)
	require.Len(t, h.Entries, 1)
}
