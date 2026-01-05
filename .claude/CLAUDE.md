# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Diego Capacity Analyzer is a full-stack dashboard for analyzing Tanzu Application Service (TAS) / Diego cell capacity and density optimization. It connects to real Cloud Foundry environments via a Go backend that integrates with BOSH, CF, and vSphere APIs.

## Architecture

```text
diego-capacity-analyzer/
├── backend/                    # Go backend service
│   ├── main.go                 # Entry point, HTTP server
│   ├── handlers/               # HTTP handlers (health, dashboard, infrastructure)
│   ├── services/
│   │   ├── boshapi.go          # BOSH API client (VM info, vitals)
│   │   ├── cfapi.go            # CF API client (apps, processes)
│   │   ├── logcache.go         # Log Cache client (memory metrics)
│   │   ├── vsphere.go          # vCenter integration
│   │   ├── scenario.go         # Scenario calculator
│   │   └── planning.go         # Planning calculator
│   ├── models/                 # Data structures
│   ├── config/                 # Configuration management
│   ├── cache/                  # In-memory caching
│   ├── logger/                 # Structured logging
│   ├── middleware/             # HTTP middleware
│   └── e2e/                    # End-to-end tests
├── frontend/                   # React frontend
│   ├── src/
│   │   ├── components/         # React components
│   │   │   └── wizard/         # Scenario wizard
│   │   │       └── steps/      # Wizard step components
│   │   ├── contexts/           # React contexts (Auth, Toast)
│   │   ├── services/           # API clients
│   │   ├── config/             # App configuration
│   │   └── utils/              # Utility functions
│   ├── public/samples/         # Sample infrastructure files
│   └── vite.config.js
├── docs/                       # Documentation
├── .env.example                # Example environment configuration
├── generate-env.sh             # Generate .env from Ops Manager
└── Makefile                    # Build and development targets
```

## Development Commands

Use the Makefile for all development tasks:

```bash
# Setup
make frontend-install           # Install frontend dependencies
cp .env.example .env            # Configure environment (or use generate-env.sh)

# Development
make backend-run                # Build and run backend (PORT=8080)
make backend-dev                # Backend with auto-reload (watchexec/air/go run)
make frontend-dev               # Frontend dev server (PORT=5173)

# Custom ports
make backend-run BACKEND_PORT=9090
make frontend-dev FRONTEND_PORT=3000

# Testing
make test                       # Run all tests
make backend-test               # Backend only
make frontend-test              # Frontend only
make lint                       # Run all linters
make check                      # Tests + linters

# Build
make build                      # Build backend and frontend
make clean                      # Remove build artifacts
```

Run `make help` to see all available targets.

## Environment Configuration

### Using generate-env.sh (Recommended)

Automatically derive credentials from Ops Manager:

```bash
# Set Ops Manager credentials
export OM_TARGET=opsman.example.com
export OM_USERNAME=admin
export OM_PASSWORD=<password>
# Or use client credentials: OM_CLIENT_ID and OM_CLIENT_SECRET

# Optional: SSH key for non-routable BOSH networks
export OM_PRIVATE_KEY=~/.ssh/opsman_key

# Generate .env file
./generate-env.sh
```

### Manual Configuration

See `.env.example` for all available options. Key variables:

```bash
# Required: Cloud Foundry
CF_API_URL=https://api.sys.example.com
CF_USERNAME=admin
CF_PASSWORD=secret

# Optional: BOSH (enables Diego cell metrics)
BOSH_ENVIRONMENT=https://10.0.0.6:25555
BOSH_CLIENT=ops_manager
BOSH_CLIENT_SECRET=secret

# Optional: vSphere (enables infrastructure discovery)
VSPHERE_HOST=vcenter.example.com
VSPHERE_DATACENTER=Datacenter-Name
VSPHERE_USERNAME=administrator@vsphere.local
VSPHERE_PASSWORD=secret
```

## API Endpoints

```text
GET  /api/health                    # Health check
GET  /api/dashboard                 # Dashboard data (cells, apps, segments)
GET  /api/infrastructure            # Live vSphere infrastructure
POST /api/infrastructure/manual     # Manual infrastructure input
POST /api/infrastructure/state      # Set infrastructure state directly
GET  /api/infrastructure/status     # Data source status
POST /api/infrastructure/planning   # Calculate max deployable cells
POST /api/scenario/compare          # Compare current vs proposed scenarios
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

### Scenario Calculator (`services/scenario.go`)

- Compares current vs proposed capacity scenarios
- Calculates memory/CPU deltas and percentage changes

### Planning Calculator (`services/planning.go`)

- Calculates maximum deployable Diego cells
- N-1 HA calculations for host failure tolerance
- Multi-resource bottleneck detection (memory/CPU/disk)

## Technology Stack

### Backend
- **Go 1.23+** - HTTP server with standard library
- **govmomi** - vSphere/vCenter API client
- **socks5-proxy** - SSH tunneling for BOSH

### Frontend
- **React 18** - UI framework
- **Vite 5** - Build tool and dev server
- **Tailwind CSS** - Utility-first styling
- **Recharts** - Data visualization
- **Vitest** - Testing framework
- **Lucide React** - Icons

## Testing

```bash
make test                       # Run all tests
make backend-test               # Backend Go tests
make backend-test-verbose       # Verbose output
make frontend-test              # Frontend Vitest
make frontend-test-coverage     # With coverage report
```

## Data Flow

1. Frontend authenticates user via CF UAA (OAuth2 password grant)
2. Frontend calls backend API endpoints
3. Backend queries BOSH for Diego cell VM info and vitals
4. Backend queries CF API for apps and process stats
5. Backend queries Log Cache for actual memory usage
6. Backend queries vSphere for infrastructure data (optional)
7. Backend aggregates data and returns dashboard/planning response
8. Frontend renders capacity analysis and recommendations
