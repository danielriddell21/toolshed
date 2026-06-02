// Package life implements Conway's Game of Life on a toroidal board, seeded
// deterministically from a directory tree. The rules engine is pure and
// filesystem-free (see SeedFromHashes) so it can be unit tested without any
// I/O; SeedFromTree layers a deterministic directory walk on top.
package life

import (
	"hash/fnv"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/danielriddell21/toolshed/internal/grid"
)

// Board is a Game of Life universe. Its space is a torus: the left edge is
// adjacent to the right edge and the top to the bottom, so gliders that leave
// one side reappear on the other.
type Board struct {
	g       *grid.Grid[bool]
	gen     int
	scratch *grid.Grid[bool] // reused double buffer for Step
}

// NewBoard returns a w by h board (dimensions clamped to a minimum of 1 by the
// underlying grid). All cells start dead and the generation counter at zero.
func NewBoard(w, h int) *Board {
	return &Board{
		g:       grid.New[bool](w, h),
		scratch: grid.New[bool](w, h),
	}
}

// W returns the board width.
func (b *Board) W() int { return b.g.W }

// H returns the board height.
func (b *Board) H() int { return b.g.H }

// Alive reports whether the cell at (x,y) is live, wrapping toroidally.
func (b *Board) Alive(x, y int) bool { return b.g.At(x, y) }

// Set writes the liveness of the cell at (x,y), wrapping toroidally.
func (b *Board) Set(x, y int, v bool) { b.g.Set(x, y, v) }

// Generation returns the number of completed Steps.
func (b *Board) Generation() int { return b.gen }

// neighbors counts the live cells in the 8 toroidal neighbors of (x,y).
func (b *Board) neighbors(x, y int) int {
	n := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if b.g.At(x+dx, y+dy) {
				n++
			}
		}
	}
	return n
}

// Step advances the board one generation using the standard B3/S23 rules on a
// torus. The next generation is computed into a scratch grid and then swapped
// in, so every cell sees the previous generation's state (double buffering).
func (b *Board) Step() {
	for y := range b.g.H {
		for x := range b.g.W {
			n := b.neighbors(x, y)
			alive := b.g.At(x, y)
			// Survival: a live cell with 2 or 3 neighbors stays alive.
			// Birth: a dead cell with exactly 3 neighbors becomes alive.
			next := n == 3 || (alive && n == 2)
			b.scratch.Set(x, y, next)
		}
	}
	b.g, b.scratch = b.scratch, b.g
	b.gen++
}

// Population returns the number of live cells.
func (b *Board) Population() int {
	pop := 0
	for _, c := range b.g.Cells() {
		if c {
			pop++
		}
	}
	return pop
}

// Clear kills every cell. It does not reset the generation counter.
func (b *Board) Clear() {
	cells := b.g.Cells()
	for i := range cells {
		cells[i] = false
	}
}

// SeedFromTree walks root and places a live cell for each path it finds. The
// walk is made deterministic by collecting every path, sorting it, and then
// hashing in order, so the same directory tree always yields the same board
// regardless of filesystem iteration order. Returns an error if root cannot be
// walked.
func SeedFromTree(b *Board, root string) error {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return err
	}
	slices.Sort(paths)
	SeedFromHashes(b, paths)
	return nil
}

// SeedFromHashes is the filesystem-free seeding core. For each path string it
// computes an FNV-64a hash and maps it to a cell coordinate, marking that cell
// alive. Given the same paths and board dimensions it always produces the same
// board, which makes it directly testable without touching disk.
func SeedFromHashes(b *Board, paths []string) {
	for _, p := range paths {
		h := fnv.New64a()
		h.Write([]byte(p))
		sum := h.Sum64()
		x := int(sum % uint64(b.W()))
		y := int((sum / uint64(b.W())) % uint64(b.H()))
		b.Set(x, y, true)
	}
}
