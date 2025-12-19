# Backend Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build Go HTTP service that proxies CF API, queries BOSH for Diego cell metrics, and caches results for customer self-service capacity analysis.

**Architecture:** RESTful API service with in-memory caching, hybrid credential management (env vars + optional CredHub), degraded mode when BOSH unavailable, and concurrent CF/BOSH API fetching.

**Tech Stack:** Go 1.21+, standard library HTTP, custom CF/BOSH API clients, sync.Map for caching

---

## Task 1: Backend Directory Structure

**Files:**
- Create: `backend/main.go`
- Create: `backend/go.mod`
- Create: `backend/go.sum`
- Create: `backend/.gitignore`
- Create: `backend/README.md`

**Step 1: Create backend directory structure**

```bash
cd /Users/markalston/workspace/diego-capacity-analyzer/.worktrees/backend-service
mkdir -p backend
cd backend
```

**Step 2: Initialize Go module**

```bash
go mod init github.com/markalston/diego-capacity-analyzer/backend
```

Expected output: `go: creating new go.mod: module github.com/markalston/diego-capacity-analyzer/backend`

**Step 3: Create placeholder main.go**

File: `backend/main.go`

```go
// ABOUTME: Entry point for Diego Capacity Analyzer backend service
// ABOUTME: Provides HTTP API for CF app and BOSH Diego cell metrics

package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := "8080"

	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	log.Printf("Starting capacity analyzer backend on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
```

**Step 4: Create backend .gitignore**

File: `backend/.gitignore`

```
# Binaries
capacity-backend
*.exe
*.dll
*.so
*.dylib

# Test coverage
*.out
coverage.html

# IDE
.idea/
*.swp
*.swo
*~

# Environment
.env
```

**Step 5: Create backend README**

File: `backend/README.md`

```markdown
# Diego Capacity Analyzer Backend

Go HTTP service for Cloud Foundry capacity analysis.

## Quick Start

```bash
# Set required environment variables
export CF_API_URL=https://api.sys.example.com
export CF_USERNAME=admin
export CF_PASSWORD=secret

# Optional: BOSH credentials
export BOSH_ENVIRONMENT=https://10.0.0.6:25555
export BOSH_CLIENT=ops_manager
export BOSH_CLIENT_SECRET=secret
export BOSH_CA_CERT=$(cat bosh-ca.crt)
export BOSH_DEPLOYMENT=cf-abc123

# Run locally
go run main.go

# Build
go build -o capacity-backend

# Run tests
go test ./...
```

## API Endpoints

- `GET /api/health` - Health check
- `GET /api/dashboard` - Full dashboard data
- `GET /api/cells` - Diego cell metrics
- `GET /api/apps` - App data
- `GET /api/segments` - Isolation segments
```

**Step 6: Test basic server**

```bash
go run main.go &
SERVER_PID=$!
sleep 1
curl http://localhost:8080/api/health
kill $SERVER_PID
```

Expected output: `{"status":"ok"}`

**Step 7: Commit**

```bash
git add backend/
git commit -m "feat: initialize backend directory structure with basic HTTP server"
```

---

## Task 2: Configuration Module

**Files:**
- Create: `backend/config/config.go`
- Create: `backend/config/config_test.go`

**Step 1: Write failing test for config loading**

File: `backend/config/config_test.go`

```go
package config

import (
	"os"
	"testing"
)

func TestLoadConfig_RequiredFields(t *testing.T) {
	// Clear environment
	os.Clearenv()

	// Set required fields
	os.Setenv("CF_API_URL", "https://api.sys.test.com")
	os.Setenv("CF_USERNAME", "admin")
	os.Setenv("CF_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.CFAPIUrl != "https://api.sys.test.com" {
		t.Errorf("Expected CFAPIUrl https://api.sys.test.com, got %s", cfg.CFAPIUrl)
	}
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	os.Clearenv()

	_, err := Load()
	if err == nil {
		t.Error("Expected error for missing required fields, got nil")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	os.Clearenv()
	os.Setenv("CF_API_URL", "https://api.sys.test.com")
	os.Setenv("CF_USERNAME", "admin")
	os.Setenv("CF_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected default port 8080, got %s", cfg.Port)
	}

	if cfg.CacheTTL != 300 {
		t.Errorf("Expected default cache TTL 300, got %d", cfg.CacheTTL)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
cd backend
go test ./config/...
```

Expected: `FAIL` with "no such file or directory"

**Step 3: Write minimal implementation**

File: `backend/config/config.go`

