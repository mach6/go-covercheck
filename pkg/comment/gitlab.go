package comment

import (
	"context"
	"fmt"
	"net/http"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GitLabPoster posts notes to a GitLab merge request.
type GitLabPoster struct {
	client  *gitlab.Client
	project string
	number  int64
}

// NewGitLabPoster creates a GitLabPoster. repository is a "group/project" path or a numeric
// project ID. baseURL is only needed for self-hosted GitLab.
func NewGitLabPoster(baseURL, token, repository string, number int) (*GitLabPoster, error) {
	opts := []gitlab.ClientOptionFunc{gitlab.WithHTTPClient(&http.Client{Timeout: httpTimeout})}
	if baseURL != "" {
		opts = append(opts, gitlab.WithBaseURL(baseURL))
	}
	client, err := gitlab.NewClient(token, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gitlab client: %w", err)
	}
	return &GitLabPoster{client: client, project: repository, number: int64(number)}, nil
}

// ListComments returns all notes on the merge request.
func (g *GitLabPoster) ListComments(ctx context.Context) ([]Comment, error) {
	var out []Comment
	opts := &gitlab.ListMergeRequestNotesOptions{ListOptions: gitlab.ListOptions{PerPage: listPageSize}}
	for {
		notes, resp, err := g.client.Notes.ListMergeRequestNotes(g.project, g.number, opts, gitlab.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		for _, n := range notes {
			out = append(out, Comment{ID: n.ID, Body: n.Body})
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

// CreateComment adds a note to the merge request.
func (g *GitLabPoster) CreateComment(ctx context.Context, body string) error {
	opts := &gitlab.CreateMergeRequestNoteOptions{Body: &body}
	if _, _, err := g.client.Notes.CreateMergeRequestNote(g.project, g.number, opts, gitlab.WithContext(ctx)); err != nil {
		return fmt.Errorf("failed to create gitlab note: %w", err)
	}
	return nil
}

// UpdateComment edits an existing note.
func (g *GitLabPoster) UpdateComment(ctx context.Context, id int64, body string) error {
	opts := &gitlab.UpdateMergeRequestNoteOptions{Body: &body}
	_, _, err := g.client.Notes.UpdateMergeRequestNote(g.project, g.number, id, opts, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to update gitlab note: %w", err)
	}
	return nil
}
