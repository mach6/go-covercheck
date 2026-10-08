package compute //nolint:testpackage

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
	"gopkg.in/yaml.v3"
)

// functionProfiles returns profiles for the sources under testdata/: package
// alpha declares One and Two in alpha.go and Three in three.go; package beta
// declares the method T.M. Only One and T.M executed.
func functionProfiles(prefix string) []*cover.Profile {
	return []*cover.Profile{
		{
			FileName: prefix + "testdata/alpha/alpha.go",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 3, StartCol: 16, EndLine: 5, EndCol: 2, NumStmt: 1, Count: 1},
				{StartLine: 7, StartCol: 16, EndLine: 9, EndCol: 2, NumStmt: 1, Count: 0},
			},
		},
		{
			FileName: prefix + "testdata/alpha/three.go",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 3, StartCol: 18, EndLine: 5, EndCol: 2, NumStmt: 1, Count: 0},
			},
		},
		{
			FileName: prefix + "testdata/beta/beta.go",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 5, StartCol: 15, EndLine: 5, EndCol: 16, NumStmt: 0, Count: 1},
			},
		},
	}
}

func fileByName(t *testing.T, r Results, name string) ByFile {
	t.Helper()
	for _, f := range r.ByFile {
		if f.File == name {
			return f
		}
	}
	require.Failf(t, "file not found", "%s", name)
	return ByFile{}
}

func packageByName(t *testing.T, r Results, name string) ByPackage {
	t.Helper()
	for _, p := range r.ByPackage {
		if p.Package == name {
			return p
		}
	}
	require.Failf(t, "package not found", "%s", name)
	return ByPackage{}
}

func TestCollectResults_FunctionCoverage(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()

	r, _ := CollectResults(functionProfiles(""), cfg)

	alpha := fileByName(t, r, "testdata/alpha/alpha.go")
	require.Equal(t, "1/2", alpha.Functions)
	require.InDelta(t, 50.0, alpha.FunctionPercentage, 0.01)

	three := fileByName(t, r, "testdata/alpha/three.go")
	require.Equal(t, "0/1", three.Functions)
	require.InDelta(t, 0.0, three.FunctionPercentage, 0.01)
	require.Zero(t, three.FunctionThreshold, "function threshold is disabled by default")

	beta := fileByName(t, r, "testdata/beta/beta.go")
	require.Equal(t, "1/1", beta.Functions, "zero-statement method still counts as executed")

	alphaPkg := packageByName(t, r, "testdata/alpha")
	require.Equal(t, "1/3", alphaPkg.Functions)
	require.InDelta(t, 33.33, alphaPkg.FunctionPercentage, 0.01)
	require.Equal(t, "1/1", packageByName(t, r, "testdata/beta").Functions)

	require.Equal(t, "2/4", r.ByTotal.Functions.Coverage)
	require.InDelta(t, 50.0, r.ByTotal.Functions.Percentage, 0.01)
	require.False(t, r.ByTotal.Functions.Failed)
}

// Profiles usually carry import paths (module prefix included); the source
// must be found the same way line coverage finds it.
func TestCollectResults_FunctionCoverageResolvesImportPaths(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()

	r, _ := CollectResults(functionProfiles("example.com/mod/pkg/compute/"), cfg)
	require.Equal(t, "2/4", r.ByTotal.Functions.Coverage)
}

func TestCollectResults_FunctionCoverageMissingSource(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()
	cfg.FunctionThreshold = 90
	require.NoError(t, cfg.Validate())

	profiles := []*cover.Profile{{
		FileName: "does/not/exist.go",
		Mode:     "set",
		Blocks:   []cover.ProfileBlock{{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 1, Count: 1}},
	}}

	r, failed := CollectResults(profiles, cfg)
	require.False(t, failed)
	require.Equal(t, "0/0", r.ByFile[0].Functions)
	require.Equal(t, "0/0", r.ByTotal.Functions.Coverage)
	require.InDelta(t, 100.0, r.ByTotal.Functions.Percentage, 0.01, "0/0 is neutral, not a failure")
}

