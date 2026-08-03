package scene

import (
	"math"

	"github.com/danielriddell21/toolshed/internal/render"
)

type Vec3 struct{ X, Y, Z float64 }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

func (a Vec3) Len() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Unit() Vec3 {
	if l := a.Len(); l > 1e-12 {
		return a.Scale(1 / l)
	}
	return Vec3{}
}

func Lerp(a, b Vec3, t float64) Vec3 { return a.Add(b.Sub(a).Scale(t)) }

// FovY is the vertical field of view every camera in this package uses; callers
// that need to work out how far back to sit have to match it.
const FovY = 1.0

type Camera struct {
	Eye    Vec3
	Target Vec3
	Up     Vec3
	FovY   float64
	Near   float64
	// PanX and PanY slide the projection centre without moving the camera, in
	// units of half a frame. Recentring a subject by swinging the camera instead
	// would change the perspective as it went, which never settles.
	PanX float64
	PanY float64
}

// Orbit places the camera on a sphere around a point: yaw sweeps around the
// vertical axis, pitch lifts it off the ground, and radius pulls it back.
func Orbit(target Vec3, yaw, pitch, radius float64) Camera {
	offset := Vec3{
		X: radius * math.Cos(pitch) * math.Cos(yaw),
		Y: radius * math.Cos(pitch) * math.Sin(yaw),
		Z: radius * math.Sin(pitch),
	}
	return Camera{
		Eye:    target.Add(offset),
		Target: target,
		Up:     Vec3{Z: 1},
		FovY:   FovY,
		Near:   0.1,
	}
}

// FitOrbit frames a set of points as tightly as an orbiting camera can: it backs
// off until the widest span just fits, then pans the projection centre so the
// subject sits in the middle. A pitched camera projects a box asymmetrically, so
// fitting by distance alone leaves one edge jammed against the frame and the
// opposite one half empty. margin is the slack around the result, 1 being snug.
func FitOrbit(target Vec3, yaw, pitch float64, points []Vec3, w, h int, pixelAspect, margin float64) Camera {
	if len(points) == 0 || w <= 0 || h <= 0 {
		return Orbit(target, yaw, pitch, 1)
	}

	radius := 0.0
	for _, p := range points {
		radius = math.Max(radius, p.Sub(target).Len())
	}

	span := func(dist float64) float64 {
		lo, hi, ok := Orbit(target, yaw, pitch, dist).bounds(points, w, h, pixelAspect)
		if !ok {
			return math.Inf(1)
		}
		return math.Max(hi.X-lo.X, hi.Y-lo.Y) / 2
	}

	// The projected span shrinks monotonically as the camera backs away, so the
	// snug distance can be bracketed and bisected. Stepping the distance by the
	// span directly overshoots and settles into a two-cycle instead.
	far := math.Max(radius, 1e-6) / math.Sin(FovY/2)
	near := far
	for span(near) < 1 && near > 1e-9 {
		near /= 2
	}
	for range 48 {
		mid := (near + far) / 2
		if span(mid) > 1 {
			near = mid
		} else {
			far = mid
		}
	}
	dist := far

	cam := Orbit(target, yaw, pitch, dist*math.Max(margin, 0.01))
	if lo, hi, ok := cam.bounds(points, w, h, pixelAspect); ok {
		cam.PanX, cam.PanY = (lo.X+hi.X)/2, (lo.Y+hi.Y)/2
	}
	return cam
}

// bounds is the projected extent of the points in units of half a frame, so 1 is
// exactly the edge whatever the resolution.
func (c Camera) bounds(points []Vec3, w, h int, pixelAspect float64) (lo, hi Vec3, ok bool) {
	fwd := c.Target.Sub(c.Eye).Unit()
	right := fwd.Cross(c.Up).Unit()
	up := right.Cross(fwd)
	tanV := math.Tan(c.FovY / 2)
	squeeze := 1 / pixelAspect
	aspect := float64(w) / float64(h)

	lo = Vec3{X: math.Inf(1), Y: math.Inf(1)}
	hi = Vec3{X: math.Inf(-1), Y: math.Inf(-1)}
	for _, p := range points {
		d := p.Sub(c.Eye)
		z := d.Dot(fwd)
		if z <= c.Near {
			return lo, hi, false
		}
		x := d.Dot(right) * squeeze / (tanV * z * aspect)
		y := d.Dot(up) / (tanV * z)
		lo.X, hi.X = math.Min(lo.X, x), math.Max(hi.X, x)
		lo.Y, hi.Y = math.Min(lo.Y, y), math.Max(hi.Y, y)
	}
	return lo, hi, true
}

