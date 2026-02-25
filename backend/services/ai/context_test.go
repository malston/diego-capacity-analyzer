// ABOUTME: Table-driven tests for BuildContext covering full, partial, and missing data scenarios
// ABOUTME: Validates section ordering, threshold flags, missing-data markers, credential safety, and nil safety

package ai

import (
	"fmt"
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

// credentialSentinels returns sentinel values matching every credential field in config.Config.
// These are distinctive strings that would be trivially detectable if they leaked into output.
func credentialSentinels() map[string]string {
	return map[string]string{
		"CFPassword":        "CREDENTIAL_CF_PASSWORD_VALUE",
		"BOSHSecret":        "CREDENTIAL_BOSH_SECRET_VALUE",
		"BOSHCACert":        "CREDENTIAL_BOSH_CA_CERT_VALUE",
		"CredHubSecret":     "CREDENTIAL_CREDHUB_SECRET_VALUE",
		"VSpherePassword":   "CREDENTIAL_VSPHERE_PASSWORD_VALUE",
		"OAuthClientSecret": "CREDENTIAL_OAUTH_CLIENT_SECRET_VALUE",
		"AIAPIKey":          "CREDENTIAL_AI_API_KEY_VALUE",
	}
}

func TestBuildContext_CredentialSafety(t *testing.T) {
	// BuildContext accepts only ContextInput (not config.Config) -- the type system
	// prevents credential leakage structurally. This test is a secondary safety net:
	// it verifies that even with realistic topology data, no credential-like sentinel
	// values appear in the output.

	sentinels := credentialSentinels()

	// Build realistic input with allowed topology data (hostnames, deployment names)
	input := ContextInput{
		Dashboard: &models.DashboardResponse{
			Cells: []models.DiegoCell{
				{ID: "cell-1", Name: "diego_cell/0", MemoryMB: 32768, AllocatedMB: 20000, UsedMB: 15000, IsolationSegment: ""},
				{ID: "cell-2", Name: "diego_cell/1", MemoryMB: 32768, AllocatedMB: 18000, UsedMB: 14000, IsolationSegment: "prod-segment"},
			},
			Apps: []models.App{
				{Name: "web-frontend", Instances: 4, RequestedMB: 2048, ActualMB: 1500},
				{Name: "api-gateway", Instances: 2, RequestedMB: 1024, ActualMB: 800, IsolationSegment: "prod-segment"},
			},
			Metadata: models.Metadata{
				Timestamp:     time.Now(),
				BOSHAvailable: true,
			},
		},
		Infra: &models.InfrastructureState{
			Source: "vsphere",
			Name:   "prod-datacenter",
			Clusters: []models.ClusterState{
				{
					Name:                         "esxi-cluster-01",
					HostCount:                    4,
					MemoryGB:                     512,
					HAUsableMemoryGB:             384,
					HAStatus:                     "ok",
					HAHostFailuresSurvived:       1,
					HostMemoryUtilizationPercent: 65.0,
					VCPURatio:                    3.5,
				},
			},
			TotalHostCount: 4,
			TotalMemoryGB:  512,
			HAStatus:       "ok",
		},
		BOSHConfigured:    true,
		VSphereConfigured: true,
		LogCacheAvailable: true,
	}

	result := BuildContext(input)

	// Verify no sentinel credential values appear in output
	for field, sentinel := range sentinels {
		if strings.Contains(result, sentinel) {
			t.Errorf("credential sentinel for %s (%q) found in output -- credential leakage detected", field, sentinel)
		}
	}

	// Verify allowed topology data DOES appear (proves output is non-trivial)
	// Note: InfrastructureState.Name is not rendered in output -- only cluster names are.
	allowedStrings := []string{
		"esxi-cluster-01",
		"web-frontend",
		"api-gateway",
		"prod-segment",
	}
	for _, allowed := range allowedStrings {
		if !strings.Contains(result, allowed) {
			t.Errorf("expected topology string %q to appear in output but it did not", allowed)
		}
	}

	// Document the structural guarantee: BuildContext accepts ContextInput, not config.Config.
	// This is enforced at compile time. The test below would fail to compile if someone
	// changed the function signature to accept config.Config.
	var fn func(ContextInput) string = BuildContext
	_ = fn // compile-time type check
}

func TestBuildContext_SegmentAggregation(t *testing.T) {
	// Three segments: shared (empty string), iso-seg-1, iso-seg-2
	// with known memory values for verifying per-segment math.
	input := ContextInput{
		Dashboard: &models.DashboardResponse{
			Cells: []models.DiegoCell{
				// Shared segment (empty string): 3 cells
				{ID: "s1", MemoryMB: 32768, AllocatedMB: 20000, UsedMB: 15000, IsolationSegment: ""},
				{ID: "s2", MemoryMB: 32768, AllocatedMB: 18000, UsedMB: 14000, IsolationSegment: ""},
				{ID: "s3", MemoryMB: 16384, AllocatedMB: 10000, UsedMB: 8000, IsolationSegment: ""},
				// iso-seg-1: 2 cells
				{ID: "i1", MemoryMB: 32768, AllocatedMB: 25000, UsedMB: 20000, IsolationSegment: "iso-seg-1"},
				{ID: "i2", MemoryMB: 32768, AllocatedMB: 22000, UsedMB: 17000, IsolationSegment: "iso-seg-1"},
				// iso-seg-2: 1 cell
				{ID: "j1", MemoryMB: 65536, AllocatedMB: 50000, UsedMB: 40000, IsolationSegment: "iso-seg-2"},
			},
			Apps: []models.App{
				{Name: "app-1", Instances: 1, RequestedMB: 512},
			},
			Metadata: models.Metadata{
				Timestamp:     time.Now(),
				BOSHAvailable: true,
			},
		},
		BOSHConfigured:    true,
		VSphereConfigured: false,
		LogCacheAvailable: false,
	}

	result := BuildContext(input)

	// Verify segment counts appear
	// shared: 3 cells, 81920 MB total (32768+32768+16384), 48000 MB allocated
	// iso-seg-1: 2 cells, 65536 MB total, 47000 MB allocated
	// iso-seg-2: 1 cell, 65536 MB total, 50000 MB allocated
	// Totals: 6 cells, 212992 MB total

	// Check shared segment appears with "shared" label and correct count
	if !strings.Contains(result, "**shared**: 3 cells") {
		t.Errorf("expected shared segment with 3 cells in output.\nOutput:\n%s", result)
	}
	// shared total memory: 32768+32768+16384 = 81920
	if !strings.Contains(result, "81920 MB total") {
		t.Errorf("expected shared segment total memory 81920 MB.\nOutput:\n%s", result)
	}

	// Check iso-seg-1: 2 cells, 65536 MB total
	if !strings.Contains(result, "**iso-seg-1**: 2 cells") {
		t.Errorf("expected iso-seg-1 with 2 cells.\nOutput:\n%s", result)
	}
	if !strings.Contains(result, "65536 MB total, 47000 MB allocated") {
		t.Errorf("expected iso-seg-1 total memory 65536 and allocated 47000.\nOutput:\n%s", result)
	}

	// Check iso-seg-2: 1 cell, 65536 MB total
	if !strings.Contains(result, "**iso-seg-2**: 1 cell") {
		t.Errorf("expected iso-seg-2 with 1 cell.\nOutput:\n%s", result)
	}

	// Check overall totals: 6 cells, 212992 MB
	if !strings.Contains(result, "**Totals**: 6 cells, 212992 MB") {
		t.Errorf("expected overall totals of 6 cells and 212992 MB.\nOutput:\n%s", result)
	}

	// Verify ordering: shared appears before iso-seg-1, iso-seg-1 before iso-seg-2
	sharedIdx := strings.Index(result, "**shared**")
	isoSeg1Idx := strings.Index(result, "**iso-seg-1**")
	isoSeg2Idx := strings.Index(result, "**iso-seg-2**")
	if sharedIdx < 0 || isoSeg1Idx < 0 || isoSeg2Idx < 0 {
		t.Fatal("one or more segments missing from output")
	}
	if sharedIdx >= isoSeg1Idx {
		t.Error("shared segment should appear before iso-seg-1")
	}
	if isoSeg1Idx >= isoSeg2Idx {
		t.Error("iso-seg-1 should appear before iso-seg-2 (alphabetical)")
	}
}

func TestBuildContext_MarkerCompleteness(t *testing.T) {
	tests := []struct {
		name     string
		input    ContextInput
		contains []string
	}{
		{
			name: "BOSH not configured",
			input: ContextInput{
				Dashboard: &models.DashboardResponse{
					Apps:     []models.App{{Name: "a", Instances: 1, RequestedMB: 128}},
					Metadata: models.Metadata{Timestamp: time.Now()},
				},
				BOSHConfigured:    false,
				VSphereConfigured: false,
			},
			contains: []string{"BOSH: NOT CONFIGURED"},
		},
		{
			name: "BOSH configured but unavailable",
			input: ContextInput{
				Dashboard: &models.DashboardResponse{
					Apps:     []models.App{{Name: "a", Instances: 1, RequestedMB: 128}},
					Metadata: models.Metadata{Timestamp: time.Now(), BOSHAvailable: false},
				},
				BOSHConfigured: true,
			},
			contains: []string{"BOSH: UNAVAILABLE"},
		},
		{
			name: "BOSH configured and available",
			input: ContextInput{
				Dashboard: &models.DashboardResponse{
					Apps:     []models.App{{Name: "a", Instances: 1, RequestedMB: 128}},
					Cells:    []models.DiegoCell{{ID: "c1", MemoryMB: 1024, IsolationSegment: ""}},
					Metadata: models.Metadata{Timestamp: time.Now(), BOSHAvailable: true},
				},
				BOSHConfigured: true,
			},
			contains: []string{"BOSH: available"},
		},
		{
			name: "vSphere not configured",
			input: ContextInput{
				Dashboard: &models.DashboardResponse{
					Apps:     []models.App{{Name: "a", Instances: 1, RequestedMB: 128}},
					Metadata: models.Metadata{Timestamp: time.Now()},
				},
				VSphereConfigured: false,
			},
			contains: []string{"vSphere: NOT CONFIGURED"},
		},
		{
			name: "vSphere configured but unavailable",
			input: ContextInput{
				Dashboard: &models.DashboardResponse{
					Apps:     []models.App{{Name: "a", Instances: 1, RequestedMB: 128}},
					Metadata: models.Metadata{Timestamp: time.Now()},
				},
				VSphereConfigured: true,
				Infra:             nil,
			},
			contains: []string{"vSphere: UNAVAILABLE"},
		},
		{
			name: "vSphere configured and available",
			input: ContextInput{
				Dashboard: &models.DashboardResponse{
					Apps:     []models.App{{Name: "a", Instances: 1, RequestedMB: 128}},
					Metadata: models.Metadata{Timestamp: time.Now()},
				},
				VSphereConfigured: true,
				Infra: &models.InfrastructureState{
					Clusters: []models.ClusterState{
						{Name: "test-cl", HostCount: 2, MemoryGB: 128, HAStatus: "ok"},
					},
					TotalHostCount: 2,
					TotalMemoryGB:  128,
					HAStatus:       "ok",
				},
			},
			contains: []string{"test-cl"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildContext(tt.input)

			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Errorf("expected %q in output.\nOutput:\n%s", want, result)
				}
			}

			// Every result must have all 5 section headings -- no section is silently omitted
			requiredSections := []string{
				"## Data Sources",
				"## Infrastructure",
				"## Diego Cells",
				"## Apps",
				"## Scenario Comparison",
			}
			for _, section := range requiredSections {
				if !strings.Contains(result, section) {
					t.Errorf("section %q missing from output -- sections must never be silently omitted", section)
				}
			}
		})
	}
}

