// Command snail is a Pomodoro timer in which a snail crosses the screen exactly
// once per session.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/danielriddell21/toolshed/internal/anim"
	"github.com/danielriddell21/toolshed/internal/sprite"
	"github.com/danielriddell21/toolshed/internal/style"
	"github.com/spf13/cobra"
)

const fps = 8

type phase int

const (
	phaseWork phase = iota
	phaseBreak
	phaseDone
)

type model struct {
	size    anim.Size
	work    time.Duration
	rest    time.Duration
	phase   phase
	elapsed time.Duration
	running bool
	last    time.Time
	rang    bool
}

func (m model) Init() tea.Cmd { return anim.Frames(fps) }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.size.Update(msg)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case " ":
			m.running = !m.running
			m.last = time.Now()
		}
		return m, nil

	case anim.TickMsg:
		m.advance(time.Time(msg))
		return m, anim.Frames(fps)
	}
	return m, nil
}

func (m *model) advance(now time.Time) {
	if m.phase == phaseDone {
		return
	}
	if m.running {
		if !m.last.IsZero() {
			m.elapsed += now.Sub(m.last)
		}
	}
	m.last = now

	if m.elapsed >= m.current() {
		switch m.phase {
		case phaseWork:
			m.bell()
			if m.rest > 0 {
				m.phase = phaseBreak
				m.elapsed = 0
				m.rang = false
			} else {
				m.phase = phaseDone
			}
		case phaseBreak:
			m.bell()
			m.phase = phaseDone
		}
	}
}

func (m *model) current() time.Duration {
	if m.phase == phaseBreak {
		return m.rest
	}
	return m.work
}

func (m *model) bell() {
	if !m.rang {
		fmt.Print("\a")
		m.rang = true
	}
}

func (m model) progress() float64 {
	d := m.current()
	if d <= 0 {
		return 1
	}
	return min(float64(m.elapsed)/float64(d), 1)
}

func (m model) remaining() time.Duration {
	r := m.current() - m.elapsed
	return max(r, 0).Round(time.Second)
}

func (m model) View() string {
	if !m.size.Ready() {
		return "Waking the snail...\n"
	}
	w := max(m.size.Width, 24)
	track := w - 6
	c := sprite.NewCanvas(w, 3)

	// Draw the ground line and the goal.
	for x := range track {
		c.Set(x, 1, '.')
	}
	c.Set(track, 1, '|')

	snail := sprite.Sprite{
		X:         m.progress() * float64(track-3),
		Y:         0,
		Dir:       sprite.Right,
		FaceRight: "@~",
		FaceLeft:  "~@",
	}
	c.DrawSprite(snail)

	var b strings.Builder
	b.WriteString(style.Title.Render("snail") + style.Note.Render("  — one crossing, your whole session"))
	b.WriteByte('\n')
	b.WriteString(c.String())
	b.WriteByte('\n')

	switch m.phase {
	case phaseWork:
		b.WriteString(style.Sprite.Render("focus  ") + fmt.Sprintf("remaining %s", m.remaining()))
	case phaseBreak:
		b.WriteString(style.Title.Render("break  ") + fmt.Sprintf("remaining %s", m.remaining()))
	case phaseDone:
		b.WriteString(style.Sprite.Render("done. the snail has arrived."))
	}
	if !m.running && m.phase != phaseDone {
		b.WriteString(style.Note.Render("  (paused)"))
	}
	b.WriteByte('\n')
	b.WriteString(style.Help.Render("space: pause/resume   q: quit"))
	return b.String()
}

func main() {
	var (
		work time.Duration
		rest time.Duration
	)
	root := &cobra.Command{
		Use:   "snail",
		Short: "A Pomodoro timer paced by a crossing snail",
		Long: "snail runs a Pomodoro session in which a single snail crosses the screen " +
			"exactly once. Its position is your progress; the clock shows what's left.",
		RunE: func(cmd *cobra.Command, args []string) error {
			m := model{
				work:    work,
				rest:    rest,
				phase:   phaseWork,
				running: true,
				last:    time.Now(),
			}
			_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
			return err
		},
	}
	root.Flags().DurationVar(&work, "duration", 25*time.Minute, "length of the focus session")
	root.Flags().DurationVar(&rest, "break", 0, "optional break session (second crossing)")
	root.SilenceUsage = true

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "snail:", err)
		os.Exit(1)
	}
}
