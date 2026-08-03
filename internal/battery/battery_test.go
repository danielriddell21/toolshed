package battery

import (
	"math"
	"testing"
)

const roomK = 298.15

func allChemistries(t *testing.T) []Chemistry {
	t.Helper()
	names := Names()
	if len(names) == 0 {
		t.Fatal("no chemistries registered")
	}
	out := make([]Chemistry, 0, len(names))
	for _, n := range names {
		ch, ok := Named(n)
		if !ok {
			t.Fatalf("Named(%q) missing", n)
		}
		out = append(out, ch)
	}
	return out
}

// atSoC parks the cell at a state of charge with both wells in equilibrium and
// the given steady current already flowing, which is the condition the datasheet
// discharge curve is measured under.
func atSoC(chem Chemistry, soc, current float64) *Cell {
	c := NewCell(chem, soc, roomK)
	c.FilteredA = current
	return c
}

func TestFitReproducesDatasheetAnchors(t *testing.T) {
	for _, chem := range allChemistries(t) {
		i0 := chem.CapacityAh * chem.FitCRate
		anchors := []struct {
			name string
			soc  float64
			want float64
		}{
			{"full", 1, chem.VFull},
			{"end of exponential zone", 1 - chem.ExpZoneFrac, chem.VExp},
			{"end of nominal zone", 1 - chem.NomZoneFrac, chem.VNom},
		}
		for _, a := range anchors {
			got := atSoC(chem, a.soc, i0).Terminal(i0)
			if math.Abs(got-a.want) > 1e-9 {
				t.Errorf("%s: %s voltage = %.9f, want %.9f", chem.Name, a.name, got, a.want)
			}
		}
	}
}

func TestOpenCircuitRisesWithCharge(t *testing.T) {
	for _, chem := range allChemistries(t) {
		prev := math.Inf(-1)
		for i := range 101 {
			soc := float64(i) / 100
			v := NewCell(chem, soc, roomK).OpenCircuit()
			if v < prev {
				t.Fatalf("%s: open-circuit voltage fell from %.4f to %.4f at soc %.2f", chem.Name, prev, v, soc)
			}
			prev = v
		}
		full := NewCell(chem, 1, roomK).OpenCircuit()
		if math.Abs(full-chem.VFull) > 0.1 {
			t.Errorf("%s: rested full voltage %.3f is far from VFull %.3f", chem.Name, full, chem.VFull)
		}
	}
}

// TestKineticsMatchesNumericIntegration checks the closed-form two-well solution
// against a fine numeric integration of the differential equations it claims to
// solve. This is the load-bearing piece of maths in the package.
func TestKineticsMatchesNumericIntegration(t *testing.T) {
	for _, chem := range allChemistries(t) {
		for _, current := range []float64{0, 0.3, 2.5, -1.7} {
			const hours = 0.25
			cell := NewCell(chem, 0.6, roomK)
			k, f := cell.RampRate(), chem.FrontFraction

			y1, y2 := cell.FrontAh, cell.DeepAh
			const steps = 400000
			h := hours / steps
			for range steps {
				// Classic RK4 on dy1/dt = -i + k(f*q - y1), dy2/dt = -(that term).
				flow := func(a, b float64) float64 { return k * (f*(a+b) - a) }
				k1 := flow(y1, y2)
				k2 := flow(y1+h/2*(-current+k1), y2-h/2*k1)
				k3 := flow(y1+h/2*(-current+k2), y2-h/2*k2)
				k4 := flow(y1+h*(-current+k3), y2-h*k3)
				avg := (k1 + 2*k2 + 2*k3 + k4) / 6
				y1 += h * (-current + avg)
				y2 -= h * avg
			}

			cell.kinetics(hours, current)
			if math.Abs(cell.FrontAh-y1) > 1e-6 || math.Abs(cell.DeepAh-y2) > 1e-6 {
				t.Errorf("%s at %.1fA: closed form (%.9f, %.9f) != integrated (%.9f, %.9f)",
					chem.Name, current, cell.FrontAh, cell.DeepAh, y1, y2)
			}
		}
	}
}

