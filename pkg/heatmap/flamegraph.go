package heatmap

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mach6/go-covercheck/pkg/compute"
	"github.com/mach6/go-covercheck/pkg/config"
)

// FlameGraph represents a coverage flame graph generator.
type FlameGraph struct {
	writer io.Writer
	config *config.Config
}

// NewFlameGraph creates a new flame graph generator.
func NewFlameGraph(writer io.Writer, cfg *config.Config) *FlameGraph {
	return &FlameGraph{
		writer: writer,
		config: cfg,
	}
}

// Generate creates a flame graph representation of the coverage results.
// This generates a text-based flame graph format that can be used with flame graph tools.
func (f *FlameGraph) Generate(results compute.Results) error {
	// Generate flame graph header
	f.writeHeader()

	// Generate package-based flame graph
	if err := f.generatePackageFlameGraph(results.ByPackage); err != nil {
		return err
	}

	return nil
}

func (f *FlameGraph) writeHeader() {
	fmt.Fprintln(f.writer, "# Coverage Flame Graph Data")
	fmt.Fprintln(f.writer, "# Format: stack_trace sample_count")
	fmt.Fprintln(f.writer, "# This data shows statement coverage across packages")
	fmt.Fprintln(f.writer, "")
}

func (f *FlameGraph) generatePackageFlameGraph(packages []compute.ByPackage) error {
	if len(packages) == 0 {
		return nil
	}

	// Sort packages by coverage percentage (descending)
	sortedPackages := make([]compute.ByPackage, len(packages))
	copy(sortedPackages, packages)
	sort.Slice(sortedPackages, func(i, j int) bool {
		return sortedPackages[i].StatementPercentage > sortedPackages[j].StatementPercentage
	})

	// Create flame graph entries
	// The format is: stacktrace weight
	// We'll use package hierarchy as the stack and coverage percentage as weight
	for _, pkg := range sortedPackages {
		// Convert package path to hierarchical stack
		stackTrace := f.packageToStack(pkg.Package)
		
		// Extract covered statements from the "covered/total" format
		weight := f.extractCoveredStatements(pkg.Statements)
		if weight == 0 {
			weight = 1 // Avoid zero weights
		}
		
		fmt.Fprintf(f.writer, "%s %d\n", stackTrace, weight)
	}

	return nil
}

// extractCoveredStatements extracts the covered count from "covered/total" string
func (f *FlameGraph) extractCoveredStatements(statements string) int {
	// statements is in format "covered/total", e.g., "25/50"
	parts := strings.Split(statements, "/")
	if len(parts) != 2 {
		return 0
	}
	
	covered := 0
	fmt.Sscanf(parts[0], "%d", &covered)
	return covered
}

// packageToStack converts a package path to a flame graph stack trace format.
// Example: "github.com/mach6/go-covercheck/pkg/compute" becomes "github.com;mach6;go-covercheck;pkg;compute"
func (f *FlameGraph) packageToStack(packagePath string) string {
	// Clean the package path and split into components
	parts := strings.Split(packagePath, "/")
	
	// Filter out empty parts
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	
	// Join with semicolons for flame graph format
	return strings.Join(cleanParts, ";")
}