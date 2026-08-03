package carpark

import (
	"math"
	"testing"
)

func TestBaysAreOrderedOutwardFromTheExit(t *testing.T) {
	s := NewStructure(3, 4)

	if want := 3 * 4 * 2; len(s.Bays) != want {
		t.Fatalf("bay count = %d, want %d", len(s.Bays), want)
	}
	if s.Bays[0].Level != 0 || s.Bays[0].Slot != 0 {
		t.Errorf("first bay = %+v, want the nearest slot on the ground level", s.Bays[0])
	}
	if last := s.Bays[len(s.Bays)-1]; last.Level != 2 {
		t.Errorf("last bay is on level %d, want the top level 2", last.Level)
	}

	for i := 1; i < len(s.Bays); i++ {
		if s.Bays[i].Level < s.Bays[i-1].Level {
			t.Fatalf("bay %d drops back to level %d", i, s.Bays[i].Level)
		}
	}
	for _, b := range s.Bays {
		if math.Abs(b.Pos.Y) <= AisleWidth/2 {
			t.Errorf("bay %+v sits in the aisle", b)
		}
		if want := float64(b.Level) * LevelHeight; b.Pos.Z != want {
			t.Errorf("bay on level %d has z %.3f, want %.3f", b.Level, b.Pos.Z, want)
		}
	}
}

func TestRampsAlternateEnds(t *testing.T) {
	if RampSign(0) == RampSign(1) {
		t.Error("consecutive ramps should climb from opposite ends")
	}
	s := NewStructure(3, 4)
	for level := range 2 {
		bottom, top := s.RampBottom(level), s.RampTop(level)
		if top.Z-bottom.Z != LevelHeight {
			t.Errorf("ramp %d rises %.3f, want one level (%.3f)", level, top.Z-bottom.Z, LevelHeight)
		}
		if math.Abs(top.X-bottom.X) != RampRun {
			t.Errorf("ramp %d runs %.3f, want %.3f", level, math.Abs(top.X-bottom.X), RampRun)
		}
	}
}

func TestRouteClimbsOneRampPerLevel(t *testing.T) {
	s := NewStructure(4, 3)

	for _, b := range s.Bays {
		path := s.PathTo(b)
		if want := 4 + 2*b.Level; len(path) != want {
			t.Fatalf("route to level %d has %d waypoints, want %d", b.Level, len(path), want)
		}
		if path[len(path)-1] != b.Pos {
			t.Errorf("route ends at %+v, want the bay at %+v", path[len(path)-1], b.Pos)
		}
		if path[0].X > -s.HalfX() {
			t.Errorf("route starts at x=%.3f, want it outside the structure", path[0].X)
		}
		for i := 1; i < len(path); i++ {
			if path[i].Z < path[i-1].Z {
				t.Fatalf("route to level %d descends at waypoint %d", b.Level, i)
			}
		}
	}
}

func TestLeavingReversesTheRoute(t *testing.T) {
	s := NewStructure(3, 3)
	b := s.Bays[len(s.Bays)-1]

	in, out := s.PathTo(b), s.PathFrom(b)
	if len(in) != len(out) {
		t.Fatalf("route lengths differ: %d in, %d out", len(in), len(out))
	}
	if out[0] != b.Pos {
		t.Errorf("departure starts at %+v, want the bay at %+v", out[0], b.Pos)
	}
	if in[0].X != out[len(out)-1].X {
		t.Errorf("departure ends at x=%.3f, want the entrance x=%.3f", out[len(out)-1].X, in[0].X)
	}
	for i := 1; i < len(out); i++ {
		if out[i].Z > out[i-1].Z {
			t.Fatalf("departure climbs at waypoint %d", i)
		}
	}
	for i := 1; i < len(in)-1; i++ {
		if in[i].Y == out[len(out)-1-i].Y {
			t.Errorf("waypoint %d shares a lane with oncoming traffic at y=%.3f", i, in[i].Y)
		}
	}
}

func TestACarDrivesUpToTheTopAndParks(t *testing.T) {
	s := NewStructure(4, 4)
	bay := s.Bays[len(s.Bays)-1]

	c := newCar(s.PathTo(bay), 0, 0, false)
	top := 0.0
	for range 20000 {
		c.step(0.02, math.Inf(1), 0)
		top = max(top, c.Speed)
		if c.done {
			break
		}
	}

	if !c.done {
		t.Fatalf("car did not arrive: %.2f of %.2f m", c.arc, c.Length())
	}
	if c.Pos != bay.Pos {
		t.Errorf("parked at %+v, want the bay at %+v", c.Pos, bay.Pos)
	}
	if c.Speed != 0 {
		t.Errorf("parked car still doing %.3f m/s", c.Speed)
	}
	if top > desiredSpeed+0.01 {
		t.Errorf("car reached %.3f m/s, above its desired %.3f", top, desiredSpeed)
	}
	if top < desiredSpeed*0.6 {
		t.Errorf("car only reached %.3f m/s on a long clear route", top)
	}
}

