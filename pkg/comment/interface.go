// Package comment posts coverage results as comments on pull requests and merge requests.
package comment

import (
	"context"
	"errors"
	"net/http"
)

// ErrNotFound is wrapped by UpdateComment when the platform reports the comment no longer
// exists (HTTP 404 or 410).
var ErrNotFound = errors.New("comment not found")

// isGone reports whether an HTTP status means the target comment no longer exists.
func isGone(status int) bool {
	return status == http.StatusNotFound || status == http.StatusGone
}

// Comment is a single comment on a pull request or merge request.
type Comment struct {
	ID   int64
	Body string
	// Author is the login or username of the account that wrote the comment.
	Author string
}

// Poster is implemented by each supported platform. A Poster is bound to a single
// pull request or merge request when it is created.
type Poster interface {
	// CurrentUser returns the login or username of the account the token authenticates as.
	CurrentUser(ctx context.Context) (string, error)
	// ListComments returns all comments on the pull request or merge request.
	ListComments(ctx context.Context) ([]Comment, error)
	// CreateComment adds a new comment with the given body.
	CreateComment(ctx context.Context, body string) error
	// UpdateComment replaces the body of the comment with the given id. The error wraps
	// ErrNotFound when the comment no longer exists.
	UpdateComment(ctx context.Context, id int64, body string) error
}
