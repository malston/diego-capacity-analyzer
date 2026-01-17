// ABOUTME: Comparison view showing current vs proposed scenario results
// ABOUTME: Uses visual panels, delta badges, and styled warnings

package comparison

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/icons"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/styles"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/widgets"
)

// Comparison displays scenario comparison results
type Comparison struct {
	result *client.ScenarioComparison
	width  int
}

// New creates a new comparison view
func New(result *client.ScenarioComparison, width int) *Comparison {
	return &Comparison{
		result: result,
		width:  width,
	}
}

// View renders the comparison
func (c *Comparison) View() string {
	if c.result == nil {
		return "No comparison data"
	}

	var sb strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Primary)
	sb.WriteString(titleStyle.Render(fmt.Sprintf("%s Scenario Comparison", icons.Chart.String())))
	sb.WriteString("\n\n")

	// Account for outer ActivePanel borders/padding (about 6 chars)
	contentWidth := c.width - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	// For side-by-side panels: each panel = (contentWidth - 2) / 2
	colWidth := (contentWidth - 2) / 2

	// If columns too narrow, stack vertically instead
	if colWidth < 25 {
		// Stack vertically
		currentPanel := c.renderScenarioPanel("Current", icons.Server, &c.result.Current, contentWidth-2)
		proposedPanel := c.renderScenarioPanel("Proposed", icons.TrendUp, &c.result.Proposed, contentWidth-2)
		sb.WriteString(currentPanel)
		sb.WriteString("\n")
		sb.WriteString(proposedPanel)
	} else {
		// Side by side
		currentPanel := c.renderScenarioPanel("Current", icons.Server, &c.result.Current, colWidth)
		proposedPanel := c.renderScenarioPanel("Proposed", icons.TrendUp, &c.result.Proposed, colWidth)
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, currentPanel, "  ", proposedPanel))
	}
	sb.WriteString("\n\n")

	// Impact summary panel - use full content width
	impactPanel := c.renderImpactPanel(contentWidth - 2)
	sb.WriteString(impactPanel)
	sb.WriteString("\n\n")

	// Warnings panel
	if len(c.result.Warnings) > 0 {
		warningsPanel := c.renderWarningsPanel(contentWidth - 2)
		sb.WriteString(warningsPanel)
	}

	return lipgloss.NewStyle().Width(c.width).Render(sb.String())
}

func (c *Comparison) renderScenarioPanel(title string, icon icons.Icon, s *client.ScenarioResult, width int) string {
	var sb strings.Builder

	// Cell info
	sb.WriteString(fmt.Sprintf("Cells:     %d\n", s.CellCount))
	sb.WriteString(fmt.Sprintf("Memory:    %d GB each\n", s.CellMemoryGB))
	sb.WriteString(fmt.Sprintf("Total:     %d GB\n", s.AppCapacityGB))
	sb.WriteString("\n")

	// Utilization with progress bar
	// Content width inside panel = width - 4 (for borders) - 2 (for "│ " and " │" padding)
	// Bar width = content width - label width (about 10 chars for " 100% ✓")
	contentWidth := width - 6
	barWidth := contentWidth - 10
	if barWidth < 10 {
		barWidth = 10 // minimum bar width
	}

	barConfig := widgets.DefaultProgressBarConfig()
	barConfig.Width = barWidth
	barConfig.ShowZones = false // Disable zones for compact display
	bar := widgets.ProgressBarWithLabel(s.UtilizationPct, barConfig, true)
	sb.WriteString(fmt.Sprintf("Utilization\n%s", bar))

	// vCPU ratio if available
	if s.VCPURatio > 0 {
		sb.WriteString(fmt.Sprintf("\nvCPU:      %.1f:1", s.VCPURatio))
	}

	return c.buildPanel(title, icon, sb.String(), width)
}

