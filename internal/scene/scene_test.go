package scene

import (
	"math"
	"testing"

	"github.com/danielriddell21/toolshed/internal/render"
)

func lookAlongX() Camera {
	return Camera{Eye: Vec3{}, Target: Vec3{X: 1}, Up: Vec3{Z: 1}, FovY: 1, Near: 0.1}
}

func newTarget(t *testing.T, w, h int) (*Target, *render.Frame) {
	t.Helper()
	f := render.NewFrame(w, h)
	return NewTarget(f), f
}

func TestVectorAlgebra(t *testing.T) {
	a, b := Vec3{X: 1, Y: 2, Z: 3}, Vec3{X: -4, Y: 5, Z: 6}

	cross := a.Cross(b)
	if got := cross.Dot(a); math.Abs(got) > 1e-12 {
		t.Errorf("cross product not perpendicular to a: %.12f", got)
	}
	if got := cross.Dot(b); math.Abs(got) > 1e-12 {
		t.Errorf("cross product not perpendicular to b: %.12f", got)
	}
	if got := a.Unit().Len(); math.Abs(got-1) > 1e-12 {
		t.Errorf("unit length = %.12f, want 1", got)
	}
	if got := (Vec3{}).Unit(); got != (Vec3{}) {
		t.Errorf("unit of the zero vector = %+v, want zero", got)
	}
	if got := Lerp(a, b, 0.5); got != (Vec3{X: -1.5, Y: 3.5, Z: 4.5}) {
		t.Errorf("midpoint = %+v", got)
	}
}

func TestProjectionIsPerspective(t *testing.T) {
	tg, f := newTarget(t, 80, 40)
	tg.Begin(lookAlongX(), render.RGB{})
	cx, cy := float64(f.W)/2, float64(f.H)/2

	x, y, invZ, ok := tg.Project(Vec3{X: 10})
	if !ok {
		t.Fatal("a point straight ahead should be visible")
	}
	if math.Abs(x-cx) > 1e-9 || math.Abs(y-cy) > 1e-9 {
		t.Errorf("dead ahead projected to (%.4f, %.4f), want the centre (%.1f, %.1f)", x, y, cx, cy)
	}
	if math.Abs(invZ-0.1) > 1e-12 {
		t.Errorf("depth = %.12f, want 1/10", invZ)
	}

	up, _, _, _ := func() (float64, float64, float64, bool) {
		_, y, z, ok := tg.Project(Vec3{X: 10, Z: 2})
		return y, 0, z, ok
	}()
	if up >= cy {
		t.Errorf("a point above the axis projected to y=%.4f, want above the centre %.1f", up, cy)
	}

	// Twice as far away should land half as far from the centre.
	_, near, _, _ := tg.Project(Vec3{X: 10, Z: 2})
	_, far, _, _ := tg.Project(Vec3{X: 20, Z: 2})
	if math.Abs((cy-far)*2-(cy-near)) > 1e-9 {
		t.Errorf("perspective divide wrong: offsets %.6f and %.6f", cy-near, cy-far)
	}

	if _, _, _, ok := tg.Project(Vec3{X: -5}); ok {
		t.Error("a point behind the camera should be rejected")
	}
}

func TestBeginClearsColourAndDepth(t *testing.T) {
	tg, f := newTarget(t, 20, 10)
	bg := render.RGB{R: 9, G: 8, B: 7}

	tg.Begin(lookAlongX(), bg)
	tg.Box(Vec3{X: 8}, Vec3{X: 4, Y: 4, Z: 4}, 0, render.RGB{R: 200})
	if f.At(10, 5) == bg {
		t.Fatal("the box did not draw over the background")
	}

	tg.Begin(lookAlongX(), bg)
	if got := f.At(10, 5); got != bg {
		t.Errorf("Begin left %+v behind, want the background %+v", got, bg)
	}
}

func TestNearerSurfacesWin(t *testing.T) {
	blueDominant := func(t *testing.T, order string, first, second render.RGB, firstX, secondX float64) {
		t.Helper()
		tg, f := newTarget(t, 60, 30)
		tg.Begin(lookAlongX(), render.RGB{})
		tg.Box(Vec3{X: firstX}, Vec3{X: 2, Y: 30, Z: 30}, 0, first)
		tg.Box(Vec3{X: secondX}, Vec3{X: 2, Y: 30, Z: 30}, 0, second)

		got := f.At(30, 15)
		if got.B <= got.R {
			t.Errorf("%s: centre pixel %+v is not the nearer blue box", order, got)
		}
	}

	red, blue := render.RGB{R: 220}, render.RGB{B: 220}
	blueDominant(t, "far first", red, blue, 40, 12)
	blueDominant(t, "near first", blue, red, 12, 40)
}

