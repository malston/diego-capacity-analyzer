// ABOUTME: Tests for the health command
// ABOUTME: Verifies health check output formatting and exit codes

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

func TestFormatHealthHuman(t *testing.T) {
	resp := &client.HealthResponse{
		CFAPI:   "ok",
		BOSHAPI: "ok",
	}

	output := formatHealthHuman("http://localhost:8080", resp)

	if !bytes.Contains([]byte(output), []byte("http://localhost:8080")) {
		t.Error("expected output to contain backend URL")
	}
	if !bytes.Contains([]byte(output), []byte("CF API:")) {
		t.Error("expected output to contain CF API label")
	}
	if !bytes.Contains([]byte(output), []byte("ok")) {
		t.Error("expected output to contain ok status")
	}
}

func TestFormatHealthJSON(t *testing.T) {
	resp := &client.HealthResponse{
		CFAPI:   "ok",
		BOSHAPI: "not_configured",
	}

	output := formatHealthJSON("http://localhost:8080", resp)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if parsed["backend"] != "http://localhost:8080" {
		t.Errorf("expected backend URL in JSON, got %v", parsed["backend"])
	}
}

func TestHealthCommand_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(client.HealthResponse{
			CFAPI:   "ok",
			BOSHAPI: "ok",
		})
	}))
	defer server.Close()

	apiURL = server.URL
	defer func() { apiURL = "" }()

	var buf bytes.Buffer
	exitCode := runHealth(context.Background(), &buf)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if !bytes.Contains(buf.Bytes(), []byte("ok")) {
		t.Error("expected ok in output")
	}
}

func TestHealthCommand_ConnectionError(t *testing.T) {
	apiURL = "http://localhost:99999"
	defer func() { apiURL = "" }()

	var buf bytes.Buffer
	exitCode := runHealth(context.Background(), &buf)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Error:")) {
		t.Error("expected error message in output")
	}
}
