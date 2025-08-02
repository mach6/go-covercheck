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

// GiteaPoster implements the Poster interface for Gitea.
type GiteaPoster struct {
	client  *http.Client
	baseURL string
}

// NewGiteaPoster creates a new Gitea poster instance.
func NewGiteaPoster(baseURL string) *GiteaPoster {
	if baseURL == "" {
		// There's no default public Gitea instance, so this should be provided
		baseURL = "https://gitea.com"
	}
	return &GiteaPoster{
		client:  &http.Client{},
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// giteaCommentRequest represents the JSON structure for creating a Gitea PR comment.
type giteaCommentRequest struct {
	Body string `json:"body"`
}

// giteaComment represents a Gitea PR comment response.
type giteaComment struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
	} `json:"user"`
}

// PostComment posts coverage results as a comment to a Gitea pull request.
func (g *GiteaPoster) PostComment(ctx context.Context, results compute.Results, cfg *config.Config) error {
	commentCfg := cfg.Comment
	if commentCfg.Platform.Token == "" {
		return fmt.Errorf("gitea token is required")
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
func (g *GiteaPoster) createComment(ctx context.Context, body string, commentCfg config.CommentConfig) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/pulls/%d/reviews",
		g.baseURL, commentCfg.Platform.Repository, commentCfg.Platform.PullRequestID)
	
	// Gitea uses a different API structure for PR comments
	reqBody := map[string]interface{}{
		"body":  body,
		"event": "COMMENT",
	}
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
	
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gitea API returned status %d", resp.StatusCode)
	}
	
	return nil
}

// updateExistingComment finds and updates an existing go-covercheck comment.
func (g *GiteaPoster) updateExistingComment(ctx context.Context, body string, commentCfg config.CommentConfig) error {
	// Get existing reviews/comments
	url := fmt.Sprintf("%s/api/v1/repos/%s/pulls/%d/reviews",
		g.baseURL, commentCfg.Platform.Repository, commentCfg.Platform.PullRequestID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Authorization", "token "+commentCfg.Platform.Token)
	
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get reviews: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gitea API returned status %d when getting reviews", resp.StatusCode)
	}
	
	var reviews []struct {
		ID   int    `json:"id"`
		Body string `json:"body"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&reviews); err != nil {
		return fmt.Errorf("failed to decode reviews response: %w", err)
	}
	
	// Find existing go-covercheck review
	var existingReview *struct {
		ID   int    `json:"id"`
		Body string `json:"body"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	}
	for _, review := range reviews {
		if strings.Contains(review.Body, "## 🚦 Coverage Report") && strings.Contains(review.Body, "go-covercheck") {
			existingReview = &review
			break
		}
	}
	
	if existingReview == nil {
		return fmt.Errorf("no existing go-covercheck review found")
	}
	
	// For Gitea, we need to submit a new review to update, as there's no direct update API
	// This is a limitation of Gitea's API compared to GitHub/GitLab
	return g.createComment(ctx, body, commentCfg)
}