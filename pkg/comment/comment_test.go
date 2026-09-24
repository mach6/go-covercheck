package comment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeServer stands in for the GitHub, GitLab, Gitea, and Gogs comment APIs, which all
// use {"id": ..., "body": ...} comment objects.
type fakeServer struct {
	*httptest.Server

	mu       sync.Mutex
	comments []Comment
	auth     []string
	requests []string
}

var commentIDPath = regexp.MustCompile(`/(\d+)$`)

func newFakeServer(t *testing.T, existing ...Comment) *fakeServer {
	t.Helper()
	f := &fakeServer{comments: existing}
	f.Server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeServer) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auth = append(f.auth, r.Header.Get("Authorization")+r.Header.Get("Private-Token"))
	f.requests = append(f.requests, r.Method+" "+r.URL.EscapedPath())
	w.Header().Set("Content-Type", "application/json")

	var in struct {
		Body string `json:"body"`
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, toJSON(f.comments))
	case http.MethodPost:
		_ = json.NewDecoder(r.Body).Decode(&in)
		c := Comment{ID: int64(len(f.comments) + 1), Body: in.Body}
		f.comments = append(f.comments, c)
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, jsonComment(c))
	case http.MethodPatch, http.MethodPut:
		_ = json.NewDecoder(r.Body).Decode(&in)
		m := commentIDPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		id, _ := strconv.ParseInt(m[1], 10, 64)
		for i := range f.comments {
			if f.comments[i].ID == id {
				f.comments[i].Body = in.Body
				writeJSON(w, jsonComment(f.comments[i]))
				return
			}
		}
		http.NotFound(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

type jsonComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

func writeJSON(w http.ResponseWriter, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(data)
}

func toJSON(comments []Comment) []jsonComment {
	out := make([]jsonComment, 0, len(comments))
	for _, c := range comments {
		out = append(out, jsonComment(c))
	}
	return out
}

func (f *fakeServer) snapshot() ([]Comment, []string, []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Comment(nil), f.comments...), append([]string(nil), f.auth...),
		append([]string(nil), f.requests...)
}

func testResults(failed bool) compute.Results {
	pct := 80.0
	if failed {
		pct = 40.0
	}
	by := compute.By{
		StatementPercentage: pct, BlockPercentage: pct, LinePercentage: pct,
		StatementThreshold: 70, BlockThreshold: 50, LineThreshold: 50,
		Failed: failed,
	}
	return compute.Results{
		ByFile:    []compute.ByFile{{By: by, File: "pkg/a/a.go"}},
		ByPackage: []compute.ByPackage{{By: by, Package: "pkg/a"}},
		ByTotal: compute.Totals{
			Statements: compute.TotalStatements{Coverage: "8/10", Percentage: pct, Threshold: 70, Failed: failed},
			Blocks:     compute.TotalBlocks{Coverage: "4/5", Percentage: pct, Threshold: 50, Failed: failed},
			Lines:      compute.TotalLines{Coverage: "16/20", Percentage: pct, Threshold: 50, Failed: failed},
		},
	}
}

func TestPostToEachPlatform(t *testing.T) {
	tests := []struct {
		platform   string
		repository string
		auth       string
		listPath   string
		updatePath string
	}{
		{
			config.CommentPlatformGitHub, "owner/repo", "Bearer tok",
			"/api/v3/repos/owner/repo/issues/7/comments",
			"PATCH /api/v3/repos/owner/repo/issues/comments/2",
		},
		{
			config.CommentPlatformGitLab, "group/sub/project", "tok",
			"/api/v4/projects/group%2Fsub%2Fproject/merge_requests/7/notes",
			"PUT /api/v4/projects/group%2Fsub%2Fproject/merge_requests/7/notes/2",
		},
		{
			config.CommentPlatformGitea, "owner/repo", "token tok",
			"/api/v1/repos/owner/repo/issues/7/comments",
			"PATCH /api/v1/repos/owner/repo/issues/comments/2",
		},
		{
			config.CommentPlatformGogs, "owner/repo", "token tok",
			"/api/v1/repos/owner/repo/issues/7/comments",
			"PATCH /api/v1/repos/owner/repo/issues/7/comments/2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			other := Comment{ID: 1, Body: "LGTM"}
			srv := newFakeServer(t, other)
			cfg := &config.CommentConfig{Enabled: true, Platform: config.PlatformConfig{
				Type: tt.platform, BaseURL: srv.URL, Token: "tok",
				Repository: tt.repository, PullRequestID: 7, IncludeColors: true,
			}}

			// first run adds a comment
			require.NoError(t, Post(context.Background(), testResults(true), true, cfg))
			comments, auth, _ := srv.snapshot()
			require.Len(t, comments, 2)
			assert.Equal(t, other, comments[0])
			assert.Contains(t, comments[1].Body, Marker)
			assert.Contains(t, comments[1].Body, "Coverage check failed")
			for _, a := range auth {
				assert.Equal(t, tt.auth, a)
			}

			// second run with update edits the same comment
			cfg.Platform.UpdateExisting = true
			require.NoError(t, Post(context.Background(), testResults(false), false, cfg))
			comments, _, requests := srv.snapshot()
			require.Len(t, comments, 2)
			assert.Equal(t, other, comments[0])
			assert.Contains(t, comments[1].Body, "Coverage check passed")
			assert.Equal(t, []string{"POST " + tt.listPath, "GET " + tt.listPath, tt.updatePath}, requests)

			// without update, a new comment is added
			cfg.Platform.UpdateExisting = false
			require.NoError(t, Post(context.Background(), testResults(false), false, cfg))
			comments, _, _ = srv.snapshot()
			assert.Len(t, comments, 3)
		})
	}
}

func TestPostReportsAPIErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"forbidden"}`, http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	for _, platform := range config.CommentPlatforms {
		t.Run(platform, func(t *testing.T) {
			cfg := &config.CommentConfig{Platform: config.PlatformConfig{
				Type: platform, BaseURL: srv.URL, Token: "tok", Repository: "o/r", PullRequestID: 1,
				UpdateExisting: true,
			}}
			assert.Error(t, Post(context.Background(), testResults(false), false, cfg))
		})
	}
}

func TestValidate(t *testing.T) {
	valid := func() config.CommentConfig {
		return config.CommentConfig{Platform: config.PlatformConfig{
			Type: "GitHub", Token: "tok", Repository: "o/r", PullRequestID: 1,
		}}
	}
	tests := []struct {
		name    string
		mutate  func(*config.CommentConfig)
		wantErr string
	}{
		{"valid", func(*config.CommentConfig) {}, ""},
		{"unknown platform", func(c *config.CommentConfig) { c.Platform.Type = "bitbucket" }, "must be one of"},
		{"missing platform", func(c *config.CommentConfig) { c.Platform.Type = "" }, "must be one of"},
		{"missing token", func(c *config.CommentConfig) { c.Platform.Token = "" }, "GITHUB_TOKEN"},
		{"missing repository", func(c *config.CommentConfig) { c.Platform.Repository = "" }, "repository is required"},
		{"bad repository", func(c *config.CommentConfig) { c.Platform.Repository = "o/r/x" }, "owner/repo"},
		{"missing pr", func(c *config.CommentConfig) { c.Platform.PullRequestID = 0 }, "request number"},
		{"gitlab nested group", func(c *config.CommentConfig) {
			c.Platform.Type = "gitlab"
			c.Platform.Repository = "a/b/c"
		}, ""},
		{"gogs needs base url", func(c *config.CommentConfig) { c.Platform.Type = "gogs" }, "base URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", "")
			cfg := valid()
			tt.mutate(&cfg)
			err := Validate(&cfg)
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTokenFromEnv(t *testing.T) {
	t.Setenv("GITLAB_TOKEN", "from-env")
	cfg := config.CommentConfig{Platform: config.PlatformConfig{
		Type: "gitlab", Repository: "g/p", PullRequestID: 1,
	}}
	require.NoError(t, Validate(&cfg))
	assert.Equal(t, "from-env", cfg.Platform.Token)

	cfg.Platform.Token = "explicit"
	require.NoError(t, Validate(&cfg))
	assert.Equal(t, "explicit", cfg.Platform.Token)
}

func TestNewPosterErrors(t *testing.T) {
	_, err := NewPoster(&config.PlatformConfig{Type: "bitbucket"})
	require.ErrorContains(t, err, "unsupported")

	_, err = NewPoster(&config.PlatformConfig{Type: "github", Repository: "bad"})
	require.ErrorContains(t, err, "owner/repo")

	_, err = NewPoster(&config.PlatformConfig{Type: "gogs", Repository: "o/r", BaseURL: "not a url"})
	require.ErrorContains(t, err, "base URL")
}

func TestFormatMarkdown(t *testing.T) {
	t.Run("passed", func(t *testing.T) {
		md := FormatMarkdown(testResults(false), false, true)
		assert.True(t, strings.HasPrefix(md, Marker))
		assert.Contains(t, md, "🟢 **Coverage check passed**")
		assert.Contains(t, md, "| Lines | 16/20 | 80.0% | 50.0% | 🟢 |")
		assert.NotContains(t, md, "Below Threshold")
	})

	t.Run("failed", func(t *testing.T) {
		md := FormatMarkdown(testResults(true), true, true)
		assert.Contains(t, md, "🔴 **Coverage check failed**")
		assert.Contains(t, md, "| Statements | 8/10 | 40.0% | 70.0% | 🔴 |")
		assert.Contains(t, md, "### Files Below Threshold")
		assert.Contains(t, md, "| `pkg/a/a.go` | 🔴 40.0% / 70.0% | 🔴 40.0% / 50.0% | 🔴 40.0% / 50.0% |")
		assert.Contains(t, md, "### Packages Below Threshold")
	})

	t.Run("no colors", func(t *testing.T) {
		md := FormatMarkdown(testResults(true), true, false)
		assert.Contains(t, md, "FAIL **Coverage check failed**")
		assert.NotContains(t, md, "🔴")
		assert.NotContains(t, md, "🟢")
	})

	t.Run("caps rows", func(t *testing.T) {
		results := testResults(true)
		base := results.ByFile[0]
		results.ByFile = nil
		for i := range maxRows + 5 {
			f := base
			f.File = "f" + strconv.Itoa(i) + ".go"
			results.ByFile = append(results.ByFile, f)
		}
		md := FormatMarkdown(results, true, true)
		assert.Contains(t, md, "…and 5 more")
		assert.Equal(t, maxRows, strings.Count(md, "| `f"))
	})
}

func TestGitHubUpdateFindsCommentOnLaterPage(t *testing.T) {
	var patched string
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("page") == "":
			w.Header().Set("Link", `<`+srv.URL+r.URL.Path+`?page=2>; rel="next"`)
			_, _ = w.Write([]byte(`[{"id":1,"body":"first page"}]`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":42,"body":"` + Marker + `"}]`))
		case r.Method == http.MethodPatch:
			patched = r.URL.Path
			_, _ = w.Write([]byte(`{"id":42}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
	}))
	t.Cleanup(srv.Close)

	p, err := NewGitHubPoster(srv.URL, "tok", "o/r", 1)
	require.NoError(t, err)
	require.NoError(t, upsert(context.Background(), p, "new", true))
	assert.Equal(t, "/api/v3/repos/o/r/issues/comments/42", patched)
}
