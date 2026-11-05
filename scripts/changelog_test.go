package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const changelog = `# Changelog

Intro.

## [Unreleased]

### Added

* A new thing.

### Fixed

* A fix.

## [v1.1.0] - 2026-01-01

### Added

* Old thing.

## [v1.0.0] - 2025-12-01

* First.

[Unreleased]: https://example.com/repo/compare/v1.1.0...HEAD
[v1.1.0]: https://example.com/repo/compare/v1.0.0...v1.1.0
[v1.0.0]: https://example.com/repo/releases/tag/v1.0.0
`

const emptyUnreleased = `# Changelog

## [Unreleased]

## [v1.1.0] - 2026-01-01

* Old thing.

[Unreleased]: https://example.com/repo/compare/v1.1.0...HEAD
[v1.1.0]: https://example.com/repo/releases/tag/v1.1.0
`

// result is what running scripts/changelog.sh left behind.
type result struct {
	file  string // the changelog file
	after string // its content afterwards
	out   string // the combined output
	err   error
}

// script runs scripts/changelog.sh on the changelog file. A directory in binDir is put first on PATH.
func script(t *testing.T, file, binDir string, args ...string) (string, error) {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not available")
	}

	cmd := exec.CommandContext(t.Context(), bash, append([]string{"changelog.sh"}, args...)...) //nolint:gosec // test data
	cmd.Env = append(os.Environ(), "CHANGELOG="+file)
	if binDir != "" {
		cmd.Env = append(cmd.Env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// run runs scripts/changelog.sh on a changelog file with the given content.
func run(t *testing.T, content, binDir string, args ...string) result {
	t.Helper()
	file := filepath.Join(t.TempDir(), "CHANGELOG.md")
	require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

	out, err := script(t, file, binDir, args...)

	data, readErr := os.ReadFile(file) //nolint:gosec // a file made by this test
	require.NoError(t, readErr)
	return result{file: file, after: string(data), out: out, err: err}
}

// fakeChglog installs a git-chglog that prints the given text, and returns the directory it is in.
func fakeChglog(t *testing.T, output string) string {
	t.Helper()
	dir := t.TempDir()
	body := "#!/bin/sh\ncat <<'EOF'\n" + output + "EOF\n"
	path := filepath.Join(dir, "git-chglog")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o700)) //nolint:gosec // a test helper script
	return dir
}

func TestChangelog_ReleasesTheCuratedNotes(t *testing.T) {
	r := run(t, changelog, "", "v1.2.0", "2026-02-02")

	require.NoError(t, r.err, r.out)
	require.Equal(t, `# Changelog

Intro.

## [Unreleased]

## [v1.2.0] - 2026-02-02

### Added

* A new thing.

### Fixed

* A fix.

## [v1.1.0] - 2026-01-01

### Added

* Old thing.

## [v1.0.0] - 2025-12-01

* First.

[Unreleased]: https://example.com/repo/compare/v1.2.0...HEAD
[v1.2.0]: https://example.com/repo/compare/v1.1.0...v1.2.0
[v1.1.0]: https://example.com/repo/compare/v1.0.0...v1.1.0
[v1.0.0]: https://example.com/repo/releases/tag/v1.0.0
`, r.after)
}

func TestChangelog_DoesNothingTheSecondTime(t *testing.T) {
	first := run(t, changelog, "", "v1.2.0", "2026-02-02")
	require.NoError(t, first.err, first.out)

	out, err := script(t, first.file, "", "v1.2.0", "2026-02-03")
	require.NoError(t, err, out)
	require.Contains(t, out, "nothing to do")

	again, readErr := os.ReadFile(first.file)
	require.NoError(t, readErr)
	require.Equal(t, first.after, string(again))
}

