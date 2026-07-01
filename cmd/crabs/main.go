package main

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/danielriddell21/toolshed/internal/cli"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/sprite"
	"github.com/danielriddell21/toolshed/internal/style"
)

const (
	crabGlyph  = "(\\/)°<"
	finishPad  = 4
	holdFrames = 45
	fps        = 30
)

type model struct {
	anim.Size
	rng    *rand.Rand
	crabs  []sprite.Sprite
	bases  []float64
	speed  float64
	count  int
	loop   bool
	winner int
	hold   int
}

func newModel(count int, speed float64, loop bool, seed int64) model {
	return model{
		rng:    rand.New(rand.NewSource(seed)),
		speed:  speed,
		count:  count,
		loop:   loop,
		winner: -1,
	}
}

func (m model) Init() tea.Cmd { return anim.Frames(fps) }

func (m *model) reset() {
	m.crabs = make([]sprite.Sprite, m.count)
	m.bases = make([]float64, m.count)
	for i := range m.crabs {
		m.crabs[i] = sprite.Sprite{
			X:         0,
			Y:         float64(i*2 + 1),
			Dir:       sprite.Right,
			FaceRight: crabGlyph,
			FaceLeft:  crabGlyph,
		}
		m.bases[i] = 0.3 + m.rng.Float64()*0.7
	}
	m.winner = -1
	m.hold = 0
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Size.Update(msg)
		if m.crabs == nil {
			m.reset()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			m.reset()
		}
		return m, nil

	case anim.TickMsg:
		m.step()
		return m, anim.Frames(fps)
	}
	return m, nil
}

func (m *model) step() {
	if !m.Ready() {
		return
	}
	if m.winner >= 0 {
		if m.loop {
			if m.hold++; m.hold >= holdFrames {
				m.reset()
			}
		}
		return
	}
	finish := float64(m.Width - finishPad)
	for i := range m.crabs {
		m.crabs[i].X += m.bases[i] * m.speed * (0.5 + m.rng.Float64())
		if m.crabs[i].X >= finish {
			m.crabs[i].X = finish
			m.winner = i
		}
	}
}

func (m model) View() string {
	if !m.Ready() {
		return "Loading the crabs...\n"
	}
	w := max(m.Width, 20)
	h := max(m.count*2+1, 4)
	c := sprite.NewCanvas(w, h)

	finish := m.Width - finishPad
	for y := range h {
		c.Set(finish+1, y, '|')
	}
	for _, crab := range m.crabs {
		c.DrawSprite(crab)
	}

	var b strings.Builder
	b.WriteString(style.Title.Render("crabs") + style.Note.Render("  — a race with no stakes"))
	b.WriteByte('\n')
	b.WriteString(c.String())
	b.WriteByte('\n')
	if m.winner >= 0 {
		b.WriteString(style.Sprite.Render(fmt.Sprintf("Crab %d wins. Nothing changes.", m.winner+1)))
	} else {
		b.WriteString(style.Note.Render("They're off, in their own time."))
	}
	b.WriteByte('\n')
	b.WriteString(style.Help.Render("r: restart   q: quit"))
	return b.String()
}

func main() {
	var (
		count int
		speed float64
		loop  bool
		seed  int64
	)
	root := &cobra.Command{
		Use:   "crabs",
		Short: "Race ASCII crabs across the terminal",
		Long: "crabs runs a race between two or more ASCII crabs at randomised speeds. " +
			"There is a winner. There is no prize.",
		RunE: func(cmd *cobra.Command, args []string) error {
			count = max(count, 1)
			seed = cli.DefaultSeed(seed)
			m := newModel(count, speed, loop, seed)
			if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
				return fmt.Errorf("run program: %w", err)
			}
			return nil
		},
	}
	root.Flags().IntVar(&count, "count", 2, "number of crabs")
	root.Flags().Float64Var(&speed, "speed", 1.0, "pace multiplier")
	root.Flags().BoolVar(&loop, "loop", false, "restart automatically after each race")
	root.Flags().Int64Var(&seed, "seed", 0, "random seed (0 = pick one)")

	cli.Execute(root)
}
