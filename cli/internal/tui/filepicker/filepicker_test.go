// ABOUTME: Tests for file picker TUI component
// ABOUTME: Validates navigation, selection, and state transitions

package filepicker

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/markalston/diego-capacity-analyzer/cli/internal/tui/samples"
)

func TestNew(t *testing.T) {
	fp := New([]string{"/path/to/file.json"}, nil)

	if fp == nil {
		t.Fatal("New() returned nil")
	}
	if fp.state != stateList {
		t.Errorf("expected initial state stateList, got %d", fp.state)
	}
}

func TestNewWithNoRecentFiles(t *testing.T) {
	fp := New(nil, nil)

	if len(fp.recentFiles) != 0 {
		t.Errorf("expected empty recent files, got %d", len(fp.recentFiles))
	}
}

func TestNewWithSamples(t *testing.T) {
	samples := []samples.SampleFile{
		{Name: "sample1.json", Path: "/samples/sample1.json"},
	}
	fp := New(nil, samples)

	if !fp.hasSamples {
		t.Error("expected hasSamples to be true")
	}
}

func TestViewContainsRecentFiles(t *testing.T) {
	fp := New([]string{"/path/to/recent.json"}, nil)
	fp.width = 80
	fp.height = 24

	view := fp.View()

	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestNavigateDown(t *testing.T) {
	fp := New([]string{"/path/to/file1.json", "/path/to/file2.json"}, nil)
	fp.width = 80
	fp.height = 24

	initialCursor := fp.cursor

	// Send down key
	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, _ := fp.Update(msg)
	updated := model.(*FilePicker)

	if updated.cursor != initialCursor+1 {
		t.Errorf("expected cursor to move down, got %d", updated.cursor)
	}
}

func TestNavigateUp(t *testing.T) {
	fp := New([]string{"/path/to/file1.json", "/path/to/file2.json"}, nil)
	fp.width = 80
	fp.height = 24
	fp.cursor = 1

	// Send up key
	msg := tea.KeyMsg{Type: tea.KeyUp}
	model, _ := fp.Update(msg)
	updated := model.(*FilePicker)

	if updated.cursor != 0 {
		t.Errorf("expected cursor to move up to 0, got %d", updated.cursor)
	}
}

func TestSelectRecentFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	os.WriteFile(testFile, []byte(`{"test": true}`), 0644)

	fp := New([]string{testFile}, nil)
	fp.width = 80
	fp.height = 24
	fp.cursor = 0 // Select first recent file

	// Send enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := fp.Update(msg)

	if cmd == nil {
		t.Fatal("expected command to be returned")
	}

	// Execute the command to get the message
	resultMsg := cmd()
	if resultMsg == nil {
		t.Fatal("command returned nil message")
	}

	selected, ok := resultMsg.(FileSelectedMsg)
	if !ok {
		t.Fatalf("expected FileSelectedMsg, got %T", resultMsg)
	}

	if selected.Path != testFile {
		t.Errorf("expected path %s, got %s", testFile, selected.Path)
	}
}

func TestSelectEnterPath(t *testing.T) {
	fp := New([]string{"/path/to/file.json"}, nil)
	fp.width = 80
	fp.height = 24
	// Move cursor to "Enter path..." option
	fp.cursor = 1 // After recent files

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	model, _ := fp.Update(msg)
	updated := model.(*FilePicker)

	if updated.state != stateInput {
		t.Errorf("expected state stateInput, got %d", updated.state)
	}
}

func TestBackFromInputReturnsToList(t *testing.T) {
	fp := New(nil, nil)
	fp.width = 80
	fp.height = 24
	fp.state = stateInput

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	model, _ := fp.Update(msg)
	updated := model.(*FilePicker)

	if updated.state != stateList {
		t.Errorf("expected state stateList after Esc, got %d", updated.state)
	}
}

func TestBackFromListReturnsCancelMsg(t *testing.T) {
	fp := New(nil, nil)
	fp.width = 80
	fp.height = 24
	fp.state = stateList

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := fp.Update(msg)

	if cmd == nil {
		t.Fatal("expected command for cancel")
	}

	resultMsg := cmd()
	if _, ok := resultMsg.(CancelledMsg); !ok {
		t.Errorf("expected CancelledMsg, got %T", resultMsg)
	}
}

func TestErrorState(t *testing.T) {
	fp := New(nil, nil)
	fp.width = 80
	fp.height = 24
	fp.SetError("File not found")

	if fp.err != "File not found" {
		t.Errorf("expected error message, got %s", fp.err)
	}

	view := fp.View()
	if view == "" {
		t.Error("View() should still render with error")
	}
}

func TestWindowSizeUpdate(t *testing.T) {
	fp := New(nil, nil)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	model, _ := fp.Update(msg)
	updated := model.(*FilePicker)

	if updated.width != 100 {
		t.Errorf("expected width 100, got %d", updated.width)
	}
	if updated.height != 50 {
		t.Errorf("expected height 50, got %d", updated.height)
	}
}

func TestViewWithZeroWidth(t *testing.T) {
	// Regression test: View() should not panic when width is 0
	// (before WindowSizeMsg is received)
	fp := New([]string{"/path/to/recent.json"}, nil)
	// Deliberately leave width and height at 0

	// This should not panic
	view := fp.View()
	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/Documents/test.json", home + "/Documents/test.json"},
		{"~", home},
		{"/absolute/path.json", "/absolute/path.json"},
		{"relative/path.json", "relative/path.json"},
		{"./local.json", "./local.json"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := expandPath(tc.input)
			if result != tc.expected {
				t.Errorf("expandPath(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
