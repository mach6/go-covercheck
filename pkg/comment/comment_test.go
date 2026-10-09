package comment

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
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
	// allowed, when set, restricts the server to these exact "METHOD escaped-path" requests;
	// anything else is answered with 418 so a wrong route fails the test.
	allowed map[string]bool
	// patchStatus, when non-zero, is returned for every PATCH/PUT instead of editing.
	patchStatus int
	// userStatus, when non-zero, is returned for GET /user instead of the user.
	userStatus int
	// userAs, when set, replaces fakeUser as the account GET /user returns.
	userAs string
}

// fakeUser is the account the fake server authenticates every request as.
const fakeUser = "bot"

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
	if f.allowed != nil && !f.allowed[r.Method+" "+r.URL.EscapedPath()] {
		w.WriteHeader(http.StatusTeapot)
		return
	}
	if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/user") {
		if f.userStatus != 0 {
			http.Error(w, `{"message":"Resource not accessible by integration"}`, f.userStatus)
			return
		}
		name := fakeUser
		if f.userAs != "" {
			name = f.userAs
		}
		writeJSON(w, jsonUser{ID: 1, Login: name, Username: name})
		return
	}
	f.handleComments(w, r)
}

func (f *fakeServer) handleComments(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Body string `json:"body"`
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, toJSON(f.comments))
	case http.MethodPost:
		_ = json.NewDecoder(r.Body).Decode(&in)
		c := Comment{ID: int64(len(f.comments) + 1), Body: in.Body, Author: fakeUser}
		f.comments = append(f.comments, c)
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, newJSONComment(c))
	case http.MethodPatch, http.MethodPut:
		if f.patchStatus != 0 {
			w.WriteHeader(f.patchStatus)
			return
		}
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
				writeJSON(w, newJSONComment(f.comments[i]))
				return
			}
		}
		http.NotFound(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// jsonUser carries the fields each platform uses to name a user.
type jsonUser struct {
	ID       int64  `json:"id"`
	Login    string `json:"login"`
	Username string `json:"username"`
}

// jsonComment serves the comment author as "user" (GitHub, Gitea, Gogs) and "author" (GitLab).
type jsonComment struct {
	ID     int64     `json:"id"`
	Body   string    `json:"body"`
	User   *jsonUser `json:"user"`
	Author *jsonUser `json:"author"`
}

func newJSONComment(c Comment) jsonComment {
	u := &jsonUser{Login: c.Author, Username: c.Author}
	return jsonComment{ID: c.ID, Body: c.Body, User: u, Author: u}
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
		out = append(out, newJSONComment(c))
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
		userPath   string
		updatePath string
	}{
		{
			config.CommentPlatformGitHub, "owner/repo", "Bearer tok",
			"/api/v3/repos/owner/repo/issues/7/comments", "/api/v3/user",
			"PATCH /api/v3/repos/owner/repo/issues/comments/2",
		},
		{
			config.CommentPlatformGitLab, "group/sub/project", "tok",
			"/api/v4/projects/group%2Fsub%2Fproject/merge_requests/7/notes", "/api/v4/user",
			"PUT /api/v4/projects/group%2Fsub%2Fproject/merge_requests/7/notes/2",
		},
		{
			config.CommentPlatformGitea, "owner/repo", "token tok",
			"/api/v1/repos/owner/repo/issues/7/comments", "/api/v1/user",
			"PATCH /api/v1/repos/owner/repo/issues/comments/2",
		},
		{
			config.CommentPlatformGogs, "owner/repo", "token tok",
			"/api/v1/repos/owner/repo/issues/7/comments", "/api/v1/user",
			"PATCH /api/v1/repos/owner/repo/issues/7/comments/2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			other := Comment{ID: 1, Body: "LGTM", Author: "human"}
			srv := newFakeServer(t, other)
			srv.allowed = map[string]bool{
				"POST " + tt.listPath: true, "GET " + tt.listPath: true,
				"GET " + tt.userPath: true, tt.updatePath: true,
			}
			cfg := &config.CommentConfig{Enabled: true, Platform: config.PlatformConfig{
				Type: tt.platform, BaseURL: srv.URL, Token: "tok",
				Repository: tt.repository, PullRequestID: 7, IncludeColors: true,
			}}

			// first run adds a comment
			require.NoError(t, Post(context.Background(), testResults(true), true, cfg))
			comments, auth, _ := srv.snapshot()
			require.Len(t, comments, 2)
			require.Equal(t, other, comments[0])
			require.Contains(t, comments[1].Body, Marker)
			require.Contains(t, comments[1].Body, "Coverage check failed")
			for _, a := range auth {
				require.Equal(t, tt.auth, a)
			}

			// second run with update edits the same comment
			cfg.Platform.UpdateExisting = true
			require.NoError(t, Post(context.Background(), testResults(false), false, cfg))
			comments, _, requests := srv.snapshot()
			require.Len(t, comments, 2)
			require.Equal(t, other, comments[0])
			require.Contains(t, comments[1].Body, "Coverage check passed")
			require.Equal(t,
				[]string{"POST " + tt.listPath, "GET " + tt.userPath, "GET " + tt.listPath, tt.updatePath}, requests)

			// without update, a new comment is added
			cfg.Platform.UpdateExisting = false
			require.NoError(t, Post(context.Background(), testResults(false), false, cfg))
			comments, _, _ = srv.snapshot()
			require.Len(t, comments, 3)
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
			require.Error(t, Post(context.Background(), testResults(false), false, cfg))
		})
	}
}

