// Package markov trains an n-gram Markov chain on text and generates new text.
package markov

import (
	"math"
	"math/rand"
	"slices"
	"strings"
	"unicode"
)

// sep joins state tokens into a map key. It is a byte unlikely to appear in
// real text, so distinct token sequences never collide.
const sep = "\x00"

// Tokenize splits text into word and punctuation tokens. Each punctuation rune
// becomes its own standalone token; runs of word characters are kept as-is.
// Whitespace is discarded.
func Tokenize(s string) []string {
	var tokens []string
	var word strings.Builder

	flush := func() {
		if word.Len() > 0 {
			tokens = append(tokens, word.String())
			word.Reset()
		}
	}

	runes := []rune(s)
	for i, r := range runes {
		switch {
		case unicode.IsSpace(r):
			flush()
		case isWordRune(r):
			word.WriteRune(r)
		case isApostrophe(r) && word.Len() > 0 && i+1 < len(runes) && isWordRune(runes[i+1]):
			// Intra-word apostrophe (e.g. "don't"): keep attached.
			word.WriteRune(r)
		default:
			// Punctuation / symbol: standalone token.
			flush()
			tokens = append(tokens, string(r))
		}
	}
	flush()
	return tokens
}

// isWordRune reports whether r belongs to a word. Letters, digits, and a few
// intra-word marks (apostrophe-like, underscore) stay attached to the word.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// isApostrophe reports whether r is a straight or typographic apostrophe.
func isApostrophe(r rune) bool {
	return r == '\'' || r == '’'
}

// IsSentenceEnd reports whether tok is a sentence-ending punctuation token.
func IsSentenceEnd(tok string) bool {
	return tok == "." || tok == "!" || tok == "?"
}

// next is a single observed successor token and how often it followed a state.
type next struct {
	tok   string
	count int
}

// Chain is an in-memory n-gram Markov model.
type Chain struct {
	order  int
	trans  map[string][]next
	starts []string // states that begin a sentence (preferred starts + fallback)
}

// NewChain returns an empty chain of the given order. Order is clamped to >= 1.
func NewChain(order int) *Chain {
	return &Chain{
		order: max(order, 1),
		trans: make(map[string][]next),
	}
}

// stateKey builds the map key for the order-length window ending the slice.
func stateKey(window []string) string {
	return strings.Join(window, sep)
}

// addTransition records that tok followed the given state.
func (c *Chain) addTransition(state string, tok string) {
	succ := c.trans[state]
	for i := range succ {
		if succ[i].tok == tok {
			succ[i].count++
			return
		}
	}
	c.trans[state] = append(succ, next{tok: tok, count: 1})
}

// Train adds the transitions implied by a token stream to the model.
func (c *Chain) Train(tokens []string) {
	if len(tokens) <= c.order {
		return
	}

	for i := 0; i+c.order < len(tokens); i++ {
		window := tokens[i : i+c.order]
		state := stateKey(window)
		c.addTransition(state, tokens[i+c.order])

		// Mark sentence-start states: the very first window, and any window
		// that begins immediately after a sentence-ending token.
		if i == 0 || IsSentenceEnd(tokens[i-1]) {
			c.starts = append(c.starts, state)
		}
	}
}

// TrainText tokenizes s and trains on the resulting stream.
func (c *Chain) TrainText(s string) {
	c.Train(Tokenize(s))
}

// pickStart chooses an initial state, preferring sentence starts. Returns ""
// if the model is empty.
func (c *Chain) pickStart(rng *rand.Rand) string {
	if len(c.starts) > 0 {
		return c.starts[rng.Intn(len(c.starts))]
	}
	return c.randomState(rng)
}

// randomState returns a uniformly random state that has successors, or "" if
// none exist. It iterates a snapshot of keys for deterministic selection.
func (c *Chain) randomState(rng *rand.Rand) string {
	if len(c.trans) == 0 {
		return ""
	}
	keys := make([]string, 0, len(c.trans))
	for k := range c.trans {
		keys = append(keys, k)
	}
	// Sorting keeps selection deterministic regardless of map iteration order.
	slices.Sort(keys)
	return keys[rng.Intn(len(keys))]
}

// chooseNext picks a successor of state using temperature-weighted sampling.
// weight = count^(1/temp). Returns ("", false) when state has no successors.
func chooseNext(rng *rand.Rand, succ []next, temp float64) (string, bool) {
	if len(succ) == 0 {
		return "", false
	}
	if temp <= 0 {
		temp = 1e-6 // avoid divide-by-zero; effectively argmax-ish
	}

	exp := 1.0 / temp
	weights := make([]float64, len(succ))
	var total float64
	for i, s := range succ {
		w := math.Pow(float64(s.count), exp)
		weights[i] = w
		total += w
	}
	if total <= 0 {
		return succ[rng.Intn(len(succ))].tok, true
	}

	r := rng.Float64() * total
	for i, w := range weights {
		r -= w
		if r < 0 {
			return succ[i].tok, true
		}
	}
	return succ[len(succ)-1].tok, true
}

// Generate emits up to n tokens, sampling with the given rng and temperature.
// It begins from a sentence-start state when possible and, on hitting a dead
// end, restarts from a random valid state so it never stalls. Output is fully
// deterministic for a given rng seed, model, n, and temp.
func (c *Chain) Generate(rng *rand.Rand, n int, temp float64) []string {
	if n <= 0 || len(c.trans) == 0 {
		return nil
	}

	state := c.pickStart(rng)
	if state == "" {
		return nil
	}
	window := strings.Split(state, sep)

	out := make([]string, 0, n)
	// Seed output with the start window (trimmed to n).
	for _, t := range window {
		if len(out) >= n {
			return out
		}
		out = append(out, t)
	}

	for len(out) < n {
		state = stateKey(window)
		tok, ok := chooseNext(rng, c.trans[state], temp)
		if !ok {
			// Dead end: fall back to a fresh start state.
			state = c.randomState(rng)
			if state == "" {
				break
			}
			window = strings.Split(state, sep)
			continue
		}
		out = append(out, tok)
		// Slide the window forward by one.
		window = append(window[1:], tok)
	}
	return out
}

// Render joins tokens into a string with sensible spacing: no space before
// closing punctuation, no space after opening punctuation, single spaces
// between words. Capitalization is left untouched.
func Render(tokens []string) string {
	var b strings.Builder
	prevOpen := false // previous token was an opener like "("
	for i, tok := range tokens {
		if i > 0 && !isCloser(tok) && !prevOpen {
			b.WriteByte(' ')
		}
		b.WriteString(tok)
		prevOpen = isOpener(tok)
	}
	return b.String()
}

// isCloser reports whether tok should hug the preceding token (no space before).
func isCloser(tok string) bool {
	switch tok {
	case ".", ",", "!", "?", ";", ":", ")", "]", "}", "'", "\"", "%":
		return true
	}
	return false
}

// isOpener reports whether tok should hug the following token (no space after).
func isOpener(tok string) bool {
	switch tok {
	case "(", "[", "{":
		return true
	}
	return false
}
