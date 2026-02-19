package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLogCacheClient_GetAppMemoryMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"envelopes": {
				"batch": [
					{
						"timestamp": "1000000000",
						"source_id": "instance-0",
						"gauge": {
							"metrics": {
								"memory": {"value": 104857600, "unit": "bytes"}
							}
						}
					},
					{
						"timestamp": "1000000001",
						"source_id": "instance-1",
						"gauge": {
							"metrics": {
								"memory": {"value": 209715200, "unit": "bytes"}
							}
						}
					}
				]
			}
		}`))
	}))
	defer server.Close()

	// Construct client pointing at test server (override the URL derivation)
	client := &LogCacheClient{
		logCacheURL: server.URL,
		token:       "test-token",
		client:      server.Client(),
	}

	metrics, err := client.GetAppMemoryMetrics(context.Background(), "a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if metrics.InstanceCount != 2 {
		t.Errorf("Expected 2 instances, got %d", metrics.InstanceCount)
	}
	if metrics.MemoryBytesAvg == 0 {
		t.Error("Expected non-zero MemoryBytesAvg")
	}
}

func TestLogCacheClient_GetAppMemoryMetrics_Unauthenticated(t *testing.T) {
	client := &LogCacheClient{
		logCacheURL: "http://localhost:0",
		token:       "",
		client:      http.DefaultClient,
	}

	_, err := client.GetAppMemoryMetrics(context.Background(), "some-guid")
	if err == nil {
		t.Error("Expected error for unauthenticated request")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("Expected 'not authenticated' error, got %v", err)
	}
}

func TestLogCacheClient_GetAppMemoryMetrics_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"envelopes":{"batch":[]}}`))
	}))
	defer server.Close()

	client := &LogCacheClient{
		logCacheURL: server.URL,
		token:       "test-token",
		client:      server.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetAppMemoryMetrics(ctx, "a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1")
	if err == nil {
		t.Fatal("Expected error when context is cancelled, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context canceled error, got: %v", err)
	}
}

func TestLogCacheClient_GetAppMemoryMetrics_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"envelopes":{"batch":[]}}`))
	}))
	defer server.Close()

	client := &LogCacheClient{
		logCacheURL: server.URL,
		token:       "test-token",
		client:      server.Client(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.GetAppMemoryMetrics(ctx, "a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1")
	if err == nil {
		t.Fatal("Expected error when context times out, got nil")
	}
}

func TestLogCacheClient_GetAppMemoryPromQL_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"result":[]}}`))
	}))
	defer server.Close()

	client := &LogCacheClient{
		logCacheURL: server.URL,
		token:       "test-token",
		client:      server.Client(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetAppMemoryPromQL(ctx, "a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1")
	if err == nil {
		t.Fatal("Expected error when context is cancelled, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected context canceled error, got: %v", err)
	}
}
