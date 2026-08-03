package battery

import "math"

const sweepMaxHours = 72.0

type Run struct {
	CRate       float64
	CurrentA    float64
	Hours       float64
	DeliveredAh float64
	EnergyWh    float64
	MeanV       float64
	EndTempC    float64
	Throttled   bool
}

func RunToEmpty(chem Chemistry, cRate, ambientK, dtSeconds float64) Run {
	s := NewSim(chem, 1, ambientK, 1, cRate)
	s.SetMode(Draining)

	var energyWh float64
	var throttled bool
	for s.Phase != PhaseEmpty && s.ElapsedS < sweepMaxHours*3600 {
		s.Advance(dtSeconds)
		energyWh += s.CurrentA * s.Cell.Terminal(s.CurrentA) * dtSeconds / 3600
		throttled = throttled || s.Phase == PhaseHeatLimited
	}

	delivered := s.Cell.ThroughputAh
	meanV := 0.0
	if delivered > 0 {
		meanV = energyWh / delivered
	}
	return Run{
		CRate:       cRate,
		CurrentA:    cRate * chem.CapacityAh,
		Hours:       s.ElapsedS / 3600,
		DeliveredAh: delivered,
		EnergyWh:    energyWh,
		MeanV:       meanV,
		EndTempC:    s.Cell.TempC(),
		Throttled:   throttled,
	}
}

func PeukertExponent(runs []Run) float64 {
	// Peukert's law says runtime falls off as t = a*I^-n, so a least-squares fit
	// of ln(t) against ln(I) has slope -n. n = 1 would be an ideal cell that
	// delivers its rated capacity no matter how hard it is drained. Runs the
	// pack throttled for heat never held their nominal current, so they say
	// nothing about the rate-capacity effect and are left out of the fit.
	var sx, sy, sxx, sxy, n float64
	for _, r := range runs {
		if r.CurrentA <= 0 || r.Hours <= 0 || r.Throttled {
			continue
		}
		x, y := math.Log(r.CurrentA), math.Log(r.Hours)
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
		n++
	}
	if n < 2 {
		return math.NaN()
	}
	return -(n*sxy - sx*sy) / (n*sxx - sx*sx)
}
