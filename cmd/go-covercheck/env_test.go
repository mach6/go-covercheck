package main

import (
	"os"
	"testing"
)

// TestMain starts every test from the same environment. Coverage depends on variables that differ between a
// laptop and a CI runner (color, terminal width, CI detection, tokens), so they are cleared here and a test
// that needs one sets it with t.Setenv. Without this, the same tests cover different lines in different places.
func TestMain(m *testing.M) {
	for _, name := range []string{
		"NO_COLOR", "FORCE_COLOR", "COLUMNS", "NO_PAGER", "TERM",
		"CI", "ACT", "GITHUB_ACTIONS", "GITEA_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS", "DRONE",
		"BITBUCKET_BUILD_NUMBER", "BUILDKITE", "APPVEYOR", "TEAMCITY_VERSION", "JENKINS_URL", "TF_BUILD",
		"HEROKU_TEST_RUN_ID", "CIRRUS_CI", "CODEBUILD_BUILD_ID", "GOCD_SERVER_HOST", "SEMAPHORE", "WERCKER",
		"HUDSON_URL", "GITHUB_TOKEN", "GITLAB_TOKEN", "GITEA_TOKEN",
	} {
		_ = os.Unsetenv(name)
	}
	os.Exit(m.Run())
}
