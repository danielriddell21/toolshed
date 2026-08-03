package main

import (
	"math"

	"github.com/danielriddell21/toolshed/internal/carpark"
	"github.com/danielriddell21/toolshed/internal/render"
	"github.com/danielriddell21/toolshed/internal/scene"
)

const (
	deckLevels   = 4
	deckPerSide  = 6
	deckDrop     = 0.34
	kerbHeight   = 0.08
	kerbWidth    = 0.12
	sceneMinRows = 6
	superSample  = 2
	pixelAspect  = 0.5
	deepShade    = 0.55
	spinRate     = 0.22
	pitchMin     = 0.08
	pitchMax     = 1.35
	zoomMin      = 1.15
	zoomMax      = 3.4
)

var (
	sky      = render.RGB{R: 14, G: 16, B: 22}
	deckPain = render.RGB{R: 92, G: 101, B: 116}
	rampPain = render.RGB{R: 116, G: 92, B: 62}
	kerbPain = render.RGB{R: 150, G: 158, B: 170}
	glass    = render.RGB{R: 40, G: 46, B: 58}
	tarmac   = render.RGB{R: 30, G: 34, B: 42}
)

const groundDrop = 1.2

func paintOf(index int, deep bool) render.RGB {
	c := carPaint[index%len(carPaint)]
	if !deep {
		return render.RGB{R: c[0], G: c[1], B: c[2]}
	}
	return render.RGB{
		R: uint8(float64(c[0]) * deepShade),
		G: uint8(float64(c[1]) * deepShade),
		B: uint8(float64(c[2]) * deepShade),
	}
}

// camera frames the whole structure however it happens to be turned, then backs
// off by the zoom setting for a little air around it. The low default pitch is
// deliberate: it is what lets you see in through the open sides and count the
// cars parked on every deck, not just the top one.
func (m model) camera(s carpark.Structure) scene.Camera {
	elevation := s.Height() + carpark.CarHeight + deckDrop
	centre := scene.Vec3{Z: elevation / 2}

	// Only the decks are framed. The approach road runs off the edge, so cars
	// arrive from off-screen rather than the structure being squeezed into the
	// middle of the frame to make room for empty tarmac.
	corners := make([]scene.Vec3, 0, 8)
	for _, x := range []float64{-s.HalfX(), s.HalfX()} {
		for _, y := range []float64{-s.HalfY(), s.HalfY()} {
			for _, z := range []float64{-deckDrop, elevation} {
				corners = append(corners, scene.Vec3{X: x, Y: y, Z: z})
			}
		}
	}

	return scene.FitOrbit(centre, m.camYaw, m.camPitch, corners,
		m.frame.W, m.frame.H, pixelAspect, m.camZoom)
}

func (m model) drawStructure(t *scene.Target, s carpark.Structure) {
	for level := range s.Levels {
		z := float64(level) * carpark.LevelHeight
		t.Box(scene.Vec3{Z: z - deckDrop/2},
			scene.Vec3{X: 2 * s.HalfX(), Y: 2 * s.HalfY(), Z: deckDrop}, 0, deckPain)

		// The apron only exists on the ground floor: it is the way in and out.
		if level == 0 {
			t.Box(scene.Vec3{X: -s.HalfX() - carpark.ApronLength/2, Z: -deckDrop / 2},
				scene.Vec3{X: carpark.ApronLength, Y: 2 * carpark.LaneOffset * 2, Z: deckDrop}, 0, deckPain)
		}
		if level < s.Levels-1 {
			t.Slab(s.RampBottom(level), s.RampTop(level), carpark.AisleWidth, deckDrop, rampPain)
		}

		for slot := 0; slot <= s.PerSide; slot++ {
			x := -s.HalfX() + float64(slot)*carpark.BayWidth
			for _, side := range []float64{-1, 1} {
				t.Box(scene.Vec3{X: x, Y: side * (carpark.AisleWidth/2 + carpark.BayDepth/2), Z: z + kerbHeight/2},
					scene.Vec3{X: kerbWidth, Y: carpark.BayDepth, Z: kerbHeight}, 0, kerbPain)
			}
		}
	}
}

func drawCar(t *scene.Target, at scene.Vec3, yaw float64, paint render.RGB) {
	body := at
	body.Z += carpark.CarHeight * 0.3
	t.Box(body, scene.Vec3{X: carpark.CarLength, Y: carpark.CarWidth, Z: carpark.CarHeight * 0.6}, yaw, paint)

	cabin := at
	cabin.Z += carpark.CarHeight * 0.78
	t.Box(cabin, scene.Vec3{
		X: carpark.CarLength * 0.46,
		Y: carpark.CarWidth * 0.86,
		Z: carpark.CarHeight * 0.36,
	}, yaw, glass)
}

func (m model) drawTraffic(t *scene.Target, p *carpark.Park) {
	for bay, state := range p.State {
		if state != carpark.Taken {
			continue
		}
		b := p.S.Bays[bay]
		drawCar(t, b.Pos, b.Yaw(), paintOf(carpark.Paint(bay), bay >= p.Front))
	}
	for _, c := range p.Cars {
		drawCar(t, c.Pos, c.Yaw, paintOf(c.Paint, c.Bay >= p.Front))
	}
}

func (m model) sceneView(rows int) string {
	cols := max(m.Width-2, 8)
	rows = max(rows, sceneMinRows)
	m.frame.Resize(cols*2, rows*2)
	m.hires.Resize(cols*2*superSample, rows*2*superSample)

	m.target.Begin(m.camera(m.park.S), sky)
	m.drawGround(m.target, m.park.S)
	m.drawStructure(m.target, m.park.S)
	m.drawTraffic(m.target, m.park)

	render.Downsample(m.hires, m.frame)
	return m.frame.Quadrants()
}

func (m model) drawGround(t *scene.Target, s carpark.Structure) {
	// A dark apron under the whole thing stops the structure reading as if it
	// were floating in space.
	span := 4 * (s.HalfX() + carpark.ApronLength)
	t.Box(scene.Vec3{X: -carpark.ApronLength / 2, Z: -groundDrop / 2},
		scene.Vec3{X: span, Y: span, Z: groundDrop}, 0, tarmac)
}

func (m *model) orbit(dYaw, dPitch, dZoom float64) {
	m.camYaw += dYaw
	m.camPitch = math.Max(pitchMin, math.Min(pitchMax, m.camPitch+dPitch))
	m.camZoom = math.Max(zoomMin, math.Min(zoomMax, m.camZoom+dZoom))
}
