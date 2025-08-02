// Package comment provides interfaces and implementations for posting coverage results to various platforms.
package comment

import (
	"context"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

// Poster defines the interface for posting coverage results as comments to different platforms.
type Poster interface {
	// PostComment posts a coverage result as a comment to a pull request or merge request.
	// ctx provides cancellation and timeout control.
	// results contains the coverage analysis results to post.
	// cfg contains the application configuration including comment settings.
	PostComment(ctx context.Context, results compute.Results, cfg *config.Config) error
}
