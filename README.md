# TAS Capacity Analyzer

A professional dashboard for analyzing Tanzu Application Service (TAS) / Diego cell capacity, density optimization, and right-sizing recommendations.

![GitHub Release](https://img.shields.io/github/v/release/malston/diego-capacity-analyzer?label=version)
![React](https://img.shields.io/badge/react-18.2-blue.svg)
![License](https://img.shields.io/github/license/malston/diego-capacity-analyzer)

> **⚠️ Disclaimer:** This is an independent, community-maintained project and is **not** an official Broadcom or VMware product. It is not supported, endorsed, or affiliated with Broadcom, VMware, or their subsidiaries. Use at your own risk.

## Features

### Dashboard

![Dashboard showing real-time capacity metrics and Diego cells](docs/images/dashboard.gif)

- **Real-time Capacity Monitoring** - Track Diego cell memory, CPU, and utilization across all cells
- **Isolation Segment Filtering** - View metrics by isolation segment
- **What-If Overcommit Modeling** - Simulate memory overcommit changes to see potential capacity gains
- **Right-Sizing Recommendations** - Identify over-provisioned apps with specific memory recommendations

### Capacity Planning

![Capacity Planning wizard with scenario analysis](docs/images/capacity-planning.gif)

- **Scenario Analysis Wizard** - Step-based configuration: Resources → Cell Config → CPU Config → Host Config → Advanced
- **vSphere Infrastructure Discovery** - Live infrastructure data from vCenter
- **N-1 HA Calculations** - Ensure capacity survives host failure
- **Max Cell Estimation** - Calculate deployable cells based on memory/CPU constraints
- **CPU Analysis** - vCPU:pCPU ratio calculation with risk level indicators
- **Host-Level Metrics** - VMs per host, host utilization, HA capacity analysis
- **Multi-Resource Bottleneck Detection** - Identify constraining resources (memory/CPU/disk)
- **Upgrade Recommendations** - Actionable suggestions: add cells, resize cells, add hosts
- **TPS Performance Modeling** - Estimate throughput based on cell count
- **Markdown Export** - Generate analysis reports for stakeholders

### Data Sources

- **Live vSphere** - Connect to vCenter for real infrastructure data
- **JSON Upload** - Import infrastructure configurations
- **Manual Entry** - Define infrastructure manually
- **Sample Scenarios** - 9 pre-built configurations including CPU and host-constrained scenarios

### CLI Tool

The `diego-capacity` CLI provides both an interactive TUI and command-line access for scripting.

#### Interactive TUI

When run without arguments in a terminal, `diego-capacity` launches an interactive TUI:

```bash
# Launch interactive TUI
diego-capacity

# Or explicitly with a specific backend
diego-capacity --api-url http://backend:8080
```

The TUI provides:

- **Data source selection**: Choose between live vSphere, JSON file, or manual input
- **Split-pane dashboard**: Live infrastructure metrics on the left, actions on the right
- **Scenario wizard**: Step-by-step what-if analysis with real-time feedback
- **Comparison view**: Side-by-side current vs proposed with delta highlights

##### Keyboard Shortcuts

| Key | Action                         |
| --- | ------------------------------ |
| `w` | Run scenario wizard            |
| `r` | Refresh infrastructure data    |
| `b` | Go back (from comparison view) |
| `q` | Quit                           |

#### Non-Interactive Commands

For CI/CD pipelines, use subcommands with `--json`:

```bash
# Check backend health
diego-capacity health

# Show infrastructure status
diego-capacity status

# Check capacity thresholds (for CI/CD)
diego-capacity check --n1-threshold 85 --memory-threshold 90

# Scenario comparison
diego-capacity scenario --cell-memory 64 --cell-cpu 8 --cell-count 20 --json

# JSON output for parsing
diego-capacity status --json
```

**Exit Codes:**

- `0` - Success (all checks passed)
- `1` - Threshold exceeded (capacity warning)
- `2` - Error (connection failed, no data)

**Configuration:**

```bash
# Set backend URL (default: http://localhost:8080)
export DIEGO_CAPACITY_API_URL=http://backend.example.com:8080

# Or use the --api-url flag
diego-capacity status --api-url http://backend.example.com:8080
```

## Quick Start (Local Development)

```bash
# Install frontend dependencies
make frontend-install

# Configure environment (see Configuration section below)
cp .env.example .env  # or use generate-env.sh

# Start services (in separate terminals)
make backend-run      # Backend on :8080
make frontend-dev     # Frontend on :5173

# Or use custom ports
make backend-run BACKEND_PORT=9090
make frontend-dev FRONTEND_PORT=3000
```

## API Endpoints

```text
GET  /api/v1/health                    # Health check
GET  /api/v1/dashboard                 # Dashboard data (cells, apps, segments)
GET  /api/v1/infrastructure            # Live vSphere infrastructure
POST /api/v1/infrastructure/manual     # Manual infrastructure input
POST /api/v1/infrastructure/state      # Set infrastructure state directly
GET  /api/v1/infrastructure/status     # Data source status
POST /api/v1/infrastructure/planning   # Calculate max deployable cells
GET  /api/v1/infrastructure/apps       # Per-app memory/disk breakdown
POST /api/v1/scenario/compare          # Compare current vs proposed scenarios
GET  /api/v1/bottleneck                # Multi-resource bottleneck analysis
GET  /api/v1/recommendations           # Upgrade path recommendations
```

## Configuration

### Generating Configuration from Ops Manager

If you have access to an Ops Manager, use `generate-env.sh` to automatically derive all credentials:

```bash
# Option 1: Username/password authentication
export OM_TARGET=opsman.example.com
export OM_USERNAME=admin
export OM_PASSWORD=<your-password>

# Option 2: OAuth client credentials
export OM_TARGET=opsman.example.com
export OM_CLIENT_ID=<client-id>
export OM_CLIENT_SECRET=<client-secret>

# Optional: SSH key for non-routable BOSH networks (tunnels through Ops Manager)
export OM_PRIVATE_KEY=~/.ssh/opsman_key

# Generate .env file with all derived credentials
./generate-env.sh
```

The script connects to Ops Manager and derives:

- BOSH Director credentials and CA certificate
- CF API credentials and system domain
- vSphere host, datacenter, and credentials
- CredHub configuration

### Manual Configuration

If not using Ops Manager, set environment variables manually:

#### Required: Cloud Foundry API

```bash
export CF_API_URL=https://api.sys.example.com
export CF_USERNAME=admin
export CF_PASSWORD=secret
```

### Optional: BOSH Integration

```bash
export BOSH_ENVIRONMENT=https://10.0.0.6:25555
export BOSH_CLIENT=ops_manager
export BOSH_CLIENT_SECRET=secret
export BOSH_CA_CERT="$(cat /path/to/bosh-ca.pem)"
export BOSH_ALL_PROXY=ssh+socks5://ubuntu@opsman.example.com:22?private-key=/path/to/key
```

### Optional: vSphere Integration

```bash
export VSPHERE_HOST=vcenter.example.com
export VSPHERE_USERNAME=administrator@vsphere.local
export VSPHERE_PASSWORD=secret
export VSPHERE_DATACENTER=Datacenter-Name
export VSPHERE_INSECURE=false
```

### Optional: Tuning

```bash
export PORT=8080                    # HTTP server port (default: 8080)
export CACHE_TTL=300                # Default cache TTL in seconds
export DASHBOARD_CACHE_TTL=30       # Dashboard cache TTL
export VSPHERE_CACHE_TTL=300        # vSphere cache TTL
export LOG_LEVEL=info               # debug, info, warn, error
export LOG_FORMAT=text              # text, json
```

## Development

Run `make help` to see all available targets:

```bash
make build                   # Build backend, frontend, and CLI
make test                    # Run all tests
make lint                    # Run all linters
make check                   # Tests + linters
make clean                   # Remove build artifacts

# Backend development
make backend-run             # Build and run server
make backend-dev             # Auto-reload with watchexec
make backend-test            # Run Go tests

# Frontend development
make frontend-dev            # Vite dev server with HMR
make frontend-test           # Run Vitest
make frontend-test-coverage  # With coverage report

# CLI development
make cli-build               # Build diego-capacity binary
make cli-test                # Run CLI tests
make cli-install             # Install to $GOPATH/bin
```

## Testing

```bash
make test                    # Run all tests
make backend-test            # Backend only
make backend-test-verbose    # Backend with verbose output
make frontend-test           # Frontend only
make frontend-test-coverage  # Frontend with coverage
make lint                    # Run all linters
make check                   # Tests + linters
```

## CI/CD

GitHub Actions workflows run automatically:

- **CI** (`.github/workflows/ci.yml`) - Runs on PRs and pushes to main
  - Frontend: lint, test, build
  - Backend: staticcheck, test, build
  - CLI: staticcheck, test, build
- **Release** (`.github/workflows/release.yml`) - Creates releases on version tags
  - Backend: Cross-compiles for linux/darwin × amd64/arm64
  - CLI: Cross-compiles `diego-capacity` for linux/darwin × amd64/arm64

## Documentation

- **[CLI Guide](docs/CLI.md)** - CLI tool and TUI usage, CI/CD integration
- **[UI Guide](docs/UI-GUIDE.md)** - Dashboard metrics and visualizations
- **[API Reference](docs/API.md)** - Backend REST API documentation
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Cloud Foundry deployment
- **[Authentication](docs/AUTHENTICATION.md)** - OAuth2/UAA authentication flow
- **[FAQ](docs/FAQ.md)** - Common questions and answers
- **[Demo Materials](docs/demo/)** - Presentation slides, scripts, and reference materials
- **[Backend README](backend/README.md)** - Backend-specific documentation

## Project Structure

```text
├── backend/                    # Go HTTP service
│   ├── main.go                 # Entry point
│   ├── config/                 # Configuration loader
│   ├── models/                 # Data models
│   ├── cache/                  # In-memory cache with TTL
│   ├── services/               # API clients
│   │   ├── boshapi.go          # BOSH Director integration
│   │   ├── cfapi.go            # Cloud Foundry API
│   │   ├── logcache.go         # Log Cache metrics
│   │   ├── vsphere.go          # vCenter integration
│   │   ├── scenario.go         # Scenario calculator
│   │   └── planning.go         # Planning calculator
│   ├── handlers/               # HTTP handlers
│   ├── logger/                 # Structured logging
│   ├── middleware/             # HTTP middleware
│   └── manifest.yml            # CF deployment manifest
│
├── cli/                        # CLI tool with TUI (diego-capacity)
│   ├── main.go                 # Entry point
│   ├── cmd/                    # Cobra commands
│   │   ├── root.go             # Root command, TTY detection, TUI launch
│   │   ├── health.go           # Health check command
│   │   ├── status.go           # Infrastructure status
│   │   ├── check.go            # Threshold checking
│   │   └── scenario.go         # Scenario comparison
│   └── internal/
│       ├── client/             # HTTP client for backend API
│       └── tui/                # Terminal UI components
│           ├── app.go          # Root TUI model
│           ├── styles/         # Lipgloss styles
│           ├── menu/           # Data source menu
│           ├── dashboard/      # Infrastructure dashboard
│           ├── wizard/         # Scenario wizard
│           └── comparison/     # Comparison view
│
├── frontend/                   # React SPA
│   ├── src/
│   │   ├── components/         # React components
│   │   │   └── wizard/         # Scenario wizard
│   │   │       └── steps/      # Wizard step components
│   │   ├── contexts/           # React contexts (Auth, Toast)
│   │   ├── services/           # API clients
│   │   ├── config/             # App configuration
│   │   └── utils/              # Utility functions
│   ├── public/samples/         # Sample infrastructure files
│   └── manifest.yml            # CF deployment manifest
│
├── .github/workflows/          # CI/CD pipelines
├── docs/                       # Documentation
├── .env.example                # Example environment configuration
├── generate-env.sh             # Generate .env from Ops Manager
└── Makefile                    # Build and development targets
```

## Technology Stack

### Backend

- **Go 1.23** - HTTP server with standard library
- **govmomi** - vSphere/vCenter API client
- **socks5-proxy** - SSH tunneling for BOSH

### Frontend

- **React 18** - UI framework
- **Vite 5** - Build tool and dev server
- **Tailwind CSS** - Utility-first styling
- **Recharts** - Data visualization
- **Vitest** - Testing framework
- **Lucide React** - Icons
