// ABOUTME: Shared lipgloss styles for consistent TUI appearance
// ABOUTME: Defines colors, borders, and text styles used across components

package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#10B981") // Green
	Warning   = lipgloss.Color("#F59E0B") // Amber
	Danger    = lipgloss.Color("#EF4444") // Red
	Muted     = lipgloss.Color("#6B7280") // Gray
	Text      = lipgloss.Color("#F9FAFB") // Light
	BgDark    = lipgloss.Color("#1F2937") // Dark gray

	// Base styles
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Muted).
			MarginBottom(1)

	// Status indicators
	StatusOK = lipgloss.NewStyle().
			Foreground(Secondary).
			Bold(true)

	StatusWarning = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	StatusCritical = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true)

	// Panels
	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(1, 2)

	ActivePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	// Help text
	Help = lipgloss.NewStyle().
		Foreground(Muted).
		MarginTop(1)
)

// ProgressBar returns a styled progress bar string
func ProgressBar(percent float64, width int) string {
	filled := int(percent / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := ""
	for i := 0; i < width; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	color := Secondary
	if percent >= 80 {
		color = Warning
	}
	if percent >= 95 {
		color = Danger
	}

	return lipgloss.NewStyle().Foreground(color).Render(bar)
}
