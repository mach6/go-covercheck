package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShouldSkip_MatchesPrefix(t *testing.T) {
	require.True(t, shouldSkip("vendor/foo.go", []string{"vendor/"}))
	require.True(t, shouldSkip("gen/code.go", []string{"gen/"}))
}

func TestShouldSkip_MatchesExact(t *testing.T) {
	require.True(t, shouldSkip("internal/tmp_test.go", []string{"internal/tmp_test.go"}))
}

func TestShouldSkip_NoMatches(t *testing.T) {
	require.False(t, shouldSkip("main.go", []string{"generated.go"}))
	require.False(t, shouldSkip("src/foo/bar.go", nil))
}
