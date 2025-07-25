// Package test provides assets and helper functions for testing purposes.
package test

import _ "embed"

// TestCoverageOut contains the contents of the embedded `assets/coverage.out` file.
// This file is typically used for test coverage data.
//
//go:embed assets/valid.coverage.out
var TestCoverageOut string

// InvalidTestCoverageOut contains the contents of the embedded `assets/invalid.coverage.out` file.
// This file represents invalid test coverage data for testing error scenarios.
//
//go:embed assets/invalid.coverage.out
var InvalidTestCoverageOut string

// TestCoverageHistory contains the contents of the embedded `assets/history.json` file.
// This file is used to store historical test coverage data.
//
//go:embed assets/history.json
var TestCoverageHistory string

// InvalidTestCoverageHistory contains the contents of the embedded `assets/invalid.history.json` file.
// This file represents invalid historical test coverage data for testing error scenarios.
//
//go:embed assets/invalid.history.json
var InvalidTestCoverageHistory string

// TestConfig contains the contents of the embedded `assets/config.yaml` file.
// This file is used for configuration purposes in tests.
//
//go:embed assets/config.yaml
var TestConfig string

// ErrorTestConfig contains the contents of the embedded `assets/error.config.yaml` file.
// This file represents an erroneous configuration for testing error handling.
//
//go:embed assets/error.config.yaml
var ErrorTestConfig string

// InvalidTestConfig contains the contents of the embedded `assets/invalid.config.yaml` file.
// This file represents an invalid configuration for testing error scenarios.
//
//go:embed assets/invalid.config.yaml
var InvalidTestConfig string
