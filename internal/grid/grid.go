// Package grid is a generic, dense 2D buffer with toroidal (wrap-around)
// addressing. It is shared by the Game of Life board and the reaction-diffusion
// simulation, both of which treat their space as a torus.
package grid

// Grid is a row-major 2D buffer of T addressed toroidally.
type Grid[T any] struct {
	W, H  int
	cells []T
}

// New returns a w by h grid (both clamped to a minimum of 1).
func New[T any](w, h int) *Grid[T] {
	w = max(w, 1)
	h = max(h, 1)
	return &Grid[T]{W: w, H: h, cells: make([]T, w*h)}
}

// Wrap maps v into [0,n) using Euclidean modulo.
func Wrap(v, n int) int {
	if n <= 0 {
		return 0
	}
	v %= n
	if v < 0 {
		v += n
	}
	return v
}

// Idx returns the backing-slice index for (x,y), wrapping both axes.
func (g *Grid[T]) Idx(x, y int) int {
	return Wrap(y, g.H)*g.W + Wrap(x, g.W)
}

// At returns the cell at (x,y), wrapping both axes.
func (g *Grid[T]) At(x, y int) T {
	return g.cells[g.Idx(x, y)]
}

// Set writes the cell at (x,y), wrapping both axes.
func (g *Grid[T]) Set(x, y int, v T) {
	g.cells[g.Idx(x, y)] = v
}

// Cells exposes the backing slice for hot loops (no wrapping).
func (g *Grid[T]) Cells() []T { return g.cells }
