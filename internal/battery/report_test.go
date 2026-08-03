package battery

import (
	"math"
	"testing"
)

func TestPeukertExponentRecoversAKnownSlope(t *testing.T) {
	const want = 1.35
	var runs []Run
	for _, current := range []float64{0.5, 1, 2, 4, 8} {
		runs = append(runs, Run{CurrentA: current, Hours: 12 * math.Pow(current, -want)})
	}
	if got := PeukertExponent(runs); math.Abs(got-want) > 1e-9 {
		t.Errorf("PeukertExponent = %.9f, want %.9f", got, want)
	}
}

func TestPeukertExponentNeedsTwoUsableRuns(t *testing.T) {
	runs := []Run{{CurrentA: 1, Hours: 2}, {CurrentA: 0, Hours: 0}}
	if got := PeukertExponent(runs); !math.IsNaN(got) {
		t.Errorf("PeukertExponent = %.4f with one usable run, want NaN", got)
	}
}

func TestHarderDrainsDeliverLess(t *testing.T) {
	for _, chem := range allChemistries(t) {
		gentle := RunToEmpty(chem, 0.2, roomK, 2)
		hard := RunToEmpty(chem, 2, roomK, 2)

		if hard.DeliveredAh >= gentle.DeliveredAh {
			t.Errorf("%s: 2C delivered %.4f Ah, not less than 0.2C's %.4f Ah",
				chem.Name, hard.DeliveredAh, gentle.DeliveredAh)
		}
		if hard.MeanV >= gentle.MeanV {
			t.Errorf("%s: 2C held %.4f V on average, not below 0.2C's %.4f V",
				chem.Name, hard.MeanV, gentle.MeanV)
		}
		if hard.EndTempC <= gentle.EndTempC {
			t.Errorf("%s: 2C ended at %.2f C, not above 0.2C's %.2f C",
				chem.Name, hard.EndTempC, gentle.EndTempC)
		}
		if gentle.DeliveredAh > chem.CapacityAh {
			t.Errorf("%s: delivered %.4f Ah from a %.4f Ah cell",
				chem.Name, gentle.DeliveredAh, chem.CapacityAh)
		}
	}
}

func TestPeukertExponentsAreAboveOneAndOrderedByChemistry(t *testing.T) {
	exponent := func(name string) float64 {
		chem, _ := Named(name)
		var runs []Run
		for _, rate := range []float64{0.2, 0.5, 1, 2} {
			runs = append(runs, RunToEmpty(chem, rate, roomK, 2))
		}
		n := PeukertExponent(runs)
		if n < 1 {
			t.Fatalf("%s: Peukert exponent %.4f is below 1, which no real cell manages", name, n)
		}
		return n
	}

	lead, liion, lfp := exponent("lead"), exponent("liion"), exponent("lfp")
	if lead <= liion || lead <= lfp {
		t.Errorf("lead-acid (%.4f) should show a stronger rate-capacity effect than liion (%.4f) and lfp (%.4f)",
			lead, liion, lfp)
	}
	if lead > 1.6 {
		t.Errorf("lead-acid Peukert exponent %.4f is outside the plausible 1.1-1.6 range", lead)
	}
}
