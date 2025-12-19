# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Diego Capacity Analyzer is a full-stack dashboard for analyzing Tanzu Application Service (TAS) / Diego cell capacity and density optimization. It connects to real Cloud Foundry environments via a Go backend that integrates with BOSH and CF APIs.

## Architecture

```
diego-capacity-analyzer/
├── backend/                    # Go backend service
│   ├── main.go                 # Entry point, HTTP server
│   ├── handlers/               # HTTP handlers (health, dashboard)
│   ├── services/
│   │   ├── boshapi.go          # BOSH API client (VM info, vitals)
│   │   ├── cfapi.go            # CF API client (apps, processes)
│   │   └── logcache.go         # Log Cache client (memory metrics)
│   ├── models/                 # Data structures
│   ├── config/                 # Configuration management
│   └── cache/                  # In-memory caching
├── frontend/                   # React frontend
│   ├── src/
│   │   ├── TASCapacityAnalyzer.jsx  # Main dashboard component
│   │   ├── services/
│   │   │   ├── cfAuth.js       # OAuth2 authentication
│   │   │   └── cfApi.js        # CF API service
│   │   └── contexts/
│   │       └── AuthContext.jsx # Auth state management
│   └── vite.config.js
└── docs/                       # Documentation
```

## Development Commands

### Backend (Go)

```bash
cd backend

# Build
go build -o capacity-backend

# Run (requires environment variables)
./capacity-backend

# Run tests
go test ./...
```

### Frontend (React)

```bash
cd frontend

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

## Backend Configuration

The Go backend requires these environment variables:

```bash
# Required
export OM_TARGET=opsman.example.com        # Ops Manager hostname
export OM_USERNAME=admin                    # Ops Manager username
export OM_PASSWORD=secret                   # Ops Manager password
export OM_SKIP_SSL_VALIDATION=true          # Skip SSL verification

# Optional - derived from Ops Manager if not set
export BOSH_ENVIRONMENT=...                 # BOSH Director URL
export BOSH_CLIENT=...                      # BOSH UAA client
export BOSH_CLIENT_SECRET=...               # BOSH UAA secret
export BOSH_CA_CERT=...                     # BOSH CA certificate
export BOSH_ALL_PROXY=...                   # SSH+SOCKS5 proxy URL
```

## Key Backend Features

### BOSH Integration (`services/boshapi.go`)

- Authenticates with BOSH Director via UAA OAuth
- Queries all CF and isolation segment deployments (pattern: `cf-*`, `p-isolation-segment-*`)
- Retrieves Diego cell VMs with memory vitals
- Supports SSH+SOCKS5 proxy tunneling through Ops Manager

### CF API Integration (`services/cfapi.go`)

- OAuth2 authentication with CF UAA
- Fetches applications and process stats
- Maps apps to isolation segments

### Log Cache Integration (`services/logcache.go`)

- Retrieves real container memory metrics
- Provides accurate "used" memory vs "allocated" memory

## API Endpoints

```
GET /api/health     # Health check
GET /api/dashboard  # Full dashboard data (cells, apps, metrics)
POST /api/refresh   # Force data refresh
```

## CORS Configuration

CORS for the CF API is configured in the Cloud Controller, not Gorouter. To allow localhost during development, add to the BOSH manifest:

```yaml
- name: cloud_controller_ng
  properties:
    cc:
      allowed_cors_domains:
      - http://localhost:3000
      - http://127.0.0.1:3000
```

## Technology Stack

### Backend
- **Go 1.21+** - HTTP server, API clients
- **No external frameworks** - Standard library only

### Frontend
- **React 18** - UI framework
- **Vite 5** - Build tool and dev server
- **Tailwind CSS** - Styling
- **Recharts** - Data visualization

## Testing

```bash
# Backend tests
cd backend && go test ./...

# Run with verbose output
go test -v ./...

# Run specific package
go test ./services/...
```

## Data Flow

1. Frontend authenticates user via CF UAA (OAuth2 password grant)
2. Frontend calls backend API endpoints
3. Backend queries BOSH for Diego cell VM info and vitals
4. Backend queries CF API for apps and process stats
5. Backend queries Log Cache for actual memory usage
6. Backend aggregates data and returns dashboard response
7. Frontend renders capacity analysis and recommendations
