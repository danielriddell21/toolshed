package battery

const (
	coldChargeK      = zeroCelsiusK
	coldTrickleCRate = 0.1
	heatDerateFromK  = zeroCelsiusK + 45
	heatDerateToK    = zeroCelsiusK + 60
	taperCRate       = 0.05
)

type Mode int

const (
	Rest Mode = iota
	Charging
	Draining
)

func (m Mode) String() string {
	switch m {
	case Charging:
		return "charging"
	case Draining:
		return "draining"
	default:
		return "resting"
	}
}

type Phase int

const (
	PhaseIdle Phase = iota
	PhaseBulk
	PhaseTaper
	PhaseLoad
	PhaseFull
	PhaseEmpty
	PhaseColdLimited
	PhaseHeatLimited
)

func (p Phase) String() string {
	switch p {
	case PhaseBulk:
		return "constant current"
	case PhaseTaper:
		return "constant voltage"
	case PhaseLoad:
		return "under load"
	case PhaseFull:
		return "full"
	case PhaseEmpty:
		return "empty"
	case PhaseColdLimited:
		return "cold-limited"
	case PhaseHeatLimited:
		return "heat-limited"
	default:
		return "idle"
	}
}

type Sim struct {
	Cell        *Cell
	Mode        Mode
	Phase       Phase
	CurrentA    float64
	ChargeCRate float64
	LoadCRate   float64
	ElapsedS    float64
}

func NewSim(chem Chemistry, soc, ambientK, chargeCRate, loadCRate float64) *Sim {
	return &Sim{
		Cell:        NewCell(chem, soc, ambientK),
		ChargeCRate: chargeCRate,
		LoadCRate:   loadCRate,
	}
}

func (s *Sim) SetMode(m Mode) {
	s.Mode = m
	s.Phase = PhaseIdle
}

func (s *Sim) Advance(dtSeconds float64) {
	current, phase := s.demand()
	if phase == PhaseFull || phase == PhaseEmpty {
		s.Mode = Rest
	}
	s.CurrentA = current
	s.Phase = phase
	s.Cell.Step(dtSeconds, current)
	s.ElapsedS += dtSeconds
}

func (s *Sim) PowerW() float64 { return s.CurrentA * s.Cell.Terminal(s.CurrentA) }

func (s *Sim) Cycles() float64 { return s.Cell.ThroughputAh / (2 * s.Cell.Chem.CapacityAh) }

func (s *Sim) demand() (float64, Phase) {
	switch s.Mode {
	case Charging:
		return s.chargeCurrent()
	case Draining:
		return s.loadCurrent()
	default:
		// Full and empty are sticky: once a run ends, keep saying why until the
		// mode is changed again.
		if s.Phase == PhaseFull || s.Phase == PhaseEmpty {
			return 0, s.Phase
		}
		return 0, PhaseIdle
	}
}

func (s *Sim) chargeCurrent() (float64, Phase) {
	cell := s.Cell
	capacity := cell.Chem.CapacityAh
	limit := s.ChargeCRate * capacity
	limited := PhaseBulk

	if cell.TempK < coldChargeK {
		// Charging a cold cell plates lithium onto the anode, so real chargers
		// drop to a trickle until it warms up.
		limit = min(limit, coldTrickleCRate*capacity)
		limited = PhaseColdLimited
	}
	if d := heatDerate(cell.TempK); d < 1 {
		limit *= d
		limited = PhaseHeatLimited
	}

	// Constant-voltage branch: the exact current that pins the terminal voltage
	// to the ceiling, given the polarisation already built up in the cell.
	taperA := (cell.Terminal(0) - cell.Chem.VFull) / cell.Resistance()
	if taperA > -limit {
		if -taperA < taperCRate*capacity {
			return 0, PhaseFull
		}
		return min(taperA, 0), PhaseTaper
	}
	return -limit, limited
}

func (s *Sim) loadCurrent() (float64, Phase) {
	cell := s.Cell
	current := s.LoadCRate * cell.Chem.CapacityAh
	phase := PhaseLoad
	if d := heatDerate(cell.TempK); d < 1 {
		current *= d
		phase = PhaseHeatLimited
	}
	if cell.Terminal(current) <= cell.Chem.VCut {
		return 0, PhaseEmpty
	}
	return current, phase
}

func heatDerate(tempK float64) float64 {
	if tempK <= heatDerateFromK {
		return 1
	}
	if tempK >= heatDerateToK {
		return 0
	}
	return (heatDerateToK - tempK) / (heatDerateToK - heatDerateFromK)
}
