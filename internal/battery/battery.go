package battery

import "math"

const (
	gasConstant   = 8.314462618
	zeroCelsiusK  = 273.15
	minSurfaceSoC = 1e-3
)

type Chemistry struct {
	Name           string
	Blurb          string
	CapacityAh     float64
	FrontFraction  float64
	RampPerHour    float64
	VFull          float64
	VExp           float64
	VNom           float64
	VCut           float64
	ExpZoneFrac    float64
	NomZoneFrac    float64
	FitCRate       float64
	OhmRef         float64
	PolarisationS  float64
	RampEa         float64
	OhmEa          float64
	RefK           float64
	EntropyVPerK   float64
	MassKg         float64
	HeatCapJPerKgK float64
	CoolingWPerK   float64
	CoulombicEff   float64
}

type shepherd struct{ e0, k, a, b float64 }

type Cell struct {
	Chem         Chemistry
	curve        shepherd
	FrontAh      float64
	DeepAh       float64
	FilteredA    float64
	TempK        float64
	AmbientK     float64
	ThroughputAh float64
}

func NewCell(chem Chemistry, soc, ambientK float64) *Cell {
	c := &Cell{
		Chem:     chem,
		curve:    fit(chem),
		TempK:    ambientK,
		AmbientK: ambientK,
	}
	c.SetSoC(soc)
	return c
}

func fit(chem Chemistry) shepherd {
	// Solve the three-parameter Shepherd curve exactly through the datasheet
	// anchors (full, end-of-exponential, end-of-nominal) rather than using the
	// usual approximate extraction formulas, so those points are reproduced to
	// machine precision. B is fixed by requiring the exponential zone to decay
	// to 5% (e^-3) by QExp, which leaves E0, K and A as a linear 3x3 system.
	q := chem.CapacityAh
	qExp := q * chem.ExpZoneFrac
	qNom := q * chem.NomZoneFrac
	b := 3 / qExp
	i0 := q * chem.FitCRate

	alpha := func(it float64) float64 { return q / (q - it) * (it + i0) }
	a1, a2, a3 := alpha(0), alpha(qExp), alpha(qNom)
	b1, b2, b3 := 1.0, math.Exp(-3), math.Exp(-b*qNom)

	// Subtracting the full-charge anchor from the other two removes E0 and
	// leaves a 2x2 system in (K, A); solve it by Cramer's rule.
	p, r := a2-a1, chem.VExp-chem.VFull
	s, u := a3-a1, chem.VNom-chem.VFull
	dq, dt := b2-b1, b3-b1
	det := -p*dt + dq*s
	k := (r*dt - dq*u) / det
	a := (r*s - p*u) / det

	return shepherd{e0: chem.VFull + chem.OhmRef*i0 + k*a1 - a*b1, k: k, a: a, b: b}
}

func (c *Cell) SetSoC(soc float64) {
	q := clamp(soc, 0, 1) * c.Chem.CapacityAh
	c.FrontAh = q * c.Chem.FrontFraction
	c.DeepAh = q * (1 - c.Chem.FrontFraction)
}

func (c *Cell) ChargeAh() float64 { return c.FrontAh + c.DeepAh }

func (c *Cell) SoC() float64 { return c.ChargeAh() / c.Chem.CapacityAh }

func (c *Cell) FrontCapacityAh() float64 { return c.Chem.FrontFraction * c.Chem.CapacityAh }

func (c *Cell) DeepCapacityAh() float64 { return (1 - c.Chem.FrontFraction) * c.Chem.CapacityAh }

func (c *Cell) SurfaceSoC() float64 {
	// How full the bays nearest the ramp are. Terminal voltage follows this, not
	// the bulk state of charge: drain hard and the front bays empty faster than
	// the deep ones can refill them, so voltage sags under load and recovers
	// once the load is removed.
	return c.FrontAh / c.FrontCapacityAh()
}

func (c *Cell) TempC() float64 { return c.TempK - zeroCelsiusK }

func (c *Cell) Resistance() float64 {
	// Arrhenius: resistance climbs as the cell cools, which is half of why a
	// cold battery sags so badly under load.
	return c.Chem.OhmRef * math.Exp(c.Chem.OhmEa/gasConstant*(1/c.TempK-1/c.Chem.RefK))
}

