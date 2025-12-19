package services

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBOSHClient_GetDiegoCells(t *testing.T) {
	taskDone := false

	// Mock BOSH server with UAA and task endpoints
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/info":
			// Return BOSH info with UAA URL pointing to this server
			uaaURL := "https://" + r.Host
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name": "test-bosh",
				"user_authentication": map[string]interface{}{
					"type": "uaa",
					"options": map[string]interface{}{
						"url": uaaURL,
					},
				},
			})
		case "/oauth/token":
			// Return a fake token
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-token",
				"token_type":   "bearer",
				"expires_in":   3600,
			})
		case "/deployments":
			// Return list of deployments
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"name": "cf-test"},
			})
		case "/deployments/cf-test/vms":
			if r.URL.Query().Get("format") == "full" {
				// Check for Bearer token
				if r.Header.Get("Authorization") != "Bearer test-token" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				// Return a task object
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":          123,
					"state":       "queued",
					"description": "retrieve vm-stats",
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		case "/tasks/123":
			// Return task status
			if !taskDone {
				taskDone = true
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":    123,
					"state": "processing",
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":    123,
					"state": "done",
				})
			}
		case "/tasks/123/output":
			if r.URL.Query().Get("type") == "result" {
				// Return NDJSON output
				w.Write([]byte(`{"job_name":"diego_cell","index":0,"id":"cell-01","vitals":{"mem":{"kb":"16777216","percent":"60"},"cpu":{"sys":"45","user":"10","wait":"2"},"disk":{"system":{"percent":"30"}}}}
{"job_name":"router","index":0,"id":"router-01","vitals":{"mem":{"kb":"4194304","percent":"40"},"cpu":{"sys":"20","user":"5","wait":"1"},"disk":{"system":{"percent":"20"}}}}
`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewBOSHClient(server.URL, "ops_manager", "secret", "", "cf-test")

	// Disable TLS verification for test
	client.client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	cells, err := client.GetDiegoCells()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(cells) != 1 {
		t.Errorf("Expected 1 diego cell, got %d", len(cells))
	}

	if len(cells) > 0 && cells[0].Name != "diego_cell/0" {
		t.Errorf("Expected diego_cell/0, got %s", cells[0].Name)
	}
}
