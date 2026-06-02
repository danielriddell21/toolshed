// Package render provides a truecolor pixel framebuffer that draws to the
// terminal using the Unicode upper-half-block (▀). Each character cell shows
// two vertical pixels: the foreground color is the top pixel, the background
// color is the bottom pixel. This doubles vertical resolution and lets the
// pixel tools (fractal, life, sandbox) treat the terminal as a square-pixel
// canvas.
//
// The ANSI is built by hand with a strings.Builder and run-length coalescing;
// lipgloss is deliberately avoided here because per-cell styling allocates and
// re-parses on every call, far too slow for full-screen per-frame redraws.
package render

import (
	"strconv"
	"strings"
)

// RGB is a 24-bit color.
type RGB struct{ R, G, B uint8 }

// Frame is a dense, row-major RGB pixel grid. Width is in pixels (one per
// terminal column); height is in pixels and is always even so each pair of
// rows maps to one row of half-block characters.
type Frame struct {
	W, H int
	pix  []RGB
}

// NewFrame returns a frame sized w by h. Height is rounded up to an even
// number and both dimensions are clamped to a minimum of 2x2.
func NewFrame(w, h int) *Frame {
	f := &Frame{}
	f.Resize(w, h)
	return f
}

// Resize reallocates the pixel buffer only when the dimensions change. Height
// is rounded up to even.
func (f *Frame) Resize(w, h int) {
	w = max(w, 2)
	h = max(h, 2)
	if h%2 != 0 {
		h++
	}
	if w == f.W && h == f.H && f.pix != nil {
		return
	}
	f.W, f.H = w, h
	f.pix = make([]RGB, w*h)
}

// Set writes one pixel, ignoring out-of-bounds coordinates.
func (f *Frame) Set(x, y int, c RGB) {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return
	}
	f.pix[y*f.W+x] = c
}

// At returns the pixel at (x, y); out-of-bounds reads return black.
func (f *Frame) At(x, y int) RGB {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return RGB{}
	}
	return f.pix[y*f.W+x]
}

// Fill paints every pixel a single color.
func (f *Frame) Fill(c RGB) {
	for i := range f.pix {
		f.pix[i] = c
	}
}

// Pix exposes the backing slice for hot loops (e.g. fractal row workers writing
// disjoint stripes). Length is W*H, row-major.
func (f *Frame) Pix() []RGB { return f.pix }

// String renders the frame to half-block ANSI: two pixel rows per text row,
// foreground = top pixel, background = bottom pixel, '▀' as the glyph. Runs of
// cells sharing both colors reuse a single SGR sequence to cut escape volume.
func (f *Frame) String() string {
	var b strings.Builder
	// Rough preallocation: ~20 bytes per cell of SGR in the worst case.
	b.Grow(f.W * (f.H / 2) * 12)

	for y := 0; y < f.H; y += 2 {
		var lastTop, lastBot RGB
		haveLast := false
		for x := range f.W {
			top := f.pix[y*f.W+x]
			bot := f.pix[(y+1)*f.W+x]
			if !haveLast || top != lastTop || bot != lastBot {
				writeSGR(&b, top, bot)
				lastTop, lastBot, haveLast = top, bot, true
			}
			b.WriteString("▀")
		}
		b.WriteString("\x1b[0m")
		if y+2 < f.H {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// writeSGR emits a combined foreground+background truecolor escape:
// ESC[38;2;r;g;b;48;2;r;g;bm
func writeSGR(b *strings.Builder, fg, bg RGB) {
	b.WriteString("\x1b[38;2;")
	writeByte(b, fg.R)
	b.WriteByte(';')
	writeByte(b, fg.G)
	b.WriteByte(';')
	writeByte(b, fg.B)
	b.WriteString(";48;2;")
	writeByte(b, bg.R)
	b.WriteByte(';')
	writeByte(b, bg.G)
	b.WriteByte(';')
	writeByte(b, bg.B)
	b.WriteByte('m')
}

func writeByte(b *strings.Builder, v uint8) {
	b.WriteString(strconv.Itoa(int(v)))
}

// Lerp linearly interpolates between two colors; t is clamped to [0,1].
func Lerp(a, c RGB, t float64) RGB {
	t = max(0, min(1, t))
	return RGB{
		R: uint8(float64(a.R) + (float64(c.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(c.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(c.B)-float64(a.B))*t),
	}
}

// RowsToPixels returns the even pixel height for a given number of text rows,
// keeping the half-block doubling rule in one place.
func RowsToPixels(textRows int) int { return max(textRows, 1) * 2 }
