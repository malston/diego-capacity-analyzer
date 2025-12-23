# Backend Service Design - Diego Capacity Analyzer

**Date:** 2024-12-18
**Status:** Approved
**Purpose:** Add Go backend service to enable customer self-service deployment with real CF API and BOSH API integration

## Overview

This design adds a Go-based backend API service to the Diego Capacity Analyzer to solve CORS limitations and provide access to Diego cell metrics via BOSH API. The backend will run as a Cloud Foundry application alongside the React frontend.

## Problem Statement

The current React-only implementation cannot:

- Access CF API from browser due to CORS restrictions
- Fetch Diego cell metrics (not available via CF API)
- Provide customer self-service deployment (requires backend for BOSH access)

## Solution

Add a lightweight Go HTTP service that:

- Proxies CF API requests (solves CORS)
- Queries BOSH API for Diego cell metrics
- Caches results to reduce API load
- Runs as a CF app in customer environment
- Supports degraded mode when BOSH unavailable

## Architecture

### Project Structure

```
diego-capacity-analyzer/
├── frontend/                 # React app (existing src/ moved here)
│   ├── src/
│   ├── public/
│   ├── package.json
│   ├── vite.config.js
│   └── .env.example
├── backend/                  # New Go API service
│   ├── main.go              # Entry point, HTTP server
│   ├── handlers/            # HTTP request handlers
│   ├── services/            # Business logic (CF API, BOSH API)
│   ├── models/              # Data structures
│   ├── cache/               # In-memory cache with TTL
│   ├── config/              # Environment config loader
│   ├── go.mod
│   └── manifest.yml         # CF app manifest
├── docs/
│   └── plans/
│       └── 2024-12-18-backend-design.md
└── README.md                # Updated with backend setup
```

### Deployment Model

- **Backend:** Deploys as CF app (`cf push capacity-backend`)
- **Frontend:** Deploys as CF app with static buildpack or static hosting
- **Communication:** Frontend calls backend via `VITE_API_URL=https://capacity-backend.apps.customer.com`

### Technology Stack

- **Language:** Go 1.21+
- **Dependencies:** Standard library for HTTP, custom CF/BOSH API clients
- **Cache:** In-memory with `sync.Map` and TTL-based cleanup
- **Database:** None (stateless service)

## API Design

### Endpoints

#### Health Check

```
GET /api/health
```

Returns CF and BOSH connectivity status.

**Response:**

```json
{
  "cf_api": "ok" | "error",
  "bosh_api": "ok" | "degraded" | "error",
  "cache_status": {
    "cells_cached": true,
    "apps_cached": true
  }
}
```

#### Unified Dashboard Data

```
GET /api/dashboard
```

Returns all data needed for dashboard in single request.

**Response:**

```json
{
  "cells": [...],
  "apps": [...],
  "segments": [...],
  "metadata": {
    "timestamp": "2024-12-18T10:30:00Z",
    "cached": true,
    "bosh_available": true
  }
}
```

#### Individual Resources

```
GET /api/cells       # Diego cell metrics only
GET /api/apps        # App data only
GET /api/segments    # Isolation segments only
```

Allows granular refresh of specific resources.

### Data Flow

1. Frontend loads → calls `GET /api/dashboard`
2. Backend checks cache (5min TTL)
   - If cached and fresh → return immediately
   - If stale or empty → fetch from APIs
3. Backend fetches in parallel:
   - CF API: apps, processes, stats, isolation segments
   - BOSH API: VMs with vitals (retry 3x with backoff)
4. If BOSH fails after retries → degraded mode (apps only)
5. Cache results, return to frontend
6. Frontend renders dashboard

## CF API Integration

### Service Layer (`backend/services/cfapi.go`)

```go
type CFClient struct {
    apiURL   string
    username string
    password string
    token    string
    client   *http.Client
}

// Key methods:
func (c *CFClient) Authenticate() error
func (c *CFClient) GetApps() ([]App, error)
func (c *CFClient) GetProcessStats(appGUID string) ([]ProcessStats, error)
func (c *CFClient) GetIsolationSegments() ([]IsolationSegment, error)
```

### Strategy

- Use CF API v3 endpoints
- Handle pagination automatically (`/v3/apps?per_page=5000`)
- Fetch app processes and stats in parallel (goroutines with sync.WaitGroup)
- Map space GUIDs → isolation segment names via `/v3/spaces/{guid}`

## BOSH API Integration

### Service Layer (`backend/services/boshapi.go`)

