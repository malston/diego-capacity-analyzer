// ABOUTME: Enhanced progress bar with visual threshold zones
// ABOUTME: Shows green/amber/red regions for capacity-aware displays

package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBarConfig holds configuration for the progress bar
type ProgressBarConfig struct {
	Width         int
	WarnThreshold float64 // Percentage where warning zone starts (default 80)
	CritThreshold float64 // Percentage where critical zone starts (default 95)
	OKColor       lipgloss.Color
	WarnColor     lipgloss.Color
	CritColor     lipgloss.Color
	EmptyColor    lipgloss.Color
	ShowZones     bool // Show threshold markers in the bar
}

// DefaultProgressBarConfig returns sensible defaults
func DefaultProgressBarConfig() ProgressBarConfig {
	return ProgressBarConfig{
		Width:         20,
		WarnThreshold: 80,
		CritThreshold: 95,
		OKColor:       lipgloss.Color("#10B981"), // Green
		WarnColor:     lipgloss.Color("#F59E0B"), // Amber
		CritColor:     lipgloss.Color("#EF4444"), // Red
		EmptyColor:    lipgloss.Color("#374151"), // Dark gray
		ShowZones:     true,
	}
}

// ProgressBar renders an enhanced progress bar with threshold zones
func ProgressBar(percent float64, config ProgressBarConfig) string {
	if config.Width <= 0 {
		config.Width = 20
	}

	// Clamp percent
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent / 100.0 * float64(config.Width))
	if filled > config.Width {
		filled = config.Width
	}

	// Calculate zone boundaries (positions in the bar)
	warnPos := int(config.WarnThreshold / 100.0 * float64(config.Width))
	critPos := int(config.CritThreshold / 100.0 * float64(config.Width))

	var bar strings.Builder
	bar.WriteString("[")

	for i := 0; i < config.Width; i++ {
		var char string
		var color lipgloss.Color

		if i < filled {
			char = "█"
			// Color based on which zone this position is in
			if i >= critPos {
				color = config.CritColor
			} else if i >= warnPos {
				color = config.WarnColor
			} else {
				color = config.OKColor
			}
		} else {
			// Empty portion
			if config.ShowZones && (i == warnPos || i == critPos) {
				char = "│"
				color = config.EmptyColor
			} else {
				char = "░"
				color = config.EmptyColor
			}
		}

		bar.WriteString(lipgloss.NewStyle().Foreground(color).Render(char))
	}

	bar.WriteString("]")
	return bar.String()
}

// ProgressBarWithLabel renders progress bar with percentage and status icon
func ProgressBarWithLabel(percent float64, config ProgressBarConfig, showPercent bool) string {
	bar := ProgressBar(percent, config)

	if !showPercent {
		return bar
	}

	// Determine status color and icon
	var statusColor lipgloss.Color
	var statusIcon string

	if percent >= config.CritThreshold {
		statusColor = config.CritColor
		statusIcon = "✗"
	} else if percent >= config.WarnThreshold {
		statusColor = config.WarnColor
		statusIcon = "⚠"
	} else {
		statusColor = config.OKColor
		statusIcon = "✓"
	}

	percentStr := fmt.Sprintf("%3.0f%%", percent)
	styledPercent := lipgloss.NewStyle().Foreground(statusColor).Render(percentStr)
	styledIcon := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon)

	return fmt.Sprintf("%s %s %s", bar, styledPercent, styledIcon)
}

// SimpleProgressBar renders a basic colored bar without zones
func SimpleProgressBar(percent float64, width int, filledColor, emptyColor lipgloss.Color) string {
	if width <= 0 {
		width = 20
	}

	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent / 100.0 * float64(width))

	var bar strings.Builder
	bar.WriteString("[")

	filledStyle := lipgloss.NewStyle().Foreground(filledColor)
	emptyStyle := lipgloss.NewStyle().Foreground(emptyColor)

	for i := 0; i < width; i++ {
		if i < filled {
			bar.WriteString(filledStyle.Render("█"))
		} else {
			bar.WriteString(emptyStyle.Render("░"))
		}
	}

	bar.WriteString("]")
	return bar.String()
}

// CompactProgressBar renders a minimal progress bar for tight spaces
func CompactProgressBar(percent float64, width int, color lipgloss.Color) string {
	if width <= 0 {
		width = 10
	}

	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent / 100.0 * float64(width))
	empty := width - filled

	filledStr := strings.Repeat("▓", filled)
	emptyStr := strings.Repeat("░", empty)

	return lipgloss.NewStyle().Foreground(color).Render(filledStr) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#374151")).Render(emptyStr)
}