func TestCollectResults_SourceWarnings(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()

	block := []cover.ProfileBlock{{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 1, Count: 1}}

	t.Run("found source has no warnings", func(t *testing.T) {
		r, _ := CollectResults(functionProfiles(""), cfg)
		require.Empty(t, r.Warnings)
	})

	t.Run("import path resolved to source has no warnings", func(t *testing.T) {
		r, _ := CollectResults(functionProfiles("example.com/mod/pkg/compute/"), cfg)
		require.Empty(t, r.Warnings)
	})

	t.Run("missing source warns once per file", func(t *testing.T) {
		profiles := append(functionProfiles(""),
			&cover.Profile{FileName: "does/not/exist.go", Mode: "set", Blocks: block},
			&cover.Profile{FileName: "also/missing.go", Mode: "set", Blocks: block},
		)
		r, _ := CollectResults(profiles, cfg)
		require.Len(t, r.Warnings, 2)
		joined := strings.Join(r.Warnings, "\n")
		require.Contains(t, joined, "does/not/exist.go: source file not found")
		require.Contains(t, joined, "also/missing.go: source file not found")
		require.Contains(t, joined, "function coverage unavailable")
		require.Equal(t, "2/4", r.ByTotal.Functions.Coverage, "0/0 files leave the totals unchanged")
	})

	t.Run("unparsable source warns with the parse reason", func(t *testing.T) {
		profiles := []*cover.Profile{{FileName: "testdata/broken/broken.go", Mode: "set", Blocks: block}}
		r, failed := CollectResults(profiles, cfg)
		require.False(t, failed)
		require.Len(t, r.Warnings, 1)
		require.Contains(t, r.Warnings[0], "testdata/broken/broken.go: source file could not be parsed (")
		require.Contains(t, r.Warnings[0], "broken.go:3")
		require.Equal(t, "0/0", r.ByFile[0].Functions)
	})

	t.Run("warnings stay out of structured output", func(t *testing.T) {
		profiles := []*cover.Profile{{FileName: "does/not/exist.go", Mode: "set", Blocks: block}}
		r, _ := CollectResults(profiles, cfg)
		require.NotEmpty(t, r.Warnings)

		j, err := json.Marshal(r)
		require.NoError(t, err)
		require.NotContains(t, string(j), "exist.go: source")
		y, err := yaml.Marshal(r)
		require.NoError(t, err)
		require.NotContains(t, string(y), "exist.go: source")
	})
}

func TestCollectResults_FunctionThresholds(t *testing.T) {
	tests := []struct {
		name          string
		configure     func(cfg *config.Config)
		failed        bool
		alphaFailed   bool
		alphaPkgFail  bool
		totalFailed   bool
		alphaFileWant float64
	}{
		{
			name:      "disabled",
			configure: func(*config.Config) {},
		},
		{
			name: "global threshold fails files, packages and total",
			configure: func(cfg *config.Config) {
				cfg.FunctionThreshold = 60
			},
			failed: true, alphaFailed: true, alphaPkgFail: true, totalFailed: true, alphaFileWant: 60,
		},
		{
			name: "per-file override relaxes one file",
			configure: func(cfg *config.Config) {
				cfg.FunctionThreshold = 60
				cfg.PerFile.Functions = config.PerOverride{"testdata/alpha/alpha.go": 50}
			},
			failed: true, alphaPkgFail: true, totalFailed: true, alphaFileWant: 50,
		},
		{
			name: "per-package override",
			configure: func(cfg *config.Config) {
				cfg.PerPackage.Functions = config.PerOverride{"testdata/alpha": 40}
			},
			failed: true, alphaPkgFail: true,
		},
		{
			name: "total only",
			configure: func(cfg *config.Config) {
				cfg.Total = config.PerOverride{config.FunctionsSection: 51}
			},
			failed: true, totalFailed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.ApplyDefaults()
			cfg.Total = nil
			tt.configure(cfg)
			require.NoError(t, cfg.Validate())
			// Isolate function thresholds from the other metrics.
			cfg.StatementThreshold, cfg.BlockThreshold, cfg.LineThreshold = 0, 0, 0
			cfg.Total[config.StatementsSection] = 0
			cfg.Total[config.BlocksSection] = 0
			cfg.Total[config.LinesSection] = 0

			r, failed := CollectResults(functionProfiles(""), cfg)
			require.Equal(t, tt.failed, failed)

			alpha := fileByName(t, r, "testdata/alpha/alpha.go")
			require.Equal(t, tt.alphaFailed, alpha.Failed)
			require.InDelta(t, tt.alphaFileWant, alpha.FunctionThreshold, 0.01)
			require.Equal(t, tt.alphaPkgFail, packageByName(t, r, "testdata/alpha").Failed)
			require.False(t, packageByName(t, r, "testdata/beta").Failed)
			require.Equal(t, tt.totalFailed, r.ByTotal.Functions.Failed)
		})
	}
}

func TestSortResults_ByFunctions(t *testing.T) {
	cfg := &config.Config{}
	cfg.ApplyDefaults()
	r, _ := CollectResults(functionProfiles(""), cfg)

	// alpha.go 1/2 (50%), three.go 0/1 (0%), beta.go 1/1 (100%).
	testFileSorting(t, config.SortByFunctionPercent, r.ByFile,
		[]string{"testdata/alpha/three.go", "testdata/alpha/alpha.go", "testdata/beta/beta.go"},
		[]string{"testdata/beta/beta.go", "testdata/alpha/alpha.go", "testdata/alpha/three.go"})

	cfg.SortBy = config.SortByFunctions
	sortFileResults(r.ByFile, cfg)
	require.Equal(t, "testdata/alpha/three.go", r.ByFile[0].File)

	cfg.SortBy = config.SortByFunctionPercent
	cfg.SortOrder = config.SortOrderDesc
	sortPackageResults(r.ByPackage, cfg)
	require.Equal(t, "testdata/beta", r.ByPackage[0].Package)
}