```go
type BOSHClient struct {
    environment string
    client      string
    secret      string
    caCert      string
    deployment  string
    httpClient  *http.Client
}

// Key methods:
func (b *BOSHClient) Authenticate() error
func (b *BOSHClient) GetDiegoCells() ([]DiegoCell, error)
```

### Strategy

- Use `/deployments/{name}/vms?format=full` endpoint
- Parse vitals: `vm.vitals.mem`, `vm.vitals.cpu`, `vm.vitals.disk`
- Filter for job names: `diego_cell` or `compute` (Small Footprint)
- Retry logic: 3 attempts with exponential backoff (1s, 2s, 4s)
- If all retries fail → return empty cells array, log error, continue degraded

### Error Handling

- **CF API errors:** Fail entire request (can't show apps without CF API)
- **BOSH API errors:** Log warning, continue without cell data (degraded mode)
- **Network timeouts:** 30 seconds for both CF and BOSH

## Configuration & Credentials

### Environment Variables

```go
type Config struct {
    // Server config
    Port        string  // Default: 8080
    CacheTTL    int     // Default: 300 seconds (5 min)

    // CF API credentials
    CFAPIUrl    string  // Required: https://api.sys.customer.com
    CFUsername  string  // Required or from CredHub
    CFPassword  string  // Required or from CredHub

    // BOSH API credentials (optional)
    BOSHEnvironment string  // e.g., https://10.0.0.6:25555
    BOSHClient      string  // e.g., ops_manager
    BOSHSecret      string  // From CredHub preferred
    BOSHCACert      string  // Base64 encoded or from CredHub
    BOSHDeployment  string  // e.g., cf-abc123def456

    // CredHub integration (optional)
    CredHubURL      string
    CredHubClient   string
    CredHubSecret   string
}
```

### Credential Resolution (Hybrid Approach)

1. **Check CredHub** (if `CREDHUB_URL` is set):
   - Fetch `/c/capacity-analyzer/cf-password`
   - Fetch `/c/capacity-analyzer/bosh-secret`
   - Fetch `/c/capacity-analyzer/bosh-ca-cert`

2. **Fallback to Environment Variables:**
   - If CredHub unavailable or creds not found
   - Use `CF_PASSWORD`, `BOSH_CLIENT_SECRET`, etc.

3. **Validation:**
   - CF credentials: REQUIRED (fail startup if missing)
   - BOSH credentials: OPTIONAL (warn if missing, run degraded)

### Example CF Manifest

```yaml
applications:
- name: capacity-backend
  memory: 256M
  instances: 1
  buildpacks:
  - go_buildpack
  env:
    CF_API_URL: https://api.sys.customer.com
    BOSH_ENVIRONMENT: https://10.0.0.6:25555
    BOSH_DEPLOYMENT: cf-abc123def456
    CACHE_TTL: 300
  # Sensitive values set via cf set-env or CredHub binding
```

## Caching Implementation

### Cache Structure (`backend/cache/cache.go`)

```go
type CacheEntry struct {
    Data      interface{}
    ExpiresAt time.Time
}

type Cache struct {
    store sync.Map
    ttl   time.Duration
    mu    sync.RWMutex
}

// Key methods:
func (c *Cache) Get(key string) (interface{}, bool)
func (c *Cache) Set(key string, value interface{})
func (c *Cache) Clear(key string)
func (c *Cache) StartCleanup() // Background goroutine
```

### Cache Keys

- `dashboard:all` - Full dashboard data
- `cells` - Diego cell metrics only
- `apps` - App data only
- `segments` - Isolation segments only

### Cache Behavior

**On cache miss:**

- Fetch from CF API + BOSH API in parallel
- BOSH retries 3x with exponential backoff
- If BOSH fails → cache apps/segments only, mark `cells` as unavailable
- Return data with `cached: false` in metadata

**On cache hit (within TTL):**

- Return immediately from memory
- No API calls
- Include `cached: true` in metadata

**Background cleanup:**

- Goroutine runs every 1 minute
- Removes expired entries from `sync.Map`
- Prevents memory leak

### Performance Targets

- **Cold start (cache miss):** 2-5 seconds (depends on BOSH latency)
- **Warm response (cache hit):** <10ms
- **Memory footprint:** ~50MB for typical foundation (500 apps, 20 cells)
- **Concurrent requests:** Safe with `sync.Map` and `sync.RWMutex`

### TTL Tuning

- **Default:** 5 minutes (good balance for capacity planning)
- **Configurable:** Via `CACHE_TTL` env var
- **Recommendation:** 5min for production, 1min for active troubleshooting

## Error Handling & Logging

### Error Response Format

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Details string `json:"details,omitempty"`
    Code    int    `json:"code"`
}

