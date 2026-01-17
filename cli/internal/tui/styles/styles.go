// ABOUTME: Shared lipgloss styles for consistent TUI appearance
// ABOUTME: Defines colors, borders, and text styles used across components

package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors - Core palette
	Primary   = lipgloss.Color("#7C3AED") // Purple
	Secondary = lipgloss.Color("#10B981") // Green
	Warning   = lipgloss.Color("#F59E0B") // Amber
	Danger    = lipgloss.Color("#EF4444") // Red
	Muted     = lipgloss.Color("#6B7280") // Gray
	Text      = lipgloss.Color("#F9FAFB") // Light
	BgDark    = lipgloss.Color("#1F2937") // Dark gray

	// Colors - Extended palette
	Accent        = lipgloss.Color("#8B5CF6") // Lighter purple for highlights
	Surface       = lipgloss.Color("#374151") // Elevated surface background
	DeltaPositive = lipgloss.Color("#10B981") // Green - improvements
	DeltaNegative = lipgloss.Color("#F59E0B") // Amber - costs/increases
	DeltaNeutral  = lipgloss.Color("#6B7280") // Gray - no change
	Info          = lipgloss.Color("#3B82F6") // Blue - informational

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