func TestBoxesBehindTheCameraDrawNothing(t *testing.T) {
	tg, f := newTarget(t, 40, 20)
	bg := render.RGB{R: 1, G: 2, B: 3}
	tg.Begin(lookAlongX(), bg)
	tg.Box(Vec3{X: -30}, Vec3{X: 5, Y: 5, Z: 5}, 0, render.RGB{R: 255, G: 255, B: 255})

	for _, px := range f.Pix() {
		if px != bg {
			t.Fatalf("something was drawn from behind the camera: %+v", px)
		}
	}
}

func TestShadingRespondsToTheLight(t *testing.T) {
	light := Vec3{Z: -1}
	facing := shade(render.RGB{R: 200, G: 200, B: 200}, Vec3{Z: 1}, light, 0.25)
	away := shade(render.RGB{R: 200, G: 200, B: 200}, Vec3{Z: -1}, light, 0.25)

	if facing.R <= away.R {
		t.Errorf("a face turned into the light (%d) should be brighter than one turned away (%d)", facing.R, away.R)
	}
	if want := uint8(200 * 0.25); away.R != want {
		t.Errorf("unlit face = %d, want the ambient floor %d", away.R, want)
	}
}

func TestFitOrbitFramesEverythingSnugly(t *testing.T) {
	const w, h = 200, 60
	var corners []Vec3
	for _, x := range []float64{-9, 9} {
		for _, y := range []float64{-7, 7} {
			for _, z := range []float64{0, 12} {
				corners = append(corners, Vec3{X: x, Y: y, Z: z})
			}
		}
	}

	for _, yaw := range []float64{-0.9, 0, 0.7, 2.4} {
		cam := FitOrbit(Vec3{Z: 6}, yaw, 0.24, corners, w, h, 0.5, 1)

		f := render.NewFrame(w, h)
		tg := NewTarget(f)
		tg.PixelAspect = 0.5
		tg.Begin(cam, render.RGB{})

		touched := false
		for _, c := range corners {
			x, y, _, ok := tg.Project(c)
			if !ok {
				t.Fatalf("yaw %.2f: corner %+v fell behind the camera", yaw, c)
			}
			if x < -0.5 || x > float64(f.W)+0.5 || y < -0.5 || y > float64(f.H)+0.5 {
				t.Errorf("yaw %.2f: corner %+v projected off frame at (%.2f, %.2f)", yaw, c, x, y)
			}
			if x < 0.02*float64(f.W) || x > 0.98*float64(f.W) ||
				y < 0.02*float64(f.H) || y > 0.98*float64(f.H) {
				touched = true
			}
		}
		if !touched {
			t.Errorf("yaw %.2f: nothing reaches an edge, so the fit is loose", yaw)
		}
	}
}

func TestFitOrbitMarginBacksOff(t *testing.T) {
	corners := []Vec3{{X: -5, Y: -5}, {X: 5, Y: 5, Z: 4}}
	snug := FitOrbit(Vec3{}, 0.3, 0.3, corners, 100, 50, 1, 1)
	loose := FitOrbit(Vec3{}, 0.3, 0.3, corners, 100, 50, 1, 1.5)

	if loose.Eye.Sub(loose.Target).Len() <= snug.Eye.Sub(snug.Target).Len() {
		t.Error("a larger margin should sit the camera further back")
	}
}

func TestOrbitLooksAtItsTarget(t *testing.T) {
	target := Vec3{X: 3, Y: -2, Z: 1}
	cam := Orbit(target, 0.7, 0.4, 25)

	if cam.Target != target {
		t.Errorf("camera target = %+v, want %+v", cam.Target, target)
	}
	if got := cam.Eye.Sub(target).Len(); math.Abs(got-25) > 1e-9 {
		t.Errorf("orbit radius = %.9f, want 25", got)
	}
	if cam.Eye.Z <= target.Z {
		t.Errorf("a positive pitch should lift the camera: eye z %.4f vs target z %.4f", cam.Eye.Z, target.Z)
	}
}
