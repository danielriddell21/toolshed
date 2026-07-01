package markov

import (
	"math"
	"math/rand"
	"slices"
	"strings"
	"unicode"
)

const sep = "\x00"

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

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func isApostrophe(r rune) bool {
	return r == '\'' || r == '’'
}

func IsSentenceEnd(tok string) bool {
	return tok == "." || tok == "!" || tok == "?"
}

type next struct {
	tok   string
	count int
}

type Chain struct {
	order  int
	trans  map[string][]next
	starts []string
}

func NewChain(order int) *Chain {
	return &Chain{
		order: max(order, 1),
		trans: make(map[string][]next),
	}
}

func stateKey(window []string) string {
	return strings.Join(window, sep)
}

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

func (c *Chain) TrainText(s string) {
	c.Train(Tokenize(s))
}

func (c *Chain) pickStart(rng *rand.Rand) string {
	if len(c.starts) > 0 {
		return c.starts[rng.Intn(len(c.starts))]
	}
	return c.randomState(rng)
}

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

func Render(tokens []string) string {
	var b strings.Builder
	for i, tok := range tokens {
		if i > 0 && NeedsSpace(tokens[i-1], tok) {
			b.WriteByte(' ')
		}
		b.WriteString(tok)
	}
	return b.String()
}

func NeedsSpace(prevTok, tok string) bool {
	return !isCloser(tok) && !isOpener(prevTok)
}

func isCloser(tok string) bool {
	switch tok {
	case ".", ",", "!", "?", ";", ":", ")", "]", "}", "'", "\"", "%":
		return true
	}
	return false
}

func isOpener(tok string) bool {
	switch tok {
	case "(", "[", "{":
		return true
	}
	return false
}
