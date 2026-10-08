package output

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mach6/go-covercheck/pkg/compute"
)

// noChange is true when nothing changed, and nothing was added or removed.
func noChange(c *compute.Comparison) bool {
	return len(c.ByFile) == 0 && len(c.ByPackage) == 0 && c.ByTotal == nil &&
		len(c.Added.Files) == 0 && len(c.Added.Packages) == 0 &&
		len(c.Removed.Files) == 0 && len(c.Removed.Packages) == 0
}

// printComparison writes the comparison as text.
func printComparison(c *compute.Comparison) {
	fmt.Printf("\n≡ Comparing against ref: %s [commit %s]\n",
		color.New(color.FgBlue).Sprint(c.Ref),
		color.New(color.FgHiBlack).Sprint(c.Commit),
	)

	if noChange(c) {
		fmt.Println(" → No change")
		return
	}

	if len(c.ByFile) > 0 {
		fmt.Printf(" → By File\n")
		for _, f := range c.ByFile {
			printDelta(f.File, f.Delta)
		}
	}
	if len(c.ByPackage) > 0 {
		fmt.Printf(" → By Package\n")
		for _, p := range c.ByPackage {
			printDelta(p.Package, p.Delta)
		}
	}
	printChanges(c)
	if c.ByTotal != nil {
		fmt.Printf(" → By Total\n")
		printDelta("total", *c.ByTotal)
	}
}

// printChanges writes the Added and Removed sections, files before packages.
func printChanges(c *compute.Comparison) {
	fileCov := func(f compute.FileCoverage) (string, compute.Coverage) { return f.File, f.Coverage }
	pkgCov := func(p compute.PackageCoverage) (string, compute.Coverage) { return p.Package, p.Coverage }
	printAdded("Added", "Files", c.Added.Files, fileCov)
	printAdded("Added", "Packages", c.Added.Packages, pkgCov)
	printAdded("Removed", "Files", c.Removed.Files, fileCov)
	printAdded("Removed", "Packages", c.Removed.Packages, pkgCov)
}

// printAdded writes one Added or Removed section, showing the coverage of each name.
func printAdded[T any](kind, what string, items []T, get func(T) (string, compute.Coverage)) {
	if len(items) == 0 {
		return
	}
	fmt.Printf(" → %s %s\n", kind, what)
	for _, item := range items {
		name, cov := get(item)
		compareShowS()
		fmt.Printf("%s [%.1f%%]\n", name, cov.Statements)
		compareShowB()
		fmt.Printf("%s [%.1f%%]\n", name, cov.Blocks)
		if cov.Lines != nil {
			compareShowL()
			fmt.Printf("%s [%.1f%%]\n", name, *cov.Lines)
		}
	}
}

func printDelta(name string, d compute.Delta) {
	if s, ok := formatDelta(d.Statements); ok {
		compareShowS()
		fmt.Printf("%s [%s]\n", name, s)
	}
	if b, ok := formatDelta(d.Blocks); ok {
		compareShowB()
		fmt.Printf("%s [%s]\n", name, b)
	}
	if d.Lines != nil {
		if l, ok := formatDelta(*d.Lines); ok {
			compareShowL()
			fmt.Printf("%s [%s]\n", name, l)
		}
	}
}
