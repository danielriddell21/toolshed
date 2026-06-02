package render

import (
	"strings"
	"testing"
)

func TestNewFrameRoundsHeightEven(t *testing.T) {
	f := NewFrame(10, 7)
	if f.H%2 != 0 {
		t.Errorf("height not rounded even: %d", f.H)
	}
	if f.W != 10 || f.H != 8 {
		t.Errorf("got %dx%d, want 10x8", f.W, f.H)
	}
}

func TestSetAtBounds(t *testing.T) {
	f := NewFrame(4, 4)
	red := RGB{255, 0, 0}
	f.Set(1, 2, red)
	if f.At(1, 2) != red {
		t.Errorf("At(1,2) = %v, want %v", f.At(1, 2), red)
	}
	// Out of bounds is a no-op / black.
	f.Set(-1, 0, red)
	f.Set(99, 0, red)
	if f.At(99, 0) != (RGB{}) {
		t.Error("out-of-bounds read should be black")
	}
}

func TestStringHalfBlock(t *testing.T) {
	f := NewFrame(2, 2)
	top := RGB{255, 0, 0}
	bot := RGB{0, 0, 255}
	f.Set(0, 0, top)
	f.Set(0, 1, bot)
	out := f.String()
	if !strings.Contains(out, "▀") {
		t.Error("output missing half-block glyph")
	}
	if !strings.Contains(out, "38;2;255;0;0") {
		t.Errorf("missing foreground (top) SGR; got %q", out)
	}
	if !strings.Contains(out, "48;2;0;0;255") {
		t.Errorf("missing background (bottom) SGR; got %q", out)
	}
	if !strings.HasSuffix(out, "\x1b[0m") {
		t.Error("line should end with a reset")
	}
}

func TestStringCoalescesRuns(t *testing.T) {
	f := NewFrame(4, 2)
	c := RGB{10, 20, 30}
	f.Fill(c)
	out := f.String()
	// All four cells identical: the SGR prefix should appear exactly once.
	if n := strings.Count(out, "38;2;10;20;30"); n != 1 {
		t.Errorf("expected 1 coalesced SGR run, got %d", n)
	}
}

func TestLerp(t *testing.T) {
	a := RGB{0, 0, 0}
	b := RGB{200, 100, 50}
	mid := Lerp(a, b, 0.5)
	if mid.R != 100 || mid.G != 50 || mid.B != 25 {
		t.Errorf("Lerp mid = %v", mid)
	}
	if Lerp(a, b, -1) != a {
		t.Error("t<0 should clamp to a")
	}
	if Lerp(a, b, 2) != b {
		t.Error("t>1 should clamp to b")
	}
}
