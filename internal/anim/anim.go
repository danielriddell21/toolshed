package anim

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type TickMsg time.Time

func Frames(fps int) tea.Cmd {
	if fps <= 0 {
		fps = 30
	}
	interval := time.Second / time.Duration(fps)
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

type Size struct {
	Width  int
	Height int
}

func (s *Size) Update(msg tea.WindowSizeMsg) {
	s.Width = msg.Width
	s.Height = msg.Height
}

func (s Size) Ready() bool { return s.Width > 0 && s.Height > 0 }
