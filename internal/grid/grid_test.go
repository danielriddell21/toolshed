package grid

import "testing"

func TestWrap(t *testing.T) {
	cases := []struct{ v, n, want int }{
		{0, 5, 0},
		{4, 5, 4},
		{5, 5, 0},
		{6, 5, 1},
		{-1, 5, 4},
		{-5, 5, 0},
		{-6, 5, 4},
	}
	for _, c := range cases {
		if got := Wrap(c.v, c.n); got != c.want {
			t.Errorf("Wrap(%d,%d) = %d, want %d", c.v, c.n, got, c.want)
		}
	}
}

func TestToroidalAtSet(t *testing.T) {
	g := New[int](3, 3)
	g.Set(0, 0, 7)
	// (3,3) wraps to (0,0).
	if g.At(3, 3) != 7 {
		t.Errorf("toroidal read failed: %d", g.At(3, 3))
	}
	// Negative wraps too.
	g.Set(-1, -1, 9) // -> (2,2)
	if g.At(2, 2) != 9 {
		t.Errorf("negative wrap failed: %d", g.At(2, 2))
	}
}

func TestCells(t *testing.T) {
	g := New[bool](4, 2)
	if len(g.Cells()) != 8 {
		t.Errorf("Cells len = %d, want 8", len(g.Cells()))
	}
}