func TestBuildContext_TokenBudget(t *testing.T) {
	// Realistic-sized input: 50 apps, 3 isolation segments with cells, 2 clusters, scenario
	apps := make([]models.App, 50)
	for i := range apps {
		apps[i] = models.App{
			Name:             fmt.Sprintf("app-%03d", i),
			Instances:        (i%5 + 1),
			RequestedMB:      2048 - i*30,
			ActualMB:         1500 - i*20,
			IsolationSegment: []string{"", "iso-prod", "iso-staging"}[i%3],
		}
	}

	cells := make([]models.DiegoCell, 0, 19)
	// 8 shared cells
	for i := range 8 {
		cells = append(cells, models.DiegoCell{
			ID: fmt.Sprintf("shared-%d", i), MemoryMB: 32768,
			AllocatedMB: 20000 + i*1000, UsedMB: 15000 + i*500, IsolationSegment: "",
		})
	}
	// 6 iso-prod cells
	for i := range 6 {
		cells = append(cells, models.DiegoCell{
			ID: fmt.Sprintf("prod-%d", i), MemoryMB: 32768,
			AllocatedMB: 22000 + i*500, UsedMB: 18000 + i*300, IsolationSegment: "iso-prod",
		})
	}
	// 5 iso-staging cells
	for i := range 5 {
		cells = append(cells, models.DiegoCell{
			ID: fmt.Sprintf("staging-%d", i), MemoryMB: 16384,
			AllocatedMB: 10000 + i*500, UsedMB: 8000 + i*300, IsolationSegment: "iso-staging",
		})
	}

	input := ContextInput{
		Dashboard: &models.DashboardResponse{
			Cells: cells,
			Apps:  apps,
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
					Name: "cluster-east", HostCount: 6, MemoryGB: 768,
					HAUsableMemoryGB: 640, HAStatus: "ok", HAHostFailuresSurvived: 1,
					HostMemoryUtilizationPercent: 72.0, VCPURatio: 3.8,
				},
				{
					Name: "cluster-west", HostCount: 4, MemoryGB: 512,
					HAUsableMemoryGB: 384, HAStatus: "ok", HAHostFailuresSurvived: 1,
					HostMemoryUtilizationPercent: 68.0, VCPURatio: 4.2,
				},
			},
			TotalHostCount: 10,
			TotalMemoryGB:  1280,
			HAStatus:       "ok",
		},
		Scenario: &models.ScenarioComparison{
			Current: models.ScenarioResult{
				CellCount: 19, CellMemoryGB: 32, CellCPU: 4, CellDiskGB: 64,
				AppCapacityGB: 560, UtilizationPct: 78.5,
			},
			Proposed: models.ScenarioResult{
				CellCount: 24, CellMemoryGB: 32, CellCPU: 4, CellDiskGB: 64,
				AppCapacityGB: 710, UtilizationPct: 62.0,
			},
			Delta: models.ScenarioDelta{
				CapacityChangeGB:     150,
				UtilizationChangePct: -16.5,
			},
			Warnings: []models.ScenarioWarning{
				{Severity: "info", Message: "Adding 5 cells increases capacity by 150 GB"},
			},
		},
		BOSHConfigured:    true,
		VSphereConfigured: true,
		LogCacheAvailable: true,
	}

	result := BuildContext(input)

	// ~1000 tokens at ~5 chars/token = ~5000 chars. Allow some margin.
	const maxChars = 5000
	if len(result) > maxChars {
		t.Errorf("output length %d exceeds token budget proxy of %d chars (~%d tokens).\nOutput:\n%s",
			len(result), maxChars, len(result)/5, result)
	}

	// Verify top-N truncation engaged (50 apps, only 10 shown)
	if !strings.Contains(result, "Showing 10 of 50") {
		t.Errorf("expected top-N truncation message for 50 apps.\nOutput:\n%s", result)
	}
}
