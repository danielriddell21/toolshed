package carpark

import "math"

type BayState uint8

const (
	Free BayState = iota
	Reserved
	Taken
	Leaving
)

const (
	maxDriving    = 14
	settleBurst   = 6
	corridorHalf  = 1.4
	clearlyAhead  = CarLength * 0.5
	levelClear    = 1.6
	entranceClear = 9.0
	exitClear     = 8.0
	sameWayCosine = 0.3
)

type Park struct {
	S     Structure
	State []BayState
	Cars  []*Car
	Front int
}

func New(levels, perSide int, frontFraction float64) *Park {
	s := NewStructure(levels, perSide)
	front := int(math.Round(float64(len(s.Bays)) * frontFraction))
	return &Park{
		S:     s,
		State: make([]BayState, len(s.Bays)),
		Front: min(max(front, 1), len(s.Bays)-1),
	}
}

func (p *Park) DeepBays() int { return len(p.S.Bays) - p.Front }

func Paint(bay int) int { return bay * 7 }

// Sync nudges the traffic toward the charge the cell actually holds. The battery
// model is the authority; cars are dispatched to make the structure agree with
// it, so a busy ramp is the diffusion between the two wells made visible.
func (p *Park) Sync(frontFill, deepFill float64) {
	p.settle(0, p.Front, frontFill)
	p.settle(p.Front, len(p.S.Bays), deepFill)
}

func (p *Park) settle(lo, hi int, fill float64) {
	want := int(math.Round(min(max(fill, 0), 1) * float64(hi-lo)))
	if have := p.claimed(lo, hi); have < want {
		p.admit(lo, hi, want-have)
	} else if have > want {
		p.release(lo, hi, have-want)
	}
}

// admit and release dispatch at most one car each per call, so traffic leaves
// the entrance in a stream rather than a stack. Anything beyond a short burst is
// settled directly: winding the clock forward outruns what anyone can drive, and
// a queue that can never clear would misrepresent the cell.
func (p *Park) admit(lo, hi, deficit int) {
	for ; deficit > settleBurst; deficit-- {
		bay, ok := p.firstFree(lo, hi)
		if !ok {
			return
		}
		p.State[bay] = Taken
	}
	if deficit == 0 || len(p.Cars) >= maxDriving || p.entranceBusy() {
		return
	}
	if bay, ok := p.firstFree(lo, hi); ok {
		p.State[bay] = Reserved
		p.Cars = append(p.Cars, newCar(p.S.PathTo(p.S.Bays[bay]), Paint(bay), bay, false))
	}
}

func (p *Park) release(lo, hi, excess int) {
	for ; excess > settleBurst; excess-- {
		bay, ok := p.lastTaken(lo, hi)
		if !ok {
			return
		}
		p.State[bay] = Free
	}
	if excess == 0 || len(p.Cars) >= maxDriving {
		return
	}
	bay, ok := p.lastTaken(lo, hi)
	if !ok || p.levelBusy(p.S.Bays[bay].Level) {
		return
	}
	p.State[bay] = Leaving
	p.Cars = append(p.Cars, newCar(p.S.PathFrom(p.S.Bays[bay]), Paint(bay), bay, true))
}

// entranceBusy keeps arrivals to one at a time, because they all come through
// the same gate. Departures are checked per level instead: cars reversing out on
// different decks never cross, so making them queue would throttle the whole
// structure to one car at a time for no reason.
func (p *Park) entranceBusy() bool {
	for _, c := range p.Cars {
		if !c.Leaving && c.arc < entranceClear {
			return true
		}
	}
	return false
}

func (p *Park) levelBusy(level int) bool {
	for _, c := range p.Cars {
		if c.Leaving && c.arc < exitClear && p.S.Bays[c.Bay].Level == level {
			return true
		}
	}
	return false
}

func (p *Park) claimed(lo, hi int) int {
	n := 0
	for _, s := range p.State[lo:hi] {
		if s == Reserved || s == Taken {
			n++
		}
	}
	return n
}

func (p *Park) firstFree(lo, hi int) (int, bool) {
	for bay := lo; bay < hi; bay++ {
		if p.State[bay] == Free {
			return bay, true
		}
	}
	return 0, false
}

func (p *Park) lastTaken(lo, hi int) (int, bool) {
	for bay := hi - 1; bay >= lo; bay-- {
		if p.State[bay] == Taken {
			return bay, true
		}
	}
	return 0, false
}

func (p *Park) Step(dt float64) {
	for _, c := range p.Cars {
		gap, closing := p.ahead(c)
		c.step(dt, gap, closing)
	}

	kept := p.Cars[:0]
	for _, c := range p.Cars {
		if !c.done {
			kept = append(kept, c)
			continue
		}
		if c.Leaving {
			p.State[c.Bay] = Free
			continue
		}
		p.State[c.Bay] = Taken
	}
	p.Cars = kept
}

// ahead finds whatever this car has to give way to: the nearest car in front of
// it, in its lane, on its level. Cars tucked into bays fall outside the corridor
// and are ignored, which is why the queue only forms on the ramp and the aisle.
func (p *Park) ahead(c *Car) (gap, closing float64) {
	sin, cos := math.Sincos(c.Yaw)
	gap = math.Inf(1)

	for _, o := range p.Cars {
		if o == c {
			continue
		}
		d := o.Pos.Sub(c.Pos)
		if math.Abs(d.Z) > levelClear {
			continue
		}
		// Only give way to traffic going the same way. Oncoming cars are in the
		// other lane and will pass; treating them as obstacles would stop both.
		if math.Cos(o.Yaw-c.Yaw) < sameWayCosine {
			continue
		}
		// The corridor is one lane wide and the leader has to be properly in
		// front. Two cars drawn abreast at a merge would otherwise each read the
		// other as the car ahead, and both would stop for good.
		along := d.X*cos + d.Y*sin
		lateral := -d.X*sin + d.Y*cos
		if along < clearlyAhead || math.Abs(lateral) > corridorHalf {
			continue
		}
		if space := along - CarLength; space < gap {
			gap, closing = space, c.Speed-o.Speed
		}
	}
	return gap, closing
}

func (p *Park) Occupied() int {
	n := 0
	for _, s := range p.State {
		if s == Taken {
			n++
		}
	}
	return n
}
