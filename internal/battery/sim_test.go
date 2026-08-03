package battery

import (
	"math"
	"testing"
)

// The controller picks a current at the start of each step and holds it, so the
// voltage at the end of the step can sit a sliver past the limit it was aiming
// at, exactly as a real BMS overshoots between samples. A millivolt of slack at
// a one-second sample covers it.
const sampleSlackV = 1e-3

func runUntilSettled(t *testing.T, s *Sim, dt float64, limit int) {
	t.Helper()
	for range limit {
		s.Advance(dt)
		if s.Mode == Rest {
			return
		}
	}
	t.Fatalf("%s never finished %s (soc %.3f, phase %s)", s.Cell.Chem.Name, s.Mode, s.Cell.SoC(), s.Phase)
}

func TestChargeRunsConstantCurrentThenConstantVoltage(t *testing.T) {
	for _, chem := range allChemistries(t) {
		s := NewSim(chem, 0.05, roomK, 1, 1)
		s.SetMode(Charging)

		var sawBulk, sawTaper bool
		var lastTaper float64
		for range 200000 {
			s.Advance(1)
			switch s.Phase {
			case PhaseBulk:
				if sawTaper {
					t.Fatalf("%s: went back to constant current after tapering", chem.Name)
				}
				sawBulk = true
				if want := -chem.CapacityAh; math.Abs(s.CurrentA-want) > 1e-9 {
					t.Fatalf("%s: bulk current %.4f A, want %.4f A", chem.Name, s.CurrentA, want)
				}
			case PhaseTaper:
				if sawTaper && s.CurrentA < lastTaper-1e-9 {
					t.Fatalf("%s: taper current grew from %.6f to %.6f", chem.Name, lastTaper, s.CurrentA)
				}
				sawTaper, lastTaper = true, s.CurrentA
			}
			if v := s.Cell.Terminal(s.CurrentA); v > chem.VFull+sampleSlackV {
				t.Fatalf("%s: terminal voltage %.6f exceeded the %.3f V ceiling", chem.Name, v, chem.VFull)
			}
			if s.Mode == Rest {
				break
			}
		}

		if !sawBulk || !sawTaper {
			t.Fatalf("%s: expected both charge phases, got bulk=%v taper=%v", chem.Name, sawBulk, sawTaper)
		}
		if s.Phase != PhaseFull {
			t.Fatalf("%s: ended in phase %s, want full", chem.Name, s.Phase)
		}
		if s.Cell.SoC() < 0.9 {
			t.Errorf("%s: charge stopped at only %.1f%%", chem.Name, 100*s.Cell.SoC())
		}
	}
}

func TestDischargeStopsAtTheCutoff(t *testing.T) {
	for _, chem := range allChemistries(t) {
		s := NewSim(chem, 1, roomK, 1, 0.5)
		s.SetMode(Draining)

		for range 400000 {
			// The guarantee is that the controller never draws from a cell that
			// has already reached the cut-off. Where it lands by the end of the
			// step depends on how steeply that chemistry falls off the cliff.
			before := s.Cell.Terminal(s.LoadCRate * chem.CapacityAh)
			s.Advance(1)
			if s.Mode == Rest {
				break
			}
			if before < chem.VCut {
				t.Fatalf("%s: kept drawing from a cell already at %.4f V, below the %.3f V cut-off",
					chem.Name, before, chem.VCut)
			}
		}
		if v := s.Cell.Terminal(0); v < chem.VCut-0.05 {
			t.Errorf("%s: overshot the cut-off, resting at %.4f V against a %.3f V limit", chem.Name, v, chem.VCut)
		}
		if s.Phase != PhaseEmpty {
			t.Fatalf("%s: ended in phase %s, want empty", chem.Name, s.Phase)
		}
		if s.CurrentA != 0 {
			t.Errorf("%s: still drawing %.4f A when empty", chem.Name, s.CurrentA)
		}
	}
}

func TestTerminalPhasesAreStickyUntilTheModeChanges(t *testing.T) {
	chem, _ := Named("liion")
	s := NewSim(chem, 0.99, roomK, 1, 1)
	s.SetMode(Charging)
	runUntilSettled(t, s, 1, 200000)

	for range 100 {
		s.Advance(1)
	}
	if s.Phase != PhaseFull {
		t.Errorf("phase drifted to %s after finishing a charge", s.Phase)
	}

	s.SetMode(Rest)
	s.Advance(1)
	if s.Phase != PhaseIdle {
		t.Errorf("phase = %s after an explicit rest, want idle", s.Phase)
	}
}

func TestChargingBelowFreezingIsTrickled(t *testing.T) {
	chem, _ := Named("liion")
	s := NewSim(chem, 0.3, zeroCelsiusK-15, 1, 1)
	s.SetMode(Charging)
	s.Advance(1)

	if s.Phase != PhaseColdLimited {
		t.Fatalf("phase = %s, want cold-limited", s.Phase)
	}
	if limit := coldTrickleCRate * chem.CapacityAh; -s.CurrentA > limit+1e-9 {
		t.Errorf("cold charge drew %.4f A, above the %.4f A trickle limit", -s.CurrentA, limit)
	}
}

func TestHeatDerateTapersThenStops(t *testing.T) {
	chem, _ := Named("liion")
	s := NewSim(chem, 0.5, roomK, 1, 1)
	s.SetMode(Draining)

	s.Cell.TempK = heatDerateFromK + (heatDerateToK-heatDerateFromK)/2
	s.Advance(1)
	if s.Phase != PhaseHeatLimited {
		t.Fatalf("phase = %s at %.1f C, want heat-limited", s.Phase, s.Cell.TempC())
	}
	if want := 0.5 * chem.CapacityAh; math.Abs(s.CurrentA-want) > 1e-6 {
		t.Errorf("half-derated current %.4f A, want %.4f A", s.CurrentA, want)
	}

	s.Cell.TempK = heatDerateToK + 5
	s.Advance(1)
	if s.CurrentA != 0 {
		t.Errorf("current %.4f A above the shutdown temperature, want 0", s.CurrentA)
	}
}

func TestPowerAndCyclesFollowTheCurrentSign(t *testing.T) {
	chem, _ := Named("liion")
	s := NewSim(chem, 0.5, roomK, 1, 1)

	s.SetMode(Draining)
	s.Advance(1)
	if s.PowerW() <= 0 {
		t.Errorf("draining power %.4f W should be positive", s.PowerW())
	}

	s.SetMode(Charging)
	s.Advance(1)
	if s.PowerW() >= 0 {
		t.Errorf("charging power %.4f W should be negative", s.PowerW())
	}

	s.Cell.ThroughputAh = 2 * chem.CapacityAh
	if math.Abs(s.Cycles()-1) > 1e-12 {
		t.Errorf("a full charge plus a full discharge should be one cycle, got %.4f", s.Cycles())
	}
}

func TestRestingDrawsNothing(t *testing.T) {
	chem, _ := Named("lfp")
	s := NewSim(chem, 0.5, roomK, 1, 1)
	before := s.Cell.ChargeAh()

	for range 600 {
		s.Advance(1)
	}
	if s.CurrentA != 0 || s.Phase != PhaseIdle {
		t.Errorf("resting drew %.4f A in phase %s", s.CurrentA, s.Phase)
	}
	if math.Abs(s.Cell.ChargeAh()-before) > 1e-12 {
		t.Errorf("charge changed at rest: %.9f -> %.9f", before, s.Cell.ChargeAh())
	}
}
