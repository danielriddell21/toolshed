package main

import (
	"fmt"
	"strings"

	"github.com/danielriddell21/toolshed/internal/cli"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/life"
	"github.com/danielriddell21/toolshed/internal/render"
	"github.com/danielriddell21/toolshed/internal/style"
)

const statusRows = 2

var (
	deadColor = render.RGB{R: 12, G: 14, B: 20}

	liveLow  = render.RGB{R: 34, G: 211, B: 238}
	liveHigh = render.RGB{R: 251, G: 191, B: 36}
)

type model struct {
	anim.Size

	board *life.Board
	frame *render.Frame

	path    string
	speed   int
	forcedW int
	forcedH int
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

func (m model) boardSize() (int, int) {
	if m.forcedW > 0 && m.forcedH > 0 {
		return m.forcedW, m.forcedH
	}
	w := max(m.Width, 1)
	h := render.RowsToPixels(max(m.Height-statusRows, 1))
	return w, h
}

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
			n := m.board.Neighbors(x, y)
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

func parseSize(s string) (int, int, error) {
	if s == "" {
		return 0, 0, nil
	}
	w, h, err := cli.ParseDims(s)
	if err != nil {
		return 0, 0, fmt.Errorf("parse --size: %w", err)
	}
	return w, h, nil
}

func main() {
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
			if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
				return fmt.Errorf("run program: %w", err)
			}
			return nil
		},
	}
	root.Flags().StringVar(&path, "path", ".", "directory tree to seed from")
	root.Flags().IntVar(&speed, "speed", 10, "steps per second")
	root.Flags().StringVar(&size, "size", "", "force grid pixel size as WxH (default: derive from terminal)")

	cli.Execute(root)
}
