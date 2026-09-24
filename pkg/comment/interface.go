// Package comment posts coverage results as comments on pull requests and merge requests.
package comment

import "context"

// Comment is a single comment on a pull request or merge request.
type Comment struct {
	ID   int64
	Body string
}

// Poster is implemented by each supported platform. A Poster is bound to a single
// pull request or merge request when it is created.
type Poster interface {
	// ListComments returns all comments on the pull request or merge request.
	ListComments(ctx context.Context) ([]Comment, error)
	// CreateComment adds a new comment with the given body.
	CreateComment(ctx context.Context, body string) error
	// UpdateComment replaces the body of the comment with the given id.
	UpdateComment(ctx context.Context, id int64, body string) error
}
