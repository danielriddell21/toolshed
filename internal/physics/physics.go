package physics

import (
	"math"

	"github.com/danielriddell21/toolshed/internal/render"
)

type Vec struct{ X, Y float64 }

type Ball struct {
	Pos, Vel Vec
	R        float64
	Color    render.RGB
}

func (b Ball) mass() float64 { return b.R * b.R }

type World struct {
	W, H        float64
	Gravity     Vec
	Restitution float64
	Damping     float64
	Balls       []Ball
}

const tangentialDamping = 0.98

func NewWorld(w, h float64, gravity, restitution float64) *World {
	return &World{
		W:           w,
		H:           h,
		Gravity:     Vec{X: 0, Y: gravity},
		Restitution: restitution,
		Damping:     1.0,
	}
}

func (w *World) Spawn(pos, vel Vec, r float64, c render.RGB) {
	r = max(r, 1)
	pos.X = clamp(pos.X, r, w.W-r)
	pos.Y = clamp(pos.Y, r, w.H-r)
	w.Balls = append(w.Balls, Ball{Pos: pos, Vel: vel, R: r, Color: c})
}

func (w *World) Clear() { w.Balls = w.Balls[:0] }

func (w *World) ToggleGravity() { w.Gravity.Y = -w.Gravity.Y }

func (w *World) Step(dt float64) {
	damp := w.Damping
	if damp == 0 {
		damp = 1.0
	}

	// Integrate.
	for i := range w.Balls {
		b := &w.Balls[i]
		b.Vel.X += w.Gravity.X * dt
		b.Vel.Y += w.Gravity.Y * dt
		b.Vel.X *= damp
		b.Vel.Y *= damp
		b.Pos.X += b.Vel.X * dt
		b.Pos.Y += b.Vel.Y * dt
	}

	// Walls.
	for i := range w.Balls {
		w.resolveWalls(&w.Balls[i])
	}

	// Ball-ball collisions.
	for i := range w.Balls {
		for j := i + 1; j < len(w.Balls); j++ {
			w.resolvePair(&w.Balls[i], &w.Balls[j])
		}
	}

	// Re-clamp to the box: a collision can shove a ball back through a wall, so
	// clamp positions afterward to guarantee nothing ever escapes. Zero the
	// outward velocity component so a pinned ball doesn't keep driving outward.
	for i := range w.Balls {
		w.clampInside(&w.Balls[i])
	}
}

func (w *World) clampInside(b *Ball) {
	if b.Pos.X-b.R < 0 {
		b.Pos.X = b.R
		b.Vel.X = max(b.Vel.X, 0)
	} else if b.Pos.X+b.R > w.W {
		b.Pos.X = w.W - b.R
		b.Vel.X = min(b.Vel.X, 0)
	}
	if b.Pos.Y-b.R < 0 {
		b.Pos.Y = b.R
		b.Vel.Y = max(b.Vel.Y, 0)
	} else if b.Pos.Y+b.R > w.H {
		b.Pos.Y = w.H - b.R
		b.Vel.Y = min(b.Vel.Y, 0)
	}
}

func (w *World) resolveWalls(b *Ball) {
	if b.Pos.X-b.R < 0 {
		b.Pos.X = b.R
		b.Vel.X = -b.Vel.X * w.Restitution
		b.Vel.Y *= tangentialDamping
	} else if b.Pos.X+b.R > w.W {
		b.Pos.X = w.W - b.R
		b.Vel.X = -b.Vel.X * w.Restitution
		b.Vel.Y *= tangentialDamping
	}

	if b.Pos.Y-b.R < 0 {
		b.Pos.Y = b.R
		b.Vel.Y = -b.Vel.Y * w.Restitution
		b.Vel.X *= tangentialDamping
	} else if b.Pos.Y+b.R > w.H {
		b.Pos.Y = w.H - b.R
		b.Vel.Y = -b.Vel.Y * w.Restitution
		b.Vel.X *= tangentialDamping
	}
}

func (w *World) resolvePair(a, b *Ball) {
	dx := b.Pos.X - a.Pos.X
	dy := b.Pos.Y - a.Pos.Y
	dist := math.Hypot(dx, dy)
	minDist := a.R + b.R
	if dist >= minDist {
		return
	}

	// Contact normal. If the centers coincide, pick an arbitrary axis to avoid a
	// divide-by-zero and still separate them.
	var nx, ny float64
	if dist > 1e-9 {
		nx, ny = dx/dist, dy/dist
	} else {
		nx, ny, dist = 1, 0, 0
	}

	ma, mb := a.mass(), b.mass()

	// Positional correction: separate by the overlap, split by inverse mass so
	// the lighter ball moves more.
	overlap := minDist - dist
	invSum := 1/ma + 1/mb
	corrA := overlap * (1 / ma) / invSum
	corrB := overlap * (1 / mb) / invSum
	a.Pos.X -= nx * corrA
	a.Pos.Y -= ny * corrA
	b.Pos.X += nx * corrB
	b.Pos.Y += ny * corrB

	// Relative velocity along the normal.
	rvx := b.Vel.X - a.Vel.X
	rvy := b.Vel.Y - a.Vel.Y
	velAlongNormal := rvx*nx + rvy*ny
	if velAlongNormal > 0 {
		// Already separating; don't add energy.
		return
	}

	// Elastic impulse (restitution e = 1 along the normal for satisfying bounces;
	// the wall restitution governs the box, not ball-ball energy loss).
	j := -2 * velAlongNormal / invSum
	jx, jy := j*nx, j*ny
	a.Vel.X -= jx / ma
	a.Vel.Y -= jy / ma
	b.Vel.X += jx / mb
	b.Vel.Y += jy / mb
}

func clamp(v, lo, hi float64) float64 {
	if hi < lo {
		return lo
	}
	return max(lo, min(hi, v))
}
