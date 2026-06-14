// Command maze generates or loads a maze and animates a BFS / A* search
// solving it, then reveals the shortest path.
package main

import (
	"github.com/danielriddell21/toolshed/internal/buildinfo"

	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/maze"
	"github.com/danielriddell21/toolshed/internal/style"
	"github.com/spf13/cobra"
)

// cellState tracks how far the replay has touched each cell.
type cellState uint8

const (
	stUntouched cellState = iota
	stFrontier
	stVisited
	stPath
)

type model struct {
	anim.Size
	mz     *maze.Maze
	algo   maze.Algo
	speed  int
	steps  []maze.Step
	path   []maze.Point
	found  bool
	cursor int         // index into steps consumed so far
	state  []cellState // per-cell render state, indexed y*W+x
	done   bool        // replay finished, path revealed

	// precomputed per-cell styles, reused every frame
	wallStyle, openStyle, visitStyle, frontStyle, pathStyle, startStyle, endStyle lipgloss.Style
}

func newModel(mz *maze.Maze, algo maze.Algo, speed int) model {
	steps, path, found := maze.Solve(mz, algo)
	return model{
		mz:    mz,
		algo:  algo,
		speed: speed,
		steps: steps,
		path:  path,
		found: found,
		state: make([]cellState, mz.W*mz.H),

		wallStyle:  lipgloss.NewStyle().Foreground(style.Faint),
		openStyle:  lipgloss.NewStyle().Foreground(style.Faint),
		visitStyle: lipgloss.NewStyle().Foreground(style.Muted),
		frontStyle: lipgloss.NewStyle().Foreground(style.Accent),
		pathStyle:  lipgloss.NewStyle().Foreground(style.Warm).Bold(true),
		startStyle: lipgloss.NewStyle().Foreground(style.Warm).Bold(true),
		endStyle:   lipgloss.NewStyle().Foreground(style.Accent).Bold(true),
	}
}

func (m model) Init() tea.Cmd { return anim.Frames(m.speed) }

func (m *model) advance() {
	if m.done {
		return
	}
	// Consume a few steps per tick to keep larger searches lively.
	const perTick = 1
	for range perTick {
		if m.cursor >= len(m.steps) {
			m.reveal()
			return
		}
		s := m.steps[m.cursor]
		idx := s.P.Y*m.mz.W + s.P.X
		switch s.Kind {
		case maze.Frontier:
			if m.state[idx] == stUntouched {
				m.state[idx] = stFrontier
			}
		case maze.Visit:
			m.state[idx] = stVisited
		}
		m.cursor++
	}
}

func (m *model) reveal() {
	for _, p := range m.path {
		m.state[p.Y*m.mz.W+p.X] = stPath
	}
	m.done = true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Size.Update(msg)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
		return m, nil
	case anim.TickMsg:
		m.advance()
		return m, anim.Frames(m.speed)
	}
	return m, nil
}

func (m model) glyph(x, y int) (string, lipgloss.Style) {
	c := m.mz.At(x, y)
	idx := y*m.mz.W + x
	st := m.state[idx]
	switch c {
	case maze.Wall:
		return "█", m.wallStyle
	case maze.Start:
		return "S", m.startStyle
	case maze.End:
		return "E", m.endStyle
	}
	// Open cell: color by search state.
	switch st {
	case stPath:
		return "•", m.pathStyle
	case stVisited:
		return "·", m.visitStyle
	case stFrontier:
		return "▒", m.frontStyle
	default:
		return " ", m.openStyle
	}
}

func (m model) View() string {
	if !m.Ready() {
		return "Loading the maze...\n"
	}

	var b strings.Builder
	b.WriteString(style.Title.Render("maze"))
	b.WriteString(style.Note.Render("  — watch the search find its way"))
	b.WriteByte('\n')

	// Reserve rows for the header (1), blank (1), status (1), help (1).
	const chrome = 5
	maxRows := max(m.Height-chrome, 1)
	maxCols := max(m.Width, 1)
	rows := min(m.mz.H, maxRows)
	cols := min(m.mz.W, maxCols)

	for y := range rows {
		var line strings.Builder
		for x := range cols {
			g, sty := m.glyph(x, y)
			line.WriteString(sty.Render(g))
		}
		b.WriteString(line.String())
		b.WriteByte('\n')
	}

	b.WriteByte('\n')
	b.WriteString(m.status())
	b.WriteByte('\n')
	b.WriteString(style.Help.Render("q: quit"))
	return b.String()
}

func (m model) status() string {
	name := "bfs"
	if m.algo == maze.AStar {
		name = "astar"
	}
	visited := min(m.cursor, len(m.steps))
	parts := []string{
		fmt.Sprintf("algo: %s", name),
		fmt.Sprintf("steps: %d/%d", visited, len(m.steps)),
	}
	if m.done {
		if m.found {
			parts = append(parts, fmt.Sprintf("path: %d", len(m.path)))
		} else {
			parts = append(parts, "path: none")
		}
	}
	return style.Note.Render(strings.Join(parts, "   "))
}

func parseDims(s string) (int, int, error) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("bad size %q (want WxH, e.g. 31x21)", s)
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || w <= 0 {
		return 0, 0, fmt.Errorf("bad width in %q", s)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || h <= 0 {
		return 0, 0, fmt.Errorf("bad height in %q", s)
	}
	return w, h, nil
}

func loadMaze(in, generate string, seed int64) (*maze.Maze, error) {
	if generate != "" {
		w, h, err := parseDims(generate)
		if err != nil {
			return nil, err
		}
		if seed == 0 {
			seed = time.Now().UnixNano()
		}
		return maze.Generate(w, h, rand.New(rand.NewSource(seed))), nil
	}
	if in == "" {
		return nil, fmt.Errorf("either --in or --generate is required")
	}
	f, err := os.Open(in)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return maze.Parse(f)
}

func main() {
	buildinfo.HandleVersionFlag()
	var (
		in       string
		algoStr  string
		speed    int
		generate string
		seed     int64
	)
	root := &cobra.Command{
		Use:   "maze",
		Short: "Animate a BFS or A* search solving a maze",
		Long: "maze loads a maze from a file (--in) or generates one (--generate WxH), " +
			"then animates the chosen search algorithm exploring it before revealing the shortest path.",
		RunE: func(cmd *cobra.Command, args []string) error {
			algo, err := maze.ParseAlgo(algoStr)
			if err != nil {
				return err
			}
			mz, err := loadMaze(in, generate, seed)
			if err != nil {
				return err
			}
			m := newModel(mz, algo, speed)
			_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
			return err
		},
	}
	root.Flags().StringVar(&in, "in", "", "maze file to read")
	root.Flags().StringVar(&algoStr, "algo", "bfs", "search algorithm: bfs|astar")
	root.Flags().IntVar(&speed, "speed", 30, "animation steps per second")
	root.Flags().StringVar(&generate, "generate", "", "generate a maze instead of reading --in, e.g. 31x21")
	root.Flags().Int64Var(&seed, "seed", 0, "random seed for generation (0 = time-based)")
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "maze:", err)
		os.Exit(1)
	}
}
