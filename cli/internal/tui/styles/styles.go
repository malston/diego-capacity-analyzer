// ABOUTME: Shared lipgloss styles for consistent TUI appearance
// ABOUTME: Defines colors, borders, and text styles used across components

package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors - Core palette (matches frontend React theme)
	Primary   = lipgloss.Color("#06B6D4") // Cyan-500 - primary accent (matching frontend)
	Secondary = lipgloss.Color("#34D399") // Emerald-400 - success/positive
	Warning   = lipgloss.Color("#FBBF24") // Amber-400 - warnings
	Danger    = lipgloss.Color("#F87171") // Red-400 - errors/critical
	Muted     = lipgloss.Color("#9CA3AF") // Gray-400 - muted text
	Text      = lipgloss.Color("#E5E7EB") // Gray-200 - primary text
	BgDark    = lipgloss.Color("#1E293B") // Slate-800 - dark background

	// Colors - Extended palette
	Accent        = lipgloss.Color("#22D3EE") // Cyan-400 - highlights/emphasis
	Surface       = lipgloss.Color("#334155") // Slate-700 - elevated surfaces
	DeltaPositive = lipgloss.Color("#34D399") // Emerald-400 - improvements
	DeltaNegative = lipgloss.Color("#FBBF24") // Amber-400 - costs/increases
	DeltaNeutral  = lipgloss.Color("#9CA3AF") // Gray-400 - no change
	Info          = lipgloss.Color("#3B82F6") // Blue-500 - informational

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

	// Frame styles for header/footer
	HeaderStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{
			Top:         "─",
			Bottom:      "",
			Left:        "╭",
			Right:       "╮",
			TopLeft:     "",
			TopRight:    "",
			BottomLeft:  "",
			BottomRight: "",
		}).
		BorderForeground(Muted).
		Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Border(lipgloss.Border{
			Top:         "",
			Bottom:      "─",
			Left:        "╰",
			Right:       "╯",
			TopLeft:     "",
			TopRight:    "",
			BottomLeft:  "",
			BottomRight: "",
		}).
		BorderForeground(Muted).
		Padding(0, 1)

	// Key style for keyboard shortcuts
	KeyStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	// Value style for emphasized data
	ValueStyle = lipgloss.NewStyle().
			Foreground(Text).
			Bold(true)

	// Delta styles for change indicators
	DeltaPositiveStyle = lipgloss.NewStyle().
				Foreground(DeltaPositive).
				Bold(true)

	DeltaNegativeStyle = lipgloss.NewStyle().
				Foreground(DeltaNegative).
				Bold(true)
)

// ProgressBar returns a styled progress bar string (matches frontend blue progress bars)
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

	color := Info // Blue-500 for normal utilization
	if percent >= 80 {
		color = Warning
	}
	if percent >= 95 {
		color = Danger
	}

	return lipgloss.NewStyle().Foreground(color).Render(bar)
}
