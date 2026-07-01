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

const fps = 12

var thoughts = []string{
	"...",
	"again, this corner.",
	"the same water as before.",
	"I have seen this rock.",
	"is it Tuesday in here?",
	"glass. always glass.",
	"I could leave. I won't.",
}

type bubble struct{ x, y float64 }

type model struct {
	anim.Size
	rng     *rand.Rand
	fish    sprite.Sprite
	bubbles []bubble
	thought string
	boredom int
	frame   int
	still   bool
}

func (m model) Init() tea.Cmd { return anim.Frames(fps) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Size.Update(msg)
		m.fish.Y = float64(m.Height / 2)
		if m.fish.Speed == 0 && !m.still {
			m.fish.Speed = 0.6
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
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
	m.frame++
	if !m.still {
		m.fish.Advance(m.Width)
		// Occasional vertical drift keeps the tank from feeling on rails.
		if m.frame%9 == 0 {
			m.fish.Y += float64(m.rng.Intn(3) - 1)
			m.fish.Y = clampF(m.fish.Y, 1, float64(m.Height-3))
		}
	}

	// Emit a bubble now and then from the fish's mouth.
	if m.frame%14 == 0 {
		mouth := m.fish.X
		if m.fish.Dir == sprite.Right {
			mouth += float64(m.fish.Width())
		}
		m.bubbles = append(m.bubbles, bubble{x: mouth, y: m.fish.Y})
	}
	// Rise and cull bubbles.
	kept := m.bubbles[:0]
	for _, b := range m.bubbles {
		b.y -= 0.5
		if b.y > 0 {
			kept = append(kept, b)
		}
	}
	m.bubbles = kept

	// A new deadpan thought, slowly, as boredom mounts.
	if m.frame%45 == 0 {
		idx := min(m.boredom, len(thoughts)-1)
		m.thought = thoughts[idx]
		m.boredom++
	}
}

func (m model) View() string {
	if !m.Ready() {
		return "Filling the tank...\n"
	}
	w := max(m.Width, 20)
	h := max(m.Height-3, 6)
	c := sprite.NewCanvas(w, h)

	c.DrawString(3, h-2, "~ ~ ~  the gravel  ~ ~ ~")
	for _, b := range m.bubbles {
		c.Set(int(b.x+0.5), int(b.y+0.5), 'o')
	}
	c.DrawSprite(m.fish)

	var b strings.Builder
	b.WriteString(style.Title.Render("fish") + style.Note.Render("  — one fish, fully tank-aware"))
	b.WriteByte('\n')
	b.WriteString(style.Frame.Render(c.String()))
	b.WriteByte('\n')
	if m.thought != "" {
		b.WriteString(style.Note.Render("fish: " + m.thought))
		b.WriteByte('\n')
	}
	b.WriteString(style.Help.Render("q: quit"))
	return b.String()
}

func clampF(v, lo, hi float64) float64 { return max(lo, min(hi, v)) }

func main() {
	var (
		still bool
		seed  int64
	)
	root := &cobra.Command{
		Use:   "fish",
		Short: "Keep a tank with exactly one bored fish",
		Long: "fish maintains a tank containing exactly one fish. It drifts within bounds, " +
			"faces where it's going, blows the occasional bubble, and is bored.",
		RunE: func(cmd *cobra.Command, args []string) error {
			seed = cli.DefaultSeed(seed)
			m := model{
				rng:   rand.New(rand.NewSource(seed)),
				still: still,
				fish: sprite.Sprite{
					X:         3,
					Dir:       sprite.Right,
					FaceRight: "><>",
					FaceLeft:  "<><",
				},
			}
			if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
				return fmt.Errorf("run program: %w", err)
			}
			return nil
		},
	}
	root.Flags().BoolVar(&still, "still", false, "let the fish rest in place")
	root.Flags().Int64Var(&seed, "seed", 0, "random seed (0 = pick one)")

	cli.Execute(root)
}
