package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCFClient_Authenticate(t *testing.T) {
	// Mock UAA server
	uaaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"test-token","token_type":"bearer"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer uaaServer.Close()

	// Mock CF API server - use a variable to avoid closure issues
	var cfServerURL string
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v3/info" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"links":{"self":{"href":"` + cfServerURL + `"},"login":{"href":"` + uaaServer.URL + `"}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer cfServer.Close()
	cfServerURL = cfServer.URL

	client := NewCFClient(cfServer.URL, "admin", "secret", true)

	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client.token == "" {
		t.Error("Expected token to be set")
	}
}

func TestCFClient_GetApps(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/v3/apps":
			// Return paginated apps
			if strings.Contains(r.URL.RawQuery, "page=2") {
				// Second page - no more results
				w.Write([]byte(`{
					"resources": [],
					"pagination": {
						"next": null
					}
				}`))
			} else {
				// First page
				w.Write([]byte(`{
					"resources": [
						{
							"guid": "a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1",
							"name": "test-app-1",
							"state": "STARTED",
							"relationships": {
								"space": {
									"data": {"guid": "11111111-1111-1111-1111-111111111111"}
								}
							}
						},
						{
							"guid": "b2b2b2b2-b2b2-b2b2-b2b2-b2b2b2b2b2b2",
							"name": "test-app-2",
							"state": "STARTED",
							"relationships": {
								"space": {
									"data": {"guid": "22222222-2222-2222-2222-222222222222"}
								}
							}
						}
					],
					"pagination": {
						"next": {
							"href": "` + serverURL + `/v3/apps?page=2"
						}
					}
				}`))
			}

		case strings.HasPrefix(r.URL.Path, "/v3/apps/") && strings.HasSuffix(r.URL.Path, "/processes"):
			// Return process info based on app GUID
			appGUID := strings.TrimPrefix(r.URL.Path, "/v3/apps/")
			appGUID = strings.TrimSuffix(appGUID, "/processes")

			if appGUID == "a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1" {
				w.Write([]byte(`{
					"resources": [
						{
							"type": "web",
							"instances": 2,
							"memory_in_mb": 512
						}
					]
				}`))
			} else if appGUID == "b2b2b2b2-b2b2-b2b2-b2b2-b2b2b2b2b2b2" {
				w.Write([]byte(`{
					"resources": [
						{
							"type": "web",
							"instances": 3,
							"memory_in_mb": 1024
						}
					]
				}`))
			}

		case strings.HasPrefix(r.URL.Path, "/v3/spaces/") && strings.HasSuffix(r.URL.Path, "/relationships/isolation_segment"):
			// Return isolation segment relationship
			spaceGUID := strings.TrimPrefix(r.URL.Path, "/v3/spaces/")
			spaceGUID = strings.TrimSuffix(spaceGUID, "/relationships/isolation_segment")

			if spaceGUID == "11111111-1111-1111-1111-111111111111" {
				w.Write([]byte(`{
					"data": {"guid": "cccccccc-cccc-cccc-cccc-cccccccccccc"}
				}`))
			} else {
				// 22222222-2222-2222-2222-222222222222 has no isolation segment
				w.Write([]byte(`{
					"data": null
				}`))
			}

		case strings.HasPrefix(r.URL.Path, "/v3/isolation_segments/"):
			isoSegGUID := strings.TrimPrefix(r.URL.Path, "/v3/isolation_segments/")
			if isoSegGUID == "cccccccc-cccc-cccc-cccc-cccccccccccc" {
				w.Write([]byte(`{
					"guid": "cccccccc-cccc-cccc-cccc-cccccccccccc",
					"name": "production"
				}`))
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewCFClient(server.URL, "admin", "secret", true)
	client.token = "test-token"

	apps, err := client.GetApps(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(apps) != 2 {
		t.Fatalf("Expected 2 apps, got %d", len(apps))
	}

	// Verify first app
	if apps[0].Name != "test-app-1" {
		t.Errorf("Expected app name 'test-app-1', got '%s'", apps[0].Name)
	}
	if apps[0].GUID != "a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1" {
		t.Errorf("Expected app GUID 'a1a1a1a1-a1a1-a1a1-a1a1-a1a1a1a1a1a1', got '%s'", apps[0].GUID)
	}
	if apps[0].Instances != 2 {
		t.Errorf("Expected 2 instances, got %d", apps[0].Instances)
	}
	if apps[0].RequestedMB != 1024 {
		t.Errorf("Expected 1024 MB (2 * 512), got %d", apps[0].RequestedMB)
	}
	if apps[0].IsolationSegment != "production" {
		t.Errorf("Expected isolation segment 'production', got '%s'", apps[0].IsolationSegment)
	}

	// Verify second app
	if apps[1].Name != "test-app-2" {
		t.Errorf("Expected app name 'test-app-2', got '%s'", apps[1].Name)
	}
	if apps[1].Instances != 3 {
		t.Errorf("Expected 3 instances, got %d", apps[1].Instances)
	}
	if apps[1].RequestedMB != 3072 {
		t.Errorf("Expected 3072 MB (3 * 1024), got %d", apps[1].RequestedMB)
	}
	if apps[1].IsolationSegment != "default" {
		t.Errorf("Expected 'default' isolation segment, got '%s'", apps[1].IsolationSegment)
	}
}

func TestCFClient_GetIsolationSegments(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/v3/isolation_segments" {
			if strings.Contains(r.URL.RawQuery, "page=2") {
				// Second page - no more results
				w.Write([]byte(`{
					"resources": [],
					"pagination": {
						"next": null
					}
				}`))
			} else {
				// First page
				w.Write([]byte(`{
					"resources": [
						{
							"guid": "cccccccc-cccc-cccc-cccc-cccccccccccc",
							"name": "production"
						},
						{
							"guid": "dddddddd-dddd-dddd-dddd-dddddddddddd",
							"name": "development"
						}
					],
					"pagination": {
						"next": {
							"href": "` + serverURL + `/v3/isolation_segments?page=2"
						}
					}
				}`))
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewCFClient(server.URL, "admin", "secret", true)
	client.token = "test-token"

	segments, err := client.GetIsolationSegments(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(segments) != 2 {
		t.Fatalf("Expected 2 isolation segments, got %d", len(segments))
	}

	// Verify first segment
	if segments[0].GUID != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Errorf("Expected GUID 'cccccccc-cccc-cccc-cccc-cccccccccccc', got '%s'", segments[0].GUID)
	}
	if segments[0].Name != "production" {
		t.Errorf("Expected name 'production', got '%s'", segments[0].Name)
	}

	// Verify second segment
	if segments[1].GUID != "dddddddd-dddd-dddd-dddd-dddddddddddd" {
		t.Errorf("Expected GUID 'dddddddd-dddd-dddd-dddd-dddddddddddd', got '%s'", segments[1].GUID)
	}
	if segments[1].Name != "development" {
		t.Errorf("Expected name 'development', got '%s'", segments[1].Name)
	}
}

func TestCFClient_GetApps_Unauthenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewCFClient(server.URL, "admin", "secret", true)
	// Don't set token

	_, err := client.GetApps(context.Background())
	if err == nil {
		t.Error("Expected error for unauthenticated request")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("Expected 'not authenticated' error, got %v", err)
	}
}

func TestCFClient_GetIsolationSegments_Unauthenticated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewCFClient(server.URL, "admin", "secret", true)
	// Don't set token

	_, err := client.GetIsolationSegments(context.Background())
	if err == nil {
		t.Error("Expected error for unauthenticated request")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("Expected 'not authenticated' error, got %v", err)
	}
}

func TestCFClient_Authenticate_CancelledContext(t *testing.T) {
	// Server that responds slowly to simulate a real API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v3/info" {
			w.Write([]byte(`{"links":{"login":{"href":"` + "http://localhost:0" + `"}}}`))
		}
	}))
	defer server.Close()

	client := NewCFClient(server.URL, "admin", "secret", true)

	// Cancel context before the call
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Authenticate(ctx)
	if err == nil {
		t.Fatal("Expected error when context is cancelled, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestCFClient_GetApps_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"resources":[],"pagination":{"next":null}}`))
	}))
	defer server.Close()

	client := NewCFClient(server.URL, "admin", "secret", true)
	client.token = "test-token"

	// Cancel context before the call
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetApps(ctx)
	if err == nil {
		t.Fatal("Expected error when context is cancelled, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestCFClient_GetIsolationSegments_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"resources":[],"pagination":{"next":null}}`))
	}))
	defer server.Close()

	client := NewCFClient(server.URL, "admin", "secret", true)
	client.token = "test-token"

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetIsolationSegments(ctx)
	if err == nil {
		t.Fatal("Expected error when context is cancelled, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestCFClient_Authenticate_ContextTimeout(t *testing.T) {
	// Server that delays longer than the context deadline
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v3/info" {
			w.Write([]byte(`{"links":{"login":{"href":"` + serverURL + `"}}}`))
			return
		}
		if r.URL.Path == "/oauth/token" {
			w.Write([]byte(`{"access_token":"test-token","token_type":"bearer"}`))
			return
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewCFClient(server.URL, "admin", "secret", true)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := client.Authenticate(ctx)
	if err == nil {
		t.Fatal("Expected error when context times out, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
	}
}