func TestKineticsConservesCharge(t *testing.T) {
	chem, _ := Named("liion")
	for _, current := range []float64{0, 1.3, -0.9} {
		cell := NewCell(chem, 0.5, roomK)
		before := cell.ChargeAh()
		const hours = 0.1
		cell.kinetics(hours, current)
		want := before - current*hours
		if math.Abs(cell.ChargeAh()-want) > 1e-12 {
			t.Errorf("at %.1fA: total charge %.12f, want %.12f", current, cell.ChargeAh(), want)
		}
	}
}

func TestWellsStayWithinTheirCapacity(t *testing.T) {
	for _, chem := range allChemistries(t) {
		cell := NewCell(chem, 0.5, roomK)
		for i := range 2000 {
			current := chem.CapacityAh * 5
			if i%2 == 0 {
				current = -current
			}
			cell.Step(60, current)
			if cell.FrontAh < 0 || cell.FrontAh > cell.FrontCapacityAh()+1e-12 {
				t.Fatalf("%s: front bays out of range: %.6f of %.6f", chem.Name, cell.FrontAh, cell.FrontCapacityAh())
			}
			if cell.DeepAh < 0 || cell.DeepAh > cell.DeepCapacityAh()+1e-12 {
				t.Fatalf("%s: deep bays out of range: %.6f of %.6f", chem.Name, cell.DeepAh, cell.DeepCapacityAh())
			}
		}
	}
}

func TestRestingEqualisesTheWells(t *testing.T) {
	chem, _ := Named("lead")
	cell := NewCell(chem, 0.8, roomK)
	cell.FrontAh -= 1.5
	before := cell.SurfaceSoC()

	for range 6000 {
		cell.Step(10, 0)
	}
	if math.Abs(cell.SurfaceSoC()-cell.SoC()) > 1e-3 {
		t.Errorf("wells did not equalise: surface %.4f vs bulk %.4f", cell.SurfaceSoC(), cell.SoC())
	}
	if cell.SurfaceSoC() <= before {
		t.Errorf("front bays did not refill: %.4f -> %.4f", before, cell.SurfaceSoC())
	}
}

func TestRampFlowMatchesTheRefillRate(t *testing.T) {
	chem, _ := Named("lead")
	cell := NewCell(chem, 0.7, roomK)
	cell.FrontAh -= 1.0

	flow := cell.RampFlowA()
	if flow <= 0 {
		t.Fatalf("depleted front bays should draw charge down the ramp, got %.4f A", flow)
	}
	const hours = 1e-6
	before := cell.FrontAh
	cell.kinetics(hours, 0)
	measured := (cell.FrontAh - before) / hours
	if math.Abs(measured-flow) > 1e-3 {
		t.Errorf("ramp flow %.6f A does not match measured refill %.6f A", flow, measured)
	}
}

func TestVoltageSagsWithLoad(t *testing.T) {
	chem, _ := Named("liion")
	cell := atSoC(chem, 0.7, 0)
	rested := cell.Terminal(0)
	loaded := cell.Terminal(2.6)
	if loaded >= rested {
		t.Fatalf("loaded voltage %.4f should be below rested %.4f", loaded, rested)
	}
	if want := 2.6 * cell.Resistance(); math.Abs((rested-loaded)-want) > 1e-9 {
		t.Errorf("instantaneous sag %.6f V, want the ohmic drop %.6f V", rested-loaded, want)
	}
}

