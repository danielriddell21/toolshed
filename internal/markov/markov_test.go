package markov

import (
	"math/rand"
	"slices"
	"testing"
)

func TestTokenizeKeepsPunctuation(t *testing.T) {
	got := Tokenize("Hi, there.")
	want := []string{"Hi", ",", "there", "."}
	if !slices.Equal(got, want) {
		t.Fatalf("Tokenize = %q, want %q", got, want)
	}
}

func TestTokenizeApostropheStaysInWord(t *testing.T) {
	got := Tokenize("don't (stop)")
	want := []string{"don't", "(", "stop", ")"}
	if !slices.Equal(got, want) {
		t.Fatalf("Tokenize = %q, want %q", got, want)
	}
}

func TestIsSentenceEnd(t *testing.T) {
	for _, tok := range []string{".", "!", "?"} {
		if !IsSentenceEnd(tok) {
			t.Errorf("IsSentenceEnd(%q) = false, want true", tok)
		}
	}
	for _, tok := range []string{",", "word", ";"} {
		if IsSentenceEnd(tok) {
			t.Errorf("IsSentenceEnd(%q) = true, want false", tok)
		}
	}
}

func TestGenerateDeterministic(t *testing.T) {
	const corpus = "the quick brown fox jumps over the lazy dog. " +
		"the dog runs fast and the fox runs faster. quick fox, lazy dog."

	build := func() []string {
		c := NewChain(2)
		c.TrainText(corpus)
		rng := rand.New(rand.NewSource(42))
		return c.Generate(rng, 30, 1.0)
	}

	a := build()
	b := build()
	if !slices.Equal(a, b) {
		t.Fatalf("Generate not deterministic:\n a = %q\n b = %q", a, b)
	}
	if len(a) == 0 {
		t.Fatal("Generate produced no tokens")
	}
}

func TestGenerateDeadEndFallback(t *testing.T) {
	// Tiny corpus where the final window is a dead end, forcing a fallback.
	c := NewChain(2)
	c.TrainText("a b c d e")

	rng := rand.New(rand.NewSource(1))
	got := c.Generate(rng, 25, 1.0)
	if len(got) != 25 {
		t.Fatalf("Generate produced %d tokens, want 25", len(got))
	}
}

func TestGenerateEmptyChain(t *testing.T) {
	c := NewChain(2)
	rng := rand.New(rand.NewSource(1))
	if got := c.Generate(rng, 10, 1.0); got != nil {
		t.Fatalf("Generate on empty chain = %q, want nil", got)
	}
}

func TestRenderSpacing(t *testing.T) {
	got := Render([]string{"Hi", ",", "there", "."})
	want := "Hi, there."
	if got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestRenderOpenerCloser(t *testing.T) {
	got := Render([]string{"see", "(", "this", ")", "now"})
	want := "see (this) now"
	if got != want {
		t.Fatalf("Render = %q, want %q", got, want)
	}
}

func TestTemperatureBiasesCommon(t *testing.T) {
	// State "x" -> "common" 9 times, "rare" once. Low temp should favor
	// "common" overwhelmingly across samples.
	c := NewChain(1)
	toks := []string{}
	for range 9 {
		toks = append(toks, "x", "common")
	}
	toks = append(toks, "x", "rare")
	c.Train(toks)

	rng := rand.New(rand.NewSource(7))
	succ := c.trans["x"]
	common := 0
	for range 200 {
		tok, ok := chooseNext(rng, succ, 0.2)
		if !ok {
			t.Fatal("chooseNext returned !ok")
		}
		if tok == "common" {
			common++
		}
	}
	if common < 190 {
		t.Fatalf("low temp picked common only %d/200 times, expected near-all", common)
	}
}
