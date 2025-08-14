package test

import (
	"bytes"
	"os"
	"testing"
)

// CreateFile crates a file at the given path with content.
// The file will be automatically cleanedett up after the test finishes.
func CreateFile(t *testing.T, path string, content string) string {
	t.Helper()
	_ = os.WriteFile(path, []byte(content), 0600) //nolint:mnd

	t.Cleanup(func() {
		_ = os.RemoveAll(t.TempDir())
	})

	return path
}

// CreateTempFile creates a temporary file with the given filename and content.
// The file will be automatically cleaned up after the test finishes.
func CreateTempFile(t *testing.T, filename string, content string) string {
	t.Helper()
	path := t.TempDir() + "/" + filename
	return CreateFile(t, path, content)
}

// CreateTempCoverageFile creates a temporary 'coverage.out' file with the given content.
// This is used to test the coverage functionality.
func CreateTempCoverageFile(t *testing.T, content string) string {
	t.Helper()
	return CreateTempFile(t, "coverage.out", content)
}

// CreateTempConfigFile creates a temporary 'config.yaml' file with the given content.
// This is used to test the configuration functionality.
func CreateTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	return CreateTempFile(t, "config.yaml", content)
}

// CreateTempHistoryFile creates a temporary '.go-covercheck.history.json' file with the given content.
// This is used to test the history functionality.
func CreateTempHistoryFile(t *testing.T, content string) string {
	t.Helper()
	return CreateTempFile(t, ".go-covercheck.history.json", content)
}

// RepipeStdOutAndErrForTest temporarily repipes stdout and stderr to capture output during a test.
// It executes the provided function and returns the captured stdout and stderr as strings.
// This is useful for testing functions that print to stdout or stderr without affecting the actual output.
func RepipeStdOutAndErrForTest(fn func()) (string, string) {
	// repipe all stdout and stderr
	oOut := os.Stdout
	rOut, wOut, _ := os.Pipe()

	oErr := os.Stderr
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	// restore standard stdout and stderr
	_ = wOut.Close()
	_ = wErr.Close()

	os.Stdout = oOut
	os.Stderr = oErr

	// return data from repiped stdout, stderr
	var stdOut bytes.Buffer
	_, _ = stdOut.ReadFrom(rOut)

	var stdErr bytes.Buffer
	_, _ = stdErr.ReadFrom(rErr)

	return stdOut.String(), stdErr.String()
}
