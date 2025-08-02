package comment

import (
	"context"
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

// GiteaPoster implements the Poster interface for Gitea using the official SDK.
type GiteaPoster struct {
	client  *gitea.Client
	baseURL string
}

// NewGiteaPoster creates a new Gitea poster instance using the official SDK.
func NewGiteaPoster(baseURL string) *GiteaPoster {
	if baseURL == "" {
		// There's no default public Gitea instance, so this should be provided
		baseURL = "https://gitea.com"
	}

	return &GiteaPoster{
		client:  nil, // Will be initialized in PostComment with the token
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// PostComment posts coverage results as a comment to a Gitea pull request using the official SDK.
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

	// Initialize the client with token if not already done
	if g.client == nil {
		client, err := gitea.NewClient(g.baseURL, gitea.SetToken(commentCfg.Platform.Token))
		if err != nil {
			return fmt.Errorf("failed to create Gitea client: %w", err)
		}
		g.client = client
	}

	// Parse repository owner and name
	parts := strings.Split(commentCfg.Platform.Repository, "/")
	if len(parts) != 2 {
		return fmt.Errorf("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]
	prID := int64(commentCfg.Platform.PullRequestID)

	// Format the comment content
	body := FormatMarkdown(results, cfg, commentCfg.Platform.IncludeColors)

	// If updateExisting is enabled, try to find and update existing comment
	if commentCfg.Platform.UpdateExisting {
		if err := g.updateExistingComment(ctx, body, owner, repo, prID); err != nil {
			// If update fails, fall back to creating a new comment
			return g.createComment(ctx, body, owner, repo, prID)
		}
		return nil
	}

	// Create a new comment
	return g.createComment(ctx, body, owner, repo, prID)
}

// createComment creates a new comment on the pull request using the Gitea SDK.
func (g *GiteaPoster) createComment(ctx context.Context, body, owner, repo string, prID int64) error {
	options := gitea.CreatePullReviewOptions{
		State: gitea.ReviewStateComment, // Just a comment, not approval/request changes
		Body:  body,
	}

	_, _, err := g.client.CreatePullReview(owner, repo, prID, options)
	if err != nil {
		return fmt.Errorf("failed to create pull request review comment: %w", err)
	}

	return nil
}

// updateExistingComment finds and updates an existing go-covercheck comment using the Gitea SDK.
func (g *GiteaPoster) updateExistingComment(ctx context.Context, body, owner, repo string, prID int64) error {
	// Get existing reviews/comments using the SDK
	reviews, _, err := g.client.ListPullReviews(owner, repo, prID, gitea.ListPullReviewsOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pull reviews: %w", err)
	}

	// Find existing go-covercheck review
	var existingReview *gitea.PullReview
	for i := range reviews {
		if strings.Contains(reviews[i].Body, "## 🚦 Coverage Report") && strings.Contains(reviews[i].Body, "go-covercheck") {
			existingReview = reviews[i]
			break
		}
	}

	if existingReview == nil {
		return fmt.Errorf("no existing go-covercheck review found")
	}

	// For Gitea, we need to submit a new review to update, as there's no direct update API
	// This is a limitation of Gitea's API compared to GitHub/GitLab
	return g.createComment(ctx, body, owner, repo, prID)
}
