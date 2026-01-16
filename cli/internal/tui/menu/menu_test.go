// ABOUTME: Tests for data source selection menu
// ABOUTME: Validates menu rendering and selection behavior

package menu

import "testing"

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