func TestValidate(t *testing.T) {
	valid := func() config.CommentConfig {
		return config.CommentConfig{Platform: config.PlatformConfig{
			Type: "github", Token: "tok", Repository: "o/r", PullRequestID: 1,
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
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
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
	require.Equal(t, "from-env", cfg.Platform.Token)

	cfg.Platform.Token = "explicit"
	require.NoError(t, Validate(&cfg))
	require.Equal(t, "explicit", cfg.Platform.Token)
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
		require.True(t, strings.HasPrefix(md, Marker))
		require.Contains(t, md, "🟢 **Coverage check passed**")
		require.Contains(t, md, "| Lines | 16/20 | 80.0% | 50.0% | 🟢 |")
		require.NotContains(t, md, "Below Threshold")
	})

	t.Run("failed", func(t *testing.T) {
		md := FormatMarkdown(testResults(true), true, true)
		require.Contains(t, md, "🔴 **Coverage check failed**")
		require.Contains(t, md, "| Statements | 8/10 | 40.0% | 70.0% | 🔴 |")
		require.Contains(t, md, "### Files Below Threshold")
		require.Contains(t, md, "| `pkg/a/a.go` | 🔴 40.0% / 70.0% | 🔴 40.0% / 50.0% | 🔴 40.0% / 50.0% |")
		require.Contains(t, md, "### Packages Below Threshold")
	})

	t.Run("no colors", func(t *testing.T) {
		md := FormatMarkdown(testResults(true), true, false)
		require.Contains(t, md, "FAIL **Coverage check failed**")
		require.NotContains(t, md, "🔴")
		require.NotContains(t, md, "🟢")
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
		require.Contains(t, md, "…and 5 more")
		require.Equal(t, maxRows, strings.Count(md, "| `f"))
	})
}

func TestGitHubUpdateFindsCommentOnLaterPage(t *testing.T) {
	var patched string
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v3/user":
			_, _ = w.Write([]byte(`{"login":"bot"}`))
		case r.Method == http.MethodGet && r.URL.Query().Get("page") == "":
			w.Header().Set("Link", `<`+srv.URL+r.URL.Path+`?page=2>; rel="next"`)
			_, _ = w.Write([]byte(`[{"id":1,"body":"first page","user":{"login":"human"}}]`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":42,"body":"` + Marker + `","user":{"login":"bot"}}]`))
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
	require.NoError(t, upsert(context.Background(), p, "github", "", "new", true, io.Discard))
	require.Equal(t, "/api/v3/repos/o/r/issues/comments/42", patched)
}

func TestGiteaListStopsWhenServerIgnoresPaging(t *testing.T) {
	const total = 150
	all := make([]jsonComment, total)
	for i := range all {
		all[i] = jsonComment{ID: int64(i + 1), Body: "c"}
	}
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		writeJSON(w, all) // ignores page/limit, like Gitea's issue comments endpoint
	}))
	t.Cleanup(srv.Close)

	p, err := NewGiteaPoster(srv.URL, "tok", "o/r", 1)
	require.NoError(t, err)
	comments, err := p.ListComments(context.Background())
	require.NoError(t, err)
	require.Len(t, comments, total)
	require.Equal(t, 2, requests)
}

