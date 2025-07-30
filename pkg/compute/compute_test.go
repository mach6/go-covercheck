package compute //nolint:testpackage

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/cover"
)

func Test_sortResults_ByFile(t *testing.T) {
	results := []ByFile{
		{
			File: "b",
			By: By{
				Statements:          "2/2",
				Blocks:              "2/2",
				StatementPercentage: 100,
				BlockPercentage:     100,
				StatementThreshold:  0,
				BlockThreshold:      0,
				Failed:              false,
			},
		},
		{
			File: "a",
			By: By{
				Statements:          "1/2",
				Blocks:              "1/2",
				StatementPercentage: 50,
				BlockPercentage:     50,
				StatementThreshold:  0,
				BlockThreshold:      0,
				Failed:              false,
			},
		},
		{
			File: "c",
			By: By{
				Statements:          "0/3",
				Blocks:              "0/3",
				StatementPercentage: 0,
				BlockPercentage:     0,
				StatementThreshold:  0,
				BlockThreshold:      0,
				Failed:              true,
			},
		},
	}

	cfg := new(config.Config)

	// test ascending
	cfg.ApplyDefaults()
	sortFileResults(results, cfg)
	expect := []string{"a", "b", "c"}
	for i, v := range results {
		require.Equal(t, expect[i], v.File)
	}

	// test descending
	cfg.SortOrder = config.SortOrderDesc
	sortFileResults(results, cfg)
	expect = []string{"c", "b", "a"}
	for i, v := range results {
		require.Equal(t, expect[i], v.File)
	}
}

func TestCollectResults(t *testing.T) {
	profiles := make([]*cover.Profile, 0)
	profiles = append(profiles, &cover.Profile{
		FileName: "foo",
		Mode:     "set",
		Blocks: []cover.ProfileBlock{
			{
				StartLine: 0,
				StartCol:  0,
				EndLine:   10,
				EndCol:    120,
				NumStmt:   150,
				Count:     1,
			},
		},
	})
	r, failed := CollectResults(profiles, new(config.Config))
	require.False(t, failed)
	require.NotNil(t, r)
	require.InEpsilon(t, 100.0, r.ByTotal.Blocks.Percentage, 0)
	require.InEpsilon(t, 100.0, r.ByTotal.Statements.Percentage, 0)
}

func TestCollectResults_WithFailures(t *testing.T) {
	profiles := []*cover.Profile{
		{
			FileName: "pkg/low-coverage.go",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 1, Count: 0}, // uncovered
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStmt: 1, Count: 1}, // covered
			},
		},
	}

	cfg := &config.Config{
		StatementThreshold: 75.0, // 50% coverage will fail this
		BlockThreshold:     75.0, // 50% coverage will fail this
	}
	cfg.ApplyDefaults()

	r, failed := CollectResults(profiles, cfg)
	require.True(t, failed)
	require.True(t, r.ByFile[0].Failed)
	require.InEpsilon(t, 50.0, r.ByFile[0].StatementPercentage, 0.01)
	require.InEpsilon(t, 50.0, r.ByFile[0].BlockPercentage, 0.01)
}

func TestCollectResults_WithPerFileThresholds(t *testing.T) {
	profiles := []*cover.Profile{
		{
			FileName: "pkg/special.go",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 1, Count: 1}, // covered
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStmt: 1, Count: 0}, // uncovered
			},
		},
	}

	cfg := &config.Config{
		StatementThreshold: 10.0, // default low threshold
		BlockThreshold:     10.0, // default low threshold
		PerFile: config.PerThresholdOverride{
			Statements: config.PerOverride{"pkg/special.go": 75.0}, // higher threshold for this file
			Blocks:     config.PerOverride{"pkg/special.go": 75.0}, // higher threshold for this file
		},
	}
	cfg.ApplyDefaults()

	r, failed := CollectResults(profiles, cfg)
	require.True(t, failed)
	require.True(t, r.ByFile[0].Failed)
	require.InEpsilon(t, 75.0, r.ByFile[0].StatementThreshold, 0.01) // should use a per-file threshold
	require.InEpsilon(t, 75.0, r.ByFile[0].BlockThreshold, 0.01)     // should use a per-file threshold
}

