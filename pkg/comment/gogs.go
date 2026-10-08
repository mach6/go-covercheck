package comment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GogsPoster posts comments to a Gogs pull request. Gogs has no maintained Go SDK, so this
// talks to the Gogs REST API (/api/v1) directly.
//
// The routes are registered in internal/route/api/v1/api.go of github.com/gogs/gogs:
//   - GET, POST /repos/:owner/:repo/issues/:index/comments
//   - PATCH /repos/:owner/:repo/issues/:index/comments/:id
//   - GET /user
//
// The list handler (listIssueComments in internal/route/api/v1/repo_issue_comment.go) is not
// paginated; it accepts only a "since" filter and returns every comment.
type GogsPoster struct {
	client      *http.Client
	token       string
	userURL     string
	commentsURL string
}

type gogsUser struct {
	Login    string `json:"login"`
	UserName string `json:"username"`
}

type gogsComment struct {
	ID     int64     `json:"id"`
	Body   string    `json:"body"`
	Poster *gogsUser `json:"user"`
}

// name returns the login, falling back to the legacy username field.
func (u *gogsUser) name() string {
	switch {
	case u == nil:
		return ""
	case u.Login != "":
		return u.Login
	default:
		return u.UserName
	}
}

// httpError is returned for a non-2xx response.
type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

// NewGogsPoster creates a GogsPoster for the Gogs instance at baseURL.
func NewGogsPoster(baseURL, token, repository string, number int) (*GogsPoster, error) {
	owner, repo, err := splitRepository(repository)
	if err != nil {
		return nil, err
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid gogs base URL %q: %w", baseURL, err)
	}
	api := strings.TrimSuffix(baseURL, "/") + "/api/v1"
	commentsURL := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments",
		api, url.PathEscape(owner), url.PathEscape(repo), number)
	return &GogsPoster{
		client:      &http.Client{Timeout: httpTimeout},
		token:       token,
		userURL:     api + "/user",
		commentsURL: commentsURL,
	}, nil
}

// ListComments returns all comments on the pull request.
func (g *GogsPoster) ListComments(ctx context.Context) ([]Comment, error) {
	var comments []gogsComment
	if err := g.do(ctx, http.MethodGet, g.commentsURL, nil, &comments); err != nil {
		return nil, err
	}
	out := make([]Comment, 0, len(comments))
	for _, c := range comments {
		out = append(out, Comment{ID: c.ID, Body: c.Body, Author: c.Poster.name()})
	}
	return out, nil
}

// CurrentUser returns the login of the authenticated user.
func (g *GogsPoster) CurrentUser(ctx context.Context) (string, error) {
	var u gogsUser
	if err := g.do(ctx, http.MethodGet, g.userURL, nil, &u); err != nil {
		return "", fmt.Errorf("failed to look up gogs user: %w", err)
	}
	return u.name(), nil
}

// CreateComment adds a comment to the pull request.
func (g *GogsPoster) CreateComment(ctx context.Context, body string) error {
	if err := g.do(ctx, http.MethodPost, g.commentsURL, map[string]string{"body": body}, nil); err != nil {
		return fmt.Errorf("failed to create gogs comment: %w", err)
	}
	return nil
}

// UpdateComment edits an existing comment.
func (g *GogsPoster) UpdateComment(ctx context.Context, id int64, body string) error {
	if err := g.do(ctx, http.MethodPatch, fmt.Sprintf("%s/%d", g.commentsURL, id),
		map[string]string{"body": body}, nil); err != nil {
		var he *httpError
		if errors.As(err, &he) && isGone(he.status) {
			err = fmt.Errorf("%w: %w", ErrNotFound, err)
		}
		return fmt.Errorf("failed to update gogs comment: %w", err)
	}
	return nil
}

func (g *GogsPoster) do(ctx context.Context, method, uri string, in, out any) error {
	var reqBody io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, uri, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+g.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512)) //nolint:mnd
		return &httpError{
			status: resp.StatusCode,
			msg:    fmt.Sprintf("%s %s: %s: %s", method, uri, resp.Status, strings.TrimSpace(string(msg))),
		}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
