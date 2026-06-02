package palette

import (
	"slices"
	"testing"

	"github.com/danielriddell21/toolshed/internal/render"
)

func TestAtEndpoints(t *testing.T) {
	g := New(Stop{0, 0, 0, 0}, Stop{1, 255, 255, 255})
	if g.At(0) != (render.RGB{R: 0, G: 0, B: 0}) {
		t.Errorf("At(0) = %v", g.At(0))
	}
	if g.At(1) != (render.RGB{R: 255, G: 255, B: 255}) {
		t.Errorf("At(1) = %v", g.At(1))
	}
}

func TestAtMidLerp(t *testing.T) {
	g := New(Stop{0, 0, 0, 0}, Stop{1, 100, 200, 50})
	mid := g.At(0.5)
	if mid != (render.RGB{R: 50, G: 100, B: 25}) {
		t.Errorf("At(0.5) = %v", mid)
	}
}

func TestAtClamps(t *testing.T) {
	g := New(Stop{0, 10, 10, 10}, Stop{1, 250, 250, 250})
	if g.At(-5) != g.At(0) {
		t.Error("negative t should clamp to start")
	}
	if g.At(99) != g.At(1) {
		t.Error("t>1 should clamp to end")
	}
}

func TestSortsUnorderedStops(t *testing.T) {
	g := New(Stop{1, 255, 255, 255}, Stop{0, 0, 0, 0}, Stop{0.5, 128, 128, 128})
	// Should not panic and endpoints should be correct after sorting.
	if g.At(0) != (render.RGB{R: 0, G: 0, B: 0}) {
		t.Errorf("At(0) after sort = %v", g.At(0))
	}
}

func TestNamedRoundTrip(t *testing.T) {
	for _, name := range Names() {
		if _, ok := Named(name); !ok {
			t.Errorf("Names() returned %q but Named() missed it", name)
		}
	}
	if _, ok := Named("does-not-exist"); ok {
		t.Error("unknown gradient should not resolve")
	}
	if !slices.Contains(Names(), "coral") {
		t.Error("expected coral in Names()")
	}
}

func TestTooFewStopsPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected panic with one stop")
		}
	}()
	New(Stop{0, 0, 0, 0})
}
