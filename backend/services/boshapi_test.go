package services

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBOSHClient_GetDiegoCells(t *testing.T) {
	// Mock BOSH server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/deployments/cf-test/vms" && r.URL.Query().Get("format") == "full" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[
				{
					"job_name": "diego_cell",
					"index": 0,
					"id": "cell-01",
					"vitals": {
						"mem": {"kb": 16777216, "percent": 60},
						"cpu": {"sys": 45},
						"disk": {"system": {"percent": 30}}
					}
				},
				{
					"job_name": "router",
					"index": 0,
					"id": "router-01"
				}
			]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
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

	if cells[0].Name != "diego_cell/0" {
		t.Errorf("Expected diego_cell/0, got %s", cells[0].Name)
	}
}