func TestRecoveryAfterAHardDrain(t *testing.T) {
	chem, _ := Named("lead")
	cell := NewCell(chem, 0.6, roomK)
	load := chem.CapacityAh * 2

	for range 900 {
		cell.Step(1, load)
	}
	drained := cell.SurfaceSoC()
	underLoad := cell.Terminal(load)

	for range 3600 {
		cell.Step(1, 0)
	}
	if cell.SurfaceSoC() <= drained {
		t.Errorf("resting should refill the front bays: %.4f -> %.4f", drained, cell.SurfaceSoC())
	}
	if cell.Terminal(0) <= underLoad {
		t.Errorf("resting should recover voltage: %.4f -> %.4f", underLoad, cell.Terminal(0))
	}
	if cell.SurfaceSoC() > cell.SoC()+1e-6 {
		t.Errorf("front bays overfilled past the bulk level: %.4f > %.4f", cell.SurfaceSoC(), cell.SoC())
	}
}

func TestColdCellsResistMore(t *testing.T) {
	chem, _ := Named("liion")
	cold := NewCell(chem, 0.8, zeroCelsiusK-10)
	warm := NewCell(chem, 0.8, zeroCelsiusK+35)

	if cold.Resistance() <= warm.Resistance() {
		t.Errorf("cold resistance %.4f should exceed warm %.4f", cold.Resistance(), warm.Resistance())
	}
	if cold.RampRate() >= warm.RampRate() {
		t.Errorf("cold ramp rate %.4f should be below warm %.4f", cold.RampRate(), warm.RampRate())
	}
	load := chem.CapacityAh
	if cold.Terminal(load) >= warm.Terminal(load) {
		t.Errorf("cold cell should sag further: %.4f vs %.4f", cold.Terminal(load), warm.Terminal(load))
	}
}

func TestCellWarmsUnderLoadAndCoolsAtRest(t *testing.T) {
	chem, _ := Named("liion")
	cell := NewCell(chem, 0.9, roomK)

	for range 600 {
		cell.Step(1, chem.CapacityAh*3)
	}
	hot := cell.TempK
	if hot <= roomK {
		t.Fatalf("cell did not warm under load: %.2f K", hot)
	}

	for range 20000 {
		cell.Step(1, 0)
	}
	if math.Abs(cell.TempK-cell.AmbientK) > 0.05 {
		t.Errorf("cell did not settle back to ambient: %.3f K vs %.3f K", cell.TempK, cell.AmbientK)
	}
}

func TestEntropicHeatingIsSignedByDirection(t *testing.T) {
	chem, _ := Named("liion")
	joule := func(current float64) float64 {
		cell := NewCell(chem, 0.5, roomK)
		cell.Chem.OhmRef = 0
		cell.Step(1, current)
		return cell.TempK - roomK
	}
	if joule(1) <= 0 {
		t.Error("discharge should be exothermic once joule heat is removed")
	}
	if joule(-1) >= 0 {
		t.Error("charge should be endothermic once joule heat is removed")
	}
}

func TestChargeAcceptanceLosesToCoulombicEfficiency(t *testing.T) {
	chem, _ := Named("lead")
	cell := NewCell(chem, 0.2, roomK)
	before := cell.ChargeAh()

	const hours = 0.5
	current := -chem.CapacityAh * 0.2
	cell.Step(hours*3600, current)

	stored := cell.ChargeAh() - before
	pushed := -current * hours
	if stored >= pushed {
		t.Fatalf("stored %.4f Ah should be less than the %.4f Ah pushed in", stored, pushed)
	}
	if want := pushed * chem.CoulombicEff; math.Abs(stored-want) > 1e-9 {
		t.Errorf("stored %.6f Ah, want %.6f Ah", stored, want)
	}
}

func TestSetSoCClampsToTheCarPark(t *testing.T) {
	chem, _ := Named("lfp")
	cell := NewCell(chem, 0.5, roomK)

	cell.SetSoC(4)
	if cell.SoC() != 1 {
		t.Errorf("over-full soc = %.4f, want 1", cell.SoC())
	}
	cell.SetSoC(-2)
	if cell.SoC() != 0 {
		t.Errorf("negative soc = %.4f, want 0", cell.SoC())
	}
	if cell.Terminal(0) < 0 {
		t.Errorf("empty cell reported a negative voltage: %.4f", cell.Terminal(0))
	}
}
