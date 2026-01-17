// ABOUTME: Icon system with Nerd Font detection and Unicode fallback
// ABOUTME: Provides consistent iconography across different terminal capabilities

package icons

import (
	"os"
	"strings"
	"sync"
)

var (
	useNerdFonts     bool
	nerdFontDetected sync.Once
)

// detectNerdFonts checks if Nerd Fonts should be used
func detectNerdFonts() bool {
	// Explicit override via environment variable
	if env := os.Getenv("DIEGO_NERD_FONTS"); env != "" {
		return env == "1" || strings.ToLower(env) == "true"
	}

	// Check for terminals known to commonly have Nerd Fonts
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	// iTerm2, Alacritty, WezTerm, Kitty typically have Nerd Fonts
	nerdFontTerminals := []string{
		"iTerm.app",
		"alacritty",
		"WezTerm",
		"kitty",
		"ghostty",
	}

	for _, t := range nerdFontTerminals {
		if strings.Contains(termProgram, t) || strings.Contains(term, strings.ToLower(t)) {
			return true
		}
	}

	// Check for common Nerd Font environment indicators
	if os.Getenv("NERD_FONTS") == "1" {
		return true
	}

	// Default to Unicode fallback for maximum compatibility
	return false
}

// HasNerdFonts returns true if Nerd Fonts are available
func HasNerdFonts() bool {
	nerdFontDetected.Do(func() {
		useNerdFonts = detectNerdFonts()
	})
	return useNerdFonts
}

// Icon represents an icon with Nerd Font and Unicode fallback variants
type Icon struct {
	NerdFont string
	Fallback string
}

// String returns the appropriate icon based on font availability
func (i Icon) String() string {
	if HasNerdFonts() {
		return i.NerdFont
	}
	return i.Fallback
}

// Icon definitions - Nerd Font codepoints with Unicode fallbacks
var (
	// Resource types
	Memory  = Icon{"󰍛", "◆"} // nf-md-memory
	CPU     = Icon{"", "●"} // nf-oct-cpu
	Disk    = Icon{"󰋊", "■"} // nf-md-harddisk
	Server  = Icon{"󰒋", "▣"} // nf-md-server
	Cluster = Icon{"󱃾", "⬡"} // nf-md-hexagon_multiple
	Cell    = Icon{"󰆧", "□"} // nf-md-cube_outline
	Host    = Icon{"󰇄", "▢"} // nf-md-desktop_classic

	// Status indicators
	CheckOK   = Icon{"", "✓"} // nf-oct-check_circle
	Warning   = Icon{"", "⚠"} // nf-oct-alert
	Critical  = Icon{"", "✗"} // nf-oct-x_circle
	Info      = Icon{"", "ℹ"} // nf-oct-info

	// Trends and charts
	TrendUp   = Icon{"󰄬", "↗"} // nf-md-trending_up
	TrendDown = Icon{"󰄰", "↘"} // nf-md-trending_down
	Chart     = Icon{"󰄭", "▁"} // nf-md-chart_line
	Gauge     = Icon{"󰓅", "◐"} // nf-md-gauge

	// Actions
	Refresh = Icon{"󰑓", "↻"} // nf-md-refresh
	Wizard  = Icon{"󰂓", "★"} // nf-md-auto_fix
	Back    = Icon{"󰁍", "←"} // nf-md-arrow_left
	Quit    = Icon{"󰗼", "×"} // nf-md-exit_to_app

	// Application
	App      = Icon{"󰋊", "◈"} // nf-md-harddisk (capacity theme)
	Settings = Icon{"󰒓", "⚙"} // nf-md-cog

	// HA and capacity
	Shield   = Icon{"󰒃", "⛊"} // nf-md-shield_check
	Capacity = Icon{"󰋁", "▮"} // nf-md-database
)
