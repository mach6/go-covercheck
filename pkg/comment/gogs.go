package comment

import (
	"context"
	"fmt"
	"strings"

	gogs "github.com/gogits/go-gogs-client"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

// GogsPoster implements the Poster interface for Gogs using the official SDK.
type GogsPoster struct {
	client  *gogs.Client
	baseURL string
}

// NewGogsPoster creates a new Gogs poster instance using the official SDK.
func NewGogsPoster(baseURL string) *GogsPoster {
	if baseURL == "" {
		// There's no default public Gogs instance, so this should be provided
		baseURL = "https://try.gogs.io"
	}

	return &GogsPoster{
		client:  nil, // Will be initialized in PostComment with the token
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// PostComment posts coverage results as a comment to a Gogs pull request using the official SDK.
func (g *GogsPoster) PostComment(ctx context.Context, results compute.Results, cfg *config.Config) error {
	commentCfg := cfg.Comment
	if commentCfg.Platform.Token == "" {
		return fmt.Errorf("gogs token is required")
	}
	if commentCfg.Platform.Repository == "" {
		return fmt.Errorf("repository is required (format: owner/repo)")
	}
	if commentCfg.Platform.PullRequestID <= 0 {
		return fmt.Errorf("pull request ID is required")
	}

	// Initialize the client with token if not already done
	if g.client == nil {
		client := gogs.NewClient(g.baseURL, commentCfg.Platform.Token)
		g.client = client
	}

	// Parse repository owner and name
	repoParts := strings.Split(commentCfg.Platform.Repository, "/")
	if len(repoParts) != 2 {
		return fmt.Errorf("invalid repository format: %s (expected: owner/repo)", commentCfg.Platform.Repository)
	}
	owner, repo := repoParts[0], repoParts[1]

	// Generate the comment content
	markdown := FormatMarkdown(results, cfg, commentCfg.Platform.IncludeColors)

	// Check if we should update an existing comment
	if commentCfg.Platform.UpdateExisting {
		if err := g.updateExistingComment(ctx, owner, repo, commentCfg.Platform.PullRequestID, markdown); err != nil {
			// If updating fails, fall back to creating a new comment
			return g.createNewComment(ctx, owner, repo, commentCfg.Platform.PullRequestID, markdown)
		}
		return nil
	}

	// Create a new comment
	return g.createNewComment(ctx, owner, repo, commentCfg.Platform.PullRequestID, markdown)
}

// createNewComment creates a new comment on the pull request.
func (g *GogsPoster) createNewComment(ctx context.Context, owner, repo string, prID int, content string) error {
	issueCommentOption := gogs.CreateIssueCommentOption{
		Body: content,
	}

	// Convert PR ID to int64 as expected by the Gogs client
	prIDInt64 := int64(prID)

	_, err := g.client.CreateIssueComment(owner, repo, prIDInt64, issueCommentOption)
	if err != nil {
		return fmt.Errorf("failed to create Gogs comment: %w", err)
	}

	return nil
}

// updateExistingComment attempts to update an existing comment from this tool.
func (g *GogsPoster) updateExistingComment(ctx context.Context, owner, repo string, prID int, content string) error {
	// Convert PR ID to int64 as expected by the Gogs client
	prIDInt64 := int64(prID)

	// Get existing comments to find one to update
	comments, err := g.client.ListIssueComments(owner, repo, prIDInt64)
	if err != nil {
		return fmt.Errorf("failed to list Gogs comments: %w", err)
	}

	// Look for an existing comment from go-covercheck (contains the signature)
	for _, comment := range comments {
		if strings.Contains(comment.Body, "## 🚦 Coverage Report") && strings.Contains(comment.Body, "go-covercheck") {
			// Update this comment
			editOption := gogs.EditIssueCommentOption{
				Body: content,
			}
			_, err := g.client.EditIssueComment(owner, repo, prIDInt64, comment.ID, editOption)
			if err != nil {
				return fmt.Errorf("failed to update Gogs comment: %w", err)
			}
			return nil
		}
	}

	// No existing comment found to update
	return fmt.Errorf("no existing go-covercheck comment found to update")
}