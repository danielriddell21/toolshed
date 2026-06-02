// Package grayscott implements the Gray-Scott reaction-diffusion model on a
// toroidal grid. Two chemical fields U and V diffuse and react; depending on
// the feed and kill rates the system settles into spots, stripes, mazes and
// other Turing patterns. The simulation core is pure and double-buffered so it
// is cheap to test and to render to images.
package grayscott

import (
	"image"
	"image/color"
	"maps"
	"math"
	"math/rand"
	"slices"

	"github.com/danielriddell21/toolshed/internal/grid"
	"github.com/danielriddell21/toolshed/internal/palette"
)

// Params holds the Gray-Scott reaction-diffusion coefficients.
type Params struct {
	Feed, Kill, Du, Dv, Dt float64
}

// Sim is a double-buffered Gray-Scott simulation on a W by H toroidal grid.
type Sim struct {
	W, H   int
	u, v   []float64 // current fields, row-major, len W*H
	nu, nv []float64 // scratch buffers for the next step
	params Params
}

// New creates a sim seeded with U=1 everywhere and a small central square of
// V=1 perturbation. A little randomness from rng nudges the seed so runs are
// reproducible for a given seed but not perfectly symmetric.
func New(w, h int, p Params, rng *rand.Rand) *Sim {
	w = max(w, 1)
	h = max(h, 1)
	n := w * h
	s := &Sim{
		W:      w,
		H:      h,
		u:      make([]float64, n),
		v:      make([]float64, n),
		nu:     make([]float64, n),
		nv:     make([]float64, n),
		params: p,
	}
	for i := range s.u {
		s.u[i] = 1
	}
	// Central perturbation square, sized relative to the grid.
	r := max(min(w, h)/10, 3)
	cx, cy := w/2, h/2
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			x := grid.Wrap(cx+dx, w)
			y := grid.Wrap(cy+dy, h)
			idx := y*w + x
			// Jitter the seed slightly so the pattern is not perfectly
			// symmetric; rng makes this reproducible per seed.
			s.u[idx] = 0.5 + (rng.Float64()-0.5)*0.02
			s.v[idx] = 0.25 + (rng.Float64()-0.5)*0.02
		}
	}
	return s
}

// laplacian weights for the classic Gray-Scott 9-point kernel.
const (
	wCenter = -1.0
	wOrtho  = 0.2
	wDiag   = 0.05
)

// Step advances the simulation by one Gray-Scott update with toroidal wrapping.
func (s *Sim) Step() {
	w, h := s.W, s.H
	p := s.params
	for y := range h {
		yn := grid.Wrap(y-1, h) * w
		yc := y * w
		ys := grid.Wrap(y+1, h) * w
		for x := range w {
			xw := grid.Wrap(x-1, w)
			xe := grid.Wrap(x+1, w)
			c := yc + x

			lapU := wCenter*s.u[c] +
				wOrtho*(s.u[yc+xw]+s.u[yc+xe]+s.u[yn+x]+s.u[ys+x]) +
				wDiag*(s.u[yn+xw]+s.u[yn+xe]+s.u[ys+xw]+s.u[ys+xe])
			lapV := wCenter*s.v[c] +
				wOrtho*(s.v[yc+xw]+s.v[yc+xe]+s.v[yn+x]+s.v[ys+x]) +
				wDiag*(s.v[yn+xw]+s.v[yn+xe]+s.v[ys+xw]+s.v[ys+xe])

			u, v := s.u[c], s.v[c]
			reaction := u * v * v
			s.nu[c] = u + (p.Du*lapU-reaction+p.Feed*(1-u))*p.Dt
			s.nv[c] = v + (p.Dv*lapV+reaction-(p.Kill+p.Feed)*v)*p.Dt
		}
	}
	s.u, s.nu = s.nu, s.u
	s.v, s.nv = s.nv, s.v
}

// StepN advances the simulation by n steps.
func (s *Sim) StepN(n int) {
	for range n {
		s.Step()
	}
}

// V returns the current V field (row-major, len W*H).
func (s *Sim) V() []float64 { return s.v }

// U returns the current U field (row-major, len W*H).
func (s *Sim) U() []float64 { return s.u }

// Image renders the current V field to an RGBA image using gradient g. The V
// field is normalized to [0,1] across its observed range for the lookup.
func (s *Sim) Image(g palette.Gradient) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, s.W, s.H))
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range s.v {
		lo = min(lo, v)
		hi = max(hi, v)
	}
	span := hi - lo
	for y := range s.H {
		for x := range s.W {
			v := s.v[y*s.W+x]
			t := 0.0
			if span > 0 {
				t = (v - lo) / span
			}
			c := g.At(t)
			img.SetRGBA(x, y, color.RGBA{R: c.R, G: c.G, B: c.B, A: 255})
		}
	}
	return img
}

// presets maps names to well-known Gray-Scott (feed, kill) pairs. All share the
// standard diffusion and timestep coefficients.
var presets = map[string][2]float64{
	"coral":    {0.0545, 0.062},
	"spots":    {0.035, 0.065},
	"mitosis":  {0.0367, 0.0649},
	"maze":     {0.029, 0.057},
	"waves":    {0.014, 0.054},
	"solitons": {0.030, 0.062},
}

// Preset returns the named parameter set and whether it exists.
func Preset(name string) (Params, bool) {
	fk, ok := presets[name]
	if !ok {
		return Params{}, false
	}
	return Params{Feed: fk[0], Kill: fk[1], Du: 0.16, Dv: 0.08, Dt: 1.0}, true
}

// PresetNames returns the available preset names, sorted.
func PresetNames() []string {
	return slices.Sorted(maps.Keys(presets))
}
