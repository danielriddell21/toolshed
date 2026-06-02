package fractal

import (
	"math/cmplx"
	"testing"

	"github.com/danielriddell21/toolshed/internal/palette"
	"github.com/danielriddell21/toolshed/internal/render"
)

func TestEscapeSmoothInside(t *testing.T) {
	// The origin is firmly inside the Mandelbrot set: z=0, c=0 stays at 0.
	if got := EscapeSmooth(0, 0, 200); got != 1.0 {
		t.Fatalf("origin should be inside (1.0), got %v", got)
	}
}

func TestEscapeSmoothOutside(t *testing.T) {
	// 2+2i is well outside and escapes almost immediately.
	got := EscapeSmooth(0, complex(2, 2), 200)
	if got >= 1.0 {
		t.Fatalf("far point should escape (<1.0), got %v", got)
	}
	if got < 0 {
		t.Fatalf("escape time should be >= 0, got %v", got)
	}
}

func TestAdaptiveIterMonotonic(t *testing.T) {
	shallow := AdaptiveIter(baseScale)
	deep := AdaptiveIter(baseScale / 1024)
	if deep < shallow {
		t.Fatalf("deeper zoom should not reduce iterations: shallow=%d deep=%d", shallow, deep)
	}
	deeper := AdaptiveIter(baseScale / 1e9)
	if deeper < deep {
		t.Fatalf("even deeper zoom should not reduce iterations: deep=%d deeper=%d", deep, deeper)
	}
}

func TestPixelToComplexCenter(t *testing.T) {
	v := View{CenterRe: -0.5, CenterIm: 0.25, Scale: 0.01}
	w, h := 100, 100
	c := v.PixelToComplex(w/2, h/2, w, h)
	if cmplx.Abs(c-complex(v.CenterRe, v.CenterIm)) > v.Scale {
		t.Fatalf("center pixel should map near center: got %v want ~%v+%vi", c, v.CenterRe, v.CenterIm)
	}
}

func TestRenderFillsAndVaries(t *testing.T) {
	g, ok := palette.Named("ultra")
	if !ok {
		t.Fatal("ultra palette missing")
	}
	f := render.NewFrame(40, 40)
	v := View{CenterRe: -0.5, CenterIm: 0, Scale: 3.0 / 40}
	p := Params{Mode: Mandelbrot, MaxIter: AdaptiveIter(v.Scale)}

	Render(f, v, p, g) // must not panic

	pix := f.Pix()
	first := pix[0]
	allSame := true
	for _, c := range pix {
		if c != first {
			allSame = false
			break
		}
	}
	if allSame {
		t.Fatal("render across the set boundary should produce varied colors")
	}
}