func TestCollectResults_WithPerPackageThresholds(t *testing.T) {
	profiles := []*cover.Profile{
		{
			FileName: "pkg/special/file1.go",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 1, Count: 1}, // covered
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStmt: 1, Count: 0}, // uncovered
			},
		},
		{
			FileName: "pkg/other/file.go", // different package to avoid common prefix normalization
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 1, Count: 1}, // covered
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStmt: 1, Count: 1}, // covered - this package will pass
			},
		},
	}

	cfg := &config.Config{}
	cfg.ApplyDefaults()
	cfg.StatementThreshold = 10.0               // set after applying defaults
	cfg.BlockThreshold = 10.0                   // set after applying defaults
	cfg.PerPackage.Statements["special"] = 75.0 // higher threshold for special package only
	cfg.PerPackage.Blocks["special"] = 75.0     // higher threshold for special package only

	r, failed := CollectResults(profiles, cfg)
	require.True(t, failed)

	// Should have results for both packages
	require.Len(t, r.ByPackage, 2)

	// Find the special package result
	var specialPkg, otherPkg *ByPackage
	for i := range r.ByPackage {
		switch r.ByPackage[i].Package {
		case "special":
			specialPkg = &r.ByPackage[i]
		case "other":
			otherPkg = &r.ByPackage[i]
		}
	}

	require.NotNil(t, specialPkg)
	require.NotNil(t, otherPkg)

	// Special package should fail due to a higher threshold
	require.True(t, specialPkg.Failed)
	require.InEpsilon(t, 75.0, specialPkg.StatementThreshold, 0.01) // should use a per-package threshold
	require.InEpsilon(t, 75.0, specialPkg.BlockThreshold, 0.01)     // should use a per-package threshold

	// Another package should pass with a default threshold
	require.False(t, otherPkg.Failed)
	require.InEpsilon(t, 10.0, otherPkg.StatementThreshold, 0.01) // should use a default threshold
	require.InEpsilon(t, 10.0, otherPkg.BlockThreshold, 0.01)     // should use a default threshold
}

func TestCollectResults_WithMultiplePackages(t *testing.T) {
	profiles := []*cover.Profile{
		{
			FileName: "pkg/a/file.go",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 2, Count: 1},
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStmt: 3, Count: 1},
			},
		},
		{
			FileName: "pkg/b/file.go",
			Mode:     "set",
			Blocks: []cover.ProfileBlock{
				{StartLine: 1, StartCol: 1, EndLine: 1, EndCol: 10, NumStmt: 1, Count: 0},
				{StartLine: 2, StartCol: 1, EndLine: 2, EndCol: 10, NumStmt: 4, Count: 1},
			},
		},
	}

	cfg := &config.Config{}
	cfg.ApplyDefaults()

	r, failed := CollectResults(profiles, cfg)
	require.False(t, failed)
	require.Len(t, r.ByPackage, 2)
	require.Len(t, r.ByFile, 2)

	// Verify totals are aggregated correctly

	// 2+3+1+4 = 10 statements, 1 statement not covered so 9/10
	require.Equal(t, "9/10", r.ByTotal.Statements.Coverage)
	// 4 blocks, 3 covered
	require.Equal(t, "3/4", r.ByTotal.Blocks.Coverage)
}

// Helper function to test sorting of file results.
func testFileSorting(t *testing.T, sortBy string, results []ByFile, expectedAsc, expectedDesc []string) {
	t.Helper()

	// Test ascending
	cfg := &config.Config{SortBy: sortBy, SortOrder: config.SortOrderAsc}
	sortFileResults(results, cfg)
	for i, expected := range expectedAsc {
		require.Equal(t, expected, results[i].File)
	}

	// Test descending
	cfg.SortOrder = config.SortOrderDesc
	sortFileResults(results, cfg)
	for i, expected := range expectedDesc {
		require.Equal(t, expected, results[i].File)
	}
}

