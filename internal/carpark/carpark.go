package carpark

import (
	"math"

	"github.com/danielriddell21/toolshed/internal/scene"
)

const (
	BayWidth    = 2.6
	BayDepth    = 5.0
	AisleWidth  = 6.4
	LevelHeight = 3.2
	RampRun     = 10.0
	ApronLength = 16.0
	LaneOffset  = 1.6

	CarLength = 4.4
	CarWidth  = 1.9
	CarHeight = 1.45
	Wheelbase = 2.7
)

type Bay struct {
	Level int
	Side  int
	Slot  int
	Pos   scene.Vec3
}

func (b Bay) Yaw() float64 { return math.Atan2(float64(b.Side), 0) }

type Structure struct {
	Levels  int
	PerSide int
	Bays    []Bay
}

func NewStructure(levels, perSide int) Structure {
	s := Structure{Levels: max(levels, 1), PerSide: max(perSide, 1)}
	// Bays are ordered outward from the exit: level by level, and within a level
	// the two sides interleaved so a deck fills evenly rather than one flank at
	// a time.
	for level := range s.Levels {
		for slot := range s.PerSide {
			for _, side := range []int{-1, 1} {
				s.Bays = append(s.Bays, Bay{
					Level: level,
					Side:  side,
					Slot:  slot,
					Pos: scene.Vec3{
						X: -s.HalfX() + (float64(slot)+0.5)*BayWidth,
						Y: float64(side) * (AisleWidth/2 + BayDepth/2),
						Z: float64(level) * LevelHeight,
					},
				})
			}
		}
	}
	return s
}

func (s Structure) HalfX() float64 { return float64(s.PerSide) * BayWidth / 2 }

func (s Structure) HalfY() float64 { return AisleWidth/2 + BayDepth }

func (s Structure) Height() float64 { return float64(s.Levels-1) * LevelHeight }

// RampSign alternates which end of the structure each ramp climbs from, the way
// a scissor ramp does, so a car drives the length of a deck between levels.
func RampSign(level int) float64 {
	if level%2 == 0 {
		return 1
	}
	return -1
}

func (s Structure) RampBottom(level int) scene.Vec3 {
	return scene.Vec3{X: RampSign(level) * (s.HalfX() - RampRun), Z: float64(level) * LevelHeight}
}

func (s Structure) RampTop(level int) scene.Vec3 {
	return scene.Vec3{X: RampSign(level) * s.HalfX(), Z: float64(level+1) * LevelHeight}
}

// route is the run between the entrance and a bay: in along the apron, up one
// ramp per level, along the aisle of the target deck, then a turn into the bay.
// The aisle is two-way, so arrivals and departures are given opposite lanes and
// pass each other rather than meeting nose to nose.
func (s Structure) route(b Bay, lane float64) []scene.Vec3 {
	path := []scene.Vec3{
		{X: -s.HalfX() - ApronLength, Y: lane},
		{X: -s.HalfX(), Y: lane},
	}
	for level := range b.Level {
		bottom, top := s.RampBottom(level), s.RampTop(level)
		bottom.Y, top.Y = lane, lane
		path = append(path, bottom, top)
	}
	z := float64(b.Level) * LevelHeight
	return append(path,
		scene.Vec3{X: b.Pos.X, Y: lane, Z: z},
		scene.Vec3{X: b.Pos.X, Y: b.Pos.Y, Z: z})
}

func (s Structure) PathTo(b Bay) []scene.Vec3 { return s.route(b, -LaneOffset) }

func (s Structure) PathFrom(b Bay) []scene.Vec3 {
	out := s.route(b, LaneOffset)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

type Car struct {
	Pos     scene.Vec3
	Yaw     float64
	Speed   float64
	Paint   int
	Bay     int
	Leaving bool

	path  []scene.Vec3
	marks []float64
	arc   float64
	done  bool
}

func newCar(path []scene.Vec3, paint, bay int, leaving bool) *Car {
	marks := make([]float64, len(path))
	for i := 1; i < len(path); i++ {
		marks[i] = marks[i-1] + path[i].Sub(path[i-1]).Len()
	}
	heading := path[1].Sub(path[0])
	return &Car{
		Pos:     path[0],
		Yaw:     math.Atan2(heading.Y, heading.X),
		Paint:   paint,
		Bay:     bay,
		Leaving: leaving,
		path:    path,
		marks:   marks,
	}
}

func (c *Car) Length() float64 { return c.marks[len(c.marks)-1] }

func (c *Car) Remaining() float64 { return c.Length() - c.arc }

func (c *Car) Done() bool { return c.done }

// at walks the polyline to the point a given distance along it.
func (c *Car) at(arc float64) scene.Vec3 {
	arc = min(max(arc, 0), c.Length())
	for i := 1; i < len(c.marks); i++ {
		if arc <= c.marks[i] {
			span := c.marks[i] - c.marks[i-1]
			if span < 1e-9 {
				return c.path[i]
			}
			return scene.Lerp(c.path[i-1], c.path[i], (arc-c.marks[i-1])/span)
		}
	}
	return c.path[len(c.path)-1]
}

const (
	desiredSpeed = 5.5
	maxAccel     = 1.9
	comfortBrake = 2.4
	standstill   = 1.8
	headwayTime  = 1.0
	lookaheadMin = 3.0
)

// intelligentDriver is the IDM of Treiber, Hennecke & Helbing (2000): a free
// acceleration term that eases off as the car approaches its desired speed, and
// an interaction term that brakes hard as the gap to whatever is ahead closes.
func intelligentDriver(speed, gap, closing float64) float64 {
	free := 1 - math.Pow(speed/desiredSpeed, 4)
	if gap <= 0.05 {
		return -comfortBrake * 3
	}
	want := standstill + max(0, speed*headwayTime+speed*closing/(2*math.Sqrt(maxAccel*comfortBrake)))
	crowding := want / gap
	return maxAccel * (free - crowding*crowding)
}

func (c *Car) step(dt, gap, closing float64) {
	// The end of the route is just another obstacle, offset by the standstill gap
	// the driver model likes to keep so that the car comes to rest exactly on the
	// bay rather than a car's length short of it.
	if stop := c.Remaining() + standstill; stop < gap {
		gap, closing = stop, c.Speed
	}

	c.Speed = max(0, c.Speed+intelligentDriver(c.Speed, gap, closing)*dt)
	c.arc = min(c.arc+c.Speed*dt, c.Length())

	// Pure pursuit: steer at a point further along the route, so the car swings
	// through its turns instead of pivoting on the spot.
	lookahead := max(lookaheadMin, c.Speed*0.6)
	target := c.at(c.arc + lookahead)
	toward := math.Atan2(target.Y-c.Pos.Y, target.X-c.Pos.X)
	alpha := wrapAngle(toward - c.Yaw)
	c.Yaw = wrapAngle(c.Yaw + 2*c.Speed*math.Sin(alpha)/lookahead*dt)

	c.Pos.X += c.Speed * math.Cos(c.Yaw) * dt
	c.Pos.Y += c.Speed * math.Sin(c.Yaw) * dt
	c.Pos.Z = c.at(c.arc).Z

	if c.Remaining() < 0.15 && c.Speed < 0.25 {
		end := c.path[len(c.path)-1]
		c.Pos = end
		c.Speed = 0
		c.done = true
	}
}

func wrapAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}
