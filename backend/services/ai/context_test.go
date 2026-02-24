// ABOUTME: Table-driven tests for BuildContext covering full, partial, and missing data scenarios
// ABOUTME: Validates section ordering, threshold flags, missing-data markers, and nil safety

package ai

import (
	"strings"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/models"
)

func TestBuildContext(t *testing.T) {
	tests := []struct {
		name     string
		input    ContextInput
		contains []string
		excludes []string
	}{
		{
			name:  "full data with all sources populated",
			input: fullDataInput(),
			contains: []string{
				// Data Sources section
				"## Data Sources",
				"CF API: available",
				"BOSH: available",
				"vSphere: available",
				"Log Cache: available",

				// Infrastructure section
				"## Infrastructure",
				"Physical hosts and clusters backing Diego cells",
				"cluster-a",
				"cluster-b",

				// Diego Cells section
				"## Diego Cells",
				"Diego cell capacity grouped by isolation segment",
				"shared",
				"iso-seg-1",

				// Apps section
				"## Apps",
				"Top applications by memory allocation",
				"big-app-1",
				"big-app-2",

				// Scenario Comparison section
				"## Scenario Comparison",
				"Current vs proposed capacity changes",
				"Metric",
				"Current",
				"Proposed",
				"Delta",
			},
			excludes: []string{
				"NOT CONFIGURED",
				"UNAVAILABLE",
			},
		},
		{
			name:  "partial data CF and BOSH only",
			input: partialCFBOSHInput(),
			contains: []string{
				// Data Sources
				"## Data Sources",
				"CF API: available",
				"BOSH: available",
				"vSphere: NOT CONFIGURED",

				// Infrastructure section still appears with marker
				"## Infrastructure",
				"NOT CONFIGURED",

				// Scenario section still appears with marker
				"## Scenario Comparison",
				"No scenario comparison has been run",

				// Cells and Apps render normally
				"## Diego Cells",
				"## Apps",
			},
			excludes: []string{
				"cluster-a",
				"cluster-b",
			},
		},
		{
			name:  "CF only no BOSH no vSphere",
			input: cfOnlyInput(),
			contains: []string{
				// Data Sources
				"## Data Sources",
				"CF API: available",
				"BOSH: NOT CONFIGURED",
				"vSphere: NOT CONFIGURED",

				// Cells section still renders
				"## Diego Cells",

				// Apps section still renders
				"## Apps",
			},
			excludes: []string{
				"cluster-a",
			},
		},
		{
			name: "all missing nil dashboard nil infra nil scenario",
			input: ContextInput{
				Dashboard:         nil,
				Infra:             nil,
				Scenario:          nil,
				BOSHConfigured:    false,
				VSphereConfigured: false,
				LogCacheAvailable: false,
			},
			contains: []string{
				"## Data Sources",
				"## Infrastructure",
				"## Diego Cells",
				"## Apps",
				"## Scenario Comparison",
				"NOT CONFIGURED",
				"No scenario comparison has been run",
			},
			excludes: []string{
				"cluster-a",
				"big-app-1",
			},
		},
		{
			name: "BOSH configured but unavailable",
			input: ContextInput{
				Dashboard: &models.DashboardResponse{
					Apps: []models.App{
						{Name: "test-app", Instances: 1, RequestedMB: 512},
					},
					Metadata: models.Metadata{
						Timestamp:     time.Now(),
						BOSHAvailable: false,
					},
				},
				Infra:             nil,
				Scenario:          nil,
				BOSHConfigured:    true,
				VSphereConfigured: false,
				LogCacheAvailable: false,
			},
			contains: []string{
				"## Data Sources",
				"BOSH: UNAVAILABLE",
			},
			excludes: []string{
				"BOSH: NOT CONFIGURED",
				"BOSH: available",
			},
		},
		{
			name: "vSphere configured but unavailable",
			input: ContextInput{
				Dashboard: &models.DashboardResponse{
					Apps: []models.App{
						{Name: "test-app", Instances: 1, RequestedMB: 512},
					},
					Metadata: models.Metadata{
						Timestamp:     time.Now(),
						BOSHAvailable: true,
					},
				},
				Infra:             nil,
				Scenario:          nil,
				BOSHConfigured:    true,
				VSphereConfigured: true,
				LogCacheAvailable: false,
			},
			contains: []string{
				"## Data Sources",
				"vSphere: UNAVAILABLE",
				"## Infrastructure",
			},
			excludes: []string{
				"vSphere: NOT CONFIGURED",
				"vSphere: available",
			},
		},
		{
			name:  "threshold flags on high utilization",
			input: highUtilizationInput(),
			contains: []string{
				"[HIGH]",
			},
		},
		{
			name:  "critical threshold flags",
			input: criticalUtilizationInput(),
			contains: []string{
				"[CRITICAL]",
			},
		},
		{
			name:  "apps top N limited to 10",
			input: manyAppsInput(),
			contains: []string{
				"## Apps",
				"Showing 10 of 15",
			},
		},
		{
			name:  "apps with partial log cache data",
			input: partialLogCacheAppsInput(),
			contains: []string{
				"## Apps",
				"Memory usage unavailable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildContext(tt.input)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("output should contain %q but did not.\nOutput:\n%s", want, result)
				}
			}

			for _, exclude := range tt.excludes {
				if strings.Contains(result, exclude) {
					t.Errorf("output should NOT contain %q but did.\nOutput:\n%s", exclude, result)
				}
			}
		})
	}
}

