// Command life runs Conway's Game of Life seeded by a directory tree. The same
// tree always produces the same starting pattern, so a project's shape becomes
// its own initial condition. Cells live on a torus and evolve under the classic
// B3/S23 rules.
package main

import (
	"github.com/danielriddell21/toolshed/internal/buildinfo"

	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/life"
	"github.com/danielriddell21/toolshed/internal/render"
	"github.com/danielriddell21/toolshed/internal/style"
	"github.com/spf13/cobra"
)

// statusRows is how many text rows the status line plus title occupy below the
// board; the rest of the terminal height is given to the simulation.
const statusRows = 2

var (
	deadColor = render.RGB{R: 12, G: 14, B: 20}
	// liveLow/liveHigh bracket the colour ramp used to tint cells by how
	// crowded their neighbourhood is, for a little visual depth.
	liveLow  = render.RGB{R: 34, G: 211, B: 238} // style.Accent-ish cyan
	liveHigh = render.RGB{R: 251, G: 191, B: 36} // style.Warm amber
)

type model struct {
	anim.Size

	board *life.Board
	frame *render.Frame

	path    string
	speed   int
	forcedW int // >0 when --size pins the grid width
	forcedH int // >0 when --size pins the grid height
	paused  bool
}

func newModel(path string, speed, forcedW, forcedH int) model {
	return model{
		path:    path,
		speed:   speed,
		forcedW: forcedW,
		forcedH: forcedH,
	}
}

func (m model) Init() tea.Cmd { return anim.Frames(m.speed) }

// boardSize returns the grid dimensions for the current terminal size, honoring
// a forced --size if one was given.
func (m model) boardSize() (int, int) {
	if m.forcedW > 0 && m.forcedH > 0 {
		return m.forcedW, m.forcedH
	}
	w := max(m.Width, 1)
	h := render.RowsToPixels(max(m.Height-statusRows, 1))
	return w, h
}

// reseed (re)builds the board at the current target size and seeds it from the
// directory tree. We reseed on every resize so the pattern always fills the
// available space; because seeding is deterministic the visual identity of a
// given tree is preserved across sizes (the layout scales, the source does not).
func (m *model) reseed() {
	w, h := m.boardSize()
	m.board = life.NewBoard(w, h)
	m.frame = render.NewFrame(w, h)
	// Best effort: if the tree can't be walked we simply start empty rather
	// than crashing the TUI; the status line will show population 0.
	_ = life.SeedFromTree(m.board, m.path)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Size.Update(msg)
		m.reseed()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case " ":
			m.paused = !m.paused
		case "r":
			m.reseed()
		case "n":
			if m.board != nil {
				m.board.Step()
			}
		}
		return m, nil

	case anim.TickMsg:
		if !m.paused && m.board != nil {
			m.board.Step()
		}
		// Always keep the clock running so unpausing resumes immediately.
		return m, anim.Frames(m.speed)
	}
	return m, nil
}

func (m model) View() string {
	if !m.Ready() || m.board == nil {
		return "Seeding life...\n"
	}

	m.frame.Fill(deadColor)
	for y := range m.board.H() {
		for x := range m.board.W() {
			if !m.board.Alive(x, y) {
				continue
			}
			// Tint by local crowding for visual interest.
			n := neighbors(m.board, x, y)
			t := float64(n) / 8.0
			m.frame.Set(x, y, render.Lerp(liveLow, liveHigh, t))
		}
	}

	var b strings.Builder
	b.WriteString(style.Title.Render("life"))
	b.WriteString(style.Note.Render("  — your directory, evolving"))
	b.WriteByte('\n')
	b.WriteString(m.frame.String())
	b.WriteByte('\n')
	b.WriteString(m.status())
	return b.String()
}

func (m model) status() string {
	state := "running"
	if m.paused {
		state = "paused"
	}
	left := style.Note.Render(fmt.Sprintf(
		"gen %d   pop %d   %s",
		m.board.Generation(), m.board.Population(), state,
	))
	help := style.Help.Render("space: pause   n: step   r: reseed   q: quit")
	return left + "   " + help
}

// neighbors counts live toroidal neighbours of (x,y) for colour shading.
func neighbors(b *life.Board, x, y int) int {
	n := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if b.Alive(x+dx, y+dy) {
				n++
			}
		}
	}
	return n
}

// parseSize parses a "WxH" string into pixel dimensions. An empty string means
// "derive from the terminal" and returns (0,0,nil).
func parseSize(s string) (int, int, error) {
	if s == "" {
		return 0, 0, nil
	}
	parts := strings.Split(strings.ToLower(s), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid --size %q: want WxH", s)
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || w <= 0 {
		return 0, 0, fmt.Errorf("invalid --size width in %q", s)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || h <= 0 {
		return 0, 0, fmt.Errorf("invalid --size height in %q", s)
	}
	return w, h, nil
}

func main() {
	buildinfo.HandleVersionFlag()
	var (
		path  string
		speed int
		size  string
	)
	root := &cobra.Command{
		Use:   "life",
		Short: "Conway's Game of Life seeded by a directory tree",
		Long: "life seeds Conway's Game of Life from the shape of a directory tree. " +
			"The same tree always produces the same starting pattern, which then " +
			"evolves under the classic B3/S23 rules on a toroidal board.",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, h, err := parseSize(size)
			if err != nil {
				return err
			}
			m := newModel(path, max(speed, 1), w, h)
			_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
			return err
		},
	}
	root.Flags().StringVar(&path, "path", ".", "directory tree to seed from")
	root.Flags().IntVar(&speed, "speed", 10, "steps per second")
	root.Flags().StringVar(&size, "size", "", "force grid pixel size as WxH (default: derive from terminal)")
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "life:", err)
		os.Exit(1)
	}
}
