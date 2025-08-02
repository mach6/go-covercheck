package comment

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GitLabPoster implements the Poster interface for GitLab.
type GitLabPoster struct {
	client  *gitlab.Client
	baseURL string
}

// NewGitLabPoster creates a new GitLab poster instance.
func NewGitLabPoster(baseURL string) *GitLabPoster {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	// The client will be configured with auth token when PostComment is called
	client, _ := gitlab.NewClient("", gitlab.WithBaseURL(baseURL))

	return &GitLabPoster{
		client:  client,
		baseURL: baseURL,
	}
}

// PostComment posts coverage results as a note to a GitLab merge request.
func (g *GitLabPoster) PostComment(ctx context.Context, results compute.Results, cfg *config.Config) error {
	commentCfg := cfg.Comment
	if commentCfg.Platform.Token == "" {
		return fmt.Errorf("gitlab token is required")
	}
	if commentCfg.Platform.Repository == "" {
		return fmt.Errorf("repository is required (format: group/project or project-id)")
	}
	if commentCfg.Platform.PullRequestID <= 0 {
		return fmt.Errorf("merge request ID is required")
	}

	// Configure authentication
	var err error
	g.client, err = gitlab.NewClient(commentCfg.Platform.Token, gitlab.WithBaseURL(g.baseURL))
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Format the comment content
	body := FormatMarkdown(results, cfg, commentCfg.Platform.IncludeColors)

	// If updateExisting is enabled, try to find and update existing comment
	if commentCfg.Platform.UpdateExisting {
		if err := g.updateExistingNote(ctx, body, commentCfg.Platform.Repository, commentCfg.Platform.PullRequestID); err != nil {
			// If update fails, fall back to creating a new note
			return g.createNote(ctx, body, commentCfg.Platform.Repository, commentCfg.Platform.PullRequestID)
		}
		return nil
	}

	// Create a new note
	return g.createNote(ctx, body, commentCfg.Platform.Repository, commentCfg.Platform.PullRequestID)
}

// createNote creates a new note on the merge request.
func (g *GitLabPoster) createNote(ctx context.Context, body, repository string, mrID int) error {
	// Parse project ID - could be numeric ID or group/project format
	var projectID interface{}
	if pid, err := strconv.Atoi(repository); err == nil {
		projectID = pid
	} else {
		projectID = repository
	}

	options := &gitlab.CreateMergeRequestNoteOptions{
		Body: &body,
	}

	_, _, err := g.client.Notes.CreateMergeRequestNote(projectID, mrID, options, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	return nil
}

// updateExistingNote finds and updates an existing go-covercheck note.
func (g *GitLabPoster) updateExistingNote(ctx context.Context, body, repository string, mrID int) error {
	// Parse project ID - could be numeric ID or group/project format
	var projectID interface{}
	if pid, err := strconv.Atoi(repository); err == nil {
		projectID = pid
	} else {
		projectID = repository
	}

	// Get existing notes
	notes, _, err := g.client.Notes.ListMergeRequestNotes(projectID, mrID, nil, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to get notes: %w", err)
	}

	// Find existing go-covercheck note
	var existingNote *gitlab.Note
	for _, note := range notes {
		if strings.Contains(note.Body, "## 🚦 Coverage Report") &&
			strings.Contains(note.Body, "go-covercheck") {
			existingNote = note
			break
		}
	}

	if existingNote == nil {
		return fmt.Errorf("no existing go-covercheck note found")
	}

	// Update the existing note
	options := &gitlab.UpdateMergeRequestNoteOptions{
		Body: &body,
	}

	_, _, err = g.client.Notes.UpdateMergeRequestNote(projectID, mrID, existingNote.ID, options, gitlab.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	return nil
}
