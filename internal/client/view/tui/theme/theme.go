package theme

import "charm.land/lipgloss/v2"

// Catppuccin palette shared across every screen.
const (
	RoseWater = "#dc8a78"
	Teal      = "#179299"
)

// Reusable styles shared across every screen.
var (
	Title   = lipgloss.NewStyle().Foreground(lipgloss.Black).Background(lipgloss.Color(RoseWater)).Bold(true).Padding(0, 1)
	Focused = lipgloss.NewStyle().Foreground(lipgloss.Color(RoseWater))
	Blurred = lipgloss.NewStyle().Foreground(lipgloss.BrightBlack)
	Filled  = lipgloss.NewStyle().Foreground(lipgloss.Color(Teal))
)
