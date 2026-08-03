package battery

import "sort"

var chemistries = map[string]Chemistry{
	"liion": {
		Name:           "liion",
		Blurb:          "NMC 18650 cell, with its deep bays close to the ramp, so it barely notices a hard drain",
		CapacityAh:     2.6,
		FrontFraction:  0.85,
		RampPerHour:    9.0,
		VFull:          4.2,
		VExp:           3.95,
		VNom:           3.6,
		VCut:           2.75,
		ExpZoneFrac:    0.05,
		NomZoneFrac:    0.88,
		FitCRate:       0.2,
		OhmRef:         0.045,
		PolarisationS:  30,
		RampEa:         25000,
		OhmEa:          20000,
		RefK:           298.15,
		EntropyVPerK:   -0.0002,
		MassKg:         0.045,
		HeatCapJPerKgK: 1000,
		CoolingWPerK:   0.05,
		CoulombicEff:   0.995,
	},
	"lfp": {
		Name:           "lfp",
		Blurb:          "LiFePO4 26650 cell, a very flat voltage plateau over a wide, shallow car park",
		CapacityAh:     3.2,
		FrontFraction:  0.9,
		RampPerHour:    4.0,
		VFull:          3.65,
		VExp:           3.42,
		VNom:           3.25,
		VCut:           2.5,
		ExpZoneFrac:    0.04,
		NomZoneFrac:    0.9,
		FitCRate:       0.2,
		OhmRef:         0.03,
		PolarisationS:  40,
		RampEa:         28000,
		OhmEa:          22000,
		RefK:           298.15,
		EntropyVPerK:   -0.00005,
		MassKg:         0.085,
		HeatCapJPerKgK: 1100,
		CoolingWPerK:   0.07,
		CoulombicEff:   0.998,
	},
	"lead": {
		Name:           "lead",
		Blurb:          "sealed lead-acid cell, where most of the bays are a long walk from the ramp",
		CapacityAh:     7.2,
		FrontFraction:  0.7,
		RampPerHour:    0.9,
		VFull:          2.4,
		VExp:           2.25,
		VNom:           2.05,
		VCut:           1.75,
		ExpZoneFrac:    0.06,
		NomZoneFrac:    0.85,
		FitCRate:       0.05,
		OhmRef:         0.012,
		PolarisationS:  90,
		RampEa:         32000,
		OhmEa:          26000,
		RefK:           298.15,
		EntropyVPerK:   -0.00015,
		MassKg:         2.5,
		HeatCapJPerKgK: 800,
		CoolingWPerK:   0.8,
		CoulombicEff:   0.9,
	},
}

func Named(name string) (Chemistry, bool) {
	c, ok := chemistries[name]
	return c, ok
}

func Names() []string {
	names := make([]string, 0, len(chemistries))
	for name := range chemistries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
