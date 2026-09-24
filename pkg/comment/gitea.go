package comment

import (
	"context"
	"fmt"
	"net/http"

	"code.gitea.io/sdk/gitea"
)

const giteaDefaultURL = "https://gitea.com"

// GiteaPoster posts comments to a Gitea pull request.
type GiteaPoster struct {
	client      *gitea.Client
	owner, repo string
	number      int64
}

// NewGiteaPoster creates a GiteaPoster. baseURL defaults to https://gitea.com.
func NewGiteaPoster(baseURL, token, repository string, number int) (*GiteaPoster, error) {
	owner, repo, err := splitRepository(repository)
	if err != nil {
		return nil, err
	}
	if baseURL == "" {
		baseURL = giteaDefaultURL
	}
	// An empty version skips the server version lookup the SDK would otherwise make here.
	client, err := gitea.NewClient(baseURL,
		gitea.SetToken(token),
		gitea.SetGiteaVersion(""),
		gitea.SetHTTPClient(&http.Client{Timeout: httpTimeout}))
	if err != nil {
		return nil, fmt.Errorf("failed to create gitea client: %w", err)
	}
	return &GiteaPoster{client: client, owner: owner, repo: repo, number: int64(number)}, nil
}

// ListComments returns all comments on the pull request.
func (g *GiteaPoster) ListComments(ctx context.Context) ([]Comment, error) {
	g.client.SetContext(ctx)
	var out []Comment
	opts := gitea.ListIssueCommentOptions{ListOptions: gitea.ListOptions{Page: 1, PageSize: listPageSize}}
	for {
		comments, _, err := g.client.ListIssueComments(g.owner, g.repo, g.number, opts)
		if err != nil {
			return nil, err
		}
		for _, c := range comments {
			out = append(out, Comment{ID: c.ID, Body: c.Body})
		}
		if len(comments) < listPageSize {
			return out, nil
		}
		opts.Page++
	}
}

// CreateComment adds a comment to the pull request.
func (g *GiteaPoster) CreateComment(ctx context.Context, body string) error {
	g.client.SetContext(ctx)
	opts := gitea.CreateIssueCommentOption{Body: body}
	if _, _, err := g.client.CreateIssueComment(g.owner, g.repo, g.number, opts); err != nil {
		return fmt.Errorf("failed to create gitea comment: %w", err)
	}
	return nil
}

// UpdateComment edits an existing comment.
func (g *GiteaPoster) UpdateComment(ctx context.Context, id int64, body string) error {
	g.client.SetContext(ctx)
	opts := gitea.EditIssueCommentOption{Body: body}
	if _, _, err := g.client.EditIssueComment(g.owner, g.repo, id, opts); err != nil {
		return fmt.Errorf("failed to update gitea comment: %w", err)
	}
	return nil
}
