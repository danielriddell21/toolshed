package grayscott

import (
	"image"
	"math"
	"math/rand"
	"slices"
	"testing"

	"github.com/danielriddell21/toolshed/internal/palette"
)

func presetOrFatal(t *testing.T, name string) Params {
	t.Helper()
	p, ok := Preset(name)
	if !ok {
		t.Fatalf("Preset(%q) returned ok=false", name)
	}
	return p
}

func TestDeterministic(t *testing.T) {
	p := presetOrFatal(t, "coral")

	a := New(48, 48, p, rand.New(rand.NewSource(7)))
	b := New(48, 48, p, rand.New(rand.NewSource(7)))
	a.StepN(200)
	b.StepN(200)

	va, vb := a.V(), b.V()
	if !slices.Equal(va, vb) {
		t.Fatalf("same seed produced different V fields")
	}

	// A different seed should generally diverge.
	c := New(48, 48, p, rand.New(rand.NewSource(99)))
	c.StepN(200)
	if slices.Equal(va, c.V()) {
		t.Fatalf("different seed produced identical V field")
	}
}

func TestPresetLookup(t *testing.T) {
	for _, name := range PresetNames() {
		p, ok := Preset(name)
		if !ok {
			t.Errorf("Preset(%q) ok=false for listed name", name)
		}
		if p.Du != 0.16 || p.Dv != 0.08 || p.Dt != 1.0 {
			t.Errorf("Preset(%q) has unexpected diffusion/timestep: %+v", name, p)
		}
	}

	if _, ok := Preset("does-not-exist"); ok {
		t.Errorf("Preset of unknown name returned ok=true")
	}
}

func TestPresetNamesSorted(t *testing.T) {
	names := PresetNames()
	if len(names) == 0 {
		t.Fatal("PresetNames is empty")
	}
	if !slices.IsSorted(names) {
		t.Errorf("PresetNames not sorted: %v", names)
	}
}

func TestValuesFinite(t *testing.T) {
	p := presetOrFatal(t, "maze")
	s := New(64, 64, p, rand.New(rand.NewSource(1)))
	s.StepN(500)

	for i, v := range s.V() {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("V[%d] not finite: %v", i, v)
		}
	}
	for i, u := range s.U() {
		if math.IsNaN(u) || math.IsInf(u, 0) {
			t.Fatalf("U[%d] not finite: %v", i, u)
		}
	}
}

func TestImageBounds(t *testing.T) {
	p := presetOrFatal(t, "spots")
	s := New(40, 24, p, rand.New(rand.NewSource(3)))
	s.StepN(50)

	g, ok := palette.Named("coral")
	if !ok {
		t.Fatal("palette coral not found")
	}
	img := s.Image(g)
	want := image.Rect(0, 0, 40, 24)
	if img.Bounds() != want {
		t.Fatalf("Image bounds = %v, want %v", img.Bounds(), want)
	}
}

func TestFieldLengths(t *testing.T) {
	p := presetOrFatal(t, "coral")
	s := New(32, 20, p, rand.New(rand.NewSource(1)))
	if len(s.V()) != 32*20 {
		t.Errorf("len(V) = %d, want %d", len(s.V()), 32*20)
	}
	if len(s.U()) != 32*20 {
		t.Errorf("len(U) = %d, want %d", len(s.U()), 32*20)
	}
}
