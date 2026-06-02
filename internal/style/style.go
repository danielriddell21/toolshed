// Package style holds the shared Lipgloss palette and styles for the toys.
// The palette is deliberately muted: these are calm, ambient tools.
package style

import "github.com/charmbracelet/lipgloss"

// Palette colours, chosen to be legible on both dark and light terminals.
var (
	Muted  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	Accent = lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#22D3EE"}
	Warm   = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	Faint  = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#4B5563"}
)

// Common styles reused across the toys.
var (
	Title  = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	Help   = lipgloss.NewStyle().Foreground(Faint)
	Note   = lipgloss.NewStyle().Foreground(Muted).Italic(true)
	Sprite = lipgloss.NewStyle().Foreground(Warm)
	Frame  = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Faint).
		Padding(0, 1)
)