func (c *Comparison) renderImpactPanel(width int) string {
	var sb strings.Builder
	delta := c.result.Delta

	// Capacity change
	capacityChange := delta.CapacityChangeGB
	// Calculate percentage change from current capacity
	var capacityPct float64
	if c.result.Current.AppCapacityGB > 0 {
		capacityPct = (float64(capacityChange) / float64(c.result.Current.AppCapacityGB)) * 100
	}

	var capacityColor lipgloss.Color
	var capacityPrefix string
	if capacityChange > 0 {
		capacityColor = styles.DeltaPositive
		capacityPrefix = "+"
	} else if capacityChange < 0 {
		capacityColor = styles.DeltaNegative
		capacityPrefix = ""
	} else {
		capacityColor = styles.DeltaNeutral
		capacityPrefix = ""
	}

	capacityStyle := lipgloss.NewStyle().Foreground(capacityColor).Bold(true)
	sb.WriteString(fmt.Sprintf("Capacity:      %s\n",
		capacityStyle.Render(fmt.Sprintf("%s%d GB (%s%.0f%%)", capacityPrefix, capacityChange, capacityPrefix, capacityPct))))

	// Utilization change (inverted - decrease is good)
	utilChange := delta.UtilizationChangePct
	var utilColor lipgloss.Color
	var utilArrow string
	if utilChange < 0 {
		utilColor = styles.DeltaPositive // Decrease is good
		utilArrow = icons.TrendDown.String()
	} else if utilChange > 0 {
		utilColor = styles.DeltaNegative // Increase is concerning
		utilArrow = icons.TrendUp.String()
	} else {
		utilColor = styles.DeltaNeutral
		utilArrow = "→"
	}

	utilStyle := lipgloss.NewStyle().Foreground(utilColor).Bold(true)
	sb.WriteString(fmt.Sprintf("Utilization:   %.1f%% → %.1f%%  %s %s\n",
		c.result.Current.UtilizationPct,
		c.result.Proposed.UtilizationPct,
		utilStyle.Render(fmt.Sprintf("(%+.1f%%)", utilChange)),
		lipgloss.NewStyle().Foreground(utilColor).Render(utilArrow)))

	// Headroom change
	currentHeadroom := 100 - c.result.Current.UtilizationPct
	proposedHeadroom := 100 - c.result.Proposed.UtilizationPct
	headroomChange := proposedHeadroom - currentHeadroom

	var headroomColor lipgloss.Color
	if headroomChange > 0 {
		headroomColor = styles.DeltaPositive
	} else if headroomChange < 0 {
		headroomColor = styles.DeltaNegative
	} else {
		headroomColor = styles.DeltaNeutral
	}

	headroomStyle := lipgloss.NewStyle().Foreground(headroomColor).Bold(true)
	sb.WriteString(fmt.Sprintf("Headroom:      %s",
		headroomStyle.Render(fmt.Sprintf("%+.1f%% available", headroomChange))))

	return c.buildPanel("Impact Summary", icons.TrendUp, sb.String(), width)
}

func (c *Comparison) renderWarningsPanel(width int) string {
	var sb strings.Builder

	// Calculate available text width inside the panel (minus borders and padding)
	textWidth := width - 8
	if textWidth < 20 {
		textWidth = 20
	}

	for i, w := range c.result.Warnings {
		var status widgets.StatusLevel
		if w.Severity == "critical" {
			status = widgets.StatusCritical
		} else {
			status = widgets.StatusWarning
		}

		// Word-wrap long messages to fit within panel
		message := w.Message
		wrappedLines := wrapText(message, textWidth)
		for j, line := range wrappedLines {
			if j == 0 {
				sb.WriteString(widgets.StatusText(line, status))
			} else {
				// Continuation lines without the status icon
				sb.WriteString("\n  " + line)
			}
		}
		if i < len(c.result.Warnings)-1 {
			sb.WriteString("\n")
		}
	}

	return c.buildPanel("Warnings", icons.Warning, sb.String(), width)
}

// wrapText wraps text to fit within the specified width
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	lines = append(lines, currentLine)

	return lines
}

func (c *Comparison) buildPanel(title string, icon icons.Icon, content string, width int) string {
	borderStyle := lipgloss.NewStyle().Foreground(styles.Muted)
	titleStyle := lipgloss.NewStyle().Foreground(styles.Primary)

	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10
	}

	fullTitle := fmt.Sprintf("%s %s", icon.String(), title)
	titleWidth := lipgloss.Width(fullTitle)
	styledTitle := titleStyle.Render(fullTitle)

	// Top border with title - use lipgloss.Width for accurate width
	fillWidth := max(0, innerWidth-titleWidth-1)
	topBorder := "┌─ " + styledTitle + " " + strings.Repeat("─", fillWidth) + "┐"

	// Content lines with side borders
	lines := strings.Split(content, "\n")
	var contentLines []string
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth > innerWidth {
			// Truncate line to fit - use lipgloss to handle ANSI codes
			line = lipgloss.NewStyle().MaxWidth(innerWidth).Render(line)
			lineWidth = lipgloss.Width(line)
		}
		padding := max(0, innerWidth-lineWidth)
		contentLines = append(contentLines, "│ "+line+strings.Repeat(" ", padding)+" │")
	}

	// Bottom border
	bottomBorder := "└" + strings.Repeat("─", innerWidth+2) + "┘"

	allLines := []string{topBorder}
	allLines = append(allLines, contentLines...)
	allLines = append(allLines, bottomBorder)

	return borderStyle.Render(strings.Join(allLines, "\n"))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
