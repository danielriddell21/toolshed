package life

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/danielriddell21/toolshed/internal/grid"
)

type Board struct {
	g       *grid.Grid[bool]
	gen     int
	scratch *grid.Grid[bool]
}

func NewBoard(w, h int) *Board {
	return &Board{
		g:       grid.New[bool](w, h),
		scratch: grid.New[bool](w, h),
	}
}

func (b *Board) W() int { return b.g.W }

func (b *Board) H() int { return b.g.H }

func (b *Board) Alive(x, y int) bool { return b.g.At(x, y) }

func (b *Board) Set(x, y int, v bool) { b.g.Set(x, y, v) }

func (b *Board) Generation() int { return b.gen }

func (b *Board) Neighbors(x, y int) int {
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

func (b *Board) Step() {
	for y := range b.g.H {
		for x := range b.g.W {
			n := b.Neighbors(x, y)
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

func (b *Board) Population() int {
	pop := 0
	for _, c := range b.g.Cells() {
		if c {
			pop++
		}
	}
	return pop
}

func (b *Board) Clear() {
	cells := b.g.Cells()
	for i := range cells {
		cells[i] = false
	}
}

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
		return fmt.Errorf("walk %s: %w", root, err)
	}
	slices.Sort(paths)
	SeedFromHashes(b, paths)
	return nil
}

func SeedFromHashes(b *Board, paths []string) {
	for _, p := range paths {
		h := fnv.New64a()
		_, _ = h.Write([]byte(p))
		sum := h.Sum64()
		x := int(sum % uint64(b.W()))
		y := int((sum / uint64(b.W())) % uint64(b.H()))
		b.Set(x, y, true)
	}
}
