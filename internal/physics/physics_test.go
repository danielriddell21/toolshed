package physics

import (
	"math"
	"testing"

	"github.com/danielriddell21/toolshed/internal/render"
)

const fixedDt = 1.0 / 120

func inBounds(t *testing.T, w *World) {
	t.Helper()
	for i, b := range w.Balls {
		if b.Pos.X-b.R < -1e-6 || b.Pos.X+b.R > w.W+1e-6 ||
			b.Pos.Y-b.R < -1e-6 || b.Pos.Y+b.R > w.H+1e-6 {
			t.Fatalf("ball %d escaped: pos=%+v r=%v box=%vx%v", i, b.Pos, b.R, w.W, w.H)
		}
	}
}

func TestWallBounceRestitution(t *testing.T) {
	const e = 0.5
	w := NewWorld(100, 100, 500, e)
	// Start near the floor moving down; no x motion.
	w.Spawn(Vec{X: 50, Y: 90}, Vec{X: 0, Y: 200}, 5, render.RGB{})

	var impactSpeed float64
	bounced := false
	for range 1000 {
		prev := w.Balls[0].Vel.Y
		w.Step(fixedDt)
		inBounds(t, w)
		cur := w.Balls[0].Vel.Y
		// Detect the bounce: downward (positive) becomes upward (negative).
		if prev > 0 && cur < 0 {
			impactSpeed = prev
			bounced = true
			break
		}
	}
	if !bounced {
		t.Fatal("ball never bounced off the floor")
	}
	rebound := -w.Balls[0].Vel.Y // upward speed magnitude
	if rebound >= impactSpeed {
		t.Fatalf("rebound speed %v should be less than impact speed %v with e<1", rebound, impactSpeed)
	}
	// Rebound should be roughly impact*e (clamping/integration add small slop).
	if rebound > impactSpeed*e*1.2 {
		t.Fatalf("rebound %v too large vs impact*e=%v", rebound, impactSpeed*e)
	}
}

func TestMomentumConservation(t *testing.T) {
	w := NewWorld(1000, 1000, 0, 1.0)
	w.Spawn(Vec{X: 400, Y: 500}, Vec{X: 100, Y: 0}, 10, render.RGB{})
	w.Spawn(Vec{X: 600, Y: 500}, Vec{X: -100, Y: 0}, 10, render.RGB{})

	mom := func() Vec {
		var p Vec
		for _, b := range w.Balls {
			m := b.mass()
			p.X += m * b.Vel.X
			p.Y += m * b.Vel.Y
		}
		return p
	}
	before := mom()
	interacted := false
	for range 2000 {
		w.Step(fixedDt)
		inBounds(t, w)
		// A head-on elastic collision swaps the equal-mass velocities, so ball 0
		// turns around (its X velocity goes negative) at least once.
		if w.Balls[0].Vel.X < 0 {
			interacted = true
		}
	}
	after := mom()

	const tol = 1.0
	if math.Abs(after.X-before.X) > tol || math.Abs(after.Y-before.Y) > tol {
		t.Fatalf("momentum not conserved: before=%+v after=%+v", before, after)
	}
	// Sanity: the balls actually interacted (a collision reversed ball 0).
	if !interacted {
		t.Fatal("expected a collision to alter velocities")
	}
}

func TestFixedStepDeterminism(t *testing.T) {
	build := func() *World {
		w := NewWorld(300, 200, 300, 0.85)
		w.Spawn(Vec{X: 50, Y: 40}, Vec{X: 30, Y: 10}, 6, render.RGB{})
		w.Spawn(Vec{X: 120, Y: 80}, Vec{X: -20, Y: 25}, 8, render.RGB{})
		w.Spawn(Vec{X: 200, Y: 30}, Vec{X: 5, Y: -15}, 7, render.RGB{})
		return w
	}
	a, b := build(), build()
	for range 5000 {
		a.Step(fixedDt)
		b.Step(fixedDt)
	}
	for i := range a.Balls {
		if a.Balls[i].Pos != b.Balls[i].Pos || a.Balls[i].Vel != b.Balls[i].Vel {
			t.Fatalf("ball %d diverged: %+v vs %+v", i, a.Balls[i], b.Balls[i])
		}
	}
}

func TestNoEscape(t *testing.T) {
	w := NewWorld(160, 120, 600, 0.9)
	w.Spawn(Vec{X: 20, Y: 20}, Vec{X: 400, Y: -300}, 5, render.RGB{})
	w.Spawn(Vec{X: 80, Y: 60}, Vec{X: -500, Y: 200}, 9, render.RGB{})
	w.Spawn(Vec{X: 140, Y: 30}, Vec{X: 250, Y: 250}, 7, render.RGB{})
	w.Spawn(Vec{X: 100, Y: 100}, Vec{X: -350, Y: -450}, 6, render.RGB{})

	for range 20000 {
		w.Step(fixedDt)
		inBounds(t, w)
	}
}
