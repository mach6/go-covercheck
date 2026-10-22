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
	"strconv"
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
	// Name is the function name, qualified by its receiver for methods:
	// "Foo", "T.Bar" or "(*T).Bar". Generic receivers keep their type
	// parameters, as in "(*T[K,V]).Bar".
	Name string
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
			Name:      funcName(fd),
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

// funcName returns the declared name of fd, prefixed with the receiver type
// for methods.
func funcName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	recv := receiverType(fd.Recv.List[0].Type)
	if strings.HasPrefix(recv, "*") {
		return "(" + recv + ")." + fd.Name.Name
	}
	return recv + "." + fd.Name.Name
}

// receiverType renders a receiver type expression such as T, *T, *T[K] or
// T[K,V]. Anything unexpected renders as "?" so a name is always produced.
func receiverType(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + receiverType(t.X)
	case *ast.ParenExpr:
		return receiverType(t.X)
	case *ast.IndexExpr:
		return receiverType(t.X) + "[" + receiverType(t.Index) + "]"
	case *ast.IndexListExpr:
		params := make([]string, 0, len(t.Indices))
		for _, idx := range t.Indices {
			params = append(params, receiverType(idx))
		}
		return receiverType(t.X) + "[" + strings.Join(params, ",") + "]"
	default:
		return "?"
	}
}

// Uncovered returns the functions in fns that were not covered, in source order.
func Uncovered(fns []Function) []Function {
	var out []Function
	for i := range fns {
		if !fns[i].Covered {
			out = append(out, fns[i])
		}
	}
	return out
}

// FormatUncovered lists the uncovered functions in fns as "Name:line" items
// joined by ", " in source order, for example "Foo:12, (*T).Bar:30". The
// line is where the func keyword is. It returns "" when every function is
// covered. Names never contain ", ", so the list can be split on it.
func FormatUncovered(fns []Function) string {
	missed := Uncovered(fns)
	items := make([]string, 0, len(missed))
	for i := range missed {
		items = append(items, missed[i].Name+":"+strconv.Itoa(missed[i].StartLine))
	}
	return strings.Join(items, ", ")
}