// newGogsFake returns a Gogs poster backed by a fake server holding existing comments.
func newGogsFake(t *testing.T, existing ...Comment) (*fakeServer, *GogsPoster) {
	t.Helper()
	srv := newFakeServer(t, existing...)
	p, err := NewGogsPoster(srv.URL, "tok", "o/r", 1)
	require.NoError(t, err)
	return srv, p
}

func TestUpdateNotFoundFallsBackToCreate(t *testing.T) {
	// the marked comment is listed, but editing it reports 404 (deleted in the meantime)
	srv, p := newGogsFake(t, Comment{ID: 1, Body: Marker + " old", Author: fakeUser})
	srv.patchStatus = http.StatusNotFound

	var warn bytes.Buffer
	require.NoError(t, upsert(context.Background(), p, "gogs", "", "report", true, &warn))

	comments, _, _ := srv.snapshot()
	require.Len(t, comments, 2)
	require.Equal(t, "report", comments[1].Body)
	require.Contains(t, warn.String(), "note: gogs: previous coverage comment 1 no longer exists")
}

func TestUpdateGoneFallsBackToCreate(t *testing.T) {
	srv, p := newGogsFake(t, Comment{ID: 1, Body: Marker, Author: fakeUser})
	srv.patchStatus = http.StatusGone

	var warn bytes.Buffer
	require.NoError(t, upsert(context.Background(), p, "gogs", "", "report", true, &warn))
	comments, _, _ := srv.snapshot()
	require.Len(t, comments, 2)
	require.Contains(t, warn.String(), "no longer exists")
}

func TestUpdateOtherFailureDoesNotCreate(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusUnauthorized, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			srv, p := newGogsFake(t, Comment{ID: 1, Body: Marker + " old", Author: fakeUser})
			srv.patchStatus = status

			var warn bytes.Buffer
			require.NoError(t, upsert(context.Background(), p, "gogs", "", "report", true, &warn))

			comments, _, _ := srv.snapshot()
			require.Len(t, comments, 1, "no duplicate comment may be added")
			require.Equal(t, Marker+" old", comments[0].Body)
			require.Contains(t, warn.String(), "warning: gogs: failed to update coverage comment 1")
			require.Contains(t, warn.String(), strconv.Itoa(status))
		})
	}
}

func TestUpdateIgnoresCommentsByOthers(t *testing.T) {
	// a person quoting the marker must not be edited; a new comment is added instead
	srv, p := newGogsFake(t, Comment{ID: 1, Body: "> " + Marker + " quoted", Author: "human"})

	var warn bytes.Buffer
	require.NoError(t, upsert(context.Background(), p, "gogs", "", "report", true, &warn))

	comments, _, requests := srv.snapshot()
	require.Len(t, comments, 2)
	require.Equal(t, "> "+Marker+" quoted", comments[0].Body)
	require.Equal(t, "report", comments[1].Body)
	require.NotContains(t, strings.Join(requests, "\n"), "PATCH")
	require.Empty(t, warn.String())
}

func TestUpdateEditsNewestOwnComment(t *testing.T) {
	// listed newest first (as GitLab does) with a human quote in between
	srv, p := newGogsFake(t,
		Comment{ID: 9, Body: Marker + " newest", Author: fakeUser},
		Comment{ID: 7, Body: "> " + Marker, Author: "human"},
		Comment{ID: 3, Body: Marker + " oldest", Author: fakeUser},
	)
	srv.allowed = map[string]bool{
		"GET /api/v1/user":                            true,
		"GET /api/v1/repos/o/r/issues/1/comments":     true,
		"PATCH /api/v1/repos/o/r/issues/1/comments/9": true,
	}

	require.NoError(t, upsert(context.Background(), p, "gogs", "", "report", true, io.Discard))

	comments, _, _ := srv.snapshot()
	require.Len(t, comments, 3)
	require.Equal(t, "report", comments[0].Body)
	require.Equal(t, Marker+" oldest", comments[2].Body)
}