type view struct {
	eye            Vec3
	right, up, fwd Vec3
	scale, squeeze float64
	cx, cy         float64
	near           float64
}

type Target struct {
	Light       Vec3
	Ambient     float64
	PixelAspect float64

	frame *render.Frame
	depth []float64
	v     view
}

func NewTarget(f *render.Frame) *Target {
	return &Target{
		Light:       Vec3{X: -0.4, Y: -0.6, Z: -0.7}.Unit(),
		Ambient:     0.35,
		PixelAspect: 1,
		frame:       f,
	}
}

func (t *Target) Begin(cam Camera, bg render.RGB) {
	w, h := t.frame.W, t.frame.H
	if len(t.depth) != w*h {
		t.depth = make([]float64, w*h)
	}
	t.frame.Fill(bg)
	for i := range t.depth {
		t.depth[i] = 0
	}

	aspect := t.PixelAspect
	if aspect <= 0 {
		aspect = 1
	}

	fwd := cam.Target.Sub(cam.Eye).Unit()
	right := fwd.Cross(cam.Up).Unit()
	up := right.Cross(fwd)
	t.v = view{
		eye:   cam.Eye,
		right: right,
		up:    up,
		fwd:   fwd,
		scale: float64(h) / 2 / math.Tan(cam.FovY/2),
		// Pixels are only square when a cell is split in half; quadrant cells give
		// pixels half as wide as they are tall, so horizontal offsets have to be
		// stretched by the same factor or the whole scene comes out squashed.
		squeeze: 1 / aspect,
		cx:      float64(w) / 2 * (1 - cam.PanX),
		cy:      float64(h) / 2 * (1 + cam.PanY),
		near:    cam.Near,
	}
}

func (t *Target) Project(p Vec3) (x, y, invZ float64, ok bool) {
	d := p.Sub(t.v.eye)
	z := d.Dot(t.v.fwd)
	if z <= t.v.near {
		return 0, 0, 0, false
	}
	return t.v.cx + d.Dot(t.v.right)*t.v.scale*t.v.squeeze/z,
		t.v.cy - d.Dot(t.v.up)*t.v.scale/z,
		1 / z, true
}

type vertex struct{ x, y, invZ float64 }

func (t *Target) triangle(a, b, c vertex, col render.RGB) {
	area := (b.x-a.x)*(c.y-a.y) - (b.y-a.y)*(c.x-a.x)
	if math.Abs(area) < 1e-9 {
		return
	}

	minX := max(0, int(math.Floor(min(a.x, min(b.x, c.x)))))
	maxX := min(t.frame.W-1, int(math.Ceil(max(a.x, max(b.x, c.x)))))
	minY := max(0, int(math.Floor(min(a.y, min(b.y, c.y)))))
	maxY := min(t.frame.H-1, int(math.Ceil(max(a.y, max(b.y, c.y)))))

	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			fx, fy := float64(px)+0.5, float64(py)+0.5
			w0 := ((b.x-a.x)*(fy-a.y) - (b.y-a.y)*(fx-a.x)) / area
			w1 := ((c.x-b.x)*(fy-b.y) - (c.y-b.y)*(fx-b.x)) / area
			w2 := 1 - w0 - w1
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}

			// 1/z is what varies linearly across the triangle in screen space,
			// so interpolating it gives an exact depth test.
			invZ := w1*a.invZ + w2*b.invZ + w0*c.invZ
			idx := py*t.frame.W + px
			if invZ <= t.depth[idx] {
				continue
			}
			t.depth[idx] = invZ
			t.frame.Set(px, py, col)
		}
	}
}

