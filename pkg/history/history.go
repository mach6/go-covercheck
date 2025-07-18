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

type Entry struct {
	Commit    string          `json:"commit"`
	Branch    string          `json:"branch"`
	Tags      []string        `json:"tags,omitempty"`
	Label     string          `json:"label,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Results   compute.Results `json:"results"`
}

type History struct {
	Entries []Entry `json:"entries"`
}

func Load(path string) (*History, error) {
	var h History
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &h)
	if err != nil {
		return nil, err
	}

	return &h, nil
}

func Save(path string, results compute.Results, label string, limit int) error {
	entry := startEntry(label)
	entry.Results = results

	// loads previous history if it exists, so we can dedup when writing.
	history, err := Load(path)
	if err != nil {
		history = &History{Entries: []Entry{}}
	}

	updated := false
	for i, existing := range history.Entries {
		if existing.Commit == entry.Commit {
			entry.Timestamp = time.Now().UTC()
			history.Entries[i] = entry
			updated = true
			break
		}
	}

	if !updated {
		history.Entries = append(history.Entries, entry)
	}

	// Sort newest-first by Timestamp
	sort.SliceStable(history.Entries, func(i, j int) bool {
		return history.Entries[i].Timestamp.After(history.Entries[j].Timestamp)
	})

	// limit how much is written
	if limit > 0 {
		history.Entries = history.Entries[:limit]
	}

	b, _ := json.MarshalIndent(history, "", "  ")
	return os.WriteFile(path, b, 0644)
}

func FindByRef(h *History, ref string) *Entry {
	for _, entry := range h.Entries {
		if entry.Commit == ref || entry.Branch == ref || entry.Label == ref {
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

func startEntry(label string) Entry {
	var commit = "unknown"
	var branch = "unknown"
	var tags []string

	repo, err := git.PlainOpen(".")
	if err == nil {
		head, herr := repo.Head()
		if herr == nil {
			commit = head.Hash().String()

			// Detect if HEAD is pointing to a named branch
			if head.Name().IsBranch() {
				branch = head.Name().Short()
			} else {
				branch = "detached"
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
