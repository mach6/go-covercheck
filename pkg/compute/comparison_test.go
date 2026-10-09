package compute_test

import (
	"testing"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/stretchr/testify/require"
)

func ptr(f float64) *float64 { return &f }

func cov(s, b, l float64) compute.By {
	return compute.By{StatementPercentage: s, BlockPercentage: b, LinePercentage: l, Lines: "1/2"}
}

func prevResults(withLines bool) compute.Results {
	by := func(s, b, l float64) compute.By {
		by := compute.By{StatementPercentage: s, BlockPercentage: b, LinePercentage: l}
		if withLines {
			by.Lines = "1/2"
		}
		return by
	}
	totals := compute.Totals{}
	totals.Statements.Percentage = 50
	totals.Blocks.Percentage = 50
	totals.Lines.Percentage = 50
	if withLines {
		totals.Lines.Coverage = "1/2"
	}
	return compute.Results{
		ByFile: []compute.ByFile{
			{File: "a.go", By: by(50, 50, 50)},
			{File: "same.go", By: by(50, 50, 50)},
			{File: "gone.go", By: by(10, 10, 10)},
		},
		ByPackage: []compute.ByPackage{{Package: "p", By: by(50, 50, 50)}},
		ByTotal:   totals,
	}
}

func currResults() compute.Results {
	by := func(s, b, l float64) compute.By {
		return compute.By{StatementPercentage: s, BlockPercentage: b, LinePercentage: l}
	}
	totals := compute.Totals{}
	totals.Statements.Percentage = 75
	totals.Blocks.Percentage = 40
	totals.Lines.Percentage = 60
	return compute.Results{
		ByFile: []compute.ByFile{
			{File: "a.go", By: by(75, 40, 60)},
			{File: "same.go", By: by(50, 50, 50)},
			{File: "new.go", By: by(90, 90, 0)},
		},
		ByPackage: []compute.ByPackage{{Package: "p", By: by(75, 40, 60)}},
		ByTotal:   totals,
	}
}

func TestBuildComparison(t *testing.T) {
	c := compute.BuildComparison("main", "0123456", prevResults(true), currResults())

	require.Equal(t, &compute.Comparison{
		Ref:    "main",
		Commit: "0123456",
		// same.go is unchanged. new.go is added and gone.go is removed.
		ByFile: []compute.FileDelta{
			{File: "a.go", Delta: compute.Delta{Statements: 25, Blocks: -10, Lines: ptr(10)}},
		},
		ByPackage: []compute.PackageDelta{
			{Package: "p", Delta: compute.Delta{Statements: 25, Blocks: -10, Lines: ptr(10)}},
		},
		Added: compute.Changes{
			Files:    []compute.FileCoverage{{File: "new.go", Coverage: compute.Coverage{Statements: 90, Blocks: 90}}},
			Packages: []compute.PackageCoverage{},
		},
		Removed: compute.Changes{
			Files: []compute.FileCoverage{
				{File: "gone.go", Coverage: compute.Coverage{Statements: 10, Blocks: 10, Lines: ptr(10)}},
			},
			Packages: []compute.PackageCoverage{},
		},
		ByTotal: &compute.Delta{Statements: 25, Blocks: -10, Lines: ptr(10)},
	}, c)
}

func TestBuildComparison_HistoryWithoutLines(t *testing.T) {
	c := compute.BuildComparison("main", "0123456", prevResults(false), currResults())

	require.Len(t, c.ByFile, 1)
	require.Nil(t, c.ByFile[0].Lines)
	require.Nil(t, c.ByPackage[0].Lines)
	require.Nil(t, c.ByTotal.Lines)
}

func TestBuildComparison_NoChange(t *testing.T) {
	prev := prevResults(true)
	c := compute.BuildComparison("main", "0123456", prev, prev)

	// lists are empty rather than null, so JSON consumers can always iterate.
	require.Equal(t, []compute.FileDelta{}, c.ByFile)
	require.Equal(t, []compute.PackageDelta{}, c.ByPackage)
	require.Equal(t, compute.Changes{Files: []compute.FileCoverage{}, Packages: []compute.PackageCoverage{}}, c.Added)
	require.Equal(t, compute.Changes{Files: []compute.FileCoverage{}, Packages: []compute.PackageCoverage{}}, c.Removed)
	require.Nil(t, c.ByTotal)
}

func TestBuildComparison_LinesOnlyChange(t *testing.T) {
	prev := prevResults(true)
	curr := prev
	curr.ByTotal.Lines.Percentage = 55

	c := compute.BuildComparison("main", "0123456", prev, curr)

	require.Empty(t, c.ByFile)
	require.Equal(t, &compute.Delta{Lines: ptr(5)}, c.ByTotal)
}

func TestBuildComparison_UsesGivenCommit(t *testing.T) {
	for _, commit := range []string{"", "abc", "0123456"} {
		c := compute.BuildComparison("main", commit, compute.Results{}, compute.Results{})
		require.Equal(t, commit, c.Commit)
		require.Equal(t, "main", c.Ref)
	}
}

