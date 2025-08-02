package comment

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v67/github"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

// GitHubPoster implements the Poster interface for GitHub.
type GitHubPoster struct {
	client  *github.Client
	baseURL string
}

// NewGitHubPoster creates a new GitHub poster instance.
func NewGitHubPoster(baseURL string) *GitHubPoster {
	client := github.NewClient(nil)

	// Set custom base URL if provided
	if baseURL != "" && baseURL != "https://api.github.com" {
		baseURL = strings.TrimSuffix(baseURL, "/") + "/"
		var err error
		client, err = client.WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			// Fallback to default if URL is invalid
			client = github.NewClient(nil)
		}
	}

	return &GitHubPoster{
		client:  client,
		baseURL: baseURL,
	}
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

	// Configure authentication
	g.client = g.client.WithAuthToken(commentCfg.Platform.Token)

	// Parse repository owner and name
	parts := strings.Split(commentCfg.Platform.Repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repository must be in format owner/repo")
	}
	owner, repo := parts[0], parts[1]

	// Format the comment content
	body := FormatMarkdown(results, cfg, commentCfg.Platform.IncludeColors)

	// If updateExisting is enabled, try to find and update existing comment
	if commentCfg.Platform.UpdateExisting {
		if err := g.updateExistingComment(ctx, body, owner, repo, commentCfg.Platform.PullRequestID); err != nil {
			// If update fails, fall back to creating a new comment
			return g.createComment(ctx, body, owner, repo, commentCfg.Platform.PullRequestID)
		}
		return nil
	}

	// Create a new comment
	return g.createComment(ctx, body, owner, repo, commentCfg.Platform.PullRequestID)
}

// createComment creates a new comment on the pull request.
func (g *GitHubPoster) createComment(ctx context.Context, body, owner, repo string, prID int) error {
	comment := &github.IssueComment{
		Body: &body,
	}

	_, _, err := g.client.Issues.CreateComment(ctx, owner, repo, prID, comment)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

// updateExistingComment finds and updates an existing go-covercheck comment.
func (g *GitHubPoster) updateExistingComment(ctx context.Context, body, owner, repo string, prID int) error {
	// Get existing comments
	comments, _, err := g.client.Issues.ListComments(ctx, owner, repo, prID, nil)
	if err != nil {
		return fmt.Errorf("failed to get comments: %w", err)
	}

	// Find existing go-covercheck comment
	var existingComment *github.IssueComment
	for _, comment := range comments {
		if comment.Body != nil && strings.Contains(*comment.Body, "## 🚦 Coverage Report") &&
			strings.Contains(*comment.Body, "go-covercheck") {
			existingComment = comment
			break
		}
	}

	if existingComment == nil {
		return fmt.Errorf("no existing go-covercheck comment found")
	}

	// Update the existing comment
	updatedComment := &github.IssueComment{
		Body: &body,
	}

	_, _, err = g.client.Issues.EditComment(ctx, owner, repo, *existingComment.ID, updatedComment)
	if err != nil {
		return fmt.Errorf("failed to update comment: %w", err)
	}

	return nil
}
