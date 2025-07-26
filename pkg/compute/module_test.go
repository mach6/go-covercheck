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

// Test longestCommonPrefix function and the findModuleName function cannot infer a correct module name.
func Test_findModuleName_withCommonParent(t *testing.T) {
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

	cfg := &config.Config{}
	moduleName := findModuleName(profiles, cfg)
	require.Equal(t, "github.com/mach6/go-covercheck/pkg/", moduleName)

	normalizeNames(profiles, cfg)
	require.Equal(t, "foo/foo.go", profiles[0].FileName)
	require.Equal(t, "bar/bar.go", profiles[1].FileName)
}

func Test_findModuleName_withConfiguredModuleName(t *testing.T) {
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

	normalizeNames(profiles, cfg)
	require.Equal(t, "pkg/foo/foo.go", profiles[0].FileName)
	require.Equal(t, "pkg/bar/bar.go", profiles[1].FileName)
}
