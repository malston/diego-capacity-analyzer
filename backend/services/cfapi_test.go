package services

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	client := NewCFClient(cfServer.URL, "admin", "secret")

	if err := client.Authenticate(); err != nil {
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
							"guid": "app-1",
							"name": "test-app-1",
							"state": "STARTED",
							"relationships": {
								"space": {
									"data": {"guid": "space-1"}
								}
							}
						},
						{
							"guid": "app-2",
							"name": "test-app-2",
							"state": "STARTED",
							"relationships": {
								"space": {
									"data": {"guid": "space-2"}
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

			if appGUID == "app-1" {
				w.Write([]byte(`{
					"resources": [
						{
							"type": "web",
							"instances": 2,
							"memory_in_mb": 512
						}
					]
				}`))
			} else if appGUID == "app-2" {
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

			if spaceGUID == "space-1" {
				w.Write([]byte(`{
					"data": {"guid": "iso-seg-1"}
				}`))
			} else {
				// space-2 has no isolation segment
				w.Write([]byte(`{
					"data": null
				}`))
			}

		case strings.HasPrefix(r.URL.Path, "/v3/isolation_segments/"):
			isoSegGUID := strings.TrimPrefix(r.URL.Path, "/v3/isolation_segments/")
			if isoSegGUID == "iso-seg-1" {
				w.Write([]byte(`{
					"guid": "iso-seg-1",
					"name": "production"
				}`))
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	client := NewCFClient(server.URL, "admin", "secret")
	client.token = "test-token"

	apps, err := client.GetApps()
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
	if apps[0].GUID != "app-1" {
		t.Errorf("Expected app GUID 'app-1', got '%s'", apps[0].GUID)
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
	if apps[1].IsolationSegment != "" {
		t.Errorf("Expected empty isolation segment, got '%s'", apps[1].IsolationSegment)
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
							"guid": "iso-seg-1",
							"name": "production"
						},
						{
							"guid": "iso-seg-2",
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

	client := NewCFClient(server.URL, "admin", "secret")
	client.token = "test-token"

	segments, err := client.GetIsolationSegments()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(segments) != 2 {
		t.Fatalf("Expected 2 isolation segments, got %d", len(segments))
	}

	// Verify first segment
	if segments[0].GUID != "iso-seg-1" {
		t.Errorf("Expected GUID 'iso-seg-1', got '%s'", segments[0].GUID)
	}
	if segments[0].Name != "production" {
		t.Errorf("Expected name 'production', got '%s'", segments[0].Name)
	}

	// Verify second segment
	if segments[1].GUID != "iso-seg-2" {
		t.Errorf("Expected GUID 'iso-seg-2', got '%s'", segments[1].GUID)
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

	client := NewCFClient(server.URL, "admin", "secret")
	// Don't set token

	_, err := client.GetApps()
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

	client := NewCFClient(server.URL, "admin", "secret")
	// Don't set token

	_, err := client.GetIsolationSegments()
	if err == nil {
		t.Error("Expected error for unauthenticated request")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("Expected 'not authenticated' error, got %v", err)
	}
}
