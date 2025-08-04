package compute //nolint:testpackage

import (
	"github.com/mach6/go-covercheck/pkg/test"
	"path/filepath"
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

func Test_readModuleNameFromGoMod_Success(t *testing.T) {
	goModContent := `module github.com/example/myproject

go 1.21

require (
	github.com/stretchr/testify v1.8.0
)
`
	p := test.CreateTempFile(t, "go.mod", goModContent)
	t.Chdir(filepath.Dir(p))

	moduleName := readModuleNameFromGoMod()
	require.Equal(t, "github.com/example/myproject", moduleName)
}

func Test_readModuleNameFromGoMod_NoFile(t *testing.T) {
	t.Chdir(t.TempDir())

	moduleName := readModuleNameFromGoMod()
	require.Empty(t, moduleName)
}

func Test_readModuleNameFromGoMod_InvalidFormat(t *testing.T) {
	goModContent := `// no module declaration
go 1.21
`
	p := test.CreateTempFile(t, "go.mod", goModContent)
	t.Chdir(path.Dir(p))

	moduleName := readModuleNameFromGoMod()
	require.Empty(t, moduleName)
}

func Test_validateModuleNameMatchesFilePaths_Success(t *testing.T) {
	profiles := []*cover.Profile{
		{FileName: "github.com/example/myproject/pkg/foo/foo.go"},
		{FileName: "github.com/example/myproject/pkg/bar/bar.go"},
		{FileName: "github.com/example/myproject/internal/baz.go"},
	}

	valid := validateModuleNameMatchesFilePaths("github.com/example/myproject", profiles)
	require.True(t, valid)
}

func Test_validateModuleNameMatchesFilePaths_Failure(t *testing.T) {
	profiles := []*cover.Profile{
		{FileName: "github.com/example/myproject/pkg/foo/foo.go"},
		{FileName: "github.com/different/project/pkg/bar/bar.go"}, // different module
	}

	valid := validateModuleNameMatchesFilePaths("github.com/example/myproject", profiles)
	require.False(t, valid)
}

func Test_validateModuleNameMatchesFilePaths_EmptyModule(t *testing.T) {
	profiles := []*cover.Profile{
		{FileName: "github.com/example/myproject/pkg/foo/foo.go"},
	}

	valid := validateModuleNameMatchesFilePaths("", profiles)
	require.False(t, valid)
}

func Test_validateModuleNameMatchesFilePaths_EmptyProfiles(t *testing.T) {
	valid := validateModuleNameMatchesFilePaths("github.com/example/myproject", []*cover.Profile{})
	require.False(t, valid)
}

func Test_findModuleName_withGoMod_Success(t *testing.T) {
	goModContent := `module github.com/example/myproject

go 1.21
`
	p := test.CreateTempFile(t, "go.mod", goModContent)
	t.Chdir(path.Dir(p))

	profiles := []*cover.Profile{
		{FileName: "github.com/example/myproject/pkg/foo/foo.go"},
		{FileName: "github.com/example/myproject/pkg/bar/bar.go"},
	}

	cfg := &config.Config{} // No configured module name
	moduleName := findModuleName(profiles, cfg)
	require.Equal(t, "github.com/example/myproject/", moduleName)
}

func Test_findModuleName_withGoMod_Mismatch(t *testing.T) {
	goModContent := `module github.com/example/myproject

go 1.21
`
	p := test.CreateTempFile(t, "go.mod", goModContent)
	t.Chdir(path.Dir(p))

	// File paths don't match the module name
	profiles := []*cover.Profile{
		{FileName: "github.com/different/project/pkg/foo/foo.go"},
		{FileName: "github.com/different/project/pkg/bar/bar.go"},
	}

	cfg := &config.Config{} // No configured module name
	moduleName := findModuleName(profiles, cfg)
	// Should fall back to longest common prefix
	require.Equal(t, "github.com/different/project/pkg/", moduleName)
}

func Test_findModuleName_priorityOrder(t *testing.T) {
	goModContent := `module github.com/example/myproject

go 1.21
`
	p := test.CreateTempFile(t, "go.mod", goModContent)
	t.Chdir(filepath.Dir(p))

	profiles := []*cover.Profile{
		{FileName: "github.com/example/myproject/pkg/foo/foo.go"},
		{FileName: "github.com/example/myproject/pkg/bar/bar.go"},
	}

	// Test that configured module name takes priority over go.mod
	cfg := &config.Config{
		ModuleName: "github.com/configured/module",
	}
	moduleName := findModuleName(profiles, cfg)
	require.Equal(t, "github.com/configured/module/", moduleName)
}
