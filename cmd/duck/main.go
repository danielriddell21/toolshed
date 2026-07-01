package main

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/danielriddell21/toolshed/internal/cli"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/style"
)

const (
	fps       = 4
	decayIdle = 12
	maxLevel  = 4
)

var replies = [][]string{
	{"mhm.", "mm.", "right."},
	{"mhm…", "mm-hm…", "go on."},
	{"...mhm.", "...sure.", "(a long pause) ...mhm."},
	{"mhm. (eyebrow, raised)", "...is that so.", "mm. (visibly unconvinced)"},
	{"mhm. (the duck says nothing, louder)", "...", "(the duck just looks at you)"},
}

type line struct {
	you  bool
	text string
}

type model struct {
	rng     *rand.Rand
	input   string
	history []line
	level   int
	idle    int
}

func (m model) Init() tea.Cmd { return anim.Frames(fps) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD:
			return m, tea.Quit
		case tea.KeyEnter:
			m.submit()
			return m, nil
		case tea.KeyBackspace, tea.KeyDelete:
			if n := len(m.input); n > 0 {
				m.input = m.input[:n-1]
			}
			return m, nil
		case tea.KeySpace:
			m.input += " "
			return m, nil
		case tea.KeyRunes:
			// "q" on an empty line quits; otherwise it's just typing.
			if m.input == "" && string(msg.Runes) == "q" {
				return m, tea.Quit
			}
			m.input += string(msg.Runes)
			return m, nil
		}
		return m, nil

	case anim.TickMsg:
		if m.idle++; m.idle >= decayIdle {
			m.idle = 0
			m.level = max(m.level-1, 0)
		}
		return m, anim.Frames(fps)
	}
	return m, nil
}

func (m *model) submit() {
	text := strings.TrimSpace(m.input)
	m.input = ""
	if text == "" {
		return
	}
	m.history = append(m.history, line{you: true, text: text})

	// The more (and the more often) you talk, the more skeptical the duck.
	m.level = min(m.level+1, maxLevel)
	m.idle = 0
	variants := replies[m.level]
	m.history = append(m.history, line{text: variants[m.rng.Intn(len(variants))]})

	// Keep the transcript bounded.
	if len(m.history) > 40 {
		m.history = m.history[len(m.history)-40:]
	}
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(style.Title.Render("duck"))
	b.WriteString("\n\n")
	if len(m.history) == 0 {
		b.WriteString(style.Note.Render("Tell the duck about your bug. It is listening. Sort of."))
		b.WriteByte('\n')
	}
	for _, l := range m.history {
		if l.you {
			b.WriteString("you:  " + l.text + "\n")
		} else {
			b.WriteString(style.Sprite.Render("duck: "+l.text) + "\n")
		}
	}
	b.WriteString("\n> " + m.input + "_\n")
	b.WriteString(style.Help.Render("enter: speak   ctrl+d or q: leave"))
	return b.String()
}

func main() {
	var seed int64
	root := &cobra.Command{
		Use:   "duck",
		Short: "A rubber-duck companion that only says mhm",
		Long: "duck is a rubber-duck debugging companion. It replies only with \"mhm\", " +
			"growing more skeptical the more you explain. Its skepticism fades if you pause.",
		RunE: func(cmd *cobra.Command, args []string) error {
			seed = cli.DefaultSeed(seed)
			m := model{rng: rand.New(rand.NewSource(seed))}
			_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "the duck watches you go. mhm.")
			if err != nil {
				return fmt.Errorf("run program: %w", err)
			}
			return nil
		},
	}
	root.Flags().Int64Var(&seed, "seed", 0, "random seed (0 = pick one)")
	cli.Execute(root)
}
