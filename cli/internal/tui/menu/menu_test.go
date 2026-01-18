// ABOUTME: Tests for data source selection menu
// ABOUTME: Validates menu rendering and selection behavior

package menu

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	m := New(true) // vSphere configured

	if m == nil {
		t.Fatal("New() returned nil")
	}
	if len(m.options) != 3 {
		t.Errorf("expected 3 options, got %d", len(m.options))
	}
}

func TestMenuOptions(t *testing.T) {
	m := New(true) // vSphere configured

	if len(m.options) != 3 {
		t.Errorf("expected 3 options, got %d", len(m.options))
	}

	if m.options[0].label != "Live vSphere" {
		t.Errorf("expected first option 'Live vSphere', got %s", m.options[0].label)
	}
}

func TestMenuVSphereDisabled(t *testing.T) {
	m := New(false) // vSphere not configured

	if m.options[0].enabled {
		t.Error("expected vSphere option to be disabled when not configured")
	}
}

func TestMenuVSphereEnabled(t *testing.T) {
	m := New(true) // vSphere configured

	if !m.options[0].enabled {
		t.Error("expected vSphere option to be enabled when configured")
	}
}

func TestDataSourceString(t *testing.T) {
	tests := []struct {
		source   DataSource
		expected string
	}{
		{SourceVSphere, "vsphere"},
		{SourceJSON, "json"},
		{SourceManual, "manual"},
		{DataSource(99), "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			if got := tc.source.String(); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestMenuDefaultOptions(t *testing.T) {
	m := New(false)

	// JSON should always be enabled
	if !m.options[1].enabled {
		t.Error("expected JSON option to always be enabled")
	}

	// Manual should always be enabled
	if !m.options[2].enabled {
		t.Error("expected Manual option to always be enabled")
	}
}

func TestViewContainsTitle(t *testing.T) {
	m := New(true)
	m.width = 80
	m.height = 24

	view := m.View()

	if view == "" {
		t.Error("View() returned empty string")
	}
}

func TestNavigateDown(t *testing.T) {
	m := New(true)
	m.width = 80
	m.height = 24

	initialCursor := m.cursor

	// Send down key
	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, _ := m.Update(msg)
	updated := model.(*Menu)

	if updated.cursor != initialCursor+1 {
		t.Errorf("expected cursor to move down, got %d", updated.cursor)
	}
}

func TestNavigateUp(t *testing.T) {
	m := New(true)
	m.width = 80
	m.height = 24
	m.cursor = 1

	// Send up key
	msg := tea.KeyMsg{Type: tea.KeyUp}
	model, _ := m.Update(msg)
	updated := model.(*Menu)

	if updated.cursor != 0 {
		t.Errorf("expected cursor to move up to 0, got %d", updated.cursor)
	}
}

func TestSelectEnabledOption(t *testing.T) {
	m := New(true)
	m.width = 80
	m.height = 24
	m.cursor = 1 // Select "Load JSON file"

	// Send enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected command to be returned")
	}

	// Execute the command to get the message
	resultMsg := cmd()
	if resultMsg == nil {
		t.Fatal("command returned nil message")
	}

	selected, ok := resultMsg.(DataSourceSelectedMsg)
	if !ok {
		t.Fatalf("expected DataSourceSelectedMsg, got %T", resultMsg)
	}

	if selected.Source != SourceJSON {
		t.Errorf("expected SourceJSON, got %v", selected.Source)
	}
}

func TestSelectVSphereWhenEnabled(t *testing.T) {
	m := New(true) // vSphere configured
	m.width = 80
	m.height = 24
	m.cursor = 0 // Select "Live vSphere"

	// Send enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected command to be returned")
	}

	resultMsg := cmd()
	selected, ok := resultMsg.(DataSourceSelectedMsg)
	if !ok {
		t.Fatalf("expected DataSourceSelectedMsg, got %T", resultMsg)
	}

	if selected.Source != SourceVSphere {
		t.Errorf("expected SourceVSphere, got %v", selected.Source)
	}
}

func TestSelectVSphereWhenDisabled(t *testing.T) {
	m := New(false) // vSphere NOT configured
	m.width = 80
	m.height = 24
	m.cursor = 0 // Select "Live vSphere"

	// Send enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	model, cmd := m.Update(msg)
	updated := model.(*Menu)

	// Should NOT return a command (disabled option)
	if cmd != nil {
		t.Error("expected no command for disabled option")
	}

	// Should set an error
	if updated.err == "" {
		t.Error("expected error message for disabled option")
	}
}

func TestCancelReturnsMsg(t *testing.T) {
	m := New(true)
	m.width = 80
	m.height = 24

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("expected command for cancel")
	}

	resultMsg := cmd()
	if _, ok := resultMsg.(CancelledMsg); !ok {
		t.Errorf("expected CancelledMsg, got %T", resultMsg)
	}
}

func TestWindowSizeUpdate(t *testing.T) {
	m := New(true)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	model, _ := m.Update(msg)
	updated := model.(*Menu)

	if updated.width != 100 {
		t.Errorf("expected width 100, got %d", updated.width)
	}
	if updated.height != 50 {
		t.Errorf("expected height 50, got %d", updated.height)
	}
}

func TestKeyboardNavigation(t *testing.T) {
	m := New(true)
	m.width = 80
	m.height = 24

	// Test 'j' key moves down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	model, _ := m.Update(msg)
	updated := model.(*Menu)
	if updated.cursor != 1 {
		t.Errorf("expected cursor 1 after 'j', got %d", updated.cursor)
	}

	// Test 'k' key moves up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	model, _ = updated.Update(msg)
	updated = model.(*Menu)
	if updated.cursor != 0 {
		t.Errorf("expected cursor 0 after 'k', got %d", updated.cursor)
	}
}

func TestCursorBounds(t *testing.T) {
	m := New(true)
	m.width = 80
	m.height = 24

	// Try to move up when already at top
	msg := tea.KeyMsg{Type: tea.KeyUp}
	model, _ := m.Update(msg)
	updated := model.(*Menu)
	if updated.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", updated.cursor)
	}

	// Move to bottom
	m.cursor = 2
	// Try to move down when already at bottom
	msg = tea.KeyMsg{Type: tea.KeyDown}
	model, _ = m.Update(msg)
	updated = model.(*Menu)
	if updated.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", updated.cursor)
	}
}
