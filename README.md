# TAS Capacity Analyzer

A professional dashboard for analyzing Tanzu Application Service (TAS) / Diego cell capacity, density optimization, and right-sizing recommendations.

![GitHub Release](https://img.shields.io/github/v/release/malston/diego-capacity-analyzer?label=version)
![React](https://img.shields.io/badge/react-18.2-blue.svg)
![License](https://img.shields.io/github/license/malston/diego-capacity-analyzer)

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
GET  /api/health                    # Health check
GET  /api/dashboard                 # Dashboard data (cells, apps, segments)
GET  /api/infrastructure            # Live vSphere infrastructure
POST /api/infrastructure/manual     # Manual infrastructure input
POST /api/infrastructure/state      # Set infrastructure state directly
GET  /api/infrastructure/status     # Data source status
POST /api/infrastructure/planning   # Calculate max deployable cells
POST /api/scenario/compare          # Compare current vs proposed scenarios
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
export VSPHERE_INSECURE=true
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
make build                   # Build backend and frontend
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
- **Release** (`.github/workflows/release.yml`) - Creates releases on version tags
  - Cross-compiles for linux/darwin × amd64/arm64

## Documentation

- **[UI Guide](docs/UI-GUIDE.md)** - Dashboard metrics and visualizations
- **[Deployment Guide](docs/DEPLOYMENT.md)** - Cloud Foundry deployment
- **[Authentication](docs/AUTHENTICATION.md)** - OAuth2/UAA authentication flow
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
