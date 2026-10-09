package compute

import "sort"

// Delta is the change in coverage percentage points, current minus historical.
// A positive value means coverage went up.
type Delta struct {
	Statements float64  `json:"statements"      yaml:"statements"`
	Blocks     float64  `json:"blocks"          yaml:"blocks"`
	Lines      *float64 `json:"lines,omitempty" yaml:"lines,omitempty"`
}

// FileDelta is the change in coverage for one file.
type FileDelta struct {
	File  string `json:"file" yaml:"file"`
	Delta `yaml:",inline"`
}

// PackageDelta is the change in coverage for one package.
type PackageDelta struct {
	Package string `json:"package" yaml:"package"`
	Delta   `yaml:",inline"`
}

// Coverage is a coverage percentage, not a change. Lines is left out when it is not known.
type Coverage struct {
	Statements float64  `json:"statements"      yaml:"statements"`
	Blocks     float64  `json:"blocks"          yaml:"blocks"`
	Lines      *float64 `json:"lines,omitempty" yaml:"lines,omitempty"`
}

// FileCoverage is the coverage of one file that was added or removed.
type FileCoverage struct {
	File     string `json:"file" yaml:"file"`
	Coverage `yaml:",inline"`
}

// PackageCoverage is the coverage of one package that was added or removed.
type PackageCoverage struct {
	Package  string `json:"package" yaml:"package"`
	Coverage `yaml:",inline"`
}

// Changes lists files and packages that are in only one of the two runs.
// Both lists are sorted by name and are never nil.
type Changes struct {
	Files    []FileCoverage    `json:"files"    yaml:"files"`
	Packages []PackageCoverage `json:"packages" yaml:"packages"`
}

// Comparison holds the coverage changes between the current results and an earlier run.
// ByFile and ByPackage list items present in both, whose coverage changed.
// Added holds items only in the current results, with their current coverage.
// Removed holds items only in the history entry, with their historical coverage.
type Comparison struct {
	Ref       string         `json:"ref"                yaml:"ref"`
	Commit    string         `json:"commit"             yaml:"commit"`
	ByFile    []FileDelta    `json:"byFile"             yaml:"byFile"`
	ByPackage []PackageDelta `json:"byPackage"          yaml:"byPackage"`
	Added     Changes        `json:"added"              yaml:"added"`
	Removed   Changes        `json:"removed"            yaml:"removed"`
	ByTotal   *Delta         `json:"byTotal,omitempty"  yaml:"byTotal,omitempty"`
}

func newCoverage(by By) Coverage {
	c := Coverage{Statements: by.StatementPercentage, Blocks: by.BlockPercentage}
	if by.Lines != "" {
		l := by.LinePercentage
		c.Lines = &l
	}
	return c
}

// newDelta returns the delta and whether any part of it is non-zero.
// Line coverage is only compared when the historical entry has it.
func newDelta(statements, blocks float64, lines *float64) (Delta, bool) {
	d := Delta{Statements: statements, Blocks: blocks, Lines: lines}
	return d, statements != 0 || blocks != 0 || (lines != nil && *lines != 0)
}

func byDelta(curr, prev By) (Delta, bool) {
	var lines *float64
	if prev.Lines != "" {
		l := curr.LinePercentage - prev.LinePercentage
		lines = &l
	}
	return newDelta(curr.StatementPercentage-prev.StatementPercentage,
		curr.BlockPercentage-prev.BlockPercentage, lines)
}

func totalDelta(curr, prev Totals) (Delta, bool) {
	var lines *float64
	if prev.Lines.Coverage != "" {
		l := curr.Lines.Percentage - prev.Lines.Percentage
		lines = &l
	}
	return newDelta(curr.Statements.Percentage-prev.Statements.Percentage,
		curr.Blocks.Percentage-prev.Blocks.Percentage, lines)
}

// BuildComparison compares curr against prev, the results of the earlier run found for ref.
// commit is the short commit hash of that run.
func BuildComparison(ref, commit string, prev, curr Results) *Comparison {
	c := &Comparison{
		Ref:       ref,
		Commit:    commit,
		ByFile:    []FileDelta{},
		ByPackage: []PackageDelta{},
		Added:     Changes{Files: []FileCoverage{}, Packages: []PackageCoverage{}},
		Removed:   Changes{Files: []FileCoverage{}, Packages: []PackageCoverage{}},
	}

	c.compareFiles(prev.ByFile, curr.ByFile)
	c.comparePackages(prev.ByPackage, curr.ByPackage)

	// Sort by name so the output is the same on every run.
	for _, ch := range []*Changes{&c.Added, &c.Removed} {
		sort.Slice(ch.Files, func(i, j int) bool { return ch.Files[i].File < ch.Files[j].File })
		sort.Slice(ch.Packages, func(i, j int) bool { return ch.Packages[i].Package < ch.Packages[j].Package })
	}

	if d, changed := totalDelta(curr.ByTotal, prev.ByTotal); changed {
		c.ByTotal = &d
	}
	return c
}

// named is a file or package with its coverage.
type named struct {
	name string
	by   By
}

// namedDelta is the change in coverage for a named item.
type namedDelta struct {
	name  string
	delta Delta
}

// diffNamed splits prev and curr into items only in curr, items only in prev,
// and items in both whose coverage changed.
func diffNamed(prev, curr []named) ([]named, []named, []namedDelta) {
	var added, removed []named
	var changed []namedDelta
	prevBy := make(map[string]By, len(prev))
	for _, p := range prev {
		prevBy[p.name] = p.by
	}
	currNames := make(map[string]struct{}, len(curr))
	for _, cur := range curr {
		currNames[cur.name] = struct{}{}
		p, ok := prevBy[cur.name]
		if !ok {
			added = append(added, cur)
		} else if d, diffs := byDelta(cur.by, p); diffs {
			changed = append(changed, namedDelta{name: cur.name, delta: d})
		}
	}
	for _, p := range prev {
		if _, ok := currNames[p.name]; !ok {
			removed = append(removed, p)
		}
	}
	return added, removed, changed
}

func mapSlice[T, U any](in []T, f func(T) U) []U {
	out := make([]U, 0, len(in))
	for _, v := range in {
		out = append(out, f(v))
	}
	return out
}

func (c *Comparison) compareFiles(prev, curr []ByFile) {
	toItem := func(f ByFile) named { return named{name: f.File, by: f.By} }
	toCoverage := func(i named) FileCoverage { return FileCoverage{File: i.name, Coverage: newCoverage(i.by)} }
	added, removed, changed := diffNamed(mapSlice(prev, toItem), mapSlice(curr, toItem))
	c.Added.Files = mapSlice(added, toCoverage)
	c.Removed.Files = mapSlice(removed, toCoverage)
	c.ByFile = mapSlice(changed, func(d namedDelta) FileDelta { return FileDelta{File: d.name, Delta: d.delta} })
}

func (c *Comparison) comparePackages(prev, curr []ByPackage) {
	toItem := func(p ByPackage) named { return named{name: p.Package, by: p.By} }
	toCoverage := func(i named) PackageCoverage { return PackageCoverage{Package: i.name, Coverage: newCoverage(i.by)} }
	added, removed, changed := diffNamed(mapSlice(prev, toItem), mapSlice(curr, toItem))
	c.Added.Packages = mapSlice(added, toCoverage)
	c.Removed.Packages = mapSlice(removed, toCoverage)
	c.ByPackage = mapSlice(changed, func(d namedDelta) PackageDelta {
		return PackageDelta{Package: d.name, Delta: d.delta}
	})
}
