package comment

import (
	"fmt"
	"strings"

	"github.com/mach6/go-covercheck/pkg/config"
)

// NewPoster creates the Poster for the configured platform.
func NewPoster(p *config.PlatformConfig) (Poster, error) {
	switch strings.ToLower(p.Type) {
	case config.CommentPlatformGitHub:
		return NewGitHubPoster(p.BaseURL, p.Token, p.Repository, p.PullRequestID)
	case config.CommentPlatformGitLab:
		return NewGitLabPoster(p.BaseURL, p.Token, p.Repository, p.PullRequestID)
	case config.CommentPlatformGitea:
		return NewGiteaPoster(p.BaseURL, p.Token, p.Repository, p.PullRequestID)
	case config.CommentPlatformGogs:
		return NewGogsPoster(p.BaseURL, p.Token, p.Repository, p.PullRequestID)
	default:
		return nil, fmt.Errorf("unsupported comment platform %q (supported: %s)",
			p.Type, strings.Join(config.CommentPlatforms, ", "))
	}
}
