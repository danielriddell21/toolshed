// Command markov trains an n-gram Markov chain on text and generates new text.
package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/danielriddell21/toolshed/internal/markov"
	"github.com/spf13/cobra"
)

func main() {
	var (
		in     string
		order  int
		words  int
		seed   int64
		stream bool
		temp   float64
	)

	root := &cobra.Command{
		Use:   "markov",
		Short: "Train an n-gram Markov chain on text and generate more",
		Long: "markov reads one or more text files, trains an n-gram Markov chain, " +
			"and generates new text in the same style.",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := filepath.Glob(in)
			if err != nil {
				return fmt.Errorf("bad pattern %q: %w", in, err)
			}
			if len(files) == 0 {
				return fmt.Errorf("no files match %q", in)
			}

			c := markov.NewChain(order)
			for _, f := range files {
				data, err := os.ReadFile(f)
				if err != nil {
					return fmt.Errorf("reading %s: %w", f, err)
				}
				c.TrainText(string(data))
			}

			if seed == 0 {
				seed = time.Now().UnixNano()
			}
			rng := rand.New(rand.NewSource(seed))
			tokens := c.Generate(rng, words, temp)

			out := cmd.OutOrStdout()
			if !stream {
				fmt.Fprintln(out, markov.Render(tokens))
				return nil
			}

			// Stream tokens one at a time, spacing them as Render would.
			prevOpen := false
			for i, tok := range tokens {
				if i > 0 && !isCloser(tok) && !prevOpen {
					fmt.Fprint(out, " ")
				}
				fmt.Fprint(out, tok)
				prevOpen = isOpener(tok)
				time.Sleep(60 * time.Millisecond)
			}
			fmt.Fprintln(out)
			return nil
		},
	}

	root.Flags().StringVar(&in, "in", "", "input file or glob (required)")
	root.Flags().IntVar(&order, "order", 2, "n-gram order")
	root.Flags().IntVar(&words, "words", 200, "number of tokens to generate")
	root.Flags().Int64Var(&seed, "seed", 0, "random seed (0 = time-based)")
	root.Flags().BoolVar(&stream, "stream", false, "print token-by-token with a small delay")
	root.Flags().Float64Var(&temp, "temp", 1.0, "sampling temperature (<1 favors common, >1 flattens)")
	_ = root.MarkFlagRequired("in")
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "markov:", err)
		os.Exit(1)
	}
}

// isCloser / isOpener mirror the spacing rules in the markov package so the
// stream output matches Render. Kept local to main since they are tiny.
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
