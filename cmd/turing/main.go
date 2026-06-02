// Command turing runs a Gray-Scott reaction-diffusion simulation and writes the
// resulting Turing pattern to a PNG, optionally also an animated GIF of the
// evolution. It is a thin CLI wrapper around internal/grayscott.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"math/rand"
	"os"
	"strings"

	"github.com/danielriddell21/toolshed/internal/grayscott"
	"github.com/danielriddell21/toolshed/internal/palette"
	"github.com/spf13/cobra"
)

func main() {
	var (
		preset  string
		feed    float64
		kill    float64
		steps   int
		size    int
		out     string
		gifPath string
		palName string
		seed    int64
	)

	root := &cobra.Command{
		Use:   "turing",
		Short: "Generate Turing patterns via Gray-Scott reaction-diffusion",
		Long: "turing runs a Gray-Scott reaction-diffusion simulation on a toroidal " +
			"grid and writes the resulting pattern to a PNG, optionally an animated GIF.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, preset, feed, kill, steps, size, out, gifPath, palName, seed)
		},
	}

	root.Flags().StringVar(&preset, "preset", "coral", "parameter preset ("+strings.Join(grayscott.PresetNames(), ", ")+")")
	root.Flags().Float64Var(&feed, "feed", math.NaN(), "feed rate (overrides preset)")
	root.Flags().Float64Var(&kill, "kill", math.NaN(), "kill rate (overrides preset)")
	root.Flags().IntVar(&steps, "steps", 5000, "number of simulation steps")
	root.Flags().IntVar(&size, "size", 256, "square grid size (W=H)")
	root.Flags().StringVar(&out, "out", "turing.png", "output PNG path")
	root.Flags().StringVar(&gifPath, "gif", "", "also write an animated GIF to this path")
	root.Flags().StringVar(&palName, "palette", "coral", "color gradient ("+strings.Join(palette.Names(), ", ")+")")
	root.Flags().Int64Var(&seed, "seed", 1, "random seed")
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "turing:", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, preset string, feed, kill float64, steps, size int, out, gifPath, palName string, seed int64) error {
	p, ok := grayscott.Preset(preset)
	if !ok {
		return fmt.Errorf("unknown preset %q; choose one of: %s", preset, strings.Join(grayscott.PresetNames(), ", "))
	}
	if cmd.Flags().Changed("feed") {
		p.Feed = feed
	}
	if cmd.Flags().Changed("kill") {
		p.Kill = kill
	}

	grad, ok := palette.Named(palName)
	if !ok {
		return fmt.Errorf("unknown palette %q; choose one of: %s", palName, strings.Join(palette.Names(), ", "))
	}

	size = max(size, 1)
	steps = max(steps, 0)

	sim := grayscott.New(size, size, p, rand.New(rand.NewSource(seed)))

	if gifPath == "" {
		sim.StepN(steps)
		if err := writePNG(out, sim.Image(grad)); err != nil {
			return err
		}
		fmt.Printf("turing: preset=%s feed=%.4f kill=%.4f size=%d steps=%d -> %s\n",
			preset, p.Feed, p.Kill, size, steps, out)
		return nil
	}

	// Build a GIF by sampling the evolution at a regular interval that yields
	// roughly 60-120 frames.
	const targetFrames = 90
	every := max(steps/targetFrames, 1)
	gifPalette := gradientPalette(grad)

	g := &gif.GIF{}
	for done := 0; done < steps; {
		n := min(every, steps-done)
		sim.StepN(n)
		done += n
		g.Image = append(g.Image, toPaletted(sim.Image(grad), gifPalette))
		g.Delay = append(g.Delay, 5)
	}
	if len(g.Image) == 0 {
		// steps == 0: still emit a single frame of the seed.
		g.Image = append(g.Image, toPaletted(sim.Image(grad), gifPalette))
		g.Delay = append(g.Delay, 5)
	}

	if err := writePNG(out, sim.Image(grad)); err != nil {
		return err
	}
	if err := writeGIF(gifPath, g); err != nil {
		return err
	}
	fmt.Printf("turing: preset=%s feed=%.4f kill=%.4f size=%d steps=%d -> %s, %s (%d frames)\n",
		preset, p.Feed, p.Kill, size, steps, out, gifPath, len(g.Image))
	return nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}

func writeGIF(path string, g *gif.GIF) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if err := gif.EncodeAll(f, g); err != nil {
		return err
	}
	return f.Close()
}

// gradientPalette samples a 256-entry color palette from the gradient.
func gradientPalette(g palette.Gradient) color.Palette {
	pal := make(color.Palette, 256)
	for i := range pal {
		c := g.At(float64(i) / 255)
		pal[i] = color.RGBA{R: c.R, G: c.G, B: c.B, A: 255}
	}
	return pal
}

// toPaletted converts an RGBA image to a paletted image using pal.
func toPaletted(src *image.RGBA, pal color.Palette) *image.Paletted {
	dst := image.NewPaletted(src.Bounds(), pal)
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, src.At(x, y))
		}
	}
	return dst
}
