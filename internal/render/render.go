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

func Lerp(a, c RGB, t float64) RGB {
	t = max(0, min(1, t))
	return RGB{
		R: uint8(float64(a.R) + (float64(c.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(c.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(c.B)-float64(a.B))*t),
	}
}

func RowsToPixels(textRows int) int { return max(textRows, 1) * 2 }