func (t *Target) quad(p [4]Vec3, normal Vec3, col render.RGB) {
	// Cull faces pointing away from the camera before any projection work.
	if normal.Dot(p[0].Sub(t.v.eye)) >= 0 {
		return
	}

	var v [4]vertex
	for i, p := range p {
		x, y, invZ, ok := t.Project(p)
		if !ok {
			return
		}
		v[i] = vertex{x, y, invZ}
	}

	shaded := shade(col, normal, t.Light, t.Ambient)
	t.triangle(v[0], v[1], v[2], shaded)
	t.triangle(v[0], v[2], v[3], shaded)
}

func shade(c render.RGB, normal, light Vec3, ambient float64) render.RGB {
	lit := ambient + (1-ambient)*max(0, -normal.Dot(light))
	return render.RGB{
		R: uint8(min(255, float64(c.R)*lit)),
		G: uint8(min(255, float64(c.G)*lit)),
		B: uint8(min(255, float64(c.B)*lit)),
	}
}

// Box draws an axis-aligned box rotated about the vertical axis, which covers
// both the slabs of the structure and the cars driving around on it.
func (t *Target) Box(centre, size Vec3, yaw float64, col render.RGB) {
	sin, cos := math.Sincos(yaw)
	ex := Vec3{X: cos, Y: sin}.Scale(size.X / 2)
	ey := Vec3{X: -sin, Y: cos}.Scale(size.Y / 2)
	ez := Vec3{Z: 1}.Scale(size.Z / 2)

	corner := func(sx, sy, sz float64) Vec3 {
		return centre.Add(ex.Scale(sx)).Add(ey.Scale(sy)).Add(ez.Scale(sz))
	}
	nx, ny, nz := ex.Unit(), ey.Unit(), ez.Unit()

	t.quad([4]Vec3{corner(-1, -1, 1), corner(1, -1, 1), corner(1, 1, 1), corner(-1, 1, 1)}, nz, col)
	t.quad([4]Vec3{corner(-1, -1, -1), corner(-1, 1, -1), corner(1, 1, -1), corner(1, -1, -1)}, nz.Scale(-1), col)
	t.quad([4]Vec3{corner(1, -1, -1), corner(1, 1, -1), corner(1, 1, 1), corner(1, -1, 1)}, nx, col)
	t.quad([4]Vec3{corner(-1, -1, -1), corner(-1, -1, 1), corner(-1, 1, 1), corner(-1, 1, -1)}, nx.Scale(-1), col)
	t.quad([4]Vec3{corner(-1, 1, -1), corner(-1, 1, 1), corner(1, 1, 1), corner(1, 1, -1)}, ny, col)
	t.quad([4]Vec3{corner(-1, -1, -1), corner(1, -1, -1), corner(1, -1, 1), corner(-1, -1, 1)}, ny.Scale(-1), col)
}

// Slab draws a box whose top face is tilted, which is what makes a ramp read as
// a ramp rather than a step.
func (t *Target) Slab(from, to Vec3, width, thickness float64, col render.RGB) {
	axis := to.Sub(from)
	flat := Vec3{X: axis.X, Y: axis.Y}.Unit()
	side := flat.Cross(Vec3{Z: 1}).Unit().Scale(width / 2)
	drop := Vec3{Z: -thickness}

	a, b := from.Sub(side), from.Add(side)
	c, d := to.Add(side), to.Sub(side)
	normal := c.Sub(b).Cross(a.Sub(b)).Unit()
	if normal.Z < 0 {
		normal = normal.Scale(-1)
	}

	t.quad([4]Vec3{a, b, c, d}, normal, col)
	t.quad([4]Vec3{a.Add(drop), d.Add(drop), c.Add(drop), b.Add(drop)}, Vec3{Z: -1}, col)
	t.quad([4]Vec3{b, b.Add(drop), c.Add(drop), c}, side.Unit(), col)
	t.quad([4]Vec3{a, d, d.Add(drop), a.Add(drop)}, side.Unit().Scale(-1), col)
}