func (c *Cell) RampRate() float64 {
	// The other half: diffusion between the wells is thermally activated too, so
	// a cold cell also refills its front bays more slowly.
	return c.Chem.RampPerHour * math.Exp(-c.Chem.RampEa/gasConstant*(1/c.TempK-1/c.Chem.RefK))
}

func (c *Cell) RampFlowA() float64 {
	// Charge moving between the wells, in amps: positive when it is draining
	// down to the ramp, negative when a charger is pushing it back into the deep
	// bays. This is the diffusion term of the kinetic model, so it is exactly
	// the rate at which the front bays refill when nothing else is happening.
	return c.RampRate() * (c.Chem.FrontFraction*c.ChargeAh() - c.FrontAh)
}

func (c *Cell) Terminal(currentA float64) float64 {
	// Shepherd/Tremblay-Dessaint: an open-circuit term that falls as charge is
	// drawn, an exponential term for the top-of-charge knee, a polarisation term
	// driven by the lagged current, and the ohmic drop. Charging uses the
	// model's alternate polarisation denominator, which is what produces the
	// steep rise at the end of a charge.
	q := c.Chem.CapacityAh
	it := q * (1 - clamp(c.SurfaceSoC(), minSurfaceSoC, 1))

	v := c.curve.e0 - c.curve.k*(q/(q-it))*it + c.curve.a*math.Exp(-c.curve.b*it)
	if c.FilteredA >= 0 {
		v -= c.curve.k * (q / (q - it)) * c.FilteredA
	} else {
		v -= c.curve.k * (q / (it + 0.1*q)) * c.FilteredA
	}
	// The Shepherd term diverges as the front bays hit empty; a real cell just
	// collapses to zero, and a discharge always terminates at the cut-off long
	// before this matters.
	return max(0, v-c.Resistance()*currentA)
}

func (c *Cell) OpenCircuit() float64 {
	rested := *c
	rested.FilteredA = 0
	return rested.Terminal(0)
}

func (c *Cell) Step(dtSeconds, currentA float64) {
	accepted := currentA
	if accepted < 0 {
		accepted *= c.Chem.CoulombicEff
	}
	c.kinetics(dtSeconds/3600, accepted)
	c.FilteredA += (currentA - c.FilteredA) * (1 - math.Exp(-dtSeconds/c.Chem.PolarisationS))
	c.thermal(dtSeconds, currentA)
	c.ThroughputAh += math.Abs(currentA) * dtSeconds / 3600
}

func (c *Cell) kinetics(dtHours, currentA float64) {
	// The two-well kinetic battery model (Manwell & McGowan, 1993), advanced by
	// its exact solution over a step of constant current so the step size never
	// distorts the diffusion:
	//
	//	dy1/dt = -i + k(f*q - y1)    y1 = front bays, y2 = deep bays, q = y1 + y2
	//	dy2/dt =      -k(f*q - y1)
	k := c.RampRate()
	f := c.Chem.FrontFraction
	y1, y2 := c.FrontAh, c.DeepAh
	q := y1 + y2
	e := math.Exp(-k * dtHours)
	ramp := currentA * (k*dtHours - 1 + e) / k

	front := y1*e + (q*k*f-currentA)*(1-e)/k - f*ramp
	deep := y2*e + q*(1-f)*(1-e) - (1-f)*ramp

	c.FrontAh = clamp(front, 0, c.FrontCapacityAh())
	c.DeepAh = clamp(deep, 0, c.DeepCapacityAh())
}

func (c *Cell) thermal(dtSeconds, currentA float64) {
	// Irreversible joule heat plus the reversible entropic term, which warms the
	// cell on discharge and cools it on charge for a negative dU/dT, all shed to
	// ambient by Newton cooling.
	joule := currentA * currentA * c.Resistance()
	entropic := -currentA * c.TempK * c.Chem.EntropyVPerK
	cooling := c.Chem.CoolingWPerK * (c.TempK - c.AmbientK)
	c.TempK += (joule + entropic - cooling) * dtSeconds / (c.Chem.MassKg * c.Chem.HeatCapJPerKgK)
}

func clamp(v, lo, hi float64) float64 { return max(lo, min(hi, v)) }
