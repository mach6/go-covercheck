// Package history implements go-covercheck history.
package history

import (
	"encoding/json"
	"os"
	"sort"
	"time"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/mach6/go-covercheck/pkg/compute"
)

var (
	// defaultRepoPath is the default path to the git repository.
	// It is used when no specific path is provided.
	// Typically, it is the current working directory.
	//
	// Defined as a package variable to allow testing with different paths.
	defaultRepoPath = "."
)

// Entry holds details for a single go-covercheck historical outcome.
type Entry struct {
	Commit    string          `json:"commit"`
	Branch    string          `json:"branch"`
	Tags      []string        `json:"tags,omitempty"`
	Label     string          `json:"label,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Results   compute.Results `json:"results"`
}

// History holds multiple Entry details for go-covercheck historical outcomes.
type History struct {
	Entries []Entry `json:"entries"`
	path    string
}

// New creates a History collection for the path specified.
func New(path string) *History {
	return &History{
		path: path,
	}
}

// Load History from a file path.
func Load(path string) (*History, error) {
	h := &History{
		path: path,
	}
	b, err := os.ReadFile(h.path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, h)
	if err != nil {
		return nil, err
	}

	return h, nil
}

// AddResults adds results to the History, optionally with a label.
func (h *History) AddResults(results compute.Results, label string) {
	entry := startEntry(label, defaultRepoPath)
	entry.Results = results

	updated := false
	for i, existing := range h.Entries {
		if existing.Commit == entry.Commit {
			entry.Timestamp = time.Now().UTC()
			h.Entries[i] = entry
			updated = true
			break
		}
	}

	if !updated {
		h.Entries = append(h.Entries, entry)
	}

	// Sort newest-first by Timestamp
	sort.SliceStable(h.Entries, func(i, j int) bool {
		return h.Entries[i].Timestamp.After(h.Entries[j].Timestamp)
	})
}

// Save results as an Entry to all the History in the file path.
func (h *History) Save(limit int) error {
	// limit how much is written
	if limit > 0 && limit < len(h.Entries) {
		h.Entries = h.Entries[:limit]
	}

	b, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.path, b, 0600) //nolint:mnd
}

// FindByRef finds a History Entry that matches the ref string and returns it.
func (h *History) FindByRef(ref string) *Entry {
	for _, entry := range h.Entries {
		if entry.Commit == ref || entry.Commit[:7] == ref ||
			entry.Branch == ref || entry.Label == ref {
			return &entry
		}
		for _, t := range entry.Tags {
			if t == ref {
				return &entry
			}
		}
	}
	return nil
}

func startEntry(label, repoPath string) Entry {
	var commit = "unknown"
	var branch = "unknown"
	var tags []string

	repo, err := git.PlainOpen(repoPath)
	if err == nil { //nolint:nestif
		head, herr := repo.Head()
		if herr == nil {
			commit = head.Hash().String()

			// Detect if HEAD is pointing to a named branch
			branch = "detached"
			if head.Name().IsBranch() {
				branch = head.Name().Short()
			}

			tagIter, _ := repo.Tags()
			_ = tagIter.ForEach(func(ref *plumbing.Reference) error {
				if ref.Hash() == head.Hash() {
					tags = append(tags, ref.Name().Short())
				}
				return nil
			})
		}
	}

	return Entry{
		Commit:    commit,
		Branch:    branch,
		Tags:      tags,
		Label:     label,
		Timestamp: time.Now().UTC(),
	}
}
