// Package palette maps a float in [0,1] to a 24-bit color via named gradients.
// It is shared by the fractal explorer (escape-time coloring) and the
// reaction-diffusion tool (concentration coloring).
package palette

import (
	"maps"
	"slices"

	"github.com/danielriddell21/toolshed/internal/render"
)

// Stop is a color anchored at position P in [0,1].
type Stop struct {
	P       float64
	R, G, B uint8
}

// Gradient is an ordered set of stops sampled by linear RGB interpolation.
type Gradient struct {
	stops []Stop
}

// New builds a gradient from stops. They are sorted by position; it panics with
// fewer than two stops.
func New(stops ...Stop) Gradient {
	if len(stops) < 2 {
		panic("palette: a gradient needs at least two stops")
	}
	s := slices.Clone(stops)
	slices.SortFunc(s, func(a, b Stop) int {
		switch {
		case a.P < b.P:
			return -1
		case a.P > b.P:
			return 1
		default:
			return 0
		}
	})
	return Gradient{stops: s}
}

// At samples the gradient at t, clamped to [0,1].
func (g Gradient) At(t float64) render.RGB {
	t = max(0, min(1, t))
	// Find the first stop whose position is >= t.
	i, found := slices.BinarySearchFunc(g.stops, t, func(s Stop, target float64) int {
		switch {
		case s.P < target:
			return -1
		case s.P > target:
			return 1
		default:
			return 0
		}
	})
	if found {
		return render.RGB{R: g.stops[i].R, G: g.stops[i].G, B: g.stops[i].B}
	}
	if i == 0 {
		return render.RGB{R: g.stops[0].R, G: g.stops[0].G, B: g.stops[0].B}
	}
	if i >= len(g.stops) {
		last := g.stops[len(g.stops)-1]
		return render.RGB{R: last.R, G: last.G, B: last.B}
	}
	lo, hi := g.stops[i-1], g.stops[i]
	span := hi.P - lo.P
	frac := 0.0
	if span > 0 {
		frac = (t - lo.P) / span
	}
	return render.Lerp(
		render.RGB{R: lo.R, G: lo.G, B: lo.B},
		render.RGB{R: hi.R, G: hi.G, B: hi.B},
		frac,
	)
}

// registry holds the built-in named gradients.
var registry = map[string]Gradient{
	"fire": New(
		Stop{0.0, 0, 0, 0}, Stop{0.4, 153, 0, 0},
		Stop{0.7, 255, 128, 0}, Stop{0.9, 255, 230, 100}, Stop{1.0, 255, 255, 255},
	),
	"ice": New(
		Stop{0.0, 4, 8, 30}, Stop{0.4, 12, 60, 120},
		Stop{0.75, 80, 170, 220}, Stop{1.0, 230, 245, 255},
	),
	"ultra": New(
		Stop{0.0, 0, 7, 100}, Stop{0.16, 32, 107, 203}, Stop{0.42, 237, 255, 255},
		Stop{0.64, 255, 170, 0}, Stop{0.86, 0, 2, 0}, Stop{1.0, 0, 7, 100},
	),
	"grayscale": New(
		Stop{0.0, 0, 0, 0}, Stop{1.0, 255, 255, 255},
	),
	"twilight": New(
		Stop{0.0, 20, 12, 40}, Stop{0.35, 90, 50, 120},
		Stop{0.6, 200, 90, 120}, Stop{0.8, 245, 170, 120}, Stop{1.0, 250, 240, 220},
	),
	"coral": New(
		Stop{0.0, 8, 12, 40}, Stop{0.5, 30, 90, 110},
		Stop{0.7, 220, 110, 80}, Stop{0.85, 250, 190, 130}, Stop{1.0, 255, 245, 235},
	),
}

// Named returns the registered gradient and whether it exists.
func Named(name string) (Gradient, bool) {
	g, ok := registry[name]
	return g, ok
}

// Names returns the registered gradient names, sorted.
func Names() []string {
	return slices.Sorted(maps.Keys(registry))
}
