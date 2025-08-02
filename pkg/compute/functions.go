package compute

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/cover"
)

// FunctionInfo holds information about a function's location and coverage.
type FunctionInfo struct {
	Name      string
	StartLine int
	EndLine   int
	Covered   bool
}

// CountFunctionsInFile counts the number of functions declared in a Go source file.
func CountFunctionsInFile(filename string) ([]FunctionInfo, error) {
	fset := token.NewFileSet()
	
	// Parse the Go source file
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	
	var functions []FunctionInfo
	
	// Walk the AST to find function declarations
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name != nil {
				pos := fset.Position(x.Pos())
				end := fset.Position(x.End())
				
				funcName := x.Name.Name
				// Include receiver type for methods
				if x.Recv != nil && len(x.Recv.List) > 0 {
					if recv := x.Recv.List[0]; recv.Type != nil {
						funcName = getTypeName(recv.Type) + "." + funcName
					}
				}
				
				functions = append(functions, FunctionInfo{
					Name:      funcName,
					StartLine: pos.Line,
					EndLine:   end.Line,
					Covered:   false, // Will be determined later by coverage analysis
				})
			}
		}
		return true
	})
	
	return functions, nil
}

// getTypeName extracts the type name from an AST expression.
func getTypeName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + getTypeName(x.X)
	case *ast.SelectorExpr:
		return getTypeName(x.X) + "." + x.Sel.Name
	default:
		return "unknown"
	}
}

// MatchFunctionsWithCoverage determines which functions are covered based on coverage blocks.
func MatchFunctionsWithCoverage(functions []FunctionInfo, blocks []cover.ProfileBlock) []FunctionInfo {
	result := make([]FunctionInfo, len(functions))
	copy(result, functions)
	
	// For each function, check if any coverage block within its range has count > 0
	for i := range result {
		for _, block := range blocks {
			// Check if this block overlaps with the function's line range
			if block.StartLine >= result[i].StartLine && block.StartLine <= result[i].EndLine {
				if block.Count > 0 {
					result[i].Covered = true
					break
				}
			}
		}
	}
	
	return result
}

// GetFunctionCoverageForFile returns function coverage statistics for a file.
func GetFunctionCoverageForFile(filename string, coverageBlocks []cover.ProfileBlock) (totalFunctions, coveredFunctions int, err error) {
	// Skip non-Go files
	if !strings.HasSuffix(filename, ".go") {
		return 0, 0, nil
	}
	
	// Get the actual source file path from the coverage filename
	// Coverage filenames might be module-relative paths
	sourcePath := filename
	if !filepath.IsAbs(filename) {
		// Try to find the file relative to current directory
		// This is a simple approach - in a real implementation you might want
		// to handle module paths more sophisticatedly
		if _, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.PackageClauseOnly); err != nil {
			// If we can't parse the file, assume 0 functions
			return 0, 0, nil
		}
		sourcePath = filename
	}
	
	functions, err := CountFunctionsInFile(sourcePath)
	if err != nil {
		// If we can't parse the source file, we can't count functions
		// This might happen for generated files or files outside the module
		return 0, 0, nil
	}
	
	// Convert coverage blocks to our internal format
	blocks := make([]cover.ProfileBlock, len(coverageBlocks))
	for i, block := range coverageBlocks {
		blocks[i] = cover.ProfileBlock{
			StartLine: block.StartLine,
			StartCol:  block.StartCol,
			EndLine:   block.EndLine,
			EndCol:    block.EndCol,
			NumStmt:   block.NumStmt,
			Count:     block.Count,
		}
	}
	
	// Match functions with coverage
	functionsWithCoverage := MatchFunctionsWithCoverage(functions, blocks)
	
	totalFunctions = len(functionsWithCoverage)
	for _, fn := range functionsWithCoverage {
		if fn.Covered {
			coveredFunctions++
		}
	}
	
	return totalFunctions, coveredFunctions, nil
}