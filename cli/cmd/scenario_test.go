// ABOUTME: Tests for scenario compare command
// ABOUTME: Validates non-interactive scenario comparison

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

func TestScenarioCommand_JSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request payload
		var input client.ScenarioInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Verify all fields are correctly transmitted
		if input.ProposedCellMemoryGB != 64 {
			t.Errorf("expected memory 64, got %d", input.ProposedCellMemoryGB)
		}
		if input.ProposedCellCPU != 8 {
			t.Errorf("expected cpu 8, got %d", input.ProposedCellCPU)
		}
		if input.ProposedCellDiskGB != 200 {
			t.Errorf("expected disk 200, got %d", input.ProposedCellDiskGB)
		}
		if input.ProposedCellCount != 15 {
			t.Errorf("expected count 15, got %d", input.ProposedCellCount)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.ScenarioComparison{
			Current: client.ScenarioResult{
				CellCount:      10,
				CellMemoryGB:   64,
				UtilizationPct: 75.0,
			},
			Proposed: client.ScenarioResult{
				CellCount:      15,
				CellMemoryGB:   64,
				UtilizationPct: 50.0,
			},
			Delta: client.ScenarioDelta{
				CapacityChangeGB:     320,
				UtilizationChangePct: -25.0,
			},
		})
	}))
	defer server.Close()

	var out bytes.Buffer
	c := client.New(server.URL)

	err := runScenarioCompare(context.Background(), c, &out, 64, 8, 200, 15, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should output JSON
	var result client.ScenarioComparison
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Assert all key response fields
	if result.Current.CellCount != 10 {
		t.Errorf("expected current cell count 10, got %d", result.Current.CellCount)
	}
	if result.Current.UtilizationPct != 75.0 {
		t.Errorf("expected current utilization 75.0, got %.1f", result.Current.UtilizationPct)
	}
	if result.Proposed.CellCount != 15 {
		t.Errorf("expected proposed cell count 15, got %d", result.Proposed.CellCount)
	}
	if result.Proposed.UtilizationPct != 50.0 {
		t.Errorf("expected proposed utilization 50.0, got %.1f", result.Proposed.UtilizationPct)
	}
}

func TestScenarioCommand_HumanOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.ScenarioComparison{
			Current: client.ScenarioResult{
				CellCount:      10,
				CellMemoryGB:   64,
				UtilizationPct: 75.0,
			},
			Proposed: client.ScenarioResult{
				CellCount:      15,
				CellMemoryGB:   64,
				UtilizationPct: 50.0,
			},
			Delta: client.ScenarioDelta{
				CapacityChangeGB:     320,
				UtilizationChangePct: -25.0,
			},
		})
	}))
	defer server.Close()

	var out bytes.Buffer
	c := client.New(server.URL)

	err := runScenarioCompare(context.Background(), c, &out, 64, 8, 200, 15, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()

	// Validate human-readable output contains expected sections
	expectedStrings := []string{
		"Scenario Comparison",
		"Current:",
		"Proposed:",
		"Changes:",
	}

	for _, expected := range expectedStrings {
		if !bytes.Contains([]byte(output), []byte(expected)) {
			t.Errorf("expected output to contain %q", expected)
		}
	}

	// Validate it contains the actual values
	if !bytes.Contains([]byte(output), []byte("Cells: 10")) {
		t.Error("expected output to contain current cell count")
	}
	if !bytes.Contains([]byte(output), []byte("Cells: 15")) {
		t.Error("expected output to contain proposed cell count")
	}
	if !bytes.Contains([]byte(output), []byte("75.0%")) {
		t.Error("expected output to contain current utilization")
	}
	if !bytes.Contains([]byte(output), []byte("50.0%")) {
		t.Error("expected output to contain proposed utilization")
	}
}

func TestScenarioCommand_ConnectionError(t *testing.T) {
	// Use an unreachable server URL
	c := client.New("http://localhost:99999")

	var out bytes.Buffer
	err := runScenarioCompare(context.Background(), c, &out, 64, 8, 200, 15, true)

	if err == nil {
		t.Fatal("expected error when server is unreachable")
	}

	// Verify the error message indicates connection failure
	if !bytes.Contains([]byte(err.Error()), []byte("cannot connect")) {
		t.Errorf("expected error to mention connection failure, got: %v", err)
	}
}

func TestScenarioCommand_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(client.ErrorResponse{
			Error: "infrastructure not configured",
			Code:  500,
		})
	}))
	defer server.Close()

	c := client.New(server.URL)

	var out bytes.Buffer
	err := runScenarioCompare(context.Background(), c, &out, 64, 8, 200, 15, true)

	if err == nil {
		t.Fatal("expected error when server returns error")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("infrastructure not configured")) {
		t.Errorf("expected error message from server, got: %v", err)
	}
}
