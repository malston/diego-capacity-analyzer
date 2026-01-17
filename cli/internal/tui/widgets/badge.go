// ABOUTME: Status badge widgets for quick visual status indication
// ABOUTME: Provides colored inline badges and status indicators

package widgets

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/icons"
)

// StatusLevel represents the severity of a status
type StatusLevel int

const (
	StatusOK StatusLevel = iota
	StatusWarning
	StatusCritical
	StatusInfo
	StatusNeutral
)

// Badge colors
var (
	BadgeOKBg       = lipgloss.Color("#10B981")
	BadgeOKFg       = lipgloss.Color("#FFFFFF")
	BadgeWarnBg     = lipgloss.Color("#F59E0B")
	BadgeWarnFg     = lipgloss.Color("#000000")
	BadgeCritBg     = lipgloss.Color("#EF4444")
	BadgeCritFg     = lipgloss.Color("#FFFFFF")
	BadgeInfoBg     = lipgloss.Color("#3B82F6")
	BadgeInfoFg     = lipgloss.Color("#FFFFFF")
	BadgeNeutralBg  = lipgloss.Color("#6B7280")
	BadgeNeutralFg  = lipgloss.Color("#FFFFFF")
)

// Badge renders a colored status badge
func Badge(text string, level StatusLevel) string {
	var bg, fg lipgloss.Color

	switch level {
	case StatusOK:
		bg, fg = BadgeOKBg, BadgeOKFg
	case StatusWarning:
		bg, fg = BadgeWarnBg, BadgeWarnFg
	case StatusCritical:
		bg, fg = BadgeCritBg, BadgeCritFg
	case StatusInfo:
		bg, fg = BadgeInfoBg, BadgeInfoFg
	default:
		bg, fg = BadgeNeutralBg, BadgeNeutralFg
	}

	style := lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Padding(0, 1).
		Bold(true)

	return style.Render(text)
}

// StatusBadge renders a predefined status badge (OK, WARN, CRIT)
func StatusBadge(level StatusLevel) string {
	switch level {
	case StatusOK:
		return Badge("OK", StatusOK)
	case StatusWarning:
		return Badge("WARN", StatusWarning)
	case StatusCritical:
		return Badge("CRIT", StatusCritical)
	case StatusInfo:
		return Badge("INFO", StatusInfo)
	default:
		return Badge("--", StatusNeutral)
	}
}

// StatusFromPercent returns the appropriate status level for a percentage value
func StatusFromPercent(percent, warnThreshold, critThreshold float64) StatusLevel {
	if percent >= critThreshold {
		return StatusCritical
	}
	if percent >= warnThreshold {
		return StatusWarning
	}
	return StatusOK
}

// StatusIcon returns the appropriate icon for a status level
func StatusIcon(level StatusLevel) string {
	switch level {
	case StatusOK:
		return lipgloss.NewStyle().Foreground(BadgeOKBg).Render(icons.CheckOK.String())
	case StatusWarning:
		return lipgloss.NewStyle().Foreground(BadgeWarnBg).Render(icons.Warning.String())
	case StatusCritical:
		return lipgloss.NewStyle().Foreground(BadgeCritBg).Render(icons.Critical.String())
	case StatusInfo:
		return lipgloss.NewStyle().Foreground(BadgeInfoBg).Render(icons.Info.String())
	default:
		return lipgloss.NewStyle().Foreground(BadgeNeutralBg).Render("•")
	}
}

// StatusText returns styled status text with icon
func StatusText(text string, level StatusLevel) string {
	icon := StatusIcon(level)

	var color lipgloss.Color
	switch level {
	case StatusOK:
		color = BadgeOKBg
	case StatusWarning:
		color = BadgeWarnBg
	case StatusCritical:
		color = BadgeCritBg
	case StatusInfo:
		color = BadgeInfoBg
	default:
		color = BadgeNeutralBg
	}

	textStyle := lipgloss.NewStyle().Foreground(color)
	return fmt.Sprintf("%s %s", icon, textStyle.Render(text))
}

// DeltaBadge renders a change indicator with color
func DeltaBadge(delta float64, unit string, invertColors bool) string {
	var text string
	var level StatusLevel

	if delta > 0 {
		text = fmt.Sprintf("+%.0f%s", delta, unit)
		if invertColors {
			level = StatusWarning // Positive is bad (e.g., utilization increase)
		} else {
			level = StatusOK // Positive is good (e.g., capacity increase)
		}
	} else if delta < 0 {
		text = fmt.Sprintf("%.0f%s", delta, unit)
		if invertColors {
			level = StatusOK // Negative is good (e.g., utilization decrease)
		} else {
			level = StatusWarning // Negative is bad (e.g., capacity decrease)
		}
	} else {
		text = fmt.Sprintf("%.0f%s", delta, unit)
		level = StatusNeutral
	}

	return Badge(text, level)
}

// TrendIndicator returns an arrow icon for trend direction
func TrendIndicator(current, previous float64) string {
	if current > previous {
		return lipgloss.NewStyle().Foreground(BadgeWarnBg).Render(icons.TrendUp.String())
	} else if current < previous {
		return lipgloss.NewStyle().Foreground(BadgeOKBg).Render(icons.TrendDown.String())
	}
	return lipgloss.NewStyle().Foreground(BadgeNeutralBg).Render("→")
}

// RiskBadge renders a risk level badge for CPU ratios
func RiskBadge(ratio float64) string {
	if ratio <= 2.0 {
		return Badge("Conservative", StatusOK)
	} else if ratio <= 4.0 {
		return Badge("Moderate", StatusWarning)
	}
	return Badge("Aggressive", StatusCritical)
}

// RiskLevel returns the risk description for a CPU ratio
func RiskLevel(ratio float64) (string, StatusLevel) {
	if ratio <= 2.0 {
		return "Conservative", StatusOK
	} else if ratio <= 4.0 {
		return "Moderate", StatusWarning
	}
	return "Aggressive", StatusCritical
}
