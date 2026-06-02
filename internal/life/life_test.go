package life

import (
	"slices"
	"testing"
)

// setCells marks the given (x,y) coordinates alive on a fresh board.
func setCells(b *Board, cells [][2]int) {
	for _, c := range cells {
		b.Set(c[0], c[1], true)
	}
}

// liveCells returns the sorted list of live coordinates for stable comparison.
func liveCells(b *Board) [][2]int {
	var out [][2]int
	for y := range b.H() {
		for x := range b.W() {
			if b.Alive(x, y) {
				out = append(out, [2]int{x, y})
			}
		}
	}
	slices.SortFunc(out, func(a, c [2]int) int {
		if a[1] != c[1] {
			return a[1] - c[1]
		}
		return a[0] - c[0]
	})
	return out
}

func equalCells(a, b [][2]int) bool {
	return slices.Equal(
		flatten(a),
		flatten(b),
	)
}

func flatten(c [][2]int) []int {
	out := make([]int, 0, len(c)*2)
	for _, p := range c {
		out = append(out, p[0], p[1])
	}
	return out
}

func TestBlinkerOscillates(t *testing.T) {
	// Horizontal line at row 2, columns 1..3 on a 5x5 board (no wrap interference).
	b := NewBoard(5, 5)
	horizontal := [][2]int{{1, 2}, {2, 2}, {3, 2}}
	vertical := [][2]int{{2, 1}, {2, 2}, {2, 3}}
	setCells(b, horizontal)

	b.Step()
	if got := liveCells(b); !equalCells(got, vertical) {
		t.Fatalf("after 1 step want vertical %v, got %v", vertical, got)
	}

	b.Step()
	if got := liveCells(b); !equalCells(got, horizontal) {
		t.Fatalf("after 2 steps want horizontal %v, got %v", horizontal, got)
	}

	if b.Generation() != 2 {
		t.Fatalf("generation: want 2, got %d", b.Generation())
	}
}

func TestBlockStillLife(t *testing.T) {
	b := NewBoard(6, 6)
	block := [][2]int{{1, 1}, {2, 1}, {1, 2}, {2, 2}}
	setCells(b, block)

	b.Step()
	if got := liveCells(b); !equalCells(got, block) {
		t.Fatalf("block should be unchanged, got %v", got)
	}
	if b.Population() != 4 {
		t.Fatalf("population: want 4, got %d", b.Population())
	}
}

func TestGliderTranslates(t *testing.T) {
	// Glider on a large board so wrap does not interfere over 4 steps.
	b := NewBoard(20, 20)
	glider := [][2]int{{1, 0}, {2, 1}, {0, 2}, {1, 2}, {2, 2}}
	setCells(b, glider)

	for range 4 {
		b.Step()
	}

	// After one full period (4 steps) a glider translates by (+1,+1).
	want := make([][2]int, len(glider))
	for i, c := range glider {
		want[i] = [2]int{c[0] + 1, c[1] + 1}
	}
	if got := liveCells(b); !equalCells(got, want) {
		t.Fatalf("glider after 4 steps: want %v, got %v", want, got)
	}
	if b.Population() != 5 {
		t.Fatalf("glider population: want 5, got %d", b.Population())
	}
}

func TestPopulationAndGeneration(t *testing.T) {
	b := NewBoard(8, 8)
	if b.Population() != 0 {
		t.Fatalf("empty population: want 0, got %d", b.Population())
	}
	if b.Generation() != 0 {
		t.Fatalf("initial generation: want 0, got %d", b.Generation())
	}
	b.Set(0, 0, true)
	b.Set(3, 4, true)
	if b.Population() != 2 {
		t.Fatalf("population: want 2, got %d", b.Population())
	}
	b.Clear()
	if b.Population() != 0 {
		t.Fatalf("after clear: want 0, got %d", b.Population())
	}
}

func TestSeedFromHashesDeterministic(t *testing.T) {
	paths := []string{"a/b", "a/c", "d/e/f", "main.go", "README.md"}

	b1 := NewBoard(40, 30)
	b2 := NewBoard(40, 30)
	SeedFromHashes(b1, paths)
	SeedFromHashes(b2, paths)

	if !slices.Equal(b1.g.Cells(), b2.g.Cells()) {
		t.Fatal("same paths must produce byte-identical boards")
	}
	if b1.Population() == 0 {
		t.Fatal("seeding should produce live cells")
	}
}

func TestSeedFromHashesDiffers(t *testing.T) {
	a := NewBoard(40, 30)
	b := NewBoard(40, 30)
	SeedFromHashes(a, []string{"alpha", "beta", "gamma"})
	SeedFromHashes(b, []string{"one", "two", "three"})

	if slices.Equal(a.g.Cells(), b.g.Cells()) {
		t.Fatal("different paths should (very likely) produce different boards")
	}
}

func TestSeedFromTree(t *testing.T) {
	dir := t.TempDir()
	b := NewBoard(20, 20)
	if err := SeedFromTree(b, dir); err != nil {
		t.Fatalf("SeedFromTree: %v", err)
	}
	// The root itself is a path, so at least one cell is seeded.
	if b.Population() == 0 {
		t.Fatal("walking a directory should seed at least the root path")
	}

	// Deterministic across calls.
	b2 := NewBoard(20, 20)
	if err := SeedFromTree(b2, dir); err != nil {
		t.Fatalf("SeedFromTree: %v", err)
	}
	if !slices.Equal(b.g.Cells(), b2.g.Cells()) {
		t.Fatal("SeedFromTree must be deterministic for the same tree")
	}
}
