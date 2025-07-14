# go-covercheck

![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Contribute](https://img.shields.io/badge/contributions-welcome-brightgreen.svg)](CONTRIBUTING.md)
[![CI](https://github.com/mach6/go-covercheck/actions/workflows/ci.yaml/badge.svg)](https://github.com/mach6/go-covercheck/actions/workflows/ci.yaml)

🚦 A fast, flexible CLI tool for enforcing test coverage thresholds in Go projects.

> Fail builds when coverage drops below acceptable thresholds — by file, statement, or block level.

## Features

- Enforce minimum coverage thresholds for files, packages, and the entire project.
- Supports statement and block coverage separately.
- Native `table`|`json`|`yaml`|`md`|`html`|`csv`|`tsv` output. 
- Configurable via a `.go-covercheck.yml` or CLI flags.
- Sorting and colored table output.
- Colored `json` and `yaml` output.
- Built-in file or package regex filtering with `--skip`.
- Works seamlessly in CI/CD environments.

## Not Supported

The following items are noteworthy and not (currently) supported.

- Does not support configurable profile block count (how many times a section of code was hit) thresholds. The assumption 
  is any value `>=1` is enough.
- Table style is not configurable.
- Color codes (see [Color Legend](#Color-Legend)) are not configurable.
- Severity weights (see [Color Legend](#Color-Legend)) are not configurable.

## Background

I had access to a similar tool in a previous job. I took it for granted. After leaving this job and continuing to work 
in Go, I realized that I needed that tool in my life again. The closest that I was able to find online is 
[gocovergate](https://github.com/patrickhoefler/gocovergate). I [forked](https://github.com/mach6/gocovergate) it to
make it configurable. 

This held me over for a little while. However, I still wanted the functionality that I had
access to before, and I did not want to put a lot of effort into creating it.

So, I used generative AI as a starting point and a few dozen prompts later, `go-covercheck` was born. Many hours later,
the first release was ready.

## Installation

```shell
go install github.com/mach6/go-covercheck/cmd/go-covercheck@latest
```

## Usage

Create a `.go-coverheck.yml` which defines the threshold requirements.  

- This step is optional but _recommended_.
- See [full sample](samples/.go-covercheck.yml).

```yaml
statementThreshold: 65.0
blockThreshold: 60.0
```

Run the tests for a go project and produce a `coverage.out`.

```shell
go test ./... --coverprofile coverage.out
```

Run the `go-covercheck` CLI.

```text
$ go-covercheck coverage.out
┌───────────────────────────┬─────────────────┬─────────────────┬─────────────────┬─────────────────┐
│                           │    STATEMENTS   │      BLOCKS     │   STATEMENT %   │     BLOCK %     │
├───────────────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ BY FILE                   │                 │                 │                 │                 │
├───────────────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ cmd/foo.go                │             0/1 │             0/1 │             0.0 │             0.0 │
│ cmd/bar.go                │          20/110 │            7/80 │            18.2 │             8.8 │
├───────────────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ BY PACKAGE                │                 │                 │                 │                 │
├───────────────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ cmd                       │          20/111 │            7/81 │            18.0 │             8.6 │
├───────────────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│ BY TOTAL                  │                 │                 │                 │                 │
├───────────────────────────┼─────────────────┼─────────────────┼─────────────────┼─────────────────┤
│                           │          20/111 │            7/81 │            18.0 │            8.6  │
└───────────────────────────┴─────────────────┴─────────────────┴─────────────────┴─────────────────┘
✘ Coverage check failed

By File
 [S] cmd/foo.go [improvement of 70.0% required to meet 70.0% threshold]
 [B] cmd/foo.go [improvement of 50.0% required to meet 50.0% threshold]
 [S] cmd/bar.go [improvement of 51.8% required to meet 70.0% threshold]
 [B] cmd/bar.go [improvement of 41.2% required to meet 50.0% threshold]

By Package
 [S] cmd [improvement of 52.0% required to meet 70.0% threshold]
 [B] cmd [improvement of 41.4% required to meet 50.0% threshold]

By Total
 [S] total [improvement of 52.0% required to meet 70.0% threshold]
 [B] total [improvement of 41.4% required to meet 50.0% threshold]
```

Flags for the `go-covercheck` CLI.

```text
$ go-covercheck -h
go-covercheck: Coverage gatekeeper for enforcing test thresholds in Go

Usage:
  go-covercheck [coverage.out] [flags]

Flags:
  -b, --block-threshold float             global block threshold to enforce - disabled with 0 (default 50)
  -c, --config string                     path to YAML config file (default ".go-covercheck.yml")
  -f, --format string                     output format: table|json|yaml|md|html|csv|tsv (default "table")
  -h, --help                              help for go-covercheck
  -w, --no-color                          disable color output
  -u, --no-summary                        suppress failure summary and only show table output - disabled by default for json|yaml
  -t, --no-table                          suppress table output and only show failure summary - disabled by default for json|yaml
  -k, --skip stringArray                  regex string of file(s) and/or package(s) to skip
      --sort-by string                    sort-by: file|blocks|statements|statement-percent|block-percent (default "file")
      --sort-order string                 sort order: asc|desc (default "asc")
  -s, --statement-threshold float         global statement threshold to enforce - disabled with 0 (default 70)
      --term-width int                    force output to specified column width - autodetect with 0
  -a, --total-block-threshold float       total block threshold to enforce - disabled with 0
  -r, --total-statement-threshold float   total statement threshold to enforce - disabled with 0
  -v, --version                           version for go-covercheck
```

## Color Legend

By default, `go-covercheck` uses color in table format(s). The color is used to indicate severity as follows:

- % in ${\color{red}red}$ indicates the actual is `<=` `50%` of the threshold goal
- % in ${\color{yellow}yellow}$ indicates the actual is `>` `50%` and `<=` `99%` of the threshold goal 
- % in ${\color{green}green}$ indicates the actual is `>` `99%` of the threshold goal or the goal was met
- % in no color indicates the goal and actual are `0` or the goal is `0`


## License

[MIT](LICENSE)

## Contributing

Contributions are welcome. Please see [CONTRIBUTING](CONTRIBUTING.md).
