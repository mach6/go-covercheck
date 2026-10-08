// Package functions derives function-level coverage from a coverage profile.
// Every top-level function and method declaration with a body in the profiled
// source file is one function, matching the set `go tool cover -func` reports.
// A function counts as covered when at least one profile block inside its
// extent was executed. Function literals (closures) are not counted on their
// own: their blocks fall inside, and contribute to, the enclosing declaration.
// Function literals assigned to package-level variables have no enclosing
// declaration and are not counted.
package functions

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"golang.org/x/tools/cover"
)

// position is a 1-based line and byte column in a source file, the coordinate
// system shared by go/token and cover.ProfileBlock.
type position struct {
	line, col int
}

func (p position) before(o position) bool {
	return p.line < o.line || (p.line == o.line && p.col < o.col)
}

// Function is the source extent of a function or method declaration and
// whether any coverage block inside it was executed.
type Function struct {
	// StartLine and StartCol locate the func keyword.
	StartLine, StartCol int
	// EndLine and EndCol locate the position just past the closing brace.
	EndLine, EndCol int
	// Covered reports whether any profile block within the extent has a
	// non-zero count.
	Covered bool
}

func (f *Function) start() position { return position{f.StartLine, f.StartCol} }
func (f *Function) end() position   { return position{f.EndLine, f.EndCol} }

// contains reports whether block b lies within the function's extent.
func (f *Function) contains(b *cover.ProfileBlock) bool {
	blockStart := position{b.StartLine, b.StartCol}
	blockEnd := position{b.EndLine, b.EndCol}
	return !blockStart.before(f.start()) && !f.end().before(blockEnd)
}

// Collect parses sourceLines (as returned by lines.ReadSourceFile) and returns
// every function declared in it, marked covered according to p's blocks.
// It returns an error when the source does not parse, including when it is
// empty.
func Collect(p *cover.Profile, sourceLines []string) ([]Function, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, p.FileName, strings.Join(sourceLines, "\n"),
		parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	fns := make([]Function, 0, len(file.Decls))
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			// Body-less declarations are implemented in assembly or linked
			// in; the cover tool never instruments them.
			continue
		}
		start := fset.Position(fd.Pos())
		end := fset.Position(fd.End())
		fns = append(fns, Function{
			StartLine: start.Line,
			StartCol:  start.Column,
			EndLine:   end.Line,
			EndCol:    end.Column,
		})
	}

	for i := range fns {
		for j := range p.Blocks {
			if p.Blocks[j].Count > 0 && fns[i].contains(&p.Blocks[j]) {
				fns[i].Covered = true
				break
			}
		}
	}
	return fns, nil
}

// Coverage returns the number of functions and how many of them are covered.
func Coverage(fns []Function) (int, int) {
	covered := 0
	for i := range fns {
		if fns[i].Covered {
			covered++
		}
	}
	return len(fns), covered
}
