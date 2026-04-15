# 🚦 go-covercheck

![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Contribute](https://img.shields.io/badge/contributions-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![CI](https://github.com/mach6/go-covercheck/actions/workflows/ci.yaml/badge.svg)](https://github.com/mach6/go-covercheck/actions/workflows/ci.yaml)

A fast, flexible CLI tool for enforcing test coverage thresholds in Go projects.

> Fail builds when coverage drops below acceptable thresholds — by file, statement, block, or line level.

## ✨ Features

- Enforce minimum coverage thresholds for files, packages, and the entire project.
- Check coverage only on changed files in git diff.
- Supports statement, block, and 🆕 line coverage separately.
- 🆕 Inspect uncovered source code with syntax highlighting (`--inspect`).
- 🆕 Show uncovered line numbers inline in the coverage table.
- Native `table`|`json`|`yaml`|`md`|`html`|`csv`|`tsv` output.
- Configurable table styles (`default`|`light`|`bold`|`rounded`|`double`).
- Configurable via a `.go-covercheck.yml` or CLI flags.
- Sorting and colored table output.
- Colored `json` and `yaml` output.
- Built-in file or package regex filtering with `--skip`.
- Save and compare against historical results from a commit, branch, tag, or user defined label.
- Works seamlessly in CI/CD environments.

## 🚫 Not Supported

The following items are noteworthy and not (currently) supported.

- Does not support configurable profile block count (how many times a section of code was hit) thresholds. The assumption
  is any value `>=1` is enough.
- Color codes (see [Color Legend](#-color-legend)) are not configurable.
- Severity weights (see [Color Legend](#-color-legend)) are not configurable.

## 📖 Background

I had access to a similar tool in a previous job. I took it for granted. After leaving this job and continuing to work
in Go, I realized that I needed that tool in my life again. The closest that I was able to find online is
[gocovergate](https://github.com/patrickhoefler/gocovergate). I [forked](https://github.com/mach6/gocovergate) it to
make it configurable.

This held me over for a little while. However, I still wanted the functionality that I had
access to before, and I did not want to put a lot of effort into creating it.

So, I used generative AI as a starting point and a few dozen prompts later, `go-covercheck` was born. Many hours later,
the first release was ready.

## 🛠️ Installation

There are several ways to install `go-covercheck`. Choose the one that best fits your needs.

### 📦 Official Releases

You can download official releases from the [releases page](https://github.com/mach6/go-covercheck/releases).
All releases are built with the latest Go version and include build information, such as the version number and commit sha.
After downloading, you can place the binary in your `PATH` to use it globally. If you are on Linux, Freebsd or Mac OS,
don't forget to make it executable:

```shell
chmod +x go-covercheck
```

### 🍺 Homebrew (macOS/Linux)

You can install `go-covercheck` using Homebrew:

```shell
brew install mach6/tap/go-covercheck
```

### 🐹 Via `go install`

You can install `go-covercheck` using the `go install` command, if you don't care about official releases. Versions installed
this way are not stamped with build information, such as the version number or commit sha.

```shell
go install github.com/mach6/go-covercheck/cmd/go-covercheck@latest
```

### 🐳 Docker Image

Docker images are available on the github container registry. You can pull the latest image with:

```shell
docker pull ghcr.io/mach6/go-covercheck:latest
```

Or you can use a tag which maps to a specific version:

```shell
docker pull ghcr.io/mach6/go-covercheck:0.6.1
```

## 📋 Usage

### ⚙️ Configure

Create a `.go-coverheck.yml` which defines the threshold requirements. You can create this file manually or use the
`--init` flag to generate a sample config file in the current directory.

- This step is optional but _recommended_.
- See [full sample](samples/.go-covercheck.yml).

```shell
go-covercheck --init
```

Here is a sample `.go-covercheck.yml` configuration file:

```yaml
# Optional, global thresholds overriding the defaults (70 statements, 50 blocks, 50 lines)
statementThreshold: 65.0
blockThreshold: 60.0
lineThreshold: 55.0

# Optional, table style for table output format (default: light)
tableStyle: bold

# Optional, by total thresholds overriding the global values above
total:
  statements: 75.0
  blocks: 70.0
  lines: 80.0
```

### 🧪 Run Tests

Run the tests for a go project and produce a `coverage.out`.

```shell
go test ./... --coverprofile coverage.out
```

### 🚀 Run `go-covercheck`

Use the `go-covercheck` CLI to check the coverage against the thresholds defined in the config file or CLI flags.

```text
$ go-covercheck coverage.out
┌────────────┬────────────┬─────────┬──────────┬─────────────┬──────────┬─────────┐
│            │ STATEMENTS │ BLOCKS  │  LINES   │ STATEMENT % │ BLOCK %  │ LINE %  │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ BY FILE    │            │         │          │             │          │         │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ cmd/foo.go │        0/1 │     0/1 │      0/3 │         0.0 │      0.0 │     0.0 │
│ cmd/bar.go │     20/110 │    7/80 │   25/140 │        18.2 │      8.8 │    17.9 │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ BY PACKAGE │            │         │          │             │          │         │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ cmd        │     20/111 │    7/81 │   25/143 │        18.0 │      8.6 │    17.5 │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ BY TOTAL   │            │         │          │             │          │         │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│            │     20/111 │    7/81 │   25/143 │        18.0 │      8.6 │    17.5 │
└────────────┴────────────┴─────────┴──────────┴─────────────┴──────────┴─────────┘
✘ Coverage check failed
 → By File
    [S] cmd/foo.go [+70.0% required for 70.0% threshold]
    [B] cmd/foo.go [+50.0% required for 50.0% threshold]
    [L] cmd/foo.go [+50.0% required for 50.0% threshold]
    [S] cmd/bar.go [+51.8% required for 70.0% threshold]
    [B] cmd/bar.go [+41.2% required for 50.0% threshold]
    [L] cmd/bar.go [+32.1% required for 50.0% threshold]
 → By Package
    [S] cmd [+52.0% required for 70.0% threshold]
    [B] cmd [+41.4% required for 50.0% threshold]
    [L] cmd [+32.5% required for 50.0% threshold]
 → By Total
    [S] total [+52.0% required for 70.0% threshold]
    [B] total [+41.4% required for 50.0% threshold]
    [L] total [+32.5% required for 50.0% threshold]
```

Note: if the file `coverage.out` is not specified, `go-covercheck` will look for a file named `coverage.out` in the current directory.
You can also specify a different file name and path.

### 🎛️ CLI Flags

You can also use CLI flags to configure `go-covercheck` without a config file.

```text
$ go-covercheck -h
go-covercheck: Coverage gatekeeper for enforcing test thresholds in Go

Usage:
  go-covercheck [coverage.out] [flags]

Flags:
  -b, --block-threshold float             global block threshold to enforce [0=disabled] (default 50)
  -C, --compare-history string            compare current coverage against historical ref [commit|branch|tag|label]
  -c, --config string                     path to YAML config file (default ".go-covercheck.yml")
  -D, --delete-history string             delete historical entry by ref [commit|branch|tag|label]
  -d, --diff-from string                  git reference (commit/branch/tag) to diff from; enables diff-only mode
  -f, --format string                     output format [table|json|yaml|md|html|csv|tsv] (default "table")
  -h, --help                              help for go-covercheck
      --history-file string               path to go-covercheck history file (default ".go-covercheck.history.json")
      --init                              create a sample .go-covercheck.yml config file in the current directory
  -U, --inspect                           show uncovered source code
  -P, --inspect-context int               additional context lines to show around uncovered source code (default 2)
  -F, --inspect-file stringArray          show uncovered source code for a specific file; match is exact path, basename, or path-boundary suffix (implies --inspect)
  -l, --label string                      optional label name for history entry
  -n, --line-threshold float              global line threshold to enforce [0=disabled] (default 50)
  -L, --limit-history int                 limit number of historical entries to save or display [0=no limit]
  -m, --module-name string                explicitly set module name for path normalization (overrides module inference)
  -w, --no-color                          disable color output
  -u, --no-summary                        suppress failure summary and only show tabular output [disabled for json|yaml]
  -t, --no-table                          suppress tabular output and only show failure summary [disabled for json|yaml]
  -Q, --no-uncovered-lines                omit uncovered line numbers from all outputs (table column and structured json/yaml/md/csv/tsv fields); use --inspect to show them
  -H, --save-history                      add coverage result to history
  -I, --show-history                      show historical entries in tabular format
  -k, --skip stringArray                  regex string of file(s) and/or package(s) to skip
      --sort-by string                    sort-by [file|blocks|statements|lines|statement-percent|block-percent|line-percent] (default "file")
      --sort-order string                 sort order [asc|desc] (default "asc")
  -s, --statement-threshold float         global statement threshold to enforce [0=disabled] (default 70)
  -Y, --syntax-style string               syntax highlighting style for code [auto|github|github-dark|monokai|dracula|solarized-dark|vim|emacs|...]; auto picks github or github-dark based on detected terminal background (default "auto")
      --term-width int                    force output to specified column width [0=autodetect]
  -B, --total-block-threshold float       total block threshold to enforce [0=disabled]
  -N, --total-line-threshold float        total line threshold to enforce [0=disabled]
  -S, --total-statement-threshold float   total statement threshold to enforce [0=disabled]
  -v, --version                           version for go-covercheck
```

### 🔧 Integration with `go tool covdata`

For integration tests or when collecting coverage from running binaries, you can use `go tool covdata` to work with coverage data directories and convert them to the standard coverage profile format that `go-covercheck` understands.


#### 🏗️ Integration Testing Scenario

This workflow is particularly useful for integration tests where you want to assert coverage from a running binary:

```shell
# 1. Build your binary with coverage support
go build -cover -o myapp ./cmd/myapp

# 2. Set environment variable for coverage data directory
export GOCOVERDIR=./coverdata
mkdir -p $GOCOVERDIR

# 3. Run your binary (coverage data will be written to GOCOVERDIR)
./myapp &
APP_PID=$!

# 4. Run your integration tests against the running binary
curl http://localhost:8080/health
curl http://localhost:8080/api/users
# ... more integration test calls

# 5. Stop the binary (this flushes coverage data)
kill $APP_PID

# 6. Convert coverage data to standard format
go tool covdata textfmt -i=./coverdata -o=integration-coverage.out

# 7. Check coverage with go-covercheck
go-covercheck integration-coverage.out
```

## 🧬 Diff Mode (Changed Files Only)

Enforce coverage thresholds only on files that have changed in your git diff. This is perfect for gradually improving coverage in large codebases without being penalized by legacy code.

### 🎯 Use Cases

- **Pull Request Validation**: Ensure new code meets coverage standards without failing on existing legacy code
- **Incremental Coverage Improvement**: Gradually increase coverage standards for new development
- **CI/CD Integration**: Gate deployments based on coverage of changes, not entire codebase

### 🚀 Basic Usage

```shell
# Check coverage only on files changed since a specific commit
go-covercheck --diff-from HEAD~1

# Compare against a specific commit
go-covercheck --diff-from HEAD~3

# Compare against a branch
go-covercheck --diff-from main

# Compare against a tag
go-covercheck --diff-from v1.0.0

# CI/CD: Check coverage for PR changes
go-covercheck --diff-from origin/main

# Local development: Check changes since last push
go-covercheck --diff-from @{upstream}

# Release validation: Check changes since last tag
go-covercheck --diff-from $(git describe --tags --abbrev=0)
```

### 🛡️ Fallback Behavior

If git operations fail (e.g., not in a git repository, invalid reference), `go-covercheck` will automatically fall back to checking all files with a warning message.

## 🔬 Inspect Uncovered Code

Use `--inspect` (`-U`) to print the uncovered source code with syntax highlighting instead of
the coverage table. This is helpful when you want to see *exactly* what code needs tests.

```shell
# Show uncovered code for all files in the coverage profile
go-covercheck --inspect coverage.out

# Limit to specific files (exact path, basename, or path-boundary suffix; implies --inspect)
go-covercheck --inspect-file pkg/compute/compute.go coverage.out

# Show more context lines around each uncovered block (default 2)
go-covercheck --inspect --inspect-context 5 coverage.out

# Pick a chroma syntax highlighting style
go-covercheck --inspect --syntax-style monokai coverage.out
```

The default `auto` picks `github` or `github-dark` based on the detected
terminal background. Any chroma style name is accepted; see
[chroma](https://github.com/alecthomas/chroma) for the full list (e.g.
`monokai`, `dracula`, `solarized-dark`, `vim`, `emacs`).

## 📏 Line Coverage

In addition to statement and block coverage, `go-covercheck` enforces line coverage thresholds.
Configure with `lineThreshold` in `.go-covercheck.yml`, `--line-threshold` (`-n`) on the CLI,
and `--total-line-threshold` (`-N`) for the project total. Per-file and per-package overrides
go under a `lines:` map alongside `statements:` and `blocks:`. Failures are reported with the
`[L]` prefix in the summary.

### 🎨 Table Styles

You can customize the appearance of table output using the `--table-style` flag or by configuring `tableStyle` in your `.go-covercheck.yml` file.

Available table styles:
- `default` - ASCII characters (compatible with all terminals)
- `light` - Unicode light box drawing characters (default)
- `bold` - Unicode bold box drawing characters
- `rounded` - Unicode rounded corner characters
- `double` - Unicode double line characters

**Example:**
```shell
# Use bold table style via CLI
go-covercheck --table-style=bold

# Use rounded table style via config file
echo "tableStyle: rounded" >> .go-covercheck.yml
```

## 🕰️ History

History is a feature that allows you to save and compare coverage results against previous runs.

### 💾 Save History

Save the current coverage result to history with the `--save-history` flag. This will create or update a history file
(`.go-covercheck.history.json` by default) with the current coverage results. Check this file into your version control
 system (or other) to keep a shared record of coverage over time.

```shell
go-covercheck --save-history
```

Optionally, specify a label for the history entry with the `--label` flag. This is useful when the project is not in git.
```shell
go-covercheck --save-history --label "my-label"
```

### 🔍 Compare Against History

Compare the current coverage against saved history with the `--compare-history` flag.
History integrates with git to compare against a ref—a `commit`, a `branch`, or a `tag`.

```shell
$ go-covercheck --compare-history main
...
≡ Comparing against ref: main [commit e402629]
 → By Total
    [S] total [+4.5 %]
    [B] total [+7.3 %]
```

A label can also be used to compare against a specific history entry.

```shell
go-covercheck --compare-history my-label
```
### 📊 Show History

Display saved history entries in a tabular format with the `--show-history` flag. This will show all saved history entries sorted by timestamp..

```text
$ go-covercheck --show-history
┌────────────┬─────────┬───────────────────────────┬─────────────────┬─────────────────┬─────────────┐
│  TIMESTAMP │  COMMIT │           BRANCH          │       TAGS      │      LABEL      │   COVERAGE  │
├────────────┼─────────┼───────────────────────────┼─────────────────┼─────────────────┼─────────────┤
│ 2025-07-25 │ e7c7d91 │ mach6_bugfix_feat_history │                 │                 │ 536/656 [S] │
│            │         │                           │                 │                 │ 309/409 [B] │
├────────────┼─────────┼───────────────────────────┼─────────────────┼─────────────────┼─────────────┤
│ 2025-07-25 │ 4f7469a │ mach6_bugfix_feat_history │                 │                 │ 484/653 [S] │
│            │         │                           │                 │                 │ 278/413 [B] │
├────────────┼─────────┼───────────────────────────┼─────────────────┼─────────────────┼─────────────┤
│ 2025-07-18 │ e402629 │ main                      │                 │                 │ 180/648 [S] │
│            │         │                           │                 │                 │ 95/409  [B] │
├────────────┼─────────┼───────────────────────────┼─────────────────┼─────────────────┼─────────────┤
│ 2025-07-18 │ 7fccdbd │ detached                  │ v0.4.1          │                 │ 220/285 [S] │
│            │         │                           │                 │                 │ 112/164 [B] │
└────────────┴─────────┴───────────────────────────┴─────────────────┴─────────────────┴─────────────┘
≡ Showing last 4 history entries
```

### ⏳ Limit History

You can limit the number of history entries displayed or saved with the `--limit-history` flag. This is useful to avoid
overloading the output with too many entries. Set it to `0` (the default) to disable any limit.
```shell
go-covercheck --save-history --limit-history 5 -l my-label
```

```text
$ go-covercheck --show-history --limit-history 2
┌────────────┬─────────┬───────────────────────────┬─────────────────┬─────────────────┬─────────────┐
│  TIMESTAMP │  COMMIT │           BRANCH          │       TAGS      │      LABEL      │   COVERAGE  │
├────────────┼─────────┼───────────────────────────┼─────────────────┼─────────────────┼─────────────┤
│ 2025-07-25 │ e7c7d91 │ mach6_bugfix_feat_history │                 │ my-label        │ 536/656 [S] │
│            │         │                           │                 │                 │ 309/409 [B] │
├────────────┼─────────┼───────────────────────────┼─────────────────┼─────────────────┼─────────────┤
│ 2025-07-25 │ 4f7469a │ mach6_bugfix_feat_history │                 │                 │ 484/653 [S] │
│            │         │                           │                 │                 │ 278/413 [B] │
└────────────┴─────────┴───────────────────────────┴─────────────────┴─────────────────┴─────────────┘
≡ Showing last 2 history entries
```

### 🗑️ Delete History

You can delete specific history entries using the `--delete-history` flag. This allows you to remove outdated or unwanted entries from your history file. The deletion uses the same reference matching as compare and show operations.

```shell
# Delete by commit hash (full or short)
go-covercheck --delete-history e402629

# Delete by branch name
go-covercheck --delete-history main

# Delete by tag
go-covercheck --delete-history v1.0.0

# Delete by label
go-covercheck --delete-history my-label
```

When a history entry is successfully deleted, you'll see a confirmation message:

```text
$ go-covercheck --delete-history main
≡ Deleted history entry for ref: main
```

If the reference is not found, an error message will be displayed:

```text
$ go-covercheck --delete-history nonexistent
Error: no history entry found for ref: nonexistent
```

## 📤 Output Formats
`go-covercheck` supports multiple output formats. The default is `table`, but you can specify other formats using the
`--format` flag (short form `-f`) or through the `format:` field of the config file.

The available formats are:

- `json`: Outputs the coverage details in JSON format.
- `yaml`: Outputs the coverage details in YAML format.
- `md`: Outputs the coverage details in Markdown format.
- `html`: Outputs the coverage details in HTML format.
- `csv`: Outputs the coverage details in CSV format.
- `tsv`: Outputs the coverage details in TSV (Tab-Separated Values) format.
- `table`: Outputs the coverage details in a human-readable table format (default).


### 📊 Table
The `table` format provides a human-readable output with color coding to indicate coverage status. It shows coverage
details by file, package, and total. The table is sorted by file name by default, but you can change the sort order and
field using the `--sort-by` and `--sort-order` flags.

```text
$ go-covercheck -f table coverage.out -u
┌────────────┬────────────┬─────────┬──────────┬─────────────┬──────────┬─────────┐
│            │ STATEMENTS │ BLOCKS  │  LINES   │ STATEMENT % │ BLOCK %  │ LINE %  │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ BY FILE    │            │         │          │             │          │         │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ cmd/foo.go │        0/1 │     0/1 │      0/3 │         0.0 │      0.0 │     0.0 │
│ cmd/bar.go │     20/110 │    7/80 │   25/140 │        18.2 │      8.8 │    17.9 │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ BY PACKAGE │            │         │          │             │          │         │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ cmd        │     20/111 │    7/81 │   25/143 │        18.0 │      8.6 │    17.5 │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│ BY TOTAL   │            │         │          │             │          │         │
├────────────┼────────────┼─────────┼──────────┼─────────────┼──────────┼─────────┤
│            │     20/111 │    7/81 │   25/143 │        18.0 │      8.6 │    17.5 │
└────────────┴────────────┴─────────┴──────────┴─────────────┴──────────┴─────────┘
```
Note: Here the `-u` flag was used to suppress the summary lines. By default the table also
includes a column listing uncovered line numbers per file; use `-Q` (`--no-uncovered-lines`)
to suppress it.


### 📑 Other Tabular Formats

`go-covercheck` supports other tabular formats such as `csv`, `tsv`, and `md`. These formats are useful for
exporting coverage data to other tools or visualizations. The flag `-u` (or `--no-summary`) can be used to suppress the
summary lines in these formats, as well, which is necessary for generating clean output for further processing.


### 📜 JSON
The `json` format provides a structured output that is easy to read and parse. It includes coverage details by file,
package, and total. It also includes the thresholds and the actual coverage percentages.

JSON output is color-coded by default, but you can disable color with the `--no-color` flag.
The `--no-summary` flag is applied when the `json` format is used.

```json
$ go-covercheck -f json coverage.out
{
  "byFile": [
    {
      "statementCoverage": "150/150",
      "blockCoverage": "1/1",
      "lineCoverage": "200/200",
      "statementPercentage": 100,
      "blockPercentage": 100,
      "linePercentage": 100,
      "statementThreshold": 0,
      "blockThreshold": 0,
      "lineThreshold": 0,
      "failed": false,
      "file": "foo"
    }
  ],
  "byPackage": [
    {
      "statementCoverage": "150/150",
      "blockCoverage": "1/1",
      "lineCoverage": "200/200",
      "statementPercentage": 100,
      "blockPercentage": 100,
      "linePercentage": 100,
      "statementThreshold": 0,
      "blockThreshold": 0,
      "lineThreshold": 0,
      "failed": false,
      "package": "."
    }
  ],
  "byTotal": {
    "statements": {
      "coverage": "150/150",
      "threshold": 0,
      "percentage": 100,
      "failed": false
    },
    "blocks": {
      "coverage": "1/1",
      "threshold": 0,
      "percentage": 100,
      "failed": false
    },
    "lines": {
      "coverage": "200/200",
      "threshold": 0,
      "percentage": 100,
      "failed": false
    }
  }
}
```

### 📜 YAML
The `yaml` format provides a structured output that is easy to read and parse. It includes coverage details by file,
package, and total. It also includes the thresholds and the actual coverage percentages.

YAML output is color-coded by default, but you can disable color with the `--no-color` flag.
The `--no-summary` flag is applied when the `yaml` format is used.
```yaml
$ go-covercheck -f yaml coverage.out
byFile:
  - statementCoverage: 150/150
    blockCoverage: 1/1
    lineCoverage: 200/200
    statementPercentage: 100
    blockPercentage: 100
    linePercentage: 100
    statementThreshold: 0
    blockThreshold: 0
    lineThreshold: 0
    failed: false
    file: foo
byPackage:
  - statementCoverage: 150/150
    blockCoverage: 1/1
    lineCoverage: 200/200
    statementPercentage: 100
    blockPercentage: 100
    linePercentage: 100
    statementThreshold: 0
    blockThreshold: 0
    lineThreshold: 0
    failed: false
    package: .
byTotal:
  statements:
    coverage: 150/150
    threshold: 0
    percentage: 100
    failed: false
  blocks:
    coverage: 1/1
    threshold: 0
    percentage: 100
    failed: false
  lines:
    coverage: 200/200
    threshold: 0
    percentage: 100
    failed: false
```


## 🎨 Color Legend

By default, `go-covercheck` uses color in tabular format(s). The color is used to indicate severity as follows:

- % in ${\color{red}red}$ indicates the actual is `<=` `50%` of the threshold goal
- % in ${\color{yellow}yellow}$ indicates the actual is `>` `50%` and `<=` `99%` of the threshold goal
- % in ${\color{green}green}$ indicates the actual is `>` `99%` of the threshold goal or the goal was met
- % in no color indicates the goal and actual are `0` or the goal is `0`


## 📜 License

[MIT](LICENSE)

## 🤝 Contributing

Contributions are welcome. Please see [CONTRIBUTING](CONTRIBUTING.md).
