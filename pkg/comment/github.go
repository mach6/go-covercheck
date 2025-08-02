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

// GitHubPoster implements the Poster interface for GitHub.
type GitHubPoster struct {
	client  *http.Client
	baseURL string
}

// NewGitHubPoster creates a new GitHub poster instance.
func NewGitHubPoster(baseURL string) *GitHubPoster {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	return &GitHubPoster{
		client:  &http.Client{},
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// gitHubCommentRequest represents the JSON structure for creating a GitHub PR comment.
type gitHubCommentRequest struct {
	Body string `json:"body"`
}

// gitHubComment represents a GitHub PR comment response.
type gitHubComment struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

// PostComment posts coverage results as a comment to a GitHub pull request.
func (g *GitHubPoster) PostComment(ctx context.Context, results compute.Results, cfg *config.Config) error {
	commentCfg := cfg.Comment
	if commentCfg.Platform.Token == "" {
		return fmt.Errorf("github token is required")
	}
	if commentCfg.Platform.Repository == "" {
		return fmt.Errorf("repository is required (format: owner/repo)")
	}
	if commentCfg.Platform.PullRequestID <= 0 {
		return fmt.Errorf("pull request ID is required")
	}
	
	// Format the comment content
	body := FormatMarkdown(results, cfg, commentCfg.Platform.IncludeColors)
	
	// If updateExisting is enabled, try to find and update existing comment
	if commentCfg.Platform.UpdateExisting {
		if err := g.updateExistingComment(ctx, body, commentCfg); err != nil {
			// If update fails, fall back to creating a new comment
			return g.createComment(ctx, body, commentCfg)
		}
		return nil
	}
	
	// Create a new comment
	return g.createComment(ctx, body, commentCfg)
}

// createComment creates a new comment on the pull request.
func (g *GitHubPoster) createComment(ctx context.Context, body string, commentCfg config.CommentConfig) error {
	url := fmt.Sprintf("%s/repos/%s/issues/%d/comments",
		g.baseURL, commentCfg.Platform.Repository, commentCfg.Platform.PullRequestID)
	
	reqBody := gitHubCommentRequest{Body: body}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal comment request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "token "+commentCfg.Platform.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// updateExistingComment finds and updates an existing go-covercheck comment.
func (g *GitHubPoster) updateExistingComment(ctx context.Context, body string, commentCfg config.CommentConfig) error {
	// Get existing comments
	url := fmt.Sprintf("%s/repos/%s/issues/%d/comments",
		g.baseURL, commentCfg.Platform.Repository, commentCfg.Platform.PullRequestID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "token "+commentCfg.Platform.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get comments: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github API returned status %d when getting comments", resp.StatusCode)
	}
	
	var comments []gitHubComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return fmt.Errorf("failed to decode comments response: %w", err)
	}
	
	// Find existing go-covercheck comment
	var existingComment *gitHubComment
	for _, comment := range comments {
		if strings.Contains(comment.Body, "## 🚦 Coverage Report") && strings.Contains(comment.Body, "go-covercheck") {
			existingComment = &comment
			break
		}
	}
	
	if existingComment == nil {
		return fmt.Errorf("no existing go-covercheck comment found")
	}
	
	// Update the existing comment
	updateURL := fmt.Sprintf("%s/repos/%s/issues/comments/%d",
		g.baseURL, commentCfg.Platform.Repository, existingComment.ID)
	
	reqBody := gitHubCommentRequest{Body: body}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal update request: %w", err)
	}
	
	updateReq, err := http.NewRequestWithContext(ctx, "PATCH", updateURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}
	
	updateReq.Header.Set("Authorization", "token "+commentCfg.Platform.Token)
	updateReq.Header.Set("Content-Type", "application/json")
	updateReq.Header.Set("Accept", "application/vnd.github.v3+json")
	
	updateResp, err := g.client.Do(updateReq)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}
	defer updateResp.Body.Close()
	
	if updateResp.StatusCode < 200 || updateResp.StatusCode >= 300 {
		return fmt.Errorf("github API returned status %d when updating comment", updateResp.StatusCode)
	}
	
	return nil
}