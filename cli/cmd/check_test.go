// ABOUTME: Tests for the check command
// ABOUTME: Verifies threshold checking logic and exit codes

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

func TestCheckResult_AllPassed(t *testing.T) {
	results := []checkResult{
		{name: "N-1 capacity", value: 72.0, threshold: 85.0, unit: "%", passed: true},
		{name: "Memory", value: 78.0, threshold: 90.0, unit: "%", passed: true},
	}

	passed, failed := countResults(results)
	if passed != 2 {
		t.Errorf("expected 2 passed, got %d", passed)
	}
	if failed != 0 {
		t.Errorf("expected 0 failed, got %d", failed)
	}
}

func TestCheckResult_SomeFailed(t *testing.T) {
	results := []checkResult{
		{name: "N-1 capacity", value: 92.0, threshold: 85.0, unit: "%", passed: false},
		{name: "Memory", value: 78.0, threshold: 90.0, unit: "%", passed: true},
	}

	passed, failed := countResults(results)
	if passed != 1 {
		t.Errorf("expected 1 passed, got %d", passed)
	}
	if failed != 1 {
		t.Errorf("expected 1 failed, got %d", failed)
	}
}

func TestFormatCheckHuman(t *testing.T) {
	results := []checkResult{
		{name: "N-1 capacity", value: 72.0, threshold: 85.0, unit: "%", passed: true},
		{name: "Memory", value: 92.0, threshold: 90.0, unit: "%", passed: false},
	}

	output := formatCheckHuman(results)

	if !bytes.Contains([]byte(output), []byte("✓")) {
		t.Error("expected checkmark for passed test")
	}
	if !bytes.Contains([]byte(output), []byte("✗")) {
		t.Error("expected X for failed test")
	}
	if !bytes.Contains([]byte(output), []byte("FAILED")) {
		t.Error("expected FAILED summary")
	}
}

func TestFormatCheckJSON(t *testing.T) {
	results := []checkResult{
		{name: "N-1 capacity", value: 72.0, threshold: 85.0, unit: "%", passed: true},
	}

	output := formatCheckJSON(results)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["status"] != "passed" {
		t.Errorf("expected status passed, got %v", parsed["status"])
	}
}

func TestCheckCommand_AllPassed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.InfrastructureStatus{
			HasData:           true,
			N1CapacityPercent: 72.0,
			MemoryUtilization: 78.0,
		})
	}))
	defer server.Close()

	apiURL = server.URL
	n1Threshold = 85
	memoryThreshold = 90
	defer func() {
		apiURL = ""
		n1Threshold = 85
		memoryThreshold = 90
	}()

	var buf bytes.Buffer
	exitCode := runCheck(context.Background(), &buf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !bytes.Contains(buf.Bytes(), []byte("PASSED")) {
		t.Error("expected PASSED in output")
	}
}

func TestCheckCommand_ThresholdExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.InfrastructureStatus{
			HasData:           true,
			N1CapacityPercent: 92.0, // Exceeds threshold
			MemoryUtilization: 78.0,
		})
	}))
	defer server.Close()

	apiURL = server.URL
	n1Threshold = 85
	memoryThreshold = 90
	defer func() {
		apiURL = ""
		n1Threshold = 85
		memoryThreshold = 90
	}()

	var buf bytes.Buffer
	exitCode := runCheck(context.Background(), &buf)

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for threshold exceeded, got %d", exitCode)
	}
	if !bytes.Contains(buf.Bytes(), []byte("FAILED")) {
		t.Error("expected FAILED in output")
	}
}

func TestCheckCommand_NoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.InfrastructureStatus{
			HasData: false,
		})
	}))
	defer server.Close()

	apiURL = server.URL
	defer func() { apiURL = "" }()

	var buf bytes.Buffer
	exitCode := runCheck(context.Background(), &buf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2 for no data, got %d", exitCode)
	}
}

func TestCheckCommand_ConnectionError(t *testing.T) {
	apiURL = "http://localhost:99999"
	defer func() { apiURL = "" }()

	var buf bytes.Buffer
	exitCode := runCheck(context.Background(), &buf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
}

func TestValidateThresholds(t *testing.T) {
	tests := []struct {
		n1     int
		memory int
		valid  bool
	}{
		{85, 90, true},
		{0, 90, true},
		{100, 90, true},
		{-1, 90, false},
		{101, 90, false},
		{85, -1, false},
		{85, 101, false},
	}

	for _, tt := range tests {
		err := validateThresholds(tt.n1, tt.memory)
		if tt.valid && err != nil {
			t.Errorf("validateThresholds(%d, %d) expected valid, got error: %v", tt.n1, tt.memory, err)
		}
		if !tt.valid && err == nil {
			t.Errorf("validateThresholds(%d, %d) expected error, got nil", tt.n1, tt.memory)
		}
	}
}