func TestUpdateUserLookupFailureCreates(t *testing.T) {
	srv, p := newGogsFake(t, Comment{ID: 1, Body: Marker, Author: fakeUser})
	handler := srv.Config.Handler
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/user") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})

	var warn bytes.Buffer
	require.NoError(t, upsert(context.Background(), p, "gogs", "", "report", true, &warn))

	comments, _, requests := srv.snapshot()
	require.Len(t, comments, 2, "falls back to create without editing anything")
	require.Equal(t, Marker, comments[0].Body)
	require.NotContains(t, strings.Join(requests, "\n"), "PATCH")
	require.Contains(t, warn.String(), "warning: gogs: cannot identify the token's user")
}

// newGitHubFake returns a GitHub poster backed by a fake server whose GET /user answers
// userStatus (when non-zero), holding existing comments.
func newGitHubFake(t *testing.T, userStatus int, existing ...Comment) (*fakeServer, *GitHubPoster) {
	t.Helper()
	srv := newFakeServer(t, existing...)
	srv.userStatus = userStatus
	p, err := NewGitHubPoster(srv.URL, "tok", "o/r", 1)
	require.NoError(t, err)
	return srv, p
}

func TestGitHubActionsTokenUpdatesBotComment(t *testing.T) {
	// the Actions GITHUB_TOKEN cannot call GET /user; its comments are by github-actions[bot]
	t.Setenv("GITHUB_ACTIONS", "true")
	srv, p := newGitHubFake(t, http.StatusForbidden,
		Comment{ID: 1, Body: "> " + Marker + " quoted", Author: "human"},
		Comment{ID: 2, Body: Marker + " old", Author: "github-actions[bot]"},
	)

	var warn bytes.Buffer
	require.NoError(t, upsert(context.Background(), p, "github", "", "report", true, &warn))

	comments, _, requests := srv.snapshot()
	require.Len(t, comments, 2, "the bot comment is edited, not duplicated")
	require.Equal(t, "report", comments[1].Body)
	require.Equal(t, "PATCH /api/v3/repos/o/r/issues/comments/2", requests[len(requests)-1])
	require.Empty(t, warn.String())
}

func TestGitHubForbiddenUserOutsideActionsWarnsAndCreates(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	srv, p := newGitHubFake(t, http.StatusForbidden,
		Comment{ID: 2, Body: Marker + " old", Author: "github-actions[bot]"})

	var warn bytes.Buffer
	require.NoError(t, upsert(context.Background(), p, "github", "", "report", true, &warn))

	comments, _, requests := srv.snapshot()
	require.Len(t, comments, 2)
	require.Equal(t, Marker+" old", comments[0].Body)
	require.NotContains(t, strings.Join(requests, "\n"), "PATCH")
	require.Contains(t, warn.String(), "cannot identify the token's user")
	require.Contains(t, warn.String(), "--comment-author")
}

func TestGitHubActionsOtherUserErrorStillWarns(t *testing.T) {
	// only a 403 selects the Actions identity
	t.Setenv("GITHUB_ACTIONS", "true")
	srv, p := newGitHubFake(t, http.StatusInternalServerError,
		Comment{ID: 2, Body: Marker + " old", Author: "github-actions[bot]"})

	var warn bytes.Buffer
	require.NoError(t, upsert(context.Background(), p, "github", "", "report", true, &warn))

	comments, _, _ := srv.snapshot()
	require.Len(t, comments, 2)
	require.Contains(t, warn.String(), "cannot identify the token's user")
}

func TestAuthorOverrideSkipsUserLookup(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "")
	srv, p := newGitHubFake(t, http.StatusForbidden,
		Comment{ID: 1, Body: Marker + " mine", Author: "ci-bot"},
		Comment{ID: 2, Body: Marker + " yours", Author: fakeUser},
	)

	var warn bytes.Buffer
	require.NoError(t, upsert(context.Background(), p, "github", "ci-bot", "report", true, &warn))

	comments, _, requests := srv.snapshot()
	require.Len(t, comments, 2)
	require.Equal(t, "report", comments[0].Body)
	require.Equal(t, Marker+" yours", comments[1].Body, "other authors are left alone")
	require.Equal(t, []string{
		"GET /api/v3/repos/o/r/issues/1/comments",
		"PATCH /api/v3/repos/o/r/issues/comments/1",
	}, requests, "GET /user must not be called")
	require.Empty(t, warn.String())
}

