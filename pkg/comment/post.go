package comment

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

const (
	// httpTimeout bounds each API request.
	httpTimeout = 30 * time.Second
	// listPageSize is the page size requested when listing comments.
	listPageSize = 100
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
// is filled in from the platform's environment variable (e.g. GITHUB_TOKEN). The platform
// type is expected in lower case, as config.Validate leaves it.
func Validate(cfg *config.CommentConfig) error {
	p := &cfg.Platform
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
	return upsert(ctx, poster, cfg.Platform.Type, cfg.Platform.Author, body, cfg.Platform.UpdateExisting, os.Stderr)
}

// upsert creates a comment, or with updateExisting edits the newest comment that carries
// Marker and was written by the authenticated user. Comments by anyone else (for example a
// person quoting the marker) are never edited. A failed edit is reported on warn and does
// not create a comment, so a persistent failure cannot add a comment on every run; only a
// deleted comment falls back to creating a new one.
func upsert(ctx context.Context, poster Poster, platform, author, body string, updateExisting bool,
	warn io.Writer) error {
	if !updateExisting {
		return poster.CreateComment(ctx, body)
	}
	user, err := resolveAuthor(ctx, poster, author)
	if err != nil {
		notef(warn, "warning: %s: cannot identify the token's user (%v); adding a new comment instead of updating"+
			" (set --comment-author to avoid this)",
			platform, err)
		return poster.CreateComment(ctx, body)
	}
	comments, err := poster.ListComments(ctx)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}
	target := newestOwnMarked(comments, user)
	if target == nil {
		return poster.CreateComment(ctx, body)
	}
	err = poster.UpdateComment(ctx, target.ID, body)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrNotFound):
		notef(warn, "note: %s: previous coverage comment %d no longer exists; adding a new comment",
			platform, target.ID)
		return poster.CreateComment(ctx, body)
	default:
		notef(warn, "warning: %s: failed to update coverage comment %d, not adding a new one: %v",
			platform, target.ID, err)
		return nil
	}
}

// resolveAuthor returns author when configured, otherwise the authenticated user's name.
func resolveAuthor(ctx context.Context, poster Poster, author string) (string, error) {
	if author != "" {
		return author, nil
	}
	user, err := poster.CurrentUser(ctx)
	if err == nil && user == "" {
		err = errors.New("no username returned")
	}
	return user, err
}

// newestOwnMarked returns the newest comment written by user that carries Marker, or nil.
// IDs increase over time on every platform, so the highest is the newest; list order differs
// between platforms (GitLab returns newest first).
func newestOwnMarked(comments []Comment, user string) *Comment {
	var target *Comment
	for i := range comments {
		c := &comments[i]
		if c.Author == user && strings.Contains(c.Body, Marker) && (target == nil || c.ID > target.ID) {
			target = c
		}
	}
	return target
}

// notef writes a one-line message to w. Failing to write a diagnostic is not actionable.
func notef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format+"\n", args...)
}

// splitRepository splits "owner/repo" into its parts.
func splitRepository(repository string) (string, string, error) {
	owner, repo, ok := strings.Cut(repository, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", "", fmt.Errorf("repository %q must be in the form owner/repo", repository)
	}
	return owner, repo, nil
}
