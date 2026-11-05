# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking

* Line coverage is now checked by default, at 50%. Builds that passed before can fail. Set `lineThreshold` to 0 to turn it off.
* Total thresholds now follow the matching global threshold unless `total:` sets them. A config that raised a global threshold now checks the total at that level too.
* CSV and TSV output has new columns, so existing columns have moved. Read columns by name, not by position.
* Building from source needs Go 1.27 or newer.
* Library users: `compute.CollectResults` no longer tidies file names itself. Call `compute.NormalizeNames` first. The command line does this for you.
* With JSON or YAML output, `--compare-history` no longer prints text after the output. Read the new `comparison` section instead.

### Added

* Line coverage, with `--line-threshold` (`-n`), `--total-line-threshold` (`-N`) and `lineThreshold` in the config file.
* Uncovered lines are listed in reports. `--no-uncovered-lines` (`-Q`) hides them.
* `--inspect` (`-U`) shows the uncovered code itself. `--inspect-file` (`-F`) picks the files, `--inspect-context` (`-P`) sets how many lines to show around each gap, and `--syntax-style` (`-Y`) sets the colors.
* Function coverage, with `--function-threshold` (`-g`), `--total-function-threshold` (`-G`) and `functionThreshold` in the config file. It is off by default.
* Uncovered functions are listed in reports. `--no-uncovered-functions` (`-O`) hides them.
* `--diff-from` (`-d`) checks only the files changed since a git ref.
* With JSON or YAML output, `--compare-history` adds a `comparison` section, so the output stays one valid document. Comparisons also list files and packages that are new or gone since the saved run.
* `--table-style` picks the table look: default, light, bold, rounded or double.
* `--format heatmap` shows coverage as a grid of colored cells. `--heatmap-png` also saves it as an image, and `--heatmap-metric` picks which measure sets the colors.
* `--comment` posts the results as a comment on a pull or merge request on GitHub, GitLab or Gitea.
* The README now points to the GitHub Action for running go-covercheck in workflows.

### Deprecated

* Library users: `output.CompareHistory` still works but is deprecated. Use `compute.BuildComparison` and `output.FormatAndReportWithComparison`.

### Changed

* History messages (saved, deleted, removed) start with `≡` instead of `✓`.
* A warning is printed when a source file can't be found.
* Release builds use Go 1.27.

### Fixed

* The Docker image can now make secure web requests, which pull request comments need.
* History no longer fails on commit ids shorter than 7 characters.
* `--inspect-file` and source lookup match paths the same way on every operating system.

## [v0.6.1] - 2025-08-05

## What's Changed
* chore: update changelog for v0.6.0
* Add module name dectection from go.mod
* Fix empty table display issue - show message instead of empty tables

## [v0.6.0] - 2025-07-30

### Changed

* Add CHANGELOG.md in Keep a Changelog format
* Add `moduleName` configuration option to fix module inference issue 
* Add `--init` flag to create sample `.go-covercheck.yml` config file 
* Add dependabot configuration for automated dependency updates 
* ci(deps): bump softprops/action-gh-release from 1 to 2
* Remove reviewers from dependabot.yml and add CODEOWNERS file
* deps(deps): bump github.com/jedib0t/go-pretty/v6 from 6.6.7 to 6.6.8
* Add support to delete history entries by reference
* Add Homebrew support for go-covercheck installation

## [v0.5.0] - 2025-07-25

### Added
- History tracking feature to compare coverage over time
- Comprehensive test suite and documentation
- GitHub Actions CI/CD workflows with automated testing
- Docker container support with multi-stage builds
- Git-chglog configuration for automated changelog generation
- Issue and pull request templates
- Contributing guidelines and project documentation

### Changed
- Major codebase expansion with organized package structure
- Enhanced configuration system with validation
- Improved error handling and logging
- Updated dependencies and Go module structure

### Fixed
- Various bug fixes and code improvements
- Enhanced test coverage across all packages

## [v0.4.1] - 2025-07-14

### Fixed
- Fixed typo in GitHub Actions tag-release workflow file that prevented proper branch creation

## [v0.4.0] - 2025-07-14

### Added
- Support for per-package coverage thresholds
- Support for separate total coverage thresholds
- New CLI flags: `--total-statement-threshold` (`-r`) and `--total-block-threshold` (`-a`)
- Enhanced table output with BY FILE, BY PACKAGE, and BY TOTAL sections
- Improved configuration structure with per-package and total threshold overrides

### Changed
- Major refactoring of configuration, output, and summary modules
- Restructured Results type to support file, package, and total statistics
- Updated table rendering with better organization and formatting
- Enhanced error reporting with categorized failure messages
- Updated CLI flag descriptions to clarify global vs total thresholds
- Improved dependency versions

### Fixed
- Bug fix for `--sort-by blocks` functionality that was not working correctly

## [v0.3.0] - 2025-07-12

### Added
- New `--term-width` flag to force output to specified column width
- Better autodetection of color and terminal width in CI environments
- Support for various CI systems including GitHub Actions, GitLab CI, CircleCI, and more
- Intelligent color detection based on environment variables and CI context
- Support for `NO_COLOR` and `FORCE_COLOR` environment variables
- Enhanced terminal width detection with fallbacks for CI environments

### Changed
- Improved color handling logic with CI-aware detection
- Better terminal width calculation for different environments
- Updated table formatting to respect terminal width constraints
- Refactored CLI argument processing into separate functions

### Fixed
- Improved error handling for terminal size detection
- Better fallback handling for terminal width in non-TTY environments

## [v0.2.0] - 2025-07-02

### Changed
- Updated version handling to separate application version from git revision
- Improved version command output with build information
- Enhanced build process with better version management
- Refactored CLI to show detailed version information including build metadata

### Fixed
- Better handling of version information during development vs production builds

## [v0.1.0] - 2025-07-02

### Added
- Initial release of go-covercheck tool
- Coverage checking functionality for Go projects
- Support for statement and block coverage thresholds
- Multiple output formats: table, JSON, YAML, Markdown, HTML, CSV, TSV
- Configuration file support (`.go-covercheck.yml`)
- Regex-based file and package filtering with `--skip` flag
- Colored output with severity-based color coding
- Sorting capabilities by file, statements, blocks, and percentages
- Per-file threshold overrides
- Comprehensive CLI with intuitive flags and options

### Features
- Global statement and block threshold enforcement
- Per-file threshold customization
- Multiple output format support
- Configurable sorting and filtering
- Color-coded results with severity indicators
- Summary reporting for failed coverage checks

[Unreleased]: https://github.com/mach6/go-covercheck/compare/v0.6.1...HEAD
[v0.6.1]: https://github.com/mach6/go-covercheck/compare/v0.6.0...v0.6.1
[v0.6.0]: https://github.com/mach6/go-covercheck/compare/v0.5.0...v0.6.0
[v0.5.0]: https://github.com/mach6/go-covercheck/compare/v0.4.1...v0.5.0
[v0.4.1]: https://github.com/mach6/go-covercheck/compare/v0.4.0...v0.4.1
[v0.4.0]: https://github.com/mach6/go-covercheck/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/mach6/go-covercheck/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/mach6/go-covercheck/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/mach6/go-covercheck/releases/tag/v0.1.0
