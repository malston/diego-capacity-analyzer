// ABOUTME: Tests for dashboard component
// ABOUTME: Validates infrastructure metrics display

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
		HostMemoryUtilizationPercent: 75.5,
		HAStatus:                     "ok",
		HAMinHostFailuresSurvived:    1,
	}

	d := New(infra, 80, 24)
	view := d.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	// Check for key metrics in output
	tests := []string{"Hosts: 4", "Diego Cells: 10", "75.5%"}
	for _, expected := range tests {
		if !strings.Contains(view, expected) {
			t.Errorf("expected view to contain %q", expected)
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
	d := New(nil, 80, 24)

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
	if !strings.Contains(view, "Hosts: 2") {
		t.Error("expected view to show updated host count")
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
		{"ok status", "ok", "OK"},
		{"warning status", "warning", "WARNING"},
		{"critical status", "critical", "CRITICAL"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			infra := &client.InfrastructureState{
				Name:     "test",
				HAStatus: tc.haStatus,
			}
			d := New(infra, 80, 24)
			view := d.View()

			if !strings.Contains(view, tc.expected) {
				t.Errorf("expected view to contain %q for HA status %q", tc.expected, tc.haStatus)
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
		wantBreakdown string
	}{
		{
			name:          "conservative ratio",
			vcpuRatio:     2.5,
			riskLevel:     "conservative",
			totalVCPUs:    80,
			totalCPUCores: 32,
			wantRatio:     "2.5:1",
			wantRisk:      "conservative",
			wantBreakdown: "80 vCPU / 32 pCPU",
		},
		{
			name:          "moderate ratio",
			vcpuRatio:     5.0,
			riskLevel:     "moderate",
			totalVCPUs:    160,
			totalCPUCores: 32,
			wantRatio:     "5.0:1",
			wantRisk:      "moderate",
			wantBreakdown: "160 vCPU / 32 pCPU",
		},
		{
			name:          "aggressive ratio",
			vcpuRatio:     10.0,
			riskLevel:     "aggressive",
			totalVCPUs:    320,
			totalCPUCores: 32,
			wantRatio:     "10.0:1",
			wantRisk:      "aggressive",
			wantBreakdown: "320 vCPU / 32 pCPU",
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

			d := New(infra, 80, 24)
			view := d.View()

			if !strings.Contains(view, tc.wantRatio) {
				t.Errorf("expected view to contain ratio %q", tc.wantRatio)
			}
			if !strings.Contains(view, tc.wantRisk) {
				t.Errorf("expected view to contain risk level %q", tc.wantRisk)
			}
			if !strings.Contains(view, tc.wantBreakdown) {
				t.Errorf("expected view to contain breakdown %q", tc.wantBreakdown)
			}
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

	d := New(infra, 80, 24)
	view := d.View()

	if !strings.Contains(view, "Clusters: 3") {
		t.Error("expected view to show cluster count")
	}
}
