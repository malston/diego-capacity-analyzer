// ABOUTME: Compact metric block widget for dashboard displays
// ABOUTME: Combines icon, value, sparkline, and status in a bordered panel

package widgets

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/icons"
)

// MetricBlockConfig holds configuration for a metric block
type MetricBlockConfig struct {
	Width       int
	BorderColor lipgloss.Color
	TitleColor  lipgloss.Color
	ValueColor  lipgloss.Color
}

// DefaultMetricBlockConfig returns sensible defaults
func DefaultMetricBlockConfig() MetricBlockConfig {
	return MetricBlockConfig{
		Width:       22,
		BorderColor: lipgloss.Color("#6B7280"), // Muted gray
		TitleColor:  lipgloss.Color("#7C3AED"), // Purple
		ValueColor:  lipgloss.Color("#F9FAFB"), // Light
	}
}

// MetricBlock renders a compact metric display block
func MetricBlock(icon icons.Icon, title string, value string, subtitle string, config MetricBlockConfig) string {
	if config.Width <= 0 {
		config.Width = 22
	}

	// Calculate inner width (accounting for border + padding)
	innerWidth := config.Width - 4

	// Title with icon
	titleStr := fmt.Sprintf("%s %s", icon.String(), title)
	if len(titleStr) > innerWidth {
		titleStr = titleStr[:innerWidth]
	}

	// Style for title line in border
	titleStyle := lipgloss.NewStyle().Foreground(config.TitleColor)

	// Build the box manually for title-in-border effect
	topBorder := fmt.Sprintf("┌─ %s %s┐",
		titleStyle.Render(titleStr),
		strings.Repeat("─", max(0, innerWidth-len(titleStr)-1)))

	// Value line (centered or left-aligned based on length)
	valueStyle := lipgloss.NewStyle().Foreground(config.ValueColor).Bold(true)
	valueLine := fmt.Sprintf("│  %-*s│", innerWidth, valueStyle.Render(value))

	// Subtitle line
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	subtitleLine := fmt.Sprintf("│  %-*s│", innerWidth, subtitleStyle.Render(subtitle))

	bottomBorder := fmt.Sprintf("└%s┘", strings.Repeat("─", config.Width-2))

	// Apply border color
	borderStyle := lipgloss.NewStyle().Foreground(config.BorderColor)

	return strings.Join([]string{
		borderStyle.Render(topBorder),
		borderStyle.Render(valueLine),
		borderStyle.Render(subtitleLine),
		borderStyle.Render(bottomBorder),
	}, "\n")
}

// MetricBlockWithBar renders a metric block with a progress bar
func MetricBlockWithBar(icon icons.Icon, title string, percent float64, details string, config MetricBlockConfig) string {
	if config.Width <= 0 {
		config.Width = 22
	}

	innerWidth := config.Width - 4
	barWidth := innerWidth - 6 // Leave room for percentage

	// Title with icon
	titleStr := fmt.Sprintf("%s %s", icon.String(), title)

	titleStyle := lipgloss.NewStyle().Foreground(config.TitleColor)

	// Top border with title
	topBorder := fmt.Sprintf("┌─ %s %s┐",
		titleStyle.Render(titleStr),
		strings.Repeat("─", max(0, innerWidth-len(titleStr)-1)))

	// Value line with percentage
	valueStyle := lipgloss.NewStyle().Bold(true)
	var statusColor lipgloss.Color
	var statusIcon string

	if percent >= 95 {
		statusColor = lipgloss.Color("#EF4444")
		statusIcon = "✗"
	} else if percent >= 80 {
		statusColor = lipgloss.Color("#F59E0B")
		statusIcon = "⚠"
	} else {
		statusColor = lipgloss.Color("#10B981")
		statusIcon = "✓"
	}

	percentStr := fmt.Sprintf("%3.0f%%", percent)
	valueLine := fmt.Sprintf("│  %s %s %s│",
		valueStyle.Foreground(statusColor).Render(percentStr),
		lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon),
		strings.Repeat(" ", max(0, innerWidth-7)))

	// Progress bar line
	barConfig := DefaultProgressBarConfig()
	barConfig.Width = barWidth
	bar := CompactProgressBar(percent, barWidth, statusColor)
	barLine := fmt.Sprintf("│  %s│", bar+strings.Repeat(" ", max(0, innerWidth-barWidth)))

	// Details line
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	detailsLine := fmt.Sprintf("│  %-*s│", innerWidth, detailStyle.Render(truncate(details, innerWidth)))

	bottomBorder := fmt.Sprintf("└%s┘", strings.Repeat("─", config.Width-2))

	borderStyle := lipgloss.NewStyle().Foreground(config.BorderColor)

	return strings.Join([]string{
		borderStyle.Render(topBorder),
		borderStyle.Render(valueLine),
		borderStyle.Render(barLine),
		borderStyle.Render(detailsLine),
		borderStyle.Render(bottomBorder),
	}, "\n")
}

// MetricBlockWithSparkline renders a metric block with a sparkline
func MetricBlockWithSparkline(icon icons.Icon, title string, value string, sparkData []float64, subtitle string, config MetricBlockConfig) string {
	if config.Width <= 0 {
		config.Width = 22
	}

	innerWidth := config.Width - 4
	sparkWidth := 8

	titleStr := fmt.Sprintf("%s %s", icon.String(), title)
	titleStyle := lipgloss.NewStyle().Foreground(config.TitleColor)

	topBorder := fmt.Sprintf("┌─ %s %s┐",
		titleStyle.Render(titleStr),
		strings.Repeat("─", max(0, innerWidth-len(titleStr)-1)))

	// Value + sparkline
	valueStyle := lipgloss.NewStyle().Foreground(config.ValueColor).Bold(true)
	spark := Sparkline(sparkData, sparkWidth, lipgloss.Color("#7C3AED"))

	valueWithSpark := fmt.Sprintf("%s  %s", valueStyle.Render(value), spark)
	// Calculate display width (value + 2 spaces + sparkline)
	displayWidth := len(value) + 2 + sparkWidth
	padding := max(0, innerWidth-displayWidth)
	valueLine := fmt.Sprintf("│  %s%s│", valueWithSpark, strings.Repeat(" ", padding))

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	subtitleLine := fmt.Sprintf("│  %-*s│", innerWidth, subtitleStyle.Render(truncate(subtitle, innerWidth)))

	bottomBorder := fmt.Sprintf("└%s┘", strings.Repeat("─", config.Width-2))

	borderStyle := lipgloss.NewStyle().Foreground(config.BorderColor)

	return strings.Join([]string{
		borderStyle.Render(topBorder),
		borderStyle.Render(valueLine),
		borderStyle.Render(subtitleLine),
		borderStyle.Render(bottomBorder),
	}, "\n")
}

// CountBlock renders a simple count metric (like cluster/host counts)
func CountBlock(icon icons.Icon, title string, count int, label string, config MetricBlockConfig) string {
	value := fmt.Sprintf("%d", count)
	return MetricBlock(icon, title, value, label, config)
}

// truncate shortens a string to maxLen with ellipsis if needed
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