// HTTP status codes:
// 200 - Success (even if BOSH unavailable, degraded mode)
// 500 - CF API failure (critical, can't proceed)
// 503 - Service startup failure (config errors)
```

### Degraded Mode

- **BOSH API fails:** Return apps/segments with warning in metadata
- **CF API fails:** Return HTTP 500 (can't provide value without apps)
- **Philosophy:** Partial data is better than no data

### Logging Strategy

```go
// Structured logging with levels
log.Info("Starting capacity analyzer backend", "port", config.Port)
log.Warn("BOSH API unavailable, running in degraded mode", "error", err)
log.Error("CF API authentication failed", "error", err)

// Log to stdout (CF captures via Loggregator)
// Include timestamps, log levels, structured fields
```

## Testing Strategy

### Unit Tests

- `services/cfapi_test.go` - Mock CF API responses
- `services/boshapi_test.go` - Mock BOSH API responses
- `cache/cache_test.go` - Cache TTL and concurrency tests
- `handlers/handlers_test.go` - HTTP handler tests with httptest

### Integration Tests

- Test against real CF API (using test foundation)
- BOSH mock (harder to test real BOSH safely)
- Verify degraded mode when BOSH unavailable

### Manual Testing

- Deploy to CF, test with real foundation
- Verify frontend can consume API responses
- Test cache behavior with various TTL settings

## Development Workflow

```bash
# Local development
cd backend
go run main.go  # Starts on :8080

# Run tests
go test ./...

# Build
go build -o capacity-backend

# Deploy to CF
cf push
```

## Security Considerations

### Authentication

- **Frontend to Backend:** No authentication (initial version)
- **Assumption:** Network security via CF app routes
- **Future:** Can add API key if needed for external access

### Credential Storage

- **Preferred:** CredHub for BOSH credentials
- **Fallback:** Environment variables (visible to CF admins via `cf env`)
- **Recommendation:** Use CredHub in production environments

### Data Sensitivity

- Backend exposes CF app names and memory usage
- Diego cell IPs and capacity metrics
- No user data or application logs exposed

## Deployment Guide for Customers

### Prerequisites

1. CF CLI installed
2. Access to CF foundation as admin user
3. BOSH credentials from Ops Manager

### Getting BOSH Credentials

```bash
# Export OM credentials
export OM_TARGET=https://opsmgr.customer.com
export OM_USERNAME=admin
export OM_PASSWORD=...
export OM_SKIP_SSL_VALIDATION=true

# Get BOSH credentials
om curl -s -p /api/v0/deployed/director/credentials/bosh_commandline_credentials
```

### Deploy Backend

```bash
# Clone repository
git clone https://github.com/customer/diego-capacity-analyzer.git
cd diego-capacity-analyzer/backend

# Configure credentials
cf set-env capacity-backend CF_USERNAME admin
cf set-env capacity-backend CF_PASSWORD <password>
cf set-env capacity-backend BOSH_CLIENT ops_manager
cf set-env capacity-backend BOSH_CLIENT_SECRET <secret>
cf set-env capacity-backend BOSH_CA_CERT <base64-cert>

# Push backend
cf push

# Get backend URL
cf app capacity-backend
```

### Deploy Frontend

```bash
cd ../frontend

# Configure backend URL
echo "VITE_API_URL=https://capacity-backend.apps.customer.com" > .env

# Build and push
npm run build
cf push capacity-ui -p dist -b staticfile_buildpack
```

## Success Criteria

- Backend successfully authenticates to CF API
- Backend fetches Diego cell metrics from BOSH API
- Frontend can call backend without CORS errors
- Cache reduces API calls by >80% under normal load
- Degraded mode works when BOSH unavailable
- Deployment takes <15 minutes for customers

## Future Enhancements

- API key authentication for external access
- Historical trend storage (time-series database)
- Multi-foundation support (query multiple CF environments)
- Cost estimation based on IaaS pricing
- Export recommendations to Terraform/Platform Automation
- Slack/email alerts for capacity thresholds

## References

- [CF API v3 Documentation](https://v3-apidocs.cloudfoundry.org/)
- [BOSH API Documentation](https://bosh.io/docs/director-api-v1/)
- [Go Cloud Foundry Client](https://github.com/cloudfoundry-community/go-cfclient)
- [Diego Cell Architecture](https://docs.cloudfoundry.org/concepts/diego/diego-architecture.html)
