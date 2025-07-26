package compute //nolint:testpackage

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func Test_normalizeNames(t *testing.T) {
	names := []string{
		"do.main/pkg/main.go",
		"do.main/pkg/utils/utils.go",
		"do.main/pkg/foo/foo.go",
		"do.main/pkg/bar.go",
		"do.main/internal/baz.go",
	}
	expect := []string{
		"pkg/main.go",
		"pkg/utils/utils.go",
		"pkg/foo/foo.go",
		"pkg/bar.go",
		"internal/baz.go",
	}

	profiles := make([]*cover.Profile, len(names))
	for i, name := range names {
		profiles[i] = &cover.Profile{
			FileName: name,
		}
	}

	cfg := &config.Config{}
	normalizeNames(profiles, cfg)
	for i, profile := range profiles {
		require.Equal(t, expect[i], profile.FileName)
	}
}

func Test_findModuleName_withCommonParent(t *testing.T) {
	// This test demonstrates the issue described in the problem statement
	// When all packages share a common parent, the module inference is incorrect
	names := []string{
		"github.com/mach6/go-covercheck/pkg/foo/foo.go",
		"github.com/mach6/go-covercheck/pkg/bar/bar.go",
	}

	profiles := make([]*cover.Profile, len(names))
	for i, name := range names {
		profiles[i] = &cover.Profile{
			FileName: name,
		}
	}

	// Current behavior: incorrectly infers "github.com/mach6/go-covercheck/pkg/"
	cfg := &config.Config{}
	moduleName := findModuleName(profiles, cfg)
	require.Equal(t, "github.com/mach6/go-covercheck/pkg/", moduleName)

	// After normalizing with the current logic, we get wrong results
	normalizeNames(profiles, cfg)
	require.Equal(t, "foo/foo.go", profiles[0].FileName)
	require.Equal(t, "bar/bar.go", profiles[1].FileName)

	// What we want instead is:
	// Module name should be "github.com/mach6/go-covercheck" 
	// And files should be "pkg/foo/foo.go" and "pkg/bar/bar.go"
}

func Test_findModuleName_withConfiguredModuleName(t *testing.T) {
	// Test the fix: when ModuleName is configured, use it instead of inferring
	names := []string{
		"github.com/mach6/go-covercheck/pkg/foo/foo.go",
		"github.com/mach6/go-covercheck/pkg/bar/bar.go",
	}

	profiles := make([]*cover.Profile, len(names))
	for i, name := range names {
		profiles[i] = &cover.Profile{
			FileName: name,
		}
	}

	// Test with configured module name
	cfg := &config.Config{
		ModuleName: "github.com/mach6/go-covercheck",
	}
	
	moduleName := findModuleName(profiles, cfg)
	require.Equal(t, "github.com/mach6/go-covercheck/", moduleName)

	// After normalizing with the configured module name, we get correct results
	normalizeNames(profiles, cfg)
	require.Equal(t, "pkg/foo/foo.go", profiles[0].FileName)
	require.Equal(t, "pkg/bar/bar.go", profiles[1].FileName)
}
