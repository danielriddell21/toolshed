package render

import (
	"strconv"
	"strings"
)

type RGB struct{ R, G, B uint8 }

type Frame struct {
	W, H int
	pix  []RGB
}

func NewFrame(w, h int) *Frame {
	f := &Frame{}
	f.Resize(w, h)
	return f
}

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

func (f *Frame) Set(x, y int, c RGB) {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return
	}
	f.pix[y*f.W+x] = c
}

func (f *Frame) At(x, y int) RGB {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return RGB{}
	}
	return f.pix[y*f.W+x]
}

func (f *Frame) Fill(c RGB) {
	for i := range f.pix {
		f.pix[i] = c
	}
}

func (f *Frame) Pix() []RGB { return f.pix }

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

// Downsample box-filters a larger frame into a smaller one. Rendering into an
// oversized buffer and averaging it down is what takes the jagged staircase off
// every edge, which matters a great deal when a whole car is only a few cells
// across.
func Downsample(src, dst *Frame) {
	fx := max(src.W/dst.W, 1)
	fy := max(src.H/dst.H, 1)
	n := fx * fy

	for y := range dst.H {
		for x := range dst.W {
			var r, g, b int
			for sy := range fy {
				for sx := range fx {
					c := src.At(x*fx+sx, y*fy+sy)
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
				}
			}
			dst.Set(x, y, RGB{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n)})
		}
	}
}

var quadrantGlyphs = [16]rune{
	' ', '▘', '▝', '▀', '▖', '▌', '▞', '▛',
	'▗', '▚', '▐', '▜', '▄', '▙', '▟', '█',
}

// Quadrants renders the frame two pixels per cell in each direction instead of
// the one-by-two of String, doubling the horizontal resolution. A cell can only
// carry two colours, so each one picks the split of its four pixels that loses
// the least colour.
func (f *Frame) Quadrants() string {
	var b strings.Builder
	b.Grow(f.W / 2 * (f.H / 2) * 14)

	var last [2]RGB
	haveLast := false
	for y := 0; y+1 < f.H; y += 2 {
		for x := 0; x+1 < f.W; x += 2 {
			pix := [4]RGB{f.At(x, y), f.At(x+1, y), f.At(x, y+1), f.At(x+1, y+1)}
			mask, fg, bg := splitCell(pix)
			if !haveLast || fg != last[0] || bg != last[1] {
				writeSGR(&b, fg, bg)
				last, haveLast = [2]RGB{fg, bg}, true
			}
			b.WriteRune(quadrantGlyphs[mask])
		}
		b.WriteString("\x1b[0m")
		haveLast = false
		if y+2 < f.H {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func splitCell(pix [4]RGB) (mask int, fg, bg RGB) {
	best := -1
	for candidate := range 16 {
		a, an := meanOf(pix, candidate, true)
		c, cn := meanOf(pix, candidate, false)
		if an == 0 {
			a = c
		}
		if cn == 0 {
			c = a
		}

		loss := 0
		for i, p := range pix {
			if candidate&(1<<i) != 0 {
				loss += distance(p, a)
			} else {
				loss += distance(p, c)
			}
		}
		if best < 0 || loss < best {
			best, mask, fg, bg = loss, candidate, a, c
		}
	}
	return mask, fg, bg
}

func meanOf(pix [4]RGB, mask int, set bool) (RGB, int) {
	var r, g, b, n int
	for i, p := range pix {
		if (mask&(1<<i) != 0) != set {
			continue
		}
		r += int(p.R)
		g += int(p.G)
		b += int(p.B)
		n++
	}
	if n == 0 {
		return RGB{}, 0
	}
	return RGB{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n)}, n
}

func distance(a, b RGB) int {
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	return dr*dr + dg*dg + db*db
}

func Lerp(a, c RGB, t float64) RGB {
	t = max(0, min(1, t))
	return RGB{
		R: uint8(float64(a.R) + (float64(c.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(c.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(c.B)-float64(a.B))*t),
	}
}

func RowsToPixels(textRows int) int { return max(textRows, 1) * 2 }
