// ABOUTME: Tests for dashboard component
// ABOUTME: Validates infrastructure metrics display with visual widgets

package dashboard

import (
	"strings"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestDashboardView(t *testing.T) {
	infra := &client.InfrastructureState{
		Source:                       "vsphere",
		Name:                         "vcenter.test.com",
		TotalHostCount:               4,
		TotalCellCount:               10,
		TotalMemoryGB:                512,
		HostMemoryUtilizationPercent: 75.5,
		HAStatus:                     "ok",
		HAMinHostFailuresSurvived:    1,
	}

	d := New(infra, 120, 24)
	view := d.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	// Check for key content in new widget-based output
	// The new format shows metrics in compact blocks
	tests := []string{
		"Memory",        // Memory metric block title
		"Hosts",         // Host count block title
		"75.5%",         // Utilization percentage
		"vcenter.test.com", // Infrastructure name
	}
	for _, expected := range tests {
		if !strings.Contains(view, expected) {
			t.Errorf("expected view to contain %q\nView:\n%s", expected, view)
		}
	}
}

func TestDashboardNilInfra(t *testing.T) {
	d := New(nil, 80, 24)
	view := d.View()

	if !strings.Contains(view, "Loading") {
		t.Error("expected loading message when infra is nil")
	}
}

func TestDashboardUpdate(t *testing.T) {
	d := New(nil, 120, 24)

	// Initial state should show loading
	view := d.View()
	if !strings.Contains(view, "Loading") {
		t.Error("expected loading message initially")
	}

	// Update with infrastructure data
	infra := &client.InfrastructureState{
		Name:           "test-cluster",
		TotalHostCount: 2,
	}
	d.Update(infra)

	view = d.View()
	if strings.Contains(view, "Loading") {
		t.Error("should not show loading after update")
	}
	// New format shows count and label separately in metric blocks
	if !strings.Contains(view, "2") || !strings.Contains(view, "hosts") {
		t.Errorf("expected view to show host count 2\nView:\n%s", view)
	}
}

func TestDashboardSetSize(t *testing.T) {
	d := New(nil, 80, 24)

	d.SetSize(120, 40)

	if d.width != 120 {
		t.Errorf("expected width 120, got %d", d.width)
	}
	if d.height != 40 {
		t.Errorf("expected height 40, got %d", d.height)
	}
}

func TestDashboardHAStatus(t *testing.T) {
	tests := []struct {
		name     string
		haStatus string
		expected string
	}{
		// New format uses descriptive status text from StatusText widget
		{"ok status", "ok", "survive"},      // "Can survive X host failure(s)"
		{"warning status", "warning", "HA"}, // Shows HA Status panel
		{"critical status", "critical", "Cannot survive"}, // "Cannot survive host failure"
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			infra := &client.InfrastructureState{
				Name:     "test",
				HAStatus: tc.haStatus,
			}
			d := New(infra, 120, 24)
			view := d.View()

			if !strings.Contains(view, tc.expected) {
				t.Errorf("expected view to contain %q for HA status %q\nView:\n%s", tc.expected, tc.haStatus, view)
			}
		})
	}
}

func TestDashboardVCPURatio(t *testing.T) {
	tests := []struct {
		name          string
		vcpuRatio     float64
		riskLevel     string
		totalVCPUs    int
		totalCPUCores int
		wantRatio     string
		wantRisk      string
	}{
		{
			name:          "conservative ratio",
			vcpuRatio:     2.5,
			riskLevel:     "conservative",
			totalVCPUs:    80,
			totalCPUCores: 32,
			wantRatio:     "2.5:1",
			wantRisk:      "Conservative", // Widget uses title case
		},
		{
			name:          "moderate ratio",
			vcpuRatio:     5.0,
			riskLevel:     "moderate",
			totalVCPUs:    160,
			totalCPUCores: 32,
			wantRatio:     "5.0:1",
			wantRisk:      "Moderate",
		},
		{
			name:          "aggressive ratio",
			vcpuRatio:     10.0,
			riskLevel:     "aggressive",
			totalVCPUs:    320,
			totalCPUCores: 32,
			wantRatio:     "10.0:1",
			wantRisk:      "Aggressive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			infra := &client.InfrastructureState{
				Name:          "test",
				VCPURatio:     tc.vcpuRatio,
				CPURiskLevel:  tc.riskLevel,
				TotalVCPUs:    tc.totalVCPUs,
				TotalCPUCores: tc.totalCPUCores,
			}

			d := New(infra, 120, 24)
			view := d.View()

			if !strings.Contains(view, tc.wantRatio) {
				t.Errorf("expected view to contain ratio %q\nView:\n%s", tc.wantRatio, view)
			}
			if !strings.Contains(view, tc.wantRisk) {
				t.Errorf("expected view to contain risk level %q\nView:\n%s", tc.wantRisk, view)
			}
			// Note: New format doesn't show the breakdown "X vCPU / Y pCPU" inline
		})
	}
}

func TestDashboardClusters(t *testing.T) {
	infra := &client.InfrastructureState{
		Name: "test",
		Clusters: []client.ClusterState{
			{Name: "cluster-1"},
			{Name: "cluster-2"},
			{Name: "cluster-3"},
		},
	}

	d := New(infra, 120, 24)
	view := d.View()

	// New format shows cluster count in a metric block
	if !strings.Contains(view, "3") || !strings.Contains(view, "clusters") {
		t.Errorf("expected view to show cluster count 3\nView:\n%s", view)
	}
}

func TestDashboardHistoryTracking(t *testing.T) {
	infra := &client.InfrastructureState{
		Name:                         "test",
		HostMemoryUtilizationPercent: 50.0,
		VCPURatio:                    2.5,
	}

	d := New(infra, 120, 24)

	// Initial history should have one entry
	if len(d.historyMemory) != 1 {
		t.Errorf("expected 1 history entry, got %d", len(d.historyMemory))
	}

	// Update multiple times
	for i := 0; i < 10; i++ {
		infra.HostMemoryUtilizationPercent = 50.0 + float64(i)
		d.Update(infra)
	}

	// History should be capped at 8 entries
	if len(d.historyMemory) != 8 {
		t.Errorf("expected 8 history entries (capped), got %d", len(d.historyMemory))
	}
}
