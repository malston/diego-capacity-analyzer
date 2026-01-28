package services

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// Security Tests - Issue #70: SSH Private Key Path Traversal Vulnerability

func TestValidateSSHKeyPath_RejectsPathTraversal(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "simple traversal",
			path:      "../../../etc/passwd",
			wantError: true,
		},
		{
			name:      "traversal in middle",
			path:      "/home/user/../../../etc/shadow",
			wantError: true,
		},
		{
			name:      "encoded traversal",
			path:      "/home/user/..%2F..%2F..%2Fetc%2Fpasswd",
			wantError: true, // After URL decoding, this becomes ../
		},
		{
			name:      "dot-dot at end",
			path:      "/home/user/..",
			wantError: true,
		},
		{
			name:      "traversal via symlink attempt",
			path:      "/tmp/../../etc/passwd",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateSSHKeyPath(tt.path)
			if tt.wantError && err == nil {
				t.Errorf("ValidateSSHKeyPath(%q) should return error for path traversal", tt.path)
			}
		})
	}
}

func TestValidateSSHKeyPath_AcceptsValidPaths(t *testing.T) {
	// Create a temporary file to test with
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	if err := os.WriteFile(keyPath, []byte("test-key-content"), 0600); err != nil {
		t.Fatalf("Failed to create test key file: %v", err)
	}

	// Test that valid absolute path to existing file is accepted
	validPath, err := ValidateSSHKeyPath(keyPath)
	if err != nil {
		t.Errorf("ValidateSSHKeyPath(%q) returned unexpected error: %v", keyPath, err)
	}
	if validPath == "" {
		t.Error("ValidateSSHKeyPath should return non-empty path for valid file")
	}
}

func TestValidateSSHKeyPath_RejectsDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Test that directory path is rejected
	_, err := ValidateSSHKeyPath(tmpDir)
	if err == nil {
		t.Error("ValidateSSHKeyPath should reject directory paths")
	}
}

func TestValidateSSHKeyPath_RejectsNonExistent(t *testing.T) {
	// Test that non-existent file is rejected
	_, err := ValidateSSHKeyPath("/nonexistent/path/to/key")
	if err == nil {
		t.Error("ValidateSSHKeyPath should reject non-existent paths")
	}
}

func TestCreateSOCKS5DialContextFunc_RejectsTraversalInProxy(t *testing.T) {
	// Test that path traversal in BOSH_ALL_PROXY private-key param is rejected
	maliciousProxy := "socks5://user@host:1080?private-key=../../../etc/passwd"

	dialer := createSOCKS5DialContextFunc(maliciousProxy)

	// Should return nil dialer due to path validation failure
	if dialer != nil {
		t.Error("createSOCKS5DialContextFunc should return nil for path traversal in private-key")
	}
}
