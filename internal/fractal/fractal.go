// Package fractal computes escape-time fractals (Mandelbrot and Julia) into a
// render.Frame. The compute path is pure and parallel: rows are split across
// GOMAXPROCS goroutines that write disjoint stripes of the frame's pixel slice.
package fractal

import (
	"math"
	"runtime"
	"sync"

	"github.com/danielriddell21/toolshed/internal/palette"
	"github.com/danielriddell21/toolshed/internal/render"
)

// Mode selects which fractal to compute.
type Mode int

const (
	// Mandelbrot iterates z = z*z + c from z=0, with c = the pixel point.
	Mandelbrot Mode = iota
	// Julia iterates z = z*z + JuliaC from z = the pixel point.
	Julia
)

// baseScale is the reference complex-plane units per pixel used by AdaptiveIter
// to decide how much to deepen iteration as the viewport shrinks.
const baseScale = 3.0 / 200.0

const (
	baseIter = 100
	iterStep = 40
	maxIters = 4000
)

// View is the complex-plane viewport. Scale is complex-plane units per pixel
// (uniform on both axes; render.Frame pixels are ~square via half-blocks).
type View struct {
	CenterRe, CenterIm, Scale float64
}

// Params configures a single render.
type Params struct {
	Mode    Mode
	JuliaC  complex128
	MaxIter int
}

// PixelToComplex maps pixel (px,py) in a w*h frame to a complex point, with the
// frame center mapping to (CenterRe, CenterIm).
func (v View) PixelToComplex(px, py, w, h int) complex128 {
	re := v.CenterRe + (float64(px)-float64(w)/2)*v.Scale
	im := v.CenterIm + (float64(py)-float64(h)/2)*v.Scale
	return complex(re, im)
}

// EscapeSmooth returns continuous (smooth) escape time normalized to [0,1].
// It iterates z = z*z + c starting from z0. Points that never escape within
// maxIter (inside the set) return 1.0; escaping points return
// (iter + 1 - log2(log2(|z|))) / maxIter.
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

// AdaptiveIter returns an iteration cap that grows as scale shrinks (deeper
// zoom). It is base + iterStep*log2(baseScale/scale), clamped to [baseIter,
// maxIters].
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

// Render fills f. For Mandelbrot, c = the pixel point and z0 = 0. For Julia,
// z0 = the pixel point and c = p.JuliaC. Color comes from g.At(escape). Rows
// are computed in parallel across GOMAXPROCS goroutines, each writing a
// disjoint stripe of f.Pix().
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