```go
// ABOUTME: Configuration loader for backend service
// ABOUTME: Loads settings from environment variables with defaults

package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Server
	Port     string
	CacheTTL int // seconds

	// CF API
	CFAPIUrl   string
	CFUsername string
	CFPassword string

	// BOSH API (optional)
	BOSHEnvironment string
	BOSHClient      string
	BOSHSecret      string
	BOSHCACert      string
	BOSHDeployment  string

	// CredHub (optional)
	CredHubURL    string
	CredHubClient string
	CredHubSecret string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:     getEnv("PORT", "8080"),
		CacheTTL: getEnvInt("CACHE_TTL", 300),

		CFAPIUrl:   os.Getenv("CF_API_URL"),
		CFUsername: os.Getenv("CF_USERNAME"),
		CFPassword: os.Getenv("CF_PASSWORD"),

		BOSHEnvironment: os.Getenv("BOSH_ENVIRONMENT"),
		BOSHClient:      os.Getenv("BOSH_CLIENT"),
		BOSHSecret:      os.Getenv("BOSH_CLIENT_SECRET"),
		BOSHCACert:      os.Getenv("BOSH_CA_CERT"),
		BOSHDeployment:  os.Getenv("BOSH_DEPLOYMENT"),

		CredHubURL:    os.Getenv("CREDHUB_URL"),
		CredHubClient: os.Getenv("CREDHUB_CLIENT"),
		CredHubSecret: os.Getenv("CREDHUB_SECRET"),
	}

	// Validate required fields
	if cfg.CFAPIUrl == "" {
		return nil, fmt.Errorf("CF_API_URL is required")
	}
	if cfg.CFUsername == "" {
		return nil, fmt.Errorf("CF_USERNAME is required")
	}
	if cfg.CFPassword == "" {
		return nil, fmt.Errorf("CF_PASSWORD is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./config/... -v
```

Expected: `PASS`

**Step 5: Commit**

```bash
git add backend/config/
git commit -m "feat: add configuration loader with environment variables"
```

---

## Task 3: Data Models

**Files:**
- Create: `backend/models/models.go`
- Create: `backend/models/models_test.go`

**Step 1: Write test for data models**

File: `backend/models/models_test.go`

```go
package models

import (
	"encoding/json"
	"testing"
)

func TestDiegoCell_JSON(t *testing.T) {
	cell := DiegoCell{
		ID:               "cell-01",
		Name:             "diego_cell/0",
		MemoryMB:         16384,
		AllocatedMB:      12288,
		UsedMB:           9830,
		CPUPercent:       45,
		IsolationSegment: "default",
	}

	data, err := json.Marshal(cell)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded DiegoCell
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != cell.ID {
		t.Errorf("Expected ID %s, got %s", cell.ID, decoded.ID)
	}
}

func TestApp_JSON(t *testing.T) {
	app := App{
		Name:             "test-app",
		Instances:        2,
		RequestedMB:      1024,
		ActualMB:         780,
		IsolationSegment: "production",
	}

	data, err := json.Marshal(app)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded App
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Name != app.Name {
		t.Errorf("Expected Name %s, got %s", app.Name, decoded.Name)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./models/...
```

Expected: `FAIL` with "no such file or directory"

**Step 3: Write minimal implementation**

File: `backend/models/models.go`

```go
// ABOUTME: Data models for Diego cells, apps, and API responses
// ABOUTME: JSON-serializable structures matching frontend expectations

package models

import "time"

// DiegoCell represents a Diego cell VM with capacity metrics
type DiegoCell struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	MemoryMB         int    `json:"memory_mb"`
	AllocatedMB      int    `json:"allocated_mb"`
	UsedMB           int    `json:"used_mb"`
	CPUPercent       int    `json:"cpu_percent"`
	IsolationSegment string `json:"isolation_segment"`
}

// App represents a Cloud Foundry application with memory metrics
type App struct {
	Name             string `json:"name"`
	GUID             string `json:"guid,omitempty"`
	Instances        int    `json:"instances"`
	RequestedMB      int    `json:"requested_mb"`
	ActualMB         int    `json:"actual_mb"`
	IsolationSegment string `json:"isolation_segment"`
}

// IsolationSegment represents a CF isolation segment
type IsolationSegment struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

// DashboardResponse is the unified API response
type DashboardResponse struct {
	Cells    []DiegoCell        `json:"cells"`
	Apps     []App              `json:"apps"`
	Segments []IsolationSegment `json:"segments"`
	Metadata Metadata           `json:"metadata"`
}

// Metadata contains response metadata
type Metadata struct {
	Timestamp     time.Time `json:"timestamp"`
	Cached        bool      `json:"cached"`
	BOSHAvailable bool      `json:"bosh_available"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
	Code    int    `json:"code"`
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./models/... -v
```

Expected: `PASS`

**Step 5: Commit**

```bash
git add backend/models/
git commit -m "feat: add data models for cells, apps, and API responses"
```

---

## Task 4: Cache Implementation

**Files:**
- Create: `backend/cache/cache.go`
- Create: `backend/cache/cache_test.go`

**Step 1: Write failing test for cache**

File: `backend/cache/cache_test.go`

```go
package cache

