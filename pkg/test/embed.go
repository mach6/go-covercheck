package test

import _ "embed"

//go:embed assets/coverage.out
var TestCoverageOut string

//go:embed assets/invalid.coverage.out
var InvalidTestCoverageOut string

//go:embed assets/history.json
var TestCoverageHistory string

//go:embed assets/invalid.history.json
var InvalidTestCoverageHistory string

//go:embed assets/config.yaml
var TestConfig string

//go:embed assets/error.config.yaml
var ErrorTestConfig string

//go:embed assets/invalid.config.yaml
var InvalidTestConfig string
