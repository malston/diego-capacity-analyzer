// ABOUTME: Test to verify header/footer width alignment
// ABOUTME: Ensures frame renders at correct terminal width

package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
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

			for _, line := range lines {
				w := lipgloss.Width(line)

				if strings.HasPrefix(line, "╭") {
					headerFound = true
					if w != targetWidth {
						t.Errorf("Header width mismatch at width %d: expected %d, got %d", targetWidth, targetWidth, w)
						t.Logf("Header line: %q", line)
					}
				}

				if strings.HasPrefix(line, "╰") {
					footerFound = true
					if w != targetWidth {
						t.Errorf("Footer width mismatch at width %d: expected %d, got %d", targetWidth, targetWidth, w)
						t.Logf("Footer line: %q", line)
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
