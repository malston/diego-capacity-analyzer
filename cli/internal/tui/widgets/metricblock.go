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
	titleWidth := lipgloss.Width(titleStr)
	if titleWidth > innerWidth {
		titleStr = titleStr[:innerWidth]
		titleWidth = innerWidth
	}

	// Style for title line in border
	titleStyle := lipgloss.NewStyle().Foreground(config.TitleColor)
	styledTitle := titleStyle.Render(titleStr)

	// Build the box manually for title-in-border effect
	fillWidth := max(0, innerWidth-titleWidth-1)
	topBorder := "┌─ " + styledTitle + " " + strings.Repeat("─", fillWidth) + "┐"

	// Value line - calculate padding manually for styled content
	valueStyle := lipgloss.NewStyle().Foreground(config.ValueColor).Bold(true)
	styledValue := valueStyle.Render(value)
	valueWidth := lipgloss.Width(styledValue)
	valuePadding := max(0, innerWidth-valueWidth)
	valueLine := "│  " + styledValue + strings.Repeat(" ", valuePadding) + "│"

	// Subtitle line
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	styledSubtitle := subtitleStyle.Render(subtitle)
	subtitleWidth := lipgloss.Width(styledSubtitle)
	subtitlePadding := max(0, innerWidth-subtitleWidth)
	subtitleLine := "│  " + styledSubtitle + strings.Repeat(" ", subtitlePadding) + "│"

	bottomBorder := "└" + strings.Repeat("─", config.Width-2) + "┘"

	// Apply border color to entire block
	borderStyle := lipgloss.NewStyle().Foreground(config.BorderColor)

	return borderStyle.Render(strings.Join([]string{
		topBorder,
		valueLine,
		subtitleLine,
		bottomBorder,
	}, "\n"))
}

// MetricBlockWithBar renders a metric block with a progress bar
func MetricBlockWithBar(icon icons.Icon, title string, percent float64, details string, config MetricBlockConfig) string {
	if config.Width <= 0 {
		config.Width = 22
	}

	innerWidth := config.Width - 4
	barWidth := innerWidth - 4 // Leave room for bracket spacing

	// Title with icon
	titleStr := fmt.Sprintf("%s %s", icon.String(), title)
	titleWidth := lipgloss.Width(titleStr)

	titleStyle := lipgloss.NewStyle().Foreground(config.TitleColor)
	styledTitle := titleStyle.Render(titleStr)

	// Top border with title
	fillWidth := max(0, innerWidth-titleWidth-1)
	topBorder := "┌─ " + styledTitle + " " + strings.Repeat("─", fillWidth) + "┐"

	// Value line with percentage
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
	styledPercent := lipgloss.NewStyle().Bold(true).Foreground(statusColor).Render(percentStr)
	styledIcon := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon)
	valueContent := styledPercent + " " + styledIcon
	valueContentWidth := lipgloss.Width(valueContent)
	valuePadding := max(0, innerWidth-valueContentWidth)
	valueLine := "│  " + valueContent + strings.Repeat(" ", valuePadding) + "│"

	// Progress bar line
	bar := CompactProgressBar(percent, barWidth, statusColor)
	barContentWidth := lipgloss.Width(bar)
	barPadding := max(0, innerWidth-barContentWidth)
	barLine := "│  " + bar + strings.Repeat(" ", barPadding) + "│"

	// Details line
	detailStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	truncatedDetails := truncate(details, innerWidth)
	styledDetails := detailStyle.Render(truncatedDetails)
	detailsWidth := lipgloss.Width(styledDetails)
	detailsPadding := max(0, innerWidth-detailsWidth)
	detailsLine := "│  " + styledDetails + strings.Repeat(" ", detailsPadding) + "│"

	bottomBorder := "└" + strings.Repeat("─", config.Width-2) + "┘"

	borderStyle := lipgloss.NewStyle().Foreground(config.BorderColor)

	return borderStyle.Render(strings.Join([]string{
		topBorder,
		valueLine,
		barLine,
		detailsLine,
		bottomBorder,
	}, "\n"))
}

// MetricBlockWithSparkline renders a metric block with a sparkline
func MetricBlockWithSparkline(icon icons.Icon, title string, value string, sparkData []float64, subtitle string, config MetricBlockConfig) string {
	if config.Width <= 0 {
		config.Width = 22
	}

	innerWidth := config.Width - 4
	sparkWidth := 8

	titleStr := fmt.Sprintf("%s %s", icon.String(), title)
	titleWidth := lipgloss.Width(titleStr)
	titleStyle := lipgloss.NewStyle().Foreground(config.TitleColor)
	styledTitle := titleStyle.Render(titleStr)

	fillWidth := max(0, innerWidth-titleWidth-1)
	topBorder := "┌─ " + styledTitle + " " + strings.Repeat("─", fillWidth) + "┐"

	// Value + sparkline
	valueStyle := lipgloss.NewStyle().Foreground(config.ValueColor).Bold(true)
	styledValue := valueStyle.Render(value)
	spark := Sparkline(sparkData, sparkWidth, lipgloss.Color("#7C3AED"))

	valueWithSpark := styledValue + "  " + spark
	contentWidth := lipgloss.Width(valueWithSpark)
	padding := max(0, innerWidth-contentWidth)
	valueLine := "│  " + valueWithSpark + strings.Repeat(" ", padding) + "│"

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	truncatedSubtitle := truncate(subtitle, innerWidth)
	styledSubtitle := subtitleStyle.Render(truncatedSubtitle)
	subtitleWidth := lipgloss.Width(styledSubtitle)
	subtitlePadding := max(0, innerWidth-subtitleWidth)
	subtitleLine := "│  " + styledSubtitle + strings.Repeat(" ", subtitlePadding) + "│"

	bottomBorder := "└" + strings.Repeat("─", config.Width-2) + "┘"

	borderStyle := lipgloss.NewStyle().Foreground(config.BorderColor)

	return borderStyle.Render(strings.Join([]string{
		topBorder,
		valueLine,
		subtitleLine,
		bottomBorder,
	}, "\n"))
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
