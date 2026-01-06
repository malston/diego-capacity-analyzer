// ABOUTME: Tests for the status command
// ABOUTME: Verifies infrastructure status output formatting and exit codes

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/markalston/diego-capacity-analyzer/cli/internal/client"
)

func TestFormatStatusHuman_WithData(t *testing.T) {
	resp := &client.InfrastructureStatus{
		HasData:               true,
		Source:                "vsphere",
		Name:                  "vcenter.example.com",
		ClusterCount:          2,
		HostCount:             8,
		CellCount:             20,
		N1CapacityPercent:     72.0,
		MemoryUtilization:     78.5,
		ConstrainingResource:  "memory",
		HAMinFailuresSurvived: 1,
		HAStatus:              "ok",
	}

	output := formatStatusHuman(resp)

	// Check key elements are present
	checks := []string{
		"vcenter.example.com",
		"vsphere",
		"2",  // clusters
		"8",  // hosts
		"20", // cells
		"72", // N-1 capacity (rounded)
		"78", // memory utilization (rounded)
		"memory",
	}
	for _, check := range checks {
		if !bytes.Contains([]byte(output), []byte(check)) {
			t.Errorf("expected output to contain '%s'", check)
		}
	}
}

func TestFormatStatusHuman_NoData(t *testing.T) {
	resp := &client.InfrastructureStatus{
		HasData:           false,
		VSphereConfigured: false,
	}

	output := formatStatusHuman(resp)

	if !bytes.Contains([]byte(output), []byte("No infrastructure data")) {
		t.Error("expected message about no data")
	}
}

func TestFormatStatusJSON(t *testing.T) {
	resp := &client.InfrastructureStatus{
		HasData:              true,
		Source:               "vsphere",
		ConstrainingResource: "memory",
	}

	output := formatStatusJSON(resp)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["source"] != "vsphere" {
		t.Errorf("expected source in JSON, got %v", parsed["source"])
	}
}

func TestStatusCommand_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.InfrastructureStatus{
			HasData:               true,
			Source:                "vsphere",
			Name:                  "vcenter.example.com",
			ClusterCount:          2,
			HostCount:             8,
			CellCount:             20,
			N1CapacityPercent:     72.0,
			MemoryUtilization:     78.5,
			ConstrainingResource:  "memory",
			HAMinFailuresSurvived: 1,
			HAStatus:              "ok",
		})
	}))
	defer server.Close()

	apiURL = server.URL
	defer func() { apiURL = "" }()

	var buf bytes.Buffer
	exitCode := runStatus(context.Background(), &buf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !bytes.Contains(buf.Bytes(), []byte("vcenter.example.com")) {
		t.Error("expected infrastructure name in output")
	}
}

func TestStatusCommand_NoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.InfrastructureStatus{
			HasData:           false,
			VSphereConfigured: false,
		})
	}))
	defer server.Close()

	apiURL = server.URL
	defer func() { apiURL = "" }()

	var buf bytes.Buffer
	exitCode := runStatus(context.Background(), &buf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2 for no data, got %d", exitCode)
	}
}

func TestStatusCommand_ConnectionError(t *testing.T) {
	apiURL = "http://localhost:99999"
	defer func() { apiURL = "" }()

	var buf bytes.Buffer
	exitCode := runStatus(context.Background(), &buf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}

func TestCapacityStatus(t *testing.T) {
	tests := []struct {
		percent  float64
		expected string
	}{
		{70.0, "ok"},
		{85.0, "warning"},
		{95.0, "critical"},
	}

	for _, tt := range tests {
		result := capacityStatus(tt.percent, 85, 90)
		if result != tt.expected {
			t.Errorf("capacityStatus(%.1f) = %s, expected %s", tt.percent, result, tt.expected)
		}
	}
}
