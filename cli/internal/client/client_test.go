// ABOUTME: Tests for the Diego Capacity Analyzer API client
// ABOUTME: Uses httptest to mock backend responses

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealth_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			t.Errorf("expected path /api/health, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{
			CFAPI:   "ok",
			BOSHAPI: "ok",
		})
	}))
	defer server.Close()

	c := New(server.URL)
	resp, err := c.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CFAPI != "ok" {
		t.Errorf("expected CFAPI ok, got %s", resp.CFAPI)
	}
	if resp.BOSHAPI != "ok" {
		t.Errorf("expected BOSHAPI ok, got %s", resp.BOSHAPI)
	}
}

func TestHealth_ConnectionError(t *testing.T) {
	c := New("http://localhost:99999")
	_, err := c.Health(context.Background())
	if err == nil {
		t.Error("expected connection error, got nil")
	}
}

func TestHealth_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "internal error"})
	}))
	defer server.Close()

	c := New(server.URL)
	_, err := c.Health(context.Background())
	if err == nil {
		t.Error("expected error for non-OK status, got nil")
	}
}

func TestHealth_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{CFAPI: "ok", BOSHAPI: "ok"})
	}))
	defer server.Close()

	c := New(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := c.Health(ctx)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestHealth_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{CFAPI: "ok", BOSHAPI: "ok"})
	}))
	defer server.Close()

	c := New(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := c.Health(ctx)
	if err == nil {
		t.Error("expected error for timed out context, got nil")
	}
}

func TestInfrastructureStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/infrastructure/status" {
			t.Errorf("expected path /api/infrastructure/status, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InfrastructureStatus{
			HasData:               true,
			Source:                "vsphere",
			Name:                  "vcenter.example.com",
			ClusterCount:          2,
			HostCount:             8,
			CellCount:             20,
			ConstrainingResource:  "memory",
			VSphereConfigured:     true,
			MemoryUtilization:     78.5,
			N1CapacityPercent:     72.0,
			HAMinFailuresSurvived: 1,
		})
	}))
	defer server.Close()

	c := New(server.URL)
	resp, err := c.InfrastructureStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.HasData {
		t.Error("expected HasData true")
	}
	if resp.Source != "vsphere" {
		t.Errorf("expected source vsphere, got %s", resp.Source)
	}
	if resp.CellCount != 20 {
		t.Errorf("expected 20 cells, got %d", resp.CellCount)
	}
}

func TestInfrastructureStatus_NoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InfrastructureStatus{
			HasData:           false,
			VSphereConfigured: false,
		})
	}))
	defer server.Close()

	c := New(server.URL)
	resp, err := c.InfrastructureStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.HasData {
		t.Error("expected HasData false")
	}
}

func TestInfrastructureStatus_ConnectionError(t *testing.T) {
	c := New("http://localhost:99999")
	_, err := c.InfrastructureStatus(context.Background())
	if err == nil {
		t.Error("expected connection error, got nil")
	}
}

func TestInfrastructureStatus_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InfrastructureStatus{HasData: true})
	}))
	defer server.Close()

	c := New(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := c.InfrastructureStatus(ctx)
	if err == nil {
		t.Error("expected error for canceled context, got nil")
	}
}

func TestGetInfrastructure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/infrastructure" {
			t.Errorf("expected path /api/infrastructure, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InfrastructureState{
			Source:         "vsphere",
			Name:           "vcenter.example.com",
			TotalHostCount: 4,
			TotalCellCount: 10,
		})
	}))
	defer server.Close()

	c := New(server.URL)
	infra, err := c.GetInfrastructure(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infra.Source != "vsphere" {
		t.Errorf("expected source vsphere, got %s", infra.Source)
	}
	if infra.TotalHostCount != 4 {
		t.Errorf("expected 4 hosts, got %d", infra.TotalHostCount)
	}
}

func TestSetManualInfrastructure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/infrastructure/manual" {
			t.Errorf("expected path /api/infrastructure/manual, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var input ManualInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if input.Name != "Test Infra" {
			t.Errorf("expected name 'Test Infra', got %s", input.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InfrastructureState{
			Source: "manual",
			Name:   input.Name,
		})
	}))
	defer server.Close()

	c := New(server.URL)
	input := &ManualInput{
		Name: "Test Infra",
		Clusters: []ClusterInput{{
			Name:              "cluster-1",
			HostCount:         4,
			MemoryGBPerHost:   256,
			CPUCoresPerHost:   32,
			DiegoCellCount:    10,
			DiegoCellMemoryGB: 64,
			DiegoCellCPU:      8,
		}},
	}

	infra, err := c.SetManualInfrastructure(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if infra.Source != "manual" {
		t.Errorf("expected source manual, got %s", infra.Source)
	}
}
