// Command garden is a zen sand garden you rake with the arrow keys.
package main

import (
	"github.com/danielriddell21/toolshed/internal/buildinfo"

	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/style"
	"github.com/spf13/cobra"
)

// rakePatterns are the strokes the rake can leave in the sand.
var rakePatterns = []rune{'~', '=', '-', '≈', '·'}

const sand = '·' // a faint dot, the untouched sand

type model struct {
	size    anim.Size
	grid    [][]rune
	cx, cy  int
	pattern int
}

func (m model) Init() tea.Cmd { return nil }

func (m *model) ensureGrid() {
	w := max(m.size.Width, 10)
	h := max(m.size.Height-3, 5)
	if len(m.grid) == h && len(m.grid[0]) == w {
		return
	}
	m.grid = make([][]rune, h)
	for y := range m.grid {
		row := make([]rune, w)
		for x := range row {
			row[x] = sand
		}
		m.grid[y] = row
	}
	m.cx = min(m.cx, w-1)
	m.cy = min(m.cy, h-1)
}

func (m *model) reset() {
	for y := range m.grid {
		for x := range m.grid[y] {
			m.grid[y][x] = sand
		}
	}
}

func (m *model) rake() {
	if m.cy < len(m.grid) && m.cx < len(m.grid[m.cy]) {
		m.grid[m.cy][m.cx] = rakePatterns[m.pattern]
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.size.Update(msg)
		m.ensureGrid()
		m.rake()
		return m, nil

	case tea.KeyMsg:
		if !m.size.Ready() {
			return m, nil
		}
		h, w := len(m.grid), len(m.grid[0])
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up":
			m.cy = max(m.cy-1, 0)
		case "down":
			m.cy = min(m.cy+1, h-1)
		case "left":
			m.cx = max(m.cx-1, 0)
		case "right":
			m.cx = min(m.cx+1, w-1)
		case "p", "tab", " ":
			m.pattern = (m.pattern + 1) % len(rakePatterns)
		case "r":
			m.reset()
		}
		m.rake()
		return m, nil
	}
	return m, nil
}

func (m model) View() string {
	if !m.size.Ready() {
		return "Smoothing the sand...\n"
	}
	var b strings.Builder
	b.WriteString(style.Title.Render("garden") + style.Note.Render("  — rake, breathe, repeat"))
	b.WriteByte('\n')
	for y, row := range m.grid {
		if y == m.cy {
			line := make([]rune, len(row))
			copy(line, row)
			line[m.cx] = '+'
			b.WriteString(style.Sprite.Render(string(line)))
		} else {
			b.WriteString(style.Note.Render(string(row)))
		}
		b.WriteByte('\n')
	}
	b.WriteString(style.Help.Render(fmt.Sprintf(
		"arrows: rake   p: pattern (%c)   r: reset   q: quit", rakePatterns[m.pattern])))
	return b.String()
}

func main() {
	buildinfo.HandleVersionFlag()
	root := &cobra.Command{
		Use:   "garden",
		Short: "Rake a zen sand garden",
		Long: "garden is a small patch of sand you rake with the arrow keys. Cycle rake " +
			"patterns, reset when you like. There is no score and no end.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := tea.NewProgram(model{}, tea.WithAltScreen()).Run()
			return err
		},
	}
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "garden:", err)
		os.Exit(1)
	}
}
