# Contributing to go-covercheck

Thank you for helping make `go-covercheck` better. Here's how to get started:

---

## 🚧 Found a Bug?

- Search existing issues first — it may already be reported or in progress.
- If it’s new, [open an issue](https://github.com/mach6/go-covercheck/issues).
- Include:
    - Steps to reproduce
    - Expected vs. actual behavior
    - Go version and OS

---

## ✨ Want to Add a Feature?

1. Fork the repo and create a feature branch
2. Use clear and concise commit messages (see below)
3. Include tests for new functionality where appropriate
4. Submit a pull request with a descriptive title and explanation

We recommend you open a discussion or issue first to validate the idea with maintainers.

---

## 🧪 Local Development

```bash
git clone https://github.com/yourusername/go-covercheck.git
cd go-covercheck

go build
go test ./...
```

To generate a coverage report for testing:

```bash
$ go test -coverprofile=coverage.out ./...
$ go run ./cmd/go-covercheck coverage.out
```

## 🎨 Code Style

- Format your code with `gofmt` and `goimports`
- Keep it idiomatic and readable
- Avoid unnecessary dependencies
- Don’t forget to update or add documentation as needed

## ✅ Commit Guidelines

We use [Conventional Commits](https://www.conventionalcommits.org/) to maintain clear history:

- `feat:` for new features
- `fix:` for bug fixes
- `docs:` for documentation only
- `chore:` for tooling or meta work
- `refactor:` for code reorganization

Example:

```text
feat: add support for multi-file coverage filtering
```

## 📝 Changelog

User-facing changes are written under `## [Unreleased]` in `CHANGELOG.md`, in plain words, grouped as
Breaking, Added, Deprecated, Changed or Fixed.

When a version is tagged, the release workflow opens a pull request that moves those notes into a section for
the tag, and updates the compare links (`scripts/changelog.sh`, run with `make changelog`). If `[Unreleased]` is
empty, it generates the notes from the commits with `git-chglog` instead, using only `feat`, `fix`, `perf` and
`refactor` commits. Edit that pull request before merging it if you like.
