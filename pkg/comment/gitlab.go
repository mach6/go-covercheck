package comment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

// GitLabPoster implements the Poster interface for GitLab.
type GitLabPoster struct {
	client  *http.Client
	baseURL string
}

// NewGitLabPoster creates a new GitLab poster instance.
func NewGitLabPoster(baseURL string) *GitLabPoster {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	return &GitLabPoster{
		client:  &http.Client{},
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// gitLabNoteRequest represents the JSON structure for creating a GitLab MR note.
type gitLabNoteRequest struct {
	Body string `json:"body"`
}

// gitLabNote represents a GitLab MR note response.
type gitLabNote struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
	Author struct {
		Username string `json:"username"`
	} `json:"author"`
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
	
	// Format the comment content
	body := FormatMarkdown(results, cfg, commentCfg.Platform.IncludeColors)
	
	// If updateExisting is enabled, try to find and update existing comment
	if commentCfg.Platform.UpdateExisting {
		if err := g.updateExistingNote(ctx, body, commentCfg); err != nil {
			// If update fails, fall back to creating a new note
			return g.createNote(ctx, body, commentCfg)
		}
		return nil
	}
	
	// Create a new note
	return g.createNote(ctx, body, commentCfg)
}

// createNote creates a new note on the merge request.
func (g *GitLabPoster) createNote(ctx context.Context, body string, commentCfg config.CommentConfig) error {
	// URL encode the repository if it contains special characters
	repoPath := strings.ReplaceAll(commentCfg.Platform.Repository, "/", "%2F")
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/notes",
		g.baseURL, repoPath, commentCfg.Platform.PullRequestID)
	
	reqBody := gitLabNoteRequest{Body: body}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal note request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("PRIVATE-TOKEN", commentCfg.Platform.Token)
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post note: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gitlab API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// updateExistingNote finds and updates an existing go-covercheck note.
func (g *GitLabPoster) updateExistingNote(ctx context.Context, body string, commentCfg config.CommentConfig) error {
	// Get existing notes
	repoPath := strings.ReplaceAll(commentCfg.Platform.Repository, "/", "%2F")
	url := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/notes",
		g.baseURL, repoPath, commentCfg.Platform.PullRequestID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("PRIVATE-TOKEN", commentCfg.Platform.Token)
	
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get notes: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gitlab API returned status %d when getting notes", resp.StatusCode)
	}
	
	var notes []gitLabNote
	if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
		return fmt.Errorf("failed to decode notes response: %w", err)
	}
	
	// Find existing go-covercheck note
	var existingNote *gitLabNote
	for _, note := range notes {
		if strings.Contains(note.Body, "## 🚦 Coverage Report") && strings.Contains(note.Body, "go-covercheck") {
			existingNote = &note
			break
		}
	}
	
	if existingNote == nil {
		return fmt.Errorf("no existing go-covercheck note found")
	}
	
	// Update the existing note
	updateURL := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests/%d/notes/%d",
		g.baseURL, repoPath, commentCfg.Platform.PullRequestID, existingNote.ID)
	
	reqBody := gitLabNoteRequest{Body: body}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}
	
	updateReq, err := http.NewRequestWithContext(ctx, "PUT", updateURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}
	
	updateReq.Header.Set("PRIVATE-TOKEN", commentCfg.Platform.Token)
	updateReq.Header.Set("Content-Type", "application/json")
	
	updateResp, err := g.client.Do(updateReq)
	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}
	defer updateResp.Body.Close()
	
	if updateResp.StatusCode < 200 || updateResp.StatusCode >= 300 {
		return fmt.Errorf("gitlab API returned status %d when updating note", updateResp.StatusCode)
	}
	
	return nil
}