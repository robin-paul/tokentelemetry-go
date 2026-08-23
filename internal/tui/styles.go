package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Theme Palette
	ColorPrimary   = lipgloss.Color("#7C3AED") // Electric Purple / Violet
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan
	ColorAccent    = lipgloss.Color("#3B82F6") // Bright Blue
	ColorSuccess   = lipgloss.Color("#10B981") // Emerald Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorDanger    = lipgloss.Color("#EF4444") // Crimson
	ColorMuted     = lipgloss.Color("#64748B") // Slate Muted
	ColorBgCard    = lipgloss.Color("#1E293B") // Dark Slate Box
	ColorFgBright  = lipgloss.Color("#F8FAFC") // Off-white
	ColorFgMuted   = lipgloss.Color("#94A3B8") // Gray

	// Header Styles
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorFgBright).
			Background(ColorPrimary).
			Padding(0, 1)

	StyleHeader = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorMuted).
			Padding(0, 0, 1, 0)

	StyleBadgeOnline = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#064E3B")).
				Background(ColorSuccess).
				Padding(0, 1)

	StyleBadgeOffline = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorFgBright).
				Background(ColorDanger).
				Padding(0, 1)

	StyleBadgePaused = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#78350F")).
				Background(ColorWarning).
				Padding(0, 1)

	// KPI Metric Box Styles
	StyleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)

	StyleCardTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	StyleCardValue = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorFgBright)

	StyleCardSub = lipgloss.NewStyle().
			Foreground(ColorFgMuted)

	// Footer & Help Styles
	StyleFooter = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(ColorMuted).
			Padding(0, 1).
			Foreground(ColorFgMuted)

	StyleKeyHelp = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)
)
