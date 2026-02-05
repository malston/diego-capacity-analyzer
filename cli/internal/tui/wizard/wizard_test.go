// ABOUTME: Tests for scenario wizard
// ABOUTME: Validates input collection and validation

package wizard

import (
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestWizardDefaults(t *testing.T) {
	infra := &client.InfrastructureState{
		Clusters: []client.ClusterState{{
			Name:              "cluster-1",
			DiegoCellMemoryGB: 64,
			DiegoCellCPU:      8,
			DiegoCellCount:    10,
		}},
	}

	w := New(infra)

	if w.input.ProposedCellMemoryGB != 64 {
		t.Errorf("expected default memory 64, got %d", w.input.ProposedCellMemoryGB)
	}
	if w.input.ProposedCellCPU != 8 {
		t.Errorf("expected default CPU 8, got %d", w.input.ProposedCellCPU)
	}
}

func TestWizardUsesInfraValues(t *testing.T) {
	infra := &client.InfrastructureState{
		TotalHostCount: 12,
		TotalCellCount: 24,
		Clusters: []client.ClusterState{{
			Name:                         "cluster-1",
			DiegoCellMemoryGB:            32,
			DiegoCellCPU:                 4,
			DiegoCellDiskGB:              100,
			DiegoCellCount:               10,
			MemoryGBPerHost:              256,
			CPUThreadsPerHost:              16,
			HAAdmissionControlPercentage: 10,
		}},
	}

	w := New(infra)

	if w.input.ProposedCellMemoryGB != 32 {
		t.Errorf("expected memory 32 from infra, got %d", w.input.ProposedCellMemoryGB)
	}
	if w.input.ProposedCellCPU != 4 {
		t.Errorf("expected CPU 4 from infra, got %d", w.input.ProposedCellCPU)
	}
	if w.input.ProposedCellDiskGB != 100 {
		t.Errorf("expected disk 100 from infra, got %d", w.input.ProposedCellDiskGB)
	}
	if w.input.ProposedCellCount != 24 {
		t.Errorf("expected cell count 24 from infra, got %d", w.input.ProposedCellCount)
	}
	if w.input.HostCount != 12 {
		t.Errorf("expected host count 12 from infra, got %d", w.input.HostCount)
	}
	if w.input.MemoryPerHostGB != 256 {
		t.Errorf("expected memory per host 256, got %d", w.input.MemoryPerHostGB)
	}
	if w.input.HAAdmissionPct != 10 {
		t.Errorf("expected HA admission 10, got %d", w.input.HAAdmissionPct)
	}
	if w.input.PhysicalCoresPerHost != 16 {
		t.Errorf("expected physical cores per host 16, got %d", w.input.PhysicalCoresPerHost)
	}
}

func TestWizardBuildInput(t *testing.T) {
	w := &Wizard{
		input: &client.ScenarioInput{
			ProposedCellMemoryGB: 32,
			ProposedCellCPU:      4,
			ProposedCellCount:    20,
		},
	}

	input := w.GetInput()
	if input.ProposedCellCount != 20 {
		t.Errorf("expected cell count 20, got %d", input.ProposedCellCount)
	}
}

func TestWizardNilInfra(t *testing.T) {
	w := New(nil)

	// Should have reasonable defaults even without infra
	if w.input.ProposedCellMemoryGB != 64 {
		t.Errorf("expected default memory 64, got %d", w.input.ProposedCellMemoryGB)
	}
	if w.input.ProposedCellCPU != 8 {
		t.Errorf("expected default CPU 8, got %d", w.input.ProposedCellCPU)
	}
	if w.input.ProposedCellDiskGB != 200 {
		t.Errorf("expected default disk 200, got %d", w.input.ProposedCellDiskGB)
	}
	if w.input.ProposedCellCount != 10 {
		t.Errorf("expected default cell count 10, got %d", w.input.ProposedCellCount)
	}
	if w.input.OverheadPct != 7.0 {
		t.Errorf("expected default overhead 7.0, got %.1f", w.input.OverheadPct)
	}
	if len(w.input.SelectedResources) != 3 {
		t.Errorf("expected 3 selected resources, got %d", len(w.input.SelectedResources))
	}
}

func TestWizardEmptyClusters(t *testing.T) {
	infra := &client.InfrastructureState{
		Clusters: []client.ClusterState{},
	}

	w := New(infra)

	// Should fall back to defaults with empty clusters
	if w.input.ProposedCellMemoryGB != 64 {
		t.Errorf("expected default memory 64, got %d", w.input.ProposedCellMemoryGB)
	}
}

func TestValidatePositiveInt(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"10", false},
		{"1", false},
		{"0", true},
		{"-1", true},
		{"abc", true},
		{"", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			err := validatePositiveInt(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for input %q", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tc.input, err)
			}
		})
	}
}

func TestValidatePercentage(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"0", false},
		{"50", false},
		{"100", false},
		{"7.5", false},
		{"-1", true},
		{"101", true},
		{"abc", true},
		{"", true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			err := validatePercentage(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for input %q", tc.input)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for input %q: %v", tc.input, err)
			}
		})
	}
}

func TestMemoryOptionsExist(t *testing.T) {
	// Ensure we have common memory sizes
	expectedSizes := []string{"16", "32", "64", "128", "256"}
	for _, size := range expectedSizes {
		found := false
		for _, opt := range memoryOptions {
			if opt.Value == size {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected memory option %s GB not found", size)
		}
	}
}

func TestCPUOptionsExist(t *testing.T) {
	// Ensure we have common CPU sizes
	expectedSizes := []string{"4", "8", "16", "32"}
	for _, size := range expectedSizes {
		found := false
		for _, opt := range cpuOptions {
			if opt.Value == size {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected CPU option %s cores not found", size)
		}
	}
}
