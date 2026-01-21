// ABOUTME: Test to verify header/footer width alignment
// ABOUTME: Ensures frame renders at correct terminal width

package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestFrameAlignment(t *testing.T) {
	widths := []int{80, 100, 120}

	for _, targetWidth := range widths {
		t.Run(strings.ReplaceAll(string(rune(targetWidth)), "", ""), func(t *testing.T) {
			app := New(nil, false, "")

			// Simulate window size message
			model, _ := app.Update(tea.WindowSizeMsg{Width: targetWidth, Height: 30})
			app = model.(*App)

			// Get the view
			view := app.View()

			lines := strings.Split(view, "\n")
			headerFound := false
			footerFound := false

			// Frame uses width-1 to prevent wrapping on some terminals,
			// but clamps to minimum of 80 for usability
			expectedWidth := targetWidth - 1
			if expectedWidth < 80 {
				expectedWidth = 80
			}

			for _, line := range lines {
				// Header starts with ╭ at the beginning of the line
				if strings.HasPrefix(line, "╭") {
					headerFound = true
					w := lipgloss.Width(line)
					if w != expectedWidth {
						t.Errorf("Header width mismatch at width %d: expected %d, got %d", targetWidth, expectedWidth, w)
						t.Logf("Header line: %q", line)
					}
				}

				// Footer contains ╰ (may have leading spaces from content centering)
				if strings.Contains(line, "╰") {
					footerFound = true
					// Extract the footer portion starting from ╰
					footerIdx := strings.Index(line, "╰")
					footerLine := line[footerIdx:]
					w := lipgloss.Width(footerLine)
					if w != expectedWidth {
						t.Errorf("Footer width mismatch at width %d: expected %d, got %d", targetWidth, expectedWidth, w)
						t.Logf("Footer line: %q", footerLine)
					}
				}
			}

			if !headerFound {
				t.Error("Header not found in output")
			}
			if !footerFound {
				t.Error("Footer not found in output")
			}
		})
	}
}
