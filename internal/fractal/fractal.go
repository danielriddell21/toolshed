package fractal

import (
	"math"
	"runtime"
	"sync"

	"github.com/danielriddell21/toolshed/internal/palette"
	"github.com/danielriddell21/toolshed/internal/render"
)

type Mode int

const (
	Mandelbrot Mode = iota

	Julia
)

const baseScale = 3.0 / 200.0

const (
	baseIter = 100
	iterStep = 40
	maxIters = 4000
)

type View struct {
	CenterRe, CenterIm, Scale float64
}

type Params struct {
	Mode    Mode
	JuliaC  complex128
	MaxIter int
}

func (v View) PixelToComplex(px, py, w, h int) complex128 {
	re := v.CenterRe + (float64(px)-float64(w)/2)*v.Scale
	im := v.CenterIm + (float64(py)-float64(h)/2)*v.Scale
	return complex(re, im)
}

func EscapeSmooth(z0, c complex128, maxIter int) float64 {
	const bailout = 1 << 16 // large radius for a smooth gradient
	z := z0
	for i := range maxIter {
		zr, zi := real(z), imag(z)
		mag2 := zr*zr + zi*zi
		if mag2 > bailout {
			mag := math.Sqrt(mag2)
			mu := float64(i) + 1 - math.Log2(math.Log2(mag))
			t := mu / float64(maxIter)
			return max(0, min(1, t))
		}
		z = z*z + c
	}
	return 1.0
}

func AdaptiveIter(scale float64) int {
	if scale <= 0 {
		return maxIters
	}
	extra := 0.0
	if ratio := baseScale / scale; ratio > 1 {
		extra = iterStep * math.Log2(ratio)
	}
	return min(maxIters, max(baseIter, baseIter+int(extra)))
}

func Render(f *render.Frame, v View, p Params, g palette.Gradient) {
	w, h := f.W, f.H
	pix := f.Pix()

	workers := min(max(runtime.GOMAXPROCS(0), 1), h)
	rowsPer := (h + workers - 1) / workers

	var wg sync.WaitGroup
	for wkr := range workers {
		y0 := wkr * rowsPer
		if y0 >= h {
			break
		}
		y1 := min(y0+rowsPer, h)
		wg.Add(1)
		go func(y0, y1 int) {
			defer wg.Done()
			for py := y0; py < y1; py++ {
				rowBase := py * w
				for px := range w {
					pt := v.PixelToComplex(px, py, w, h)
					var z0, c complex128
					if p.Mode == Julia {
						z0, c = pt, p.JuliaC
					} else {
						z0, c = 0, pt
					}
					t := EscapeSmooth(z0, c, p.MaxIter)
					pix[rowBase+px] = g.At(t)
				}
			}
		}(y0, y1)
	}
	wg.Wait()
}
