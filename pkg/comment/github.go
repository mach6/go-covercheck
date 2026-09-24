package comment

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v92/github"
)

// GitHubPoster posts comments to a GitHub pull request.
type GitHubPoster struct {
	client      *github.Client
	owner, repo string
	number      int
}

// NewGitHubPoster creates a GitHubPoster. baseURL is only needed for GitHub Enterprise Server.
func NewGitHubPoster(baseURL, token, repository string, number int) (*GitHubPoster, error) {
	owner, repo, err := splitRepository(repository)
	if err != nil {
		return nil, err
	}
	opts := []github.ClientOptionsFunc{github.WithAuthToken(token), github.WithTimeout(httpTimeout)}
	if !isPublicGitHub(baseURL) {
		opts = append(opts, github.WithEnterpriseURLs(baseURL, baseURL))
	}
	client, err := github.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create github client (base URL %q): %w", baseURL, err)
	}
	return &GitHubPoster{client: client, owner: owner, repo: repo, number: number}, nil
}

// isPublicGitHub reports whether baseURL is empty or points at github.com, in which case the
// default API endpoint is used rather than GitHub Enterprise paths (/api/v3/).
func isPublicGitHub(baseURL string) bool {
	switch strings.TrimSuffix(strings.ToLower(baseURL), "/") {
	case "", "https://github.com", "https://api.github.com":
		return true
	}
	return false
}

// ListComments returns all comments on the pull request.
func (g *GitHubPoster) ListComments(ctx context.Context) ([]Comment, error) {
	var out []Comment
	opts := &github.IssueListCommentsOptions{ListOptions: github.ListOptions{PerPage: listPageSize}}
	for {
		comments, resp, err := g.client.Issues.ListComments(ctx, g.owner, g.repo, g.number, opts)
		if err != nil {
			return nil, err
		}
		for _, c := range comments {
			out = append(out, Comment{ID: c.GetID(), Body: c.GetBody()})
		}
		if resp.NextPage == 0 {
			return out, nil
		}
		opts.Page = resp.NextPage
	}
}

// CreateComment adds a comment to the pull request.
func (g *GitHubPoster) CreateComment(ctx context.Context, body string) error {
	_, _, err := g.client.Issues.CreateComment(ctx, g.owner, g.repo, g.number, github.IssueCommentRequest{Body: body})
	if err != nil {
		return fmt.Errorf("failed to create github comment: %w", err)
	}
	return nil
}

// UpdateComment edits an existing comment.
func (g *GitHubPoster) UpdateComment(ctx context.Context, id int64, body string) error {
	_, _, err := g.client.Issues.UpdateComment(ctx, g.owner, g.repo, id, github.IssueCommentRequest{Body: body})
	if err != nil {
		return fmt.Errorf("failed to update github comment: %w", err)
	}
	return nil
}
