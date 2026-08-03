package render

import (
	"strings"
	"testing"
)

func TestDownsampleAveragesEachBlock(t *testing.T) {
	src := NewFrame(4, 4)
	// Left half black, right half white: every 2x2 block straddles nothing, so
	// the result is two flat columns rather than a blur.
	for y := range 4 {
		for x := range 4 {
			if x >= 2 {
				src.Set(x, y, RGB{R: 200, G: 100, B: 40})
			}
		}
	}

	dst := NewFrame(2, 2)
	Downsample(src, dst)

	if got := dst.At(0, 0); got != (RGB{}) {
		t.Errorf("left block = %+v, want black", got)
	}
	if want := (RGB{R: 200, G: 100, B: 40}); dst.At(1, 0) != want {
		t.Errorf("right block = %+v, want %+v", dst.At(1, 0), want)
	}
}

func TestDownsampleBlendsAcrossAnEdge(t *testing.T) {
	src := NewFrame(2, 2)
	src.Set(0, 0, RGB{R: 100, G: 100, B: 100})
	src.Set(1, 0, RGB{R: 100, G: 100, B: 100})
	src.Set(0, 1, RGB{})
	src.Set(1, 1, RGB{})

	dst := NewFrame(1, 2)
	Downsample(src, dst)

	// A one-pixel-wide destination averages each row of two pixels.
	if got := dst.At(0, 0); got.R != 100 {
		t.Errorf("top = %+v, want the lit row", got)
	}
	if got := dst.At(0, 1); got.R != 0 {
		t.Errorf("bottom = %+v, want the dark row", got)
	}
}

func TestSplitCellFindsTheExactPartition(t *testing.T) {
	dark, light := RGB{R: 10, G: 10, B: 10}, RGB{R: 240, G: 240, B: 240}

	// Top-left and bottom-left light, right side dark: that is the left half.
	mask, fg, bg := splitCell([4]RGB{light, dark, light, dark})
	if quadrantGlyphs[mask] != '▌' {
		t.Errorf("glyph = %q, want a left half block", quadrantGlyphs[mask])
	}
	if fg != light || bg != dark {
		t.Errorf("colours = %+v on %+v, want %+v on %+v", fg, bg, light, dark)
	}

	// A flat cell has nothing to split, so it must cost nothing.
	_, fg, bg = splitCell([4]RGB{light, light, light, light})
	if fg != light || bg != light {
		t.Errorf("a flat cell resolved to %+v and %+v, want both %+v", fg, bg, light)
	}
}

func TestSplitCellNeverLosesMoreThanItMust(t *testing.T) {
	pix := [4]RGB{{R: 250}, {G: 250}, {B: 250}, {R: 250, G: 250}}
	mask, fg, bg := splitCell(pix)

	loss := 0
	for i, p := range pix {
		if mask&(1<<i) != 0 {
			loss += distance(p, fg)
		} else {
			loss += distance(p, bg)
		}
	}

	for candidate := range 16 {
		a, an := meanOf(pix, candidate, true)
		c, cn := meanOf(pix, candidate, false)
		if an == 0 {
			a = c
		}
		if cn == 0 {
			c = a
		}
		rival := 0
		for i, p := range pix {
			if candidate&(1<<i) != 0 {
				rival += distance(p, a)
			} else {
				rival += distance(p, c)
			}
		}
		if rival < loss {
			t.Fatalf("split %d loses %d, better than the chosen %d", candidate, rival, loss)
		}
	}
}

func TestQuadrantsShapeAndGlyphs(t *testing.T) {
	f := NewFrame(4, 4)
	for y := range 4 {
		f.Set(0, y, RGB{R: 250, G: 250, B: 250})
		f.Set(2, y, RGB{R: 250, G: 250, B: 250})
	}

	out := f.Quadrants()
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines for a four-pixel-tall frame, want 2", len(lines))
	}
	if strings.Count(out, "▌") != 4 {
		t.Errorf("expected four left-half cells, got %q", out)
	}
}

func TestQuadrantsPacksTwiceAsManyColumnsAsString(t *testing.T) {
	f := NewFrame(8, 4)
	for y := range 4 {
		for x := range 8 {
			if (x+y)%2 == 0 {
				f.Set(x, y, RGB{R: 200})
			}
		}
	}

	cells := 0
	for _, r := range strings.Split(f.Quadrants(), "\n")[0] {
		if r >= 0x2580 && r <= 0x259F || r == ' ' {
			cells++
		}
	}
	if cells != 4 {
		t.Errorf("an eight-pixel row became %d cells, want 4", cells)
	}
}
