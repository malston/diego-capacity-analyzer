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

### Ops Manager Variables

```bash
export OM_TARGET=opsman.example.com        # Ops Manager hostname
export OM_USERNAME=admin                    # Ops Manager username
export OM_PASSWORD=secret                   # Ops Manager password
export OM_SKIP_SSL_VALIDATION=true          # Skip SSL verification
export OM_PRIVATE_KEY=~/.ssh/opsman_key     # SSH key for BOSH proxy
```

### Deriving BOSH Variables from Ops Manager

Use the `om` CLI to set BOSH environment variables:

```bash
# Get BOSH credentials from Ops Manager
eval "$(om bosh-env 2>/dev/null)"

# Get active CA certificate
export BOSH_CA_CERT
BOSH_CA_CERT="$(om certificate-authorities -f json | jq -r '.[] | select(.active==true) | .cert_pem')"

# Set up SSH proxy through Ops Manager
export BOSH_ALL_PROXY="ssh+socks5://ubuntu@$OM_TARGET:22?private-key=$OM_PRIVATE_KEY"

# Ensure full URL for BOSH Director
export BOSH_ENVIRONMENT=https://$BOSH_ENVIRONMENT:25555

# Get CF deployment name
export BOSH_DEPLOYMENT=$(bosh deployments --json | jq -r '.Tables[0].Rows[] | select(.name | startswith("cf-")) | .name')
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

### vSphere Integration (`services/vsphere.go`)

- Connects to vCenter via govmomi to discover infrastructure
- Retrieves cluster, host, and VM inventory for capacity analysis
- Automatically detects Diego cell VMs by name pattern

**Diego Cell VM Naming:**
- Standard TAS: VMs named `diego_cell/*` or `diego-cell-*`
- Small Footprint TAS/TPCF: Diego cells run on `compute` instances (colocated)
- Detection matches: `diego_cell*`, `diego-cell*`, `compute*`, `diego*`

**Environment Variables:**
```bash
export VSPHERE_HOST=vcenter.example.com
export VSPHERE_USERNAME=administrator@vsphere.local
export VSPHERE_PASSWORD=secret
export VSPHERE_DATACENTER=Datacenter-Name
export VSPHERE_INSECURE=true  # optional, defaults to true
```

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