func TestDriversAccelerateWhenClearAndBrakeWhenClosing(t *testing.T) {
	if a := intelligentDriver(0, math.Inf(1), 0); a <= 0 {
		t.Errorf("a stopped car on a clear road should pull away, got %.3f m/s²", a)
	}
	if a := intelligentDriver(desiredSpeed, math.Inf(1), 0); math.Abs(a) > 1e-9 {
		t.Errorf("a car at its desired speed should hold it, got %.3f m/s²", a)
	}
	if a := intelligentDriver(4, 3, 4); a >= 0 {
		t.Errorf("closing fast on a near obstacle should brake, got %.3f m/s²", a)
	}
	if open, tight := intelligentDriver(4, 40, 0), intelligentDriver(4, 8, 0); tight >= open {
		t.Errorf("a tighter gap should mean less acceleration: %.3f vs %.3f", tight, open)
	}
}

func TestQueueingCarsDoNotDriveThroughEachOther(t *testing.T) {
	p := New(3, 5, 0.6)
	closest := math.Inf(1)

	for range 8000 {
		p.Sync(1, 1)
		p.Step(0.02)
		for i, a := range p.Cars {
			for _, b := range p.Cars[i+1:] {
				d := b.Pos.Sub(a.Pos)
				if math.Abs(d.Z) > levelClear {
					continue
				}
				sin, cos := math.Sincos(a.Yaw)
				along := d.X*cos + d.Y*sin
				lateral := -d.X*sin + d.Y*cos
				if along > 0 && math.Abs(lateral) < corridorHalf {
					closest = min(closest, along)
				}
			}
		}
	}

	if closest < CarLength {
		t.Errorf("cars closed to %.3f m nose to tail, shorter than the %.2f m car", closest, CarLength)
	}
}

func TestTrafficFillsAndEmptiesTheStructure(t *testing.T) {
	p := New(3, 4, 0.5)

	run := func(front, deep float64, steps int) {
		for range steps {
			p.Sync(front, deep)
			p.Step(0.05)
		}
	}

	run(1, 1, 6000)
	if got := p.Occupied(); got != len(p.S.Bays) {
		t.Errorf("filled to %d of %d bays", got, len(p.S.Bays))
	}

	run(0, 0, 8000)
	if got := p.Occupied(); got != 0 {
		t.Errorf("emptied to %d bays, want 0", got)
	}
	if len(p.Cars) != 0 {
		t.Errorf("%d cars still driving after everyone left", len(p.Cars))
	}
	for i, s := range p.State {
		if s != Free {
			t.Fatalf("bay %d left in state %d", i, s)
		}
	}
}

func TestFrontAndDeepFillIndependently(t *testing.T) {
	p := New(4, 4, 0.5)
	for range 6000 {
		p.Sync(1, 0)
		p.Step(0.05)
	}

	front, deep := 0, 0
	for bay, s := range p.State {
		if s != Taken {
			continue
		}
		if bay < p.Front {
			front++
		} else {
			deep++
		}
	}
	if front != p.Front {
		t.Errorf("front bays hold %d of %d", front, p.Front)
	}
	if deep != 0 {
		t.Errorf("deep bays hold %d cars, want none", deep)
	}
}

func TestWindingTheClockSettlesWithoutADrivingQueue(t *testing.T) {
	p := New(3, 6, 0.5)
	p.Sync(1, 1)

	if len(p.Cars) > 2 {
		t.Errorf("a jump from empty to full put %d cars on the road at once", len(p.Cars))
	}
	if p.Occupied() < len(p.S.Bays)-settleBurst*2-2 {
		t.Errorf("only %d of %d bays settled directly", p.Occupied(), len(p.S.Bays))
	}
}

func TestBayYawFacesIntoTheBay(t *testing.T) {
	s := NewStructure(1, 2)
	for _, b := range s.Bays {
		want := math.Pi / 2 * float64(b.Side)
		if math.Abs(b.Yaw()-want) > 1e-9 {
			t.Errorf("bay on side %d faces %.4f, want %.4f", b.Side, b.Yaw(), want)
		}
	}
}

func TestPaintScattersAcrossAPalette(t *testing.T) {
	// The scatter has to be a permutation of any palette whose size is coprime
	// with the stride, or whole colours would never appear in the structure.
	for _, palette := range []int{8, 9, 10, 12} {
		seen := make(map[int]bool, palette)
		for bay := range palette {
			seen[Paint(bay)%palette] = true
		}
		if len(seen) != palette {
			t.Errorf("a palette of %d only ever shows %d colours", palette, len(seen))
		}
	}
	if Paint(3)%10 == Paint(4)%10 {
		t.Error("neighbouring bays should not share a colour")
	}
}
