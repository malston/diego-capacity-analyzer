package services

import (
	"net/http"
	"net/http/httptest"
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