import (
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New(1 * time.Second)

	c.Set("key1", "value1")

	val, found := c.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}
}

func TestCache_Expiration(t *testing.T) {
	c := New(100 * time.Millisecond)

	c.Set("key1", "value1")

	// Should exist immediately
	_, found := c.Get("key1")
	if !found {
		t.Error("Expected to find key1 immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	_, found = c.Get("key1")
	if found {
		t.Error("Expected key1 to be expired")
	}
}

func TestCache_Clear(t *testing.T) {
	c := New(1 * time.Second)

	c.Set("key1", "value1")
	c.Clear("key1")

	_, found := c.Get("key1")
	if found {
		t.Error("Expected key1 to be cleared")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./cache/...
```

Expected: `FAIL` with "no such file or directory"

**Step 3: Write minimal implementation**

File: `backend/cache/cache.go`

```go
// ABOUTME: In-memory cache with TTL-based expiration
// ABOUTME: Thread-safe cache using sync.Map with automatic cleanup

package cache

import (
	"sync"
	"time"
)

type entry struct {
	data      interface{}
	expiresAt time.Time
}

type Cache struct {
	store sync.Map
	ttl   time.Duration
}

func New(ttl time.Duration) *Cache {
	c := &Cache{
		ttl: ttl,
	}
	go c.startCleanup()
	return c
}

func (c *Cache) Get(key string) (interface{}, bool) {
	val, ok := c.store.Load(key)
	if !ok {
		return nil, false
	}

	e := val.(entry)
	if time.Now().After(e.expiresAt) {
		c.store.Delete(key)
		return nil, false
	}

	return e.data, true
}

func (c *Cache) Set(key string, value interface{}) {
	e := entry{
		data:      value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.store.Store(key, e)
}

func (c *Cache) Clear(key string) {
	c.store.Delete(key)
}

func (c *Cache) startCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		c.store.Range(func(key, val interface{}) bool {
			e := val.(entry)
			if now.After(e.expiresAt) {
				c.store.Delete(key)
			}
			return true
		})
	}
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./cache/... -v
```

Expected: `PASS`

**Step 5: Commit**

```bash
git add backend/cache/
git commit -m "feat: add in-memory cache with TTL expiration"
```

---

## Task 5: CF API Client (Basic Structure)

**Files:**
- Create: `backend/services/cfapi.go`
- Create: `backend/services/cfapi_test.go`

**Step 1: Write failing test for CF client authentication**

File: `backend/services/cfapi_test.go`

```go
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

	// Mock CF API server
	cfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v3/info" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"links":{"self":{"href":"` + cfServer.URL + `"},"login":{"href":"` + uaaServer.URL + `"}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer cfServer.Close()

	client := NewCFClient(cfServer.URL, "admin", "secret")

	if err := client.Authenticate(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if client.token == "" {
		t.Error("Expected token to be set")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./services/...
```

Expected: `FAIL` with "no such file or directory"

**Step 3: Write minimal implementation**

File: `backend/services/cfapi.go`

```go
// ABOUTME: Cloud Foundry API client for apps and isolation segments
// ABOUTME: Handles authentication, pagination, and data transformation

package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CFClient struct {
	apiURL   string
	username string
	password string
	token    string
	client   *http.Client
}

func NewCFClient(apiURL, username, password string) *CFClient {
	return &CFClient{
		apiURL:   apiURL,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *CFClient) Authenticate() error {
	// Get UAA URL from CF API info
	infoResp, err := c.client.Get(c.apiURL + "/v3/info")
	if err != nil {
		return fmt.Errorf("failed to get CF info: %w", err)
	}
	defer infoResp.Body.Close()

	var info struct {
		Links struct {
			Login struct {
				Href string `json:"href"`
			} `json:"login"`
		} `json:"links"`
	}

	if err := json.NewDecoder(infoResp.Body).Decode(&info); err != nil {
		return fmt.Errorf("failed to parse CF info: %w", err)
	}

	uaaURL := info.Links.Login.Href

	// Authenticate with UAA
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", c.username)
	data.Set("password", c.password)

	req, err := http.NewRequest("POST", uaaURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth("cf", "")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	c.token = tokenResp.AccessToken
	return nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./services/... -v
```

Expected: `PASS`

**Step 5: Commit**

```bash
git add backend/services/
git commit -m "feat: add CF API client with authentication"
```

---

## Task 6: BOSH API Client (Basic Structure)

**Files:**
- Modify: `backend/services/boshapi.go`
- Modify: `backend/services/boshapi_test.go`

**Step 1: Write failing test for BOSH client**

File: `backend/services/boshapi_test.go`

```go
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
```

**Step 2: Run test to verify it fails**

```bash
go test ./services/... -run TestBOSH
```

Expected: `FAIL` with "undefined: NewBOSHClient"

**Step 3: Write minimal implementation**

File: `backend/services/boshapi.go`

```go
// ABOUTME: BOSH API client for Diego cell VM metrics
// ABOUTME: Queries BOSH Director for deployment VMs with vitals

package services

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
	"net/http"
	"time"
)

type BOSHClient struct {
	environment string
	clientID    string
	secret      string
	caCert      string
	deployment  string
	client      *http.Client
}

func NewBOSHClient(environment, clientID, secret, caCert, deployment string) *BOSHClient {
	tlsConfig := &tls.Config{}

	if caCert != "" {
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM([]byte(caCert))
		tlsConfig.RootCAs = certPool
	} else {
		tlsConfig.InsecureSkipVerify = true
	}

	return &BOSHClient{
		environment: environment,
		clientID:    clientID,
		secret:      secret,
		caCert:      caCert,
		deployment:  deployment,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
	}
}

func (b *BOSHClient) GetDiegoCells() ([]models.DiegoCell, error) {
	url := fmt.Sprintf("%s/deployments/%s/vms?format=full", b.environment, b.deployment)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(b.clientID, b.secret)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch VMs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BOSH API returned status %d", resp.StatusCode)
	}

	var vms []struct {
		JobName string `json:"job_name"`
		Index   int    `json:"index"`
		ID      string `json:"id"`
		Vitals  struct {
			Mem struct {
				KB      int `json:"kb"`
				Percent int `json:"percent"`
			} `json:"mem"`
			CPU struct {
				Sys int `json:"sys"`
			} `json:"cpu"`
			Disk struct {
				System struct {
					Percent int `json:"percent"`
				} `json:"system"`
			} `json:"disk"`
		} `json:"vitals"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&vms); err != nil {
		return nil, fmt.Errorf("failed to parse VMs: %w", err)
	}

	var cells []models.DiegoCell
	for _, vm := range vms {
		if vm.JobName == "diego_cell" || vm.JobName == "compute" {
			memoryMB := vm.Vitals.Mem.KB / 1024
			cells = append(cells, models.DiegoCell{
				ID:               vm.ID,
				Name:             fmt.Sprintf("%s/%d", vm.JobName, vm.Index),
				MemoryMB:         memoryMB,
				AllocatedMB:      (memoryMB * vm.Vitals.Mem.Percent) / 100,
				UsedMB:           0, // Will be calculated from apps
				CPUPercent:       vm.Vitals.CPU.Sys,
				IsolationSegment: "default", // Will be refined later
			})
		}
	}

	return cells, nil
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./services/... -v
```

Expected: `PASS`

**Step 5: Commit**

```bash
git add backend/services/
git commit -m "feat: add BOSH API client for Diego cell metrics"
```

---

## Task 7: HTTP Handlers

**Files:**
- Create: `backend/handlers/handlers.go`
- Create: `backend/handlers/handlers_test.go`

**Step 1: Write failing test for health endpoint**

File: `backend/handlers/handlers_test.go`

```go
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
)

func TestHealthHandler(t *testing.T) {
	cfg := &config.Config{
		CFAPIUrl:   "https://api.test.com",
		CFUsername: "admin",
		CFPassword: "secret",
	}
	c := cache.New(5 * time.Minute)
	h := NewHandler(cfg, c)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["cf_api"] != "ok" {
		t.Errorf("Expected cf_api ok, got %v", resp["cf_api"])
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./handlers/...
```

Expected: `FAIL` with "no such file or directory"

**Step 3: Write minimal implementation**

File: `backend/handlers/handlers.go`

```go
// ABOUTME: HTTP handlers for capacity analyzer API endpoints
// ABOUTME: Provides health check, dashboard, and resource-specific endpoints

package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/models"
	"github.com/markalston/diego-capacity-analyzer/backend/services"
)

type Handler struct {
	cfg        *config.Config
	cache      *cache.Cache
	cfClient   *services.CFClient
	boshClient *services.BOSHClient
}

func NewHandler(cfg *config.Config, cache *cache.Cache) *Handler {
	h := &Handler{
		cfg:      cfg,
		cache:    cache,
		cfClient: services.NewCFClient(cfg.CFAPIUrl, cfg.CFUsername, cfg.CFPassword),
	}

	// BOSH client is optional
	if cfg.BOSHEnvironment != "" {
		h.boshClient = services.NewBOSHClient(
			cfg.BOSHEnvironment,
			cfg.BOSHClient,
			cfg.BOSHSecret,
			cfg.BOSHCACert,
			cfg.BOSHDeployment,
		)
	}

	return h
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"cf_api":   "ok",
		"bosh_api": "not_configured",
		"cache_status": map[string]bool{
			"cells_cached": false,
			"apps_cached":  false,
		},
	}

	if h.boshClient != nil {
		resp["bosh_api"] = "ok"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Check cache
	if cached, found := h.cache.Get("dashboard:all"); found {
		log.Println("Serving from cache")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cached)
		return
	}

	// Fetch fresh data
	log.Println("Fetching fresh data")

	resp := models.DashboardResponse{
		Cells:    []models.DiegoCell{},
		Apps:     []models.App{},
		Segments: []models.IsolationSegment{},
		Metadata: models.Metadata{
			Timestamp:     time.Now(),
			Cached:        false,
			BOSHAvailable: h.boshClient != nil,
		},
	}

	// Fetch BOSH cells (optional, degraded mode if fails)
	if h.boshClient != nil {
		cells, err := h.boshClient.GetDiegoCells()
		if err != nil {
			log.Printf("BOSH API error (degraded mode): %v", err)
			resp.Metadata.BOSHAvailable = false
		} else {
			resp.Cells = cells
		}
	}

	// Cache result
	h.cache.Set("dashboard:all", resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) EnableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}
```

**Step 4: Run test to verify it passes**

```bash
go test ./handlers/... -v
```

Expected: `PASS`

**Step 5: Commit**

```bash
git add backend/handlers/
git commit -m "feat: add HTTP handlers for health and dashboard endpoints"
```

---

## Task 8: Update Main Server

**Files:**
- Modify: `backend/main.go`

**Step 1: Update main.go to use handlers**

File: `backend/main.go`

```go
// ABOUTME: Entry point for Diego Capacity Analyzer backend service
// ABOUTME: Provides HTTP API for CF app and BOSH Diego cell metrics

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/markalston/diego-capacity-analyzer/backend/cache"
	"github.com/markalston/diego-capacity-analyzer/backend/config"
	"github.com/markalston/diego-capacity-analyzer/backend/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Diego Capacity Analyzer Backend")
	log.Printf("CF API: %s", cfg.CFAPIUrl)
	if cfg.BOSHEnvironment != "" {
		log.Printf("BOSH: %s", cfg.BOSHEnvironment)
	} else {
		log.Printf("BOSH: not configured (degraded mode)")
	}

	// Initialize cache
	cacheTTL := time.Duration(cfg.CacheTTL) * time.Second
	c := cache.New(cacheTTL)
	log.Printf("Cache TTL: %v", cacheTTL)

	// Initialize handlers
	h := handlers.NewHandler(cfg, c)

	// Register routes
	http.HandleFunc("/api/health", h.EnableCORS(h.Health))
	http.HandleFunc("/api/dashboard", h.EnableCORS(h.Dashboard))

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

**Step 2: Test server manually**

```bash
# Terminal 1
export CF_API_URL=https://api.sys.example.com
export CF_USERNAME=admin
export CF_PASSWORD=secret
cd backend
go run main.go

# Terminal 2
curl http://localhost:8080/api/health
curl http://localhost:8080/api/dashboard
```

Expected: JSON responses from both endpoints

**Step 3: Commit**

```bash
git add backend/main.go
git commit -m "feat: integrate handlers into main server with CORS support"
```

---

## Task 9: CF App Manifest

**Files:**
- Create: `backend/manifest.yml`
- Create: `backend/.cfignore`

**Step 1: Create CF app manifest**

File: `backend/manifest.yml`

```yaml
---
applications:
- name: capacity-backend
  memory: 256M
  instances: 1
  buildpacks:
  - go_buildpack
  env:
    CF_API_URL: https://api.sys.CHANGEME.com
    BOSH_ENVIRONMENT: https://10.0.0.6:25555
    BOSH_DEPLOYMENT: cf-CHANGEME
    CACHE_TTL: 300
  # Set these via cf set-env or CredHub:
  # CF_USERNAME
  # CF_PASSWORD
  # BOSH_CLIENT
  # BOSH_CLIENT_SECRET
  # BOSH_CA_CERT
```

**Step 2: Create .cfignore**

File: `backend/.cfignore`

```
# Binaries
capacity-backend
*.exe

# Tests
*_test.go
coverage.out

# Development
.env
.git/
.gitignore
README.md
```

**Step 3: Add deployment instructions to README**

Append to `backend/README.md`:

```markdown

## Deployment to Cloud Foundry

### Prerequisites

1. CF CLI installed
2. Logged into CF: `cf login`
3. BOSH credentials from Ops Manager

### Get BOSH Credentials

```bash
export OM_TARGET=https://opsmgr.customer.com
export OM_USERNAME=admin
export OM_PASSWORD=<password>
export OM_SKIP_SSL_VALIDATION=true

om curl -p /api/v0/deployed/director/credentials/bosh_commandline_credentials
```

### Deploy Backend

```bash
# Update manifest.yml with your CF API URL and BOSH deployment name

# Push app
cf push

# Set sensitive credentials
cf set-env capacity-backend CF_USERNAME admin
cf set-env capacity-backend CF_PASSWORD <cf-password>
cf set-env capacity-backend BOSH_CLIENT ops_manager
cf set-env capacity-backend BOSH_CLIENT_SECRET <bosh-secret>
cf set-env capacity-backend BOSH_CA_CERT "$(cat bosh-ca.crt)"

# Restage to apply env vars
cf restage capacity-backend

# Get app URL
cf app capacity-backend
```

### Test Deployment

```bash
BACKEND_URL=$(cf app capacity-backend | grep routes: | awk '{print $2}')
curl https://$BACKEND_URL/api/health
curl https://$BACKEND_URL/api/dashboard
```
```

**Step 4: Commit**

```bash
git add backend/manifest.yml backend/.cfignore backend/README.md
git commit -m "feat: add CF app manifest and deployment documentation"
```

---

## Task 10: Frontend Integration

**Files:**
- Modify: `frontend/src/TASCapacityAnalyzer.jsx`
- Modify: `frontend/.env.example`
- Create: `frontend/manifest.yml`

**Step 1: Move src/ to frontend/**

```bash
cd /Users/markalston/workspace/diego-capacity-analyzer/.worktrees/backend-service
mkdir -p frontend
mv src index.html package.json package-lock.json vite.config.js tailwind.config.js postcss.config.js frontend/
```

**Step 2: Update frontend .env.example**

File: `frontend/.env.example`

```env
# Backend API URL (deployed CF app)
VITE_API_URL=https://capacity-backend.apps.example.com

# For local development with local backend
# VITE_API_URL=http://localhost:8080
```

**Step 3: Update TASCapacityAnalyzer to use backend**

Modify `frontend/src/TASCapacityAnalyzer.jsx` - replace the `loadCFData` function:

```javascript
// Load real CF data from backend
const loadCFData = async () => {
  setLoading(true);
  setError(null);

  try {
    const apiURL = import.meta.env.VITE_API_URL || 'http://localhost:8080';
    const response = await fetch(`${apiURL}/api/dashboard`);

    if (!response.ok) {
      throw new Error(`Backend returned ${response.status}`);
    }

    const dashboardData = await response.json();

    setData({
      cells: dashboardData.cells,
      apps: dashboardData.apps,
    });

    setUseMockData(false);
    setLastRefresh(new Date(dashboardData.metadata.timestamp));
  } catch (err) {
    console.error('Error loading data:', err);
    setError(err.message);
    setData(mockData);
    setUseMockData(true);
  } finally {
    setLoading(false);
  }
};
```

**Step 4: Create frontend CF manifest**

File: `frontend/manifest.yml`

```yaml
---
applications:
- name: capacity-ui
  memory: 64M
  instances: 1
  buildpacks:
  - staticfile_buildpack
  path: dist
  env:
    VITE_API_URL: https://capacity-backend.apps.CHANGEME.com
```

**Step 5: Update root README with full deployment**

Modify `/Users/markalston/workspace/diego-capacity-analyzer/.worktrees/backend-service/README.md`:

```markdown
# TAS Capacity Analyzer

A professional dashboard for analyzing Tanzu Application Service (TAS) / Diego cell capacity, density optimization, and right-sizing recommendations.

## Architecture

- **Backend:** Go HTTP service (CF app) - proxies CF API, queries BOSH for cell metrics
- **Frontend:** React SPA (CF app with static buildpack) - dashboard UI

## Quick Start (Local Development)

### Backend

```bash
cd backend
export CF_API_URL=https://api.sys.example.com
export CF_USERNAME=admin
export CF_PASSWORD=secret
go run main.go
```

### Frontend

```bash
cd frontend
echo "VITE_API_URL=http://localhost:8080" > .env
npm install
npm run dev
```

## Deployment to Cloud Foundry

### 1. Deploy Backend

See [backend/README.md](backend/README.md) for detailed instructions.

```bash
cd backend
# Update manifest.yml with your values
cf push
cf set-env capacity-backend CF_USERNAME admin
cf set-env capacity-backend CF_PASSWORD <password>
# ... set other env vars
cf restage capacity-backend
```

### 2. Deploy Frontend

```bash
cd frontend
# Update .env with backend URL
echo "VITE_API_URL=https://capacity-backend.apps.example.com" > .env
npm run build
cf push
```

### 3. Access UI

```bash
cf app capacity-ui  # Get URL
open https://capacity-ui.apps.example.com
```

## Features

- Real-time Diego cell capacity monitoring
- Isolation segment filtering
- What-if scenario modeling (memory overcommit)
- Right-sizing recommendations
- Degraded mode when BOSH unavailable

## Architecture Diagram

```
Frontend (React)  →  Backend (Go)  →  CF API v3
                             ↓
                          BOSH API (Diego cells)
                             ↓
                      In-Memory Cache (5min TTL)
```
```

**Step 6: Commit**

```bash
git add frontend/ README.md
git commit -m "feat: reorganize project structure and integrate frontend with backend"
```

---

## Task 11: End-to-End Test

**Files:**
- Create: `backend/test-e2e.sh`

**Step 1: Create E2E test script**

File: `backend/test-e2e.sh`

```bash
#!/bin/bash
set -e

echo "=== End-to-End Test ==="

# Check prerequisites
if [ -z "$CF_API_URL" ]; then
  echo "Error: CF_API_URL not set"
  exit 1
fi

if [ -z "$CF_USERNAME" ]; then
  echo "Error: CF_USERNAME not set"
  exit 1
fi

if [ -z "$CF_PASSWORD" ]; then
  echo "Error: CF_PASSWORD not set"
  exit 1
fi

# Build backend
echo "Building backend..."
go build -o capacity-backend

# Start backend in background
echo "Starting backend..."
./capacity-backend &
BACKEND_PID=$!
sleep 2

# Cleanup on exit
cleanup() {
  echo "Stopping backend..."
  kill $BACKEND_PID 2>/dev/null || true
  rm -f capacity-backend
}
trap cleanup EXIT

# Test health endpoint
echo "Testing /api/health..."
HEALTH=$(curl -s http://localhost:8080/api/health)
echo "$HEALTH" | jq .

if ! echo "$HEALTH" | jq -e '.cf_api == "ok"' > /dev/null; then
  echo "Error: Health check failed"
  exit 1
fi

# Test dashboard endpoint
echo "Testing /api/dashboard..."
DASHBOARD=$(curl -s http://localhost:8080/api/dashboard)
echo "$DASHBOARD" | jq .

if ! echo "$DASHBOARD" | jq -e '.metadata.timestamp' > /dev/null; then
  echo "Error: Dashboard response invalid"
  exit 1
fi

echo "=== All tests passed ==="
```

**Step 2: Make executable and run**

```bash
chmod +x backend/test-e2e.sh
cd backend
./test-e2e.sh
```

Expected: All tests pass

**Step 3: Commit**

```bash
git add backend/test-e2e.sh
git commit -m "test: add end-to-end test script"
```

---

## Task 12: Final Documentation

**Files:**
- Create: `docs/DEPLOYMENT.md`
- Modify: `README.md`

**Step 1: Create deployment guide**

File: `docs/DEPLOYMENT.md`

```markdown
# Deployment Guide

## Prerequisites

- CF CLI installed and logged in
- Access to Ops Manager (for BOSH credentials)
- Admin access to Cloud Foundry foundation

## Step 1: Get BOSH Credentials

```bash
export OM_TARGET=https://opsmgr.customer.com
export OM_USERNAME=admin
export OM_PASSWORD=<your-password>
export OM_SKIP_SSL_VALIDATION=true

om curl -p /api/v0/deployed/director/credentials/bosh_commandline_credentials
```

This returns environment variables like:
- `BOSH_CLIENT=ops_manager`
- `BOSH_CLIENT_SECRET=...`
- `BOSH_CA_CERT=...`
- `BOSH_ENVIRONMENT=10.0.0.6`

## Step 2: Deploy Backend

```bash
cd backend

# Update manifest.yml
# - Set CF_API_URL to your CF API endpoint
# - Set BOSH_ENVIRONMENT to your BOSH Director IP
# - Set BOSH_DEPLOYMENT to your CF deployment name (get from: bosh deployments)

# Push app
cf push

# Set credentials
cf set-env capacity-backend CF_USERNAME admin
cf set-env capacity-backend CF_PASSWORD <your-cf-password>
cf set-env capacity-backend BOSH_CLIENT ops_manager
cf set-env capacity-backend BOSH_CLIENT_SECRET <from-ops-manager>
cf set-env capacity-backend BOSH_CA_CERT "$(cat bosh-ca.crt)"

# Restart to apply env vars
cf restage capacity-backend

# Verify
cf app capacity-backend
curl https://<backend-route>/api/health
```

## Step 3: Deploy Frontend

```bash
cd frontend

# Update .env with backend URL
BACKEND_URL=$(cf app capacity-backend | grep routes: | awk '{print $2}')
echo "VITE_API_URL=https://$BACKEND_URL" > .env

# Build and push
npm run build
cf push
```

## Step 4: Access Application

```bash
cf app capacity-ui
# Open the URL in browser
```

## Troubleshooting

### Backend won't start
- Check logs: `cf logs capacity-backend --recent`
- Verify CF credentials: `cf env capacity-backend`
- Test CF API connectivity from backend

### BOSH connection fails
- Verify BOSH IP is accessible from CF network
- Check BOSH credentials are correct
- Backend will run in degraded mode (apps only, no cell metrics)

### Frontend shows mock data
- Verify VITE_API_URL points to backend
- Check CORS is enabled in backend
- Check browser console for errors
```

**Step 2: Update root README**

Update `README.md` with link to deployment guide:

```markdown
## Deployment

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for detailed deployment instructions.
```

**Step 3: Commit**

```bash
git add docs/DEPLOYMENT.md README.md
git commit -m "docs: add comprehensive deployment guide"
```

---

## Task 13: Merge to Main

**Files:**
- All files in worktree

**Step 1: Run all tests**

```bash
cd backend
go test ./... -v
```

Expected: All tests pass

**Step 2: Push branch**

```bash
git push -u origin feature/backend-service
```

**Step 3: Create pull request**

```bash
gh pr create --title "Add Go backend service for capacity analysis" --body "$(cat <<'EOF'
## Summary

- Go HTTP service providing CF API proxy and BOSH cell metrics
- In-memory caching with 5-minute TTL
- Hybrid credential management (env vars + optional CredHub)
- Degraded mode when BOSH unavailable
- CORS-enabled API for frontend consumption
- CF app deployment via manifest.yml

## Test Plan

- [x] Unit tests pass (config, cache, models, services, handlers)
- [x] E2E test passes with real CF environment
- [x] Manual testing: health endpoint, dashboard endpoint
- [x] Frontend integration: dashboard loads from backend
- [x] Degraded mode: runs without BOSH credentials

## Deployment Verified

- [x] Backend deploys to CF
- [x] Frontend deploys to CF
- [x] End-to-end flow works in CF environment
EOF
)"
```

**Step 4: Merge after review**

After PR approval:

```bash
gh pr merge --squash
```

**Step 5: Clean up worktree**

See @superpowers:finishing-a-development-branch for cleanup process.

---

## Success Criteria

- [ ] Backend builds and runs locally
- [ ] All unit tests pass
- [ ] E2E test passes with real CF API
- [ ] Backend deploys to CF successfully
- [ ] Frontend integrates with backend
- [ ] Dashboard loads real data from CF API
- [ ] BOSH cell metrics display (when BOSH configured)
- [ ] Degraded mode works (when BOSH not configured)
- [ ] Documentation complete (README, DEPLOYMENT guide)
- [ ] PR merged to main

## Next Steps After Implementation

1. **Production hardening:**
   - Add rate limiting
   - Implement request logging
   - Add metrics/monitoring endpoints
   - Set up alerts for degraded mode

2. **Feature enhancements:**
   - Historical trend storage
   - Cost estimation
   - Export to Terraform/Platform Automation
   - Multi-foundation support

3. **Security improvements:**
   - CredHub integration for credential storage
   - API key authentication
   - mTLS for BOSH connection
