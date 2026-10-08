package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/mach6/go-covercheck/pkg/comment"
	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/mach6/go-covercheck/pkg/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func commentArgs(t *testing.T, baseURL string, extra ...string) []string {
	t.Helper()
	args := make([]string, 0, 20+len(extra))
	args = append(args,
		"-w", "-s", "1", "-b", "1", "-n", "1",
		"--comment", "--comment-platform", "gogs", "--comment-base-url", baseURL,
		"--comment-token", "tok", "--comment-repository", "o/r", "--comment-pr", "3",
	)
	args = append(args, extra...)
	return append(args, test.CreateTempCoverageFile(t, test.TestCoverageOut))
}

func Test_run_CommentPosted(t *testing.T) {
	var (
		mu      sync.Mutex
		bodies  []string
		payload []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Body string `json:"body"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		mu.Lock()
		bodies = append(bodies, r.Method+" "+r.URL.Path)
		payload = append(payload, in.Body)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	t.Cleanup(srv.Close)

	cmd := setupTestCmd()
	cmd.SetArgs(commentArgs(t, srv.URL, "--comment-no-emoji"))
	_, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Empty(t, stdErr)
	require.Equal(t, []string{"POST /api/v1/repos/o/r/issues/3/comments"}, bodies)
	require.Len(t, payload, 1)
	require.Contains(t, payload[0], comment.Marker)
	require.Contains(t, payload[0], "PASS")
	require.NotContains(t, payload[0], "🟢")
}

func Test_run_CommentPostFailureIsAWarning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	cmd := setupTestCmd()
	cmd.SetArgs(commentArgs(t, srv.URL))
	_, stdErr, err := runCmdForTest(t, cmd)
	require.NoError(t, err)
	require.Contains(t, stdErr, "warning: failed to post coverage comment")
	require.Contains(t, stdErr, "401")
}

func Test_run_CommentMisconfiguredFailsEarly(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	cmd := setupTestCmd()
	cmd.SetArgs([]string{
		"--comment", "--comment-platform", "github", "--comment-repository", "o/r", "--comment-pr", "1",
		test.CreateTempCoverageFile(t, test.TestCoverageOut),
	})
	stdOut, _, err := runCmdForTest(t, cmd)
	require.ErrorContains(t, err, "GITHUB_TOKEN")
	require.Empty(t, stdOut, "no coverage output expected before configuration is validated")
}

func TestApplyConfigOverrides_CommentFlags(t *testing.T) {
	cmd := &cobra.Command{}
	initFlags(cmd)

	cfg := &config.Config{}
	cfg.ApplyDefaults()
	cfg.Comment.Platform.Repository = "from/config"

	for flag, value := range map[string]string{
		CommentFlag:         "true",
		CommentPlatformFlag: "gitlab",
		CommentBaseURLFlag:  "https://gitlab.example.com",
		CommentTokenFlag:    "tok",
		CommentPRFlag:       "9",
		CommentUpdateFlag:   "true",
		CommentAuthorFlag:   "ci-bot",
		CommentNoEmojiFlag:  "true",
	} {
		require.NoError(t, cmd.Flags().Set(flag, value))
	}
	applyConfigOverrides(cfg, cmd, false)

	require.Equal(t, config.CommentConfig{
		Enabled: true,
		Platform: config.PlatformConfig{
			Type:           "gitlab",
			BaseURL:        "https://gitlab.example.com",
			Token:          "tok",
			Repository:     "from/config",
			PullRequestID:  9,
			IncludeColors:  false,
			Author:         "ci-bot",
			UpdateExisting: true,
		},
	}, cfg.Comment)
}

func Test_run_CommentPostedEvenWhenHistoryFails(t *testing.T) {
	var posted bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		posted = true
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	t.Cleanup(srv.Close)

	cmd := setupTestCmd()
	cmd.SetArgs(commentArgs(t, srv.URL,
		"--history-file", test.CreateTempHistoryFile(t, test.TestCoverageHistory),
		"--compare-history", "no-such-ref"))
	_, _, err := runCmdForTest(t, cmd)
	require.Error(t, err)
	require.True(t, posted)
}
