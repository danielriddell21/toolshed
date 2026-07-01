package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/danielriddell21/toolshed/internal/cli"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/fractal"
	"github.com/danielriddell21/toolshed/internal/palette"
	"github.com/danielriddell21/toolshed/internal/render"
	"github.com/danielriddell21/toolshed/internal/style"
)

const (
	statusRows = 2
	panFrac    = 0.10
	zoomFactor = 0.7
)

var defaultJuliaC = complex(-0.8, 0.156)

func defaultView(mode fractal.Mode, w int) fractal.View {
	if w < 2 {
		w = 2
	}
	scale := 3.0 / float64(w)
	if mode == fractal.Julia {
		return fractal.View{CenterRe: 0, CenterIm: 0, Scale: scale}
	}
	return fractal.View{CenterRe: -0.5, CenterIm: 0, Scale: scale}
}

type model struct {
	anim.Size

	frame    *render.Frame
	view     fractal.View
	params   fractal.Params
	palName  string
	gradient palette.Gradient
	dirty    bool
}

func newModel(palName string, startJulia bool) model {
	g, _ := palette.Named(palName)
	m := model{
		frame:    render.NewFrame(2, 2),
		palName:  palName,
		gradient: g,
	}
	if startJulia {
		m.params = fractal.Params{Mode: fractal.Julia, JuliaC: defaultJuliaC}
	} else {
		m.params = fractal.Params{Mode: fractal.Mandelbrot}
	}
	m.view = defaultView(m.params.Mode, 2)
	m.dirty = true
	return m
}

func (m model) Init() tea.Cmd { return nil }

func (m model) frameDims() (w, h int) {
	w = max(m.Width, 2)
	h = render.RowsToPixels(max(m.Height-statusRows, 1))
	return w, h
}

func (m *model) recompute() {
	if !m.Ready() {
		return
	}
	w, h := m.frameDims()
	m.frame.Resize(w, h)
	m.params.MaxIter = fractal.AdaptiveIter(m.view.Scale)
	fractal.Render(m.frame, m.view, m.params, m.gradient)
	m.dirty = false
}

func (m *model) cyclePalette(dir int) {
	names := palette.Names()
	i := slices.Index(names, m.palName)
	if i < 0 {
		i = 0
	}
	i = (i + dir + len(names)) % len(names)
	m.palName = names[i]
	m.gradient, _ = palette.Named(m.palName)
	m.dirty = true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Size.Update(msg)
		m.dirty = true
		m.recompute()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "up":
		m.view.CenterIm -= panFrac * m.view.Scale * float64(m.frame.H)
		m.dirty = true
	case "down":
		m.view.CenterIm += panFrac * m.view.Scale * float64(m.frame.H)
		m.dirty = true
	case "left":
		m.view.CenterRe -= panFrac * m.view.Scale * float64(m.frame.W)
		m.dirty = true
	case "right":
		m.view.CenterRe += panFrac * m.view.Scale * float64(m.frame.W)
		m.dirty = true

	case "+", "=":
		m.view.Scale *= zoomFactor
		m.dirty = true
	case "-", "_":
		m.view.Scale /= zoomFactor
		m.dirty = true

	case "j":
		w, _ := m.frameDims()
		if m.params.Mode == fractal.Mandelbrot {
			m.params.Mode = fractal.Julia
			m.params.JuliaC = complex(m.view.CenterRe, m.view.CenterIm)
			m.view = defaultView(fractal.Julia, w)
		} else {
			m.params.Mode = fractal.Mandelbrot
			m.view = defaultView(fractal.Mandelbrot, w)
		}
		m.dirty = true

	case "[":
		m.cyclePalette(-1)
	case "]":
		m.cyclePalette(1)

	case "r":
		w, _ := m.frameDims()
		m.view = defaultView(m.params.Mode, w)
		m.dirty = true
	}

	if m.dirty {
		m.recompute()
	}
	return m, nil
}

func (m model) View() string {
	if !m.Ready() {
		return "Loading the fractal...\n"
	}

	mode := "Mandelbrot"
	if m.params.Mode == fractal.Julia {
		mode = "Julia"
	}
	zoom := 1.0 / m.view.Scale

	var status strings.Builder
	status.WriteString(style.Title.Render("fractal"))
	status.WriteString(style.Note.Render(fmt.Sprintf(
		"  %s  c=(%.6g, %.6g)  zoom=%.4g  iter=%d  palette=%s",
		mode, m.view.CenterRe, m.view.CenterIm, zoom, m.params.MaxIter, m.palName,
	)))
	if m.params.Mode == fractal.Julia {
		status.WriteString(style.Note.Render(fmt.Sprintf("  C=%.4g%+.4gi",
			real(m.params.JuliaC), imag(m.params.JuliaC))))
	}
	help := style.Help.Render("arrows: pan   +/-: zoom   j: julia   [ ]: palette   r: reset   q: quit")

	return m.frame.String() + "\n" + status.String() + "\n" + help
}

func main() {
	var (
		palName    string
		startJulia bool
	)
	root := &cobra.Command{
		Use:   "fractal",
		Short: "Interactive Mandelbrot/Julia explorer",
		Long: "fractal renders escape-time fractals in truecolor. Pan with the arrow keys, " +
			"zoom with +/-, toggle Julia mode with j, and cycle palettes with [ and ].",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, ok := palette.Named(palName); !ok {
				return fmt.Errorf("unknown palette %q (available: %s)",
					palName, strings.Join(palette.Names(), ", "))
			}
			m := newModel(palName, startJulia)
			if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
				return fmt.Errorf("run program: %w", err)
			}
			return nil
		},
	}
	root.Flags().StringVar(&palName, "palette", "ultra", "color palette (see "+strings.Join(palette.Names(), ", ")+")")
	root.Flags().BoolVar(&startJulia, "julia", false, "start in Julia mode")

	cli.Execute(root)
}
