package comment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

// Marker is embedded in every posted comment so an earlier comment can be found and updated.
const Marker = "<!-- go-covercheck-report -->"

// tokenEnvVars lists the environment variables checked, per platform, when no token is configured.
var tokenEnvVars = map[string]string{
	config.CommentPlatformGitHub: "GITHUB_TOKEN",
	config.CommentPlatformGitLab: "GITLAB_TOKEN",
	config.CommentPlatformGitea:  "GITEA_TOKEN",
	config.CommentPlatformGogs:   "GOGS_TOKEN",
}

// Validate checks that everything needed to post a comment is configured. A missing token
// is filled in from the platform's environment variable (e.g. GITHUB_TOKEN).
func Validate(cfg *config.CommentConfig) error {
	p := &cfg.Platform
	p.Type = strings.ToLower(p.Type)
	envVar, ok := tokenEnvVars[p.Type]
	if !ok {
		return fmt.Errorf("comment platform must be one of %s", strings.Join(config.CommentPlatforms, "|"))
	}
	if p.Token == "" {
		p.Token = os.Getenv(envVar)
	}
	if p.Token == "" {
		return fmt.Errorf("a %s token is required to post comments (use --comment-token or %s)", p.Type, envVar)
	}
	if p.Repository == "" {
		return errors.New("a repository is required to post comments (use --comment-repository)")
	}
	if p.Type != config.CommentPlatformGitLab {
		if _, _, err := splitRepository(p.Repository); err != nil {
			return err
		}
	}
	if p.PullRequestID <= 0 {
		return errors.New("a pull/merge request number is required to post comments (use --comment-pr)")
	}
	if p.Type == config.CommentPlatformGogs && p.BaseURL == "" {
		return errors.New("a base URL is required for gogs (use --comment-base-url)")
	}
	return nil
}

// Post formats results and posts them as a comment. failed is the overall outcome of the
// check. When UpdateExisting is set, an earlier go-covercheck comment is edited in place
// instead of adding a new one.
func Post(ctx context.Context, results compute.Results, failed bool, cfg *config.CommentConfig) error {
	if err := Validate(cfg); err != nil {
		return err
	}
	poster, err := NewPoster(&cfg.Platform)
	if err != nil {
		return err
	}
	body := FormatMarkdown(results, failed, cfg.Platform.IncludeColors)
	return upsert(ctx, poster, body, cfg.Platform.UpdateExisting)
}

func upsert(ctx context.Context, poster Poster, body string, updateExisting bool) error {
	if !updateExisting {
		return poster.CreateComment(ctx, body)
	}
	comments, err := poster.ListComments(ctx)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}
	for _, c := range comments {
		if strings.Contains(c.Body, Marker) {
			return poster.UpdateComment(ctx, c.ID, body)
		}
	}
	return poster.CreateComment(ctx, body)
}

// splitRepository splits "owner/repo" into its parts.
func splitRepository(repository string) (string, string, error) {
	owner, repo, ok := strings.Cut(repository, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", "", fmt.Errorf("repository %q must be in the form owner/repo", repository)
	}
	return owner, repo, nil
}