// fullDataInput builds a ContextInput with all sources populated.
func fullDataInput() ContextInput {
	return ContextInput{
		Dashboard: &models.DashboardResponse{
			Cells: []models.DiegoCell{
				{ID: "cell-1", Name: "diego_cell/0", MemoryMB: 32768, AllocatedMB: 20000, UsedMB: 15000, IsolationSegment: ""},
				{ID: "cell-2", Name: "diego_cell/1", MemoryMB: 32768, AllocatedMB: 18000, UsedMB: 14000, IsolationSegment: ""},
				{ID: "cell-3", Name: "diego_cell/2", MemoryMB: 32768, AllocatedMB: 22000, UsedMB: 16000, IsolationSegment: ""},
				{ID: "cell-4", Name: "diego_cell/3", MemoryMB: 32768, AllocatedMB: 19000, UsedMB: 13000, IsolationSegment: "iso-seg-1"},
				{ID: "cell-5", Name: "diego_cell/4", MemoryMB: 32768, AllocatedMB: 21000, UsedMB: 15500, IsolationSegment: "iso-seg-1"},
				{ID: "cell-6", Name: "diego_cell/5", MemoryMB: 32768, AllocatedMB: 17000, UsedMB: 12000, IsolationSegment: "iso-seg-1"},
			},
			Apps: []models.App{
				{Name: "big-app-1", Instances: 4, RequestedMB: 2048, ActualMB: 1500, IsolationSegment: ""},
				{Name: "big-app-2", Instances: 3, RequestedMB: 1024, ActualMB: 800, IsolationSegment: ""},
				{Name: "medium-app-1", Instances: 2, RequestedMB: 512, ActualMB: 400, IsolationSegment: "iso-seg-1"},
				{Name: "medium-app-2", Instances: 2, RequestedMB: 512, ActualMB: 350, IsolationSegment: ""},
				{Name: "small-app-1", Instances: 1, RequestedMB: 256, ActualMB: 200, IsolationSegment: ""},
				{Name: "small-app-2", Instances: 1, RequestedMB: 256, ActualMB: 180, IsolationSegment: "iso-seg-1"},
				{Name: "small-app-3", Instances: 1, RequestedMB: 128, ActualMB: 100, IsolationSegment: ""},
				{Name: "small-app-4", Instances: 1, RequestedMB: 128, ActualMB: 90, IsolationSegment: ""},
				{Name: "small-app-5", Instances: 1, RequestedMB: 64, ActualMB: 50, IsolationSegment: "iso-seg-1"},
				{Name: "tiny-app", Instances: 1, RequestedMB: 32, ActualMB: 20, IsolationSegment: ""},
			},
			Segments: []models.IsolationSegment{
				{GUID: "seg-1", Name: "iso-seg-1"},
			},
			Metadata: models.Metadata{
				Timestamp:     time.Now(),
				BOSHAvailable: true,
			},
		},
		Infra: &models.InfrastructureState{
			Source: "vsphere",
			Name:   "prod-dc",
			Clusters: []models.ClusterState{
				{
					Name:                         "cluster-a",
					HostCount:                    4,
					MemoryGB:                     512,
					MemoryGBPerHost:              128,
					HAUsableMemoryGB:             384,
					HAStatus:                     "ok",
					HAHostFailuresSurvived:       1,
					HostMemoryUtilizationPercent: 65.0,
					VCPURatio:                    3.5,
					DiegoCellCount:               3,
					DiegoCellMemoryGB:            32,
					TotalCellMemoryGB:            96,
				},
				{
					Name:                         "cluster-b",
					HostCount:                    3,
					MemoryGB:                     384,
					MemoryGBPerHost:              128,
					HAUsableMemoryGB:             256,
					HAStatus:                     "ok",
					HAHostFailuresSurvived:       1,
					HostMemoryUtilizationPercent: 70.0,
					VCPURatio:                    4.0,
					DiegoCellCount:               3,
					DiegoCellMemoryGB:            32,
					TotalCellMemoryGB:            96,
				},
			},
			TotalHostCount:               7,
			TotalMemoryGB:                896,
			TotalHAUsableMemoryGB:        640,
			HAStatus:                     "ok",
			HostMemoryUtilizationPercent: 67.5,
			VCPURatio:                    3.75,
		},
		Scenario: &models.ScenarioComparison{
			Current: models.ScenarioResult{
				CellCount:      6,
				CellMemoryGB:   32,
				CellCPU:        4,
				CellDiskGB:     64,
				AppCapacityGB:  178,
				UtilizationPct: 72.5,
			},
			Proposed: models.ScenarioResult{
				CellCount:      8,
				CellMemoryGB:   32,
				CellCPU:        4,
				CellDiskGB:     64,
				AppCapacityGB:  238,
				UtilizationPct: 54.2,
			},
			Delta: models.ScenarioDelta{
				CapacityChangeGB:     60,
				UtilizationChangePct: -18.3,
			},
			Warnings: []models.ScenarioWarning{
				{Severity: "info", Message: "Adding 2 cells increases capacity by 60 GB"},
			},
		},
		BOSHConfigured:    true,
		VSphereConfigured: true,
		LogCacheAvailable: true,
	}
}