func TestSortResults_ByStatementPercent(t *testing.T) {
	results := []ByFile{
		{File: "high.go", By: By{StatementPercentage: 90.0, stmtHits: 9}},
		{File: "low.go", By: By{StatementPercentage: 10.0, stmtHits: 1}},
		{File: "medium.go", By: By{StatementPercentage: 50.0, stmtHits: 5}},
	}

	testFileSorting(t, config.SortByStatementPercent, results,
		[]string{"low.go", "medium.go", "high.go"},
		[]string{"high.go", "medium.go", "low.go"})
}

func TestSortResults_ByBlockPercent(t *testing.T) {
	results := []ByFile{
		{File: "high.go", By: By{BlockPercentage: 90.0, blockHits: 9}},
		{File: "low.go", By: By{BlockPercentage: 10.0, blockHits: 1}},
		{File: "medium.go", By: By{BlockPercentage: 50.0, blockHits: 5}},
	}

	testFileSorting(t, config.SortByBlockPercent, results,
		[]string{"low.go", "medium.go", "high.go"},
		[]string{"high.go", "medium.go", "low.go"})
}

func TestSortResults_ByStatements(t *testing.T) {
	results := []ByFile{
		{File: "high.go", By: By{stmtHits: 90}},
		{File: "low.go", By: By{stmtHits: 10}},
		{File: "medium.go", By: By{stmtHits: 50}},
	}

	testFileSorting(t, config.SortByStatements, results,
		[]string{"low.go", "medium.go", "high.go"},
		[]string{"high.go", "medium.go", "low.go"})
}

func TestSortResults_ByBlocks(t *testing.T) {
	results := []ByFile{
		{File: "high.go", By: By{blockHits: 90}},
		{File: "low.go", By: By{blockHits: 10}},
		{File: "medium.go", By: By{blockHits: 50}},
	}

	testFileSorting(t, config.SortByBlocks, results,
		[]string{"low.go", "medium.go", "high.go"},
		[]string{"high.go", "medium.go", "low.go"})
}

// Helper function to test sorting of package results.
func testPackageSorting(t *testing.T, sortBy string, results []ByPackage, expectedAsc, expectedDesc []string) {
	t.Helper()

	// Test ascending
	cfg := &config.Config{SortBy: sortBy, SortOrder: config.SortOrderAsc}
	sortPackageResults(results, cfg)
	for i, expected := range expectedAsc {
		require.Equal(t, expected, results[i].Package)
	}

	// Test descending
	cfg.SortOrder = config.SortOrderDesc
	sortPackageResults(results, cfg)
	for i, expected := range expectedDesc {
		require.Equal(t, expected, results[i].Package)
	}
}

func TestSortPackageResults_ByStatementPercent(t *testing.T) {
	results := []ByPackage{
		{Package: "pkg/high", By: By{StatementPercentage: 90.0, stmtHits: 9}},
		{Package: "pkg/low", By: By{StatementPercentage: 10.0, stmtHits: 1}},
		{Package: "pkg/medium", By: By{StatementPercentage: 50.0, stmtHits: 5}},
	}

	testPackageSorting(t, config.SortByStatementPercent, results,
		[]string{"pkg/low", "pkg/medium", "pkg/high"},
		[]string{"pkg/high", "pkg/medium", "pkg/low"})
}

func TestSortPackageResults_ByPackageName(t *testing.T) {
	results := []ByPackage{
		{Package: "pkg/z"},
		{Package: "pkg/a"},
		{Package: "pkg/m"},
	}

	// Test ascending (default sort by package name)
	cfg := &config.Config{SortBy: config.SortByFile, SortOrder: config.SortOrderAsc}
	sortPackageResults(results, cfg)
	require.Equal(t, "pkg/a", results[0].Package)
	require.Equal(t, "pkg/m", results[1].Package)
	require.Equal(t, "pkg/z", results[2].Package)

	// Test descending
	cfg.SortOrder = config.SortOrderDesc
	sortPackageResults(results, cfg)
	require.Equal(t, "pkg/z", results[0].Package)
	require.Equal(t, "pkg/m", results[1].Package)
	require.Equal(t, "pkg/a", results[2].Package)
}
