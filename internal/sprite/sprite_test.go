package sprite

import (
	"strings"
	"testing"
)

func TestAdvanceBounces(t *testing.T) {
	const maxX = 10
	s := Sprite{X: 0, Speed: 3, Dir: Right, FaceRight: ">", FaceLeft: "<"}
	right := float64(maxX - s.Width())

	// Over many steps the sprite must stay in bounds and reverse off each wall.
	hitLeft, hitRight := false, false
	for range 200 {
		prev := s.Dir
		s.Advance(maxX)
		if s.X < 0 || s.X > right {
			t.Fatalf("sprite escaped bounds: x=%v (right=%v)", s.X, right)
		}
		if s.X == 0 && s.Dir == Right {
			hitLeft = true
		}
		if s.X == right && s.Dir == Left {
			hitRight = true
		}
		// Direction only ever flips at a wall.
		if s.Dir != prev && s.X != 0 && s.X != right {
			t.Fatalf("direction flipped off a wall at x=%v", s.X)
		}
	}
	if !hitLeft || !hitRight {
		t.Errorf("expected to bounce off both walls; left=%v right=%v", hitLeft, hitRight)
	}
}

func TestGlyphFacing(t *testing.T) {
	s := Sprite{Dir: Right, FaceRight: "><>", FaceLeft: "<><"}
	if s.Glyph() != "><>" {
		t.Errorf("right glyph = %q", s.Glyph())
	}
	s.Dir = Left
	if s.Glyph() != "<><" {
		t.Errorf("left glyph = %q", s.Glyph())
	}
	if s.Width() != 3 {
		t.Errorf("width = %d, want 3", s.Width())
	}
}

func TestCanvasDraw(t *testing.T) {
	c := NewCanvas(5, 2)
	c.DrawString(1, 0, "abc")
	c.Set(0, 1, 'x')
	out := c.String()
	lines := strings.Split(out, "\n")
	if lines[0] != " abc" {
		t.Errorf("row 0 = %q", lines[0])
	}
	if lines[1] != "x" {
		t.Errorf("row 1 = %q", lines[1])
	}
}

func TestCanvasClipsOutOfBounds(t *testing.T) {
	c := NewCanvas(3, 1)
	c.DrawString(2, 0, "hello") // only 'h' fits
	if got := c.String(); got != "  h" {
		t.Errorf("clipped draw = %q", got)
	}
}