func TestGiteaActionsUserIsMatched(t *testing.T) {
	// a Gitea Actions task token answers GET /user as the "gitea-actions" system user, which is
	// also the author of the comments it posts, so no special casing is needed.
	srv := newFakeServer(t, Comment{ID: 1, Body: Marker + " old", Author: "gitea-actions"})
	srv.userAs = "gitea-actions"
	p, err := NewGiteaPoster(srv.URL, "tok", "o/r", 1)
	require.NoError(t, err)

	require.NoError(t, upsert(context.Background(), p, "gitea", "", "report", true, io.Discard))

	comments, _, _ := srv.snapshot()
	require.Len(t, comments, 1)
	require.Equal(t, "report", comments[0].Body)
}

func TestUpdateNotFoundSentinelPerPlatform(t *testing.T) {
	srv := newFakeServer(t)
	for _, platform := range config.CommentPlatforms {
		t.Run(platform, func(t *testing.T) {
			p, err := NewPoster(&config.PlatformConfig{
				Type: platform, BaseURL: srv.URL, Token: "tok", Repository: "o/r", PullRequestID: 1,
			})
			require.NoError(t, err)
			// the fake answers 404 for a comment id it does not hold
			err = p.UpdateComment(context.Background(), 12345, "x")
			require.ErrorIs(t, err, ErrNotFound)
		})
	}
}

func TestUpdateForbiddenIsNotNotFound(t *testing.T) {
	srv := newFakeServer(t)
	srv.patchStatus = http.StatusForbidden
	for _, platform := range config.CommentPlatforms {
		t.Run(platform, func(t *testing.T) {
			p, err := NewPoster(&config.PlatformConfig{
				Type: platform, BaseURL: srv.URL, Token: "tok", Repository: "o/r", PullRequestID: 1,
			})
			require.NoError(t, err)
			err = p.UpdateComment(context.Background(), 1, "x")
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrNotFound)
		})
	}
}

func TestCurrentUserPerPlatform(t *testing.T) {
	srv := newFakeServer(t)
	for _, platform := range config.CommentPlatforms {
		t.Run(platform, func(t *testing.T) {
			p, err := NewPoster(&config.PlatformConfig{
				Type: platform, BaseURL: srv.URL, Token: "tok", Repository: "o/r", PullRequestID: 1,
			})
			require.NoError(t, err)
			user, err := p.CurrentUser(context.Background())
			require.NoError(t, err)
			require.Equal(t, fakeUser, user)
		})
	}
}

func TestListCommentsReportsAuthor(t *testing.T) {
	srv := newFakeServer(t, Comment{ID: 1, Body: "hi", Author: "human"})
	for _, platform := range config.CommentPlatforms {
		t.Run(platform, func(t *testing.T) {
			p, err := NewPoster(&config.PlatformConfig{
				Type: platform, BaseURL: srv.URL, Token: "tok", Repository: "o/r", PullRequestID: 1,
			})
			require.NoError(t, err)
			comments, err := p.ListComments(context.Background())
			require.NoError(t, err)
			require.Equal(t, []Comment{{ID: 1, Body: "hi", Author: "human"}}, comments)
		})
	}
}

func TestIsPublicGitHub(t *testing.T) {
	for _, u := range []string{"", "https://github.com", "https://api.github.com/", "HTTPS://API.GITHUB.COM"} {
		require.True(t, isPublicGitHub(u), u)
	}
	require.False(t, isPublicGitHub("https://github.example.com"))
}

func TestCodeCell(t *testing.T) {
	require.Equal(t, "`pkg/a.go`", codeCell("pkg/a.go"))
	require.Equal(t, "`pkg/a\\|b.go`", codeCell("pkg/a|b.go"))
	require.Equal(t, "`` pkg/a`b.go ``", codeCell("pkg/a`b.go"))
}
