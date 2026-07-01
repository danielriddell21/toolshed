package grid

type Grid[T any] struct {
	W, H  int
	cells []T
}

func New[T any](w, h int) *Grid[T] {
	w = max(w, 1)
	h = max(h, 1)
	return &Grid[T]{W: w, H: h, cells: make([]T, w*h)}
}

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

func (g *Grid[T]) Idx(x, y int) int {
	return Wrap(y, g.H)*g.W + Wrap(x, g.W)
}

func (g *Grid[T]) At(x, y int) T {
	return g.cells[g.Idx(x, y)]
}

func (g *Grid[T]) Set(x, y int, v T) {
	g.cells[g.Idx(x, y)] = v
}

func (g *Grid[T]) Cells() []T { return g.cells }