// partialCFBOSHInput builds input with CF + BOSH but no vSphere or scenario.
func partialCFBOSHInput() ContextInput {
	return ContextInput{
		Dashboard: &models.DashboardResponse{
			Cells: []models.DiegoCell{
				{ID: "cell-1", Name: "diego_cell/0", MemoryMB: 32768, AllocatedMB: 20000, UsedMB: 15000, IsolationSegment: ""},
				{ID: "cell-2", Name: "diego_cell/1", MemoryMB: 32768, AllocatedMB: 18000, UsedMB: 14000, IsolationSegment: ""},
			},
			Apps: []models.App{
				{Name: "app-1", Instances: 2, RequestedMB: 1024, ActualMB: 800},
				{Name: "app-2", Instances: 1, RequestedMB: 512, ActualMB: 400},
			},
			Metadata: models.Metadata{
				Timestamp:     time.Now(),
				BOSHAvailable: true,
			},
		},
		Infra:             nil,
		Scenario:          nil,
		BOSHConfigured:    true,
		VSphereConfigured: false,
		LogCacheAvailable: true,
	}
}

// cfOnlyInput builds input with CF data but no BOSH or vSphere.
func cfOnlyInput() ContextInput {
	return ContextInput{
		Dashboard: &models.DashboardResponse{
			Apps: []models.App{
				{Name: "cf-app-1", Instances: 2, RequestedMB: 1024},
				{Name: "cf-app-2", Instances: 1, RequestedMB: 512},
			},
			Metadata: models.Metadata{
				Timestamp:     time.Now(),
				BOSHAvailable: false,
			},
		},
		Infra:             nil,
		Scenario:          nil,
		BOSHConfigured:    false,
		VSphereConfigured: false,
		LogCacheAvailable: false,
	}
}