func TestBuildComparison_AddedOnly(t *testing.T) {
	prev := compute.Results{ByFile: []compute.ByFile{{File: "a.go", By: cov(50, 50, 50)}}}
	curr := compute.Results{ByFile: []compute.ByFile{
		{File: "b.go", By: cov(80, 70, 60)},
		{File: "a.go", By: cov(50, 50, 50)},
		{File: "0.go", By: cov(1, 2, 3)},
	}}
	c := compute.BuildComparison("main", "0123456", prev, curr)

	// sorted by name, not results order.
	require.Equal(t, []compute.FileCoverage{
		{File: "0.go", Coverage: compute.Coverage{Statements: 1, Blocks: 2, Lines: ptr(3)}},
		{File: "b.go", Coverage: compute.Coverage{Statements: 80, Blocks: 70, Lines: ptr(60)}},
	}, c.Added.Files)
	require.Empty(t, c.Removed.Files)
	require.Empty(t, c.ByFile)
}

func TestBuildComparison_RemovedOnly(t *testing.T) {
	prev := compute.Results{ByFile: []compute.ByFile{
		{File: "z.go", By: cov(10, 20, 30)},
		{File: "a.go", By: cov(50, 50, 50)},
		{File: "keep.go", By: cov(50, 50, 50)},
	}}
	curr := compute.Results{ByFile: []compute.ByFile{{File: "keep.go", By: cov(50, 50, 50)}}}
	c := compute.BuildComparison("main", "0123456", prev, curr)

	require.Empty(t, c.Added.Files)
	require.Equal(t, []compute.FileCoverage{
		{File: "a.go", Coverage: compute.Coverage{Statements: 50, Blocks: 50, Lines: ptr(50)}},
		{File: "z.go", Coverage: compute.Coverage{Statements: 10, Blocks: 20, Lines: ptr(30)}},
	}, c.Removed.Files)
}

func TestBuildComparison_RenamedFile(t *testing.T) {
	prev := compute.Results{ByFile: []compute.ByFile{{File: "old.go", By: cov(50, 50, 50)}}}
	curr := compute.Results{ByFile: []compute.ByFile{{File: "new.go", By: cov(50, 50, 50)}}}
	c := compute.BuildComparison("main", "0123456", prev, curr)

	require.Len(t, c.Added.Files, 1)
	require.Equal(t, "new.go", c.Added.Files[0].File)
	require.Len(t, c.Removed.Files, 1)
	require.Equal(t, "old.go", c.Removed.Files[0].File)
	require.Empty(t, c.ByFile)
}

func TestBuildComparison_PackageAddedWithItsFiles(t *testing.T) {
	prev := compute.Results{
		ByFile:    []compute.ByFile{{File: "p/a.go", By: cov(50, 50, 50)}},
		ByPackage: []compute.ByPackage{{Package: "p", By: cov(50, 50, 50)}},
	}
	curr := compute.Results{
		ByFile: []compute.ByFile{
			{File: "p/a.go", By: cov(50, 50, 50)},
			{File: "q/b.go", By: cov(70, 70, 70)},
			{File: "q/c.go", By: cov(90, 90, 90)},
		},
		ByPackage: []compute.ByPackage{
			{Package: "p", By: cov(50, 50, 50)},
			{Package: "q", By: cov(80, 80, 80)},
		},
	}
	c := compute.BuildComparison("main", "0123456", prev, curr)

	require.Equal(t, []compute.PackageCoverage{
		{Package: "q", Coverage: compute.Coverage{Statements: 80, Blocks: 80, Lines: ptr(80)}},
	}, c.Added.Packages)
	require.Len(t, c.Added.Files, 2)
	require.Empty(t, c.Removed.Packages)
	require.Empty(t, c.ByPackage)
}

func TestBuildComparison_HistoryWithoutLinesRemoved(t *testing.T) {
	by := compute.By{StatementPercentage: 10, BlockPercentage: 20}
	prev := compute.Results{
		ByFile:    []compute.ByFile{{File: "gone.go", By: by}},
		ByPackage: []compute.ByPackage{{Package: "gone", By: by}},
	}
	c := compute.BuildComparison("main", "0123456", prev, compute.Results{})

	require.Equal(t, compute.Coverage{Statements: 10, Blocks: 20}, c.Removed.Files[0].Coverage)
	require.Nil(t, c.Removed.Files[0].Lines)
	require.Nil(t, c.Removed.Packages[0].Lines)
}

func TestBuildComparison_PackageDeltaWithoutChange(t *testing.T) {
	prev := compute.Results{ByPackage: []compute.ByPackage{{Package: "p", By: cov(50, 50, 50)}}}
	curr := compute.Results{ByPackage: []compute.ByPackage{{Package: "p", By: cov(50, 50, 52)}}}
	c := compute.BuildComparison("main", "0123456", prev, curr)

	require.Equal(t, []compute.PackageDelta{{Package: "p", Delta: compute.Delta{Lines: ptr(2)}}}, c.ByPackage)
}
