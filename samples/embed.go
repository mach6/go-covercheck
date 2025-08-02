// Package samples provides embedded sample configuration files for go-covercheck.
package samples

import _ "embed"

// SampleConfigYAML contains the embedded sample .go-covercheck.yml configuration file content.
//
//go:embed .go-covercheck.yml
var SampleConfigYAML string