func TestChangelog_WorksWithoutCompareLinks(t *testing.T) {
	r := run(t, "# Changelog\n\n## [Unreleased]\n\n* A thing.\n\n## [v1.0.0] - 2025-12-01\n\n* First.\n",
		"", "v1.1.0", "2026-01-01")

	require.NoError(t, r.err, r.out)
	require.Equal(t, "# Changelog\n\n## [Unreleased]\n\n## [v1.1.0] - 2026-01-01\n\n* A thing.\n\n"+
		"## [v1.0.0] - 2025-12-01\n\n* First.\n", r.after)
}

func TestChangelog_NothingBelowUnreleased(t *testing.T) {
	r := run(t, "# Changelog\n\n## [Unreleased]\n\n* A thing.\n", "", "v1.0.0", "2026-01-01")

	require.NoError(t, r.err, r.out)
	require.Equal(t, "# Changelog\n\n## [Unreleased]\n\n## [v1.0.0] - 2026-01-01\n\n* A thing.\n\n", r.after)
}

func TestChangelog_DateDefaultsToToday(t *testing.T) {
	r := run(t, changelog, "", "v9.9.9")

	require.NoError(t, r.err, r.out)
	require.Regexp(t, `(?m)^## \[v9\.9\.9\] - \d{4}-\d{2}-\d{2}$`, r.after)
}

func TestChangelog_GeneratesTheNotesWhenNothingIsCurated(t *testing.T) {
	generated := `## [Unreleased]


## [v1.2.0] - 2026-02-02
### Feat
- **output:** add a thing
- add another thing

### Fix
- **docker:** fix a thing


[Unreleased]: https://example.com/repo/compare/v1.2.0...HEAD
[v1.2.0]: https://example.com/repo/compare/v1.1.0...v1.2.0
`
	r := run(t, emptyUnreleased, fakeChglog(t, generated), "v1.2.0", "2026-02-02")

	require.NoError(t, r.err, r.out)
	require.Equal(t, `# Changelog

## [Unreleased]

## [v1.2.0] - 2026-02-02

### Feat
- **output:** add a thing
- add another thing

### Fix
- **docker:** fix a thing

## [v1.1.0] - 2026-01-01

* Old thing.

[Unreleased]: https://example.com/repo/compare/v1.2.0...HEAD
[v1.2.0]: https://example.com/repo/compare/v1.1.0...v1.2.0
[v1.1.0]: https://example.com/repo/releases/tag/v1.1.0
`, r.after)
}

func TestChangelog_GeneratedNotesWithAnUnbracketedHeading(t *testing.T) {
	// git-chglog writes the heading without brackets when it cannot find the previous tag.
	generated := "## [Unreleased]\n\n\n## v1.2.0 - 2026-02-02\n### Feat\n- add a thing\n\n\n[Unreleased]: x\n"
	r := run(t, emptyUnreleased, fakeChglog(t, generated), "v1.2.0", "2026-02-02")

	require.NoError(t, r.err, r.out)
	require.Contains(t, r.after, "## [v1.2.0] - 2026-02-02\n\n### Feat\n- add a thing\n\n## [v1.1.0]")
}

func TestChangelog_FailsWhenThereIsNothingToRelease(t *testing.T) {
	// git-chglog finds no changes for the tag.
	generated := "## [Unreleased]\n\n\n[Unreleased]: https://example.com/repo/compare/v1.1.0...HEAD\n"
	r := run(t, emptyUnreleased, fakeChglog(t, generated), "v1.2.0", "2026-02-02")

	require.Error(t, r.err)
	require.Contains(t, r.out, "found no changes")
	require.Equal(t, emptyUnreleased, r.after, "the file is left alone")
}

func TestChangelog_FailsWithoutAnUnreleasedSection(t *testing.T) {
	content := "# Changelog\n\n## [v1.0.0] - 2025-12-01\n\n* First.\n"
	r := run(t, content, "", "v1.1.0", "2026-01-01")

	require.Error(t, r.err)
	require.Contains(t, r.out, "no [Unreleased] section")
	require.Equal(t, content, r.after)
}

func TestChangelog_NeedsATag(t *testing.T) {
	r := run(t, changelog, "")

	require.Error(t, r.err)
	require.Contains(t, r.out, "usage")
}