// highUtilizationInput builds input with utilization >80% to test [HIGH] flags.
func highUtilizationInput() ContextInput {
	return ContextInput{
		Dashboard: &models.DashboardResponse{
			Cells: []models.DiegoCell{
				{ID: "cell-1", MemoryMB: 32768, AllocatedMB: 28000, IsolationSegment: ""},
			},
			Apps: []models.App{
				{Name: "app-1", Instances: 1, RequestedMB: 1024},
			},
			Metadata: models.Metadata{
				Timestamp:     time.Now(),
				BOSHAvailable: true,
			},
		},
		Infra: &models.InfrastructureState{
			Clusters: []models.ClusterState{
				{
					Name:                         "high-util-cluster",
					HostCount:                    3,
					MemoryGB:                     384,
					HAUsableMemoryGB:             256,
					HAStatus:                     "ok",
					HostMemoryUtilizationPercent: 85.0,
					VCPURatio:                    5.0,
				},
			},
			TotalHostCount: 3,
			TotalMemoryGB:  384,
			HAStatus:       "ok",
		},
		BOSHConfigured:    true,
		VSphereConfigured: true,
		LogCacheAvailable: false,
	}
}

// criticalUtilizationInput builds input with utilization >90% to test [CRITICAL] flags.
func criticalUtilizationInput() ContextInput {
	return ContextInput{
		Dashboard: &models.DashboardResponse{
			Cells: []models.DiegoCell{
				{ID: "cell-1", MemoryMB: 32768, AllocatedMB: 30000, IsolationSegment: ""},
			},
			Apps: []models.App{
				{Name: "app-1", Instances: 1, RequestedMB: 1024},
			},
			Metadata: models.Metadata{
				Timestamp:     time.Now(),
				BOSHAvailable: true,
			},
		},
		Infra: &models.InfrastructureState{
			Clusters: []models.ClusterState{
				{
					Name:                         "crit-cluster",
					HostCount:                    3,
					MemoryGB:                     384,
					HAUsableMemoryGB:             256,
					HAStatus:                     "at-risk",
					HostMemoryUtilizationPercent: 95.0,
					VCPURatio:                    10.0,
				},
			},
			TotalHostCount: 3,
			TotalMemoryGB:  384,
			HAStatus:       "at-risk",
		},
		BOSHConfigured:    true,
		VSphereConfigured: true,
		LogCacheAvailable: false,
	}
}

// manyAppsInput builds input with more than 10 apps to test top-N truncation.
func manyAppsInput() ContextInput {
	apps := make([]models.App, 15)
	for i := range apps {
		apps[i] = models.App{
			Name:        "app-" + string(rune('a'+i)),
			Instances:   2,
			RequestedMB: 1024 - i*50,
			ActualMB:    800 - i*40,
		}
	}
	return ContextInput{
		Dashboard: &models.DashboardResponse{
			Apps: apps,
			Metadata: models.Metadata{
				Timestamp:     time.Now(),
				BOSHAvailable: true,
			},
		},
		BOSHConfigured:    true,
		VSphereConfigured: false,
		LogCacheAvailable: true,
	}
}

// partialLogCacheAppsInput builds input where some apps have ActualMB data and others don't.
func partialLogCacheAppsInput() ContextInput {
	return ContextInput{
		Dashboard: &models.DashboardResponse{
			Apps: []models.App{
				{Name: "tracked-app-1", Instances: 2, RequestedMB: 1024, ActualMB: 800},
				{Name: "tracked-app-2", Instances: 1, RequestedMB: 512, ActualMB: 400},
				{Name: "untracked-app-1", Instances: 1, RequestedMB: 256, ActualMB: 0},
				{Name: "untracked-app-2", Instances: 1, RequestedMB: 128, ActualMB: 0},
			},
			Metadata: models.Metadata{
				Timestamp:     time.Now(),
				BOSHAvailable: true,
			},
		},
		BOSHConfigured:    true,
		VSphereConfigured: false,
		LogCacheAvailable: true,
	}
}
