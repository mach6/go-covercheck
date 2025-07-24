package test

import (
	"bytes"
	"os"
	"testing"
)

func CreateTempFile(t *testing.T, filename string, content string) string {
	t.Helper()
	path := t.TempDir() + "/" + filename
	_ = os.WriteFile(path, []byte(content), 0600)

	t.Cleanup(func() {
		_ = os.RemoveAll(t.TempDir())
	})

	return path
}

func CreateTempCoverageFile(t *testing.T, content string) string {
	t.Helper()
	return CreateTempFile(t, "coverage.out", content)
}

func CreateTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	return CreateTempFile(t, "config.yaml", content)
}

func CreateTempHistoryFile(t *testing.T, content string) string {
	t.Helper()
	return CreateTempFile(t, ".go-covercheck.history.json", content)
}

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
