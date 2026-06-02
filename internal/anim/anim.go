// Package anim provides the shared tick-based animation loop for the toys: a
// steady framerate driver and a small holder for terminal dimensions.
package anim

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TickMsg is delivered once per animation frame. It carries the frame's
// timestamp so models can compute elapsed time precisely.
type TickMsg time.Time

// Frames produces a tea.Cmd that emits a TickMsg at the given framerate. Pass
// its result back into your Update loop after every TickMsg to keep ticking.
func Frames(fps int) tea.Cmd {
	if fps <= 0 {
		fps = 30
	}
	interval := time.Second / time.Duration(fps)
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Size tracks the usable terminal dimensions. Models embed it and call Update
// from their tea.WindowSizeMsg branch.
type Size struct {
	Width  int
	Height int
}

// Update records a new terminal size.
func (s *Size) Update(msg tea.WindowSizeMsg) {
	s.Width = msg.Width
	s.Height = msg.Height
}

// Ready reports whether a size has been received yet.
func (s Size) Ready() bool { return s.Width > 0 && s.Height > 0 }
