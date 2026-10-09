package fncov

type Stack[T any] struct{ items []T }

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() T {
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v
}

// Empty has no statements; the cover tool still emits a zero-statement block.
func Empty() {}

func EmptyUncalled() {}

// Outer is called, but the closure it returns never is.
func Outer() func() int {
	return func() int {
		return 1
	}
}

// pkgClosure has no enclosing declaration, so it is not a counted function.
var pkgClosure = func() int { return 2 }

func Map[T, U any](in []T, f func(T) U) []U {
	out := make([]U, 0, len(in))
	for _, v := range in {
		out = append(out, f(v))
	}
	return out
}

// a and b share a line; only b is called.
func a() int { return 1 }; func b() int { return 2 }
