# Diego Capacity Analyzer Backend

Go HTTP service for Cloud Foundry capacity analysis.

## Quick Start

```bash
# Set required environment variables
export CF_API_URL=https://api.sys.example.com
export CF_USERNAME=admin
export CF_PASSWORD=secret

# Optional: BOSH credentials (enables Diego cell metrics)
export BOSH_ENVIRONMENT=https://10.0.0.6:25555
export BOSH_CLIENT=ops_manager
export BOSH_CLIENT_SECRET=secret
export BOSH_CA_CERT="$(cat bosh-ca.crt)"
export BOSH_DEPLOYMENT=cf-abc123

# Optional: vSphere credentials (enables infrastructure discovery)
export VSPHERE_HOST=vcenter.example.com
export VSPHERE_USERNAME=administrator@vsphere.local
export VSPHERE_PASSWORD=secret
export VSPHERE_DATACENTER=Datacenter-Name

# Run locally
go run main.go

# Build
go build -o capacity-backend

# Run tests
go test ./...
```

## API Endpoints

All endpoints use the `/api/v1/` prefix:

```text
# Health & Status
GET  /api/v1/health                    # Health check
GET  /api/v1/dashboard                 # Dashboard data (cells, apps, segments)

# Infrastructure
GET  /api/v1/infrastructure            # Live vSphere infrastructure
POST /api/v1/infrastructure/manual     # Manual infrastructure input
POST /api/v1/infrastructure/state      # Set infrastructure state directly
GET  /api/v1/infrastructure/status     # Data source status
POST /api/v1/infrastructure/planning   # Calculate max deployable cells
GET  /api/v1/infrastructure/apps       # Per-app memory/disk breakdown

# Scenario
POST /api/v1/scenario/compare          # Compare current vs proposed scenarios

# Analysis
GET  /api/v1/bottleneck                # Multi-resource bottleneck analysis
GET  /api/v1/recommendations           # Upgrade path recommendations
```

Legacy `/api/` routes (without `/v1/`) are supported for backward compatibility.

## Environment Variables

### Required: Cloud Foundry API

| Variable      | Description                                                 |
| ------------- | ----------------------------------------------------------- |
| `CF_API_URL`  | Cloud Foundry API URL (e.g., `https://api.sys.example.com`) |
| `CF_USERNAME` | CF admin username                                           |
| `CF_PASSWORD` | CF admin password                                           |

### Optional: BOSH Integration

| Variable             | Description                                                                                   |
| -------------------- | --------------------------------------------------------------------------------------------- |
| `BOSH_ENVIRONMENT`   | BOSH Director URL (e.g., `https://10.0.0.6:25555`)                                            |
| `BOSH_CLIENT`        | BOSH UAA client ID                                                                            |
| `BOSH_CLIENT_SECRET` | BOSH UAA client secret                                                                        |
| `BOSH_CA_CERT`       | BOSH Director CA certificate (PEM format)                                                     |
| `BOSH_DEPLOYMENT`    | BOSH deployment name (e.g., `cf-abc123`)                                                      |
| `BOSH_ALL_PROXY`     | SOCKS5 proxy for BOSH access (e.g., `ssh+socks5://ubuntu@opsman:22?private-key=/path/to/key`) |

### Optional: vSphere Integration

| Variable             | Description             | Default |
| -------------------- | ----------------------- | ------- |
| `VSPHERE_HOST`       | vCenter hostname        |         |
| `VSPHERE_USERNAME`   | vCenter username        |         |
| `VSPHERE_PASSWORD`   | vCenter password        |         |
| `VSPHERE_DATACENTER` | vCenter datacenter name |         |
| `VSPHERE_INSECURE`   | Skip TLS verification   | `true`  |

### Optional: CredHub Integration

| Variable         | Description               |
| ---------------- | ------------------------- |
| `CREDHUB_URL`    | CredHub API URL           |
| `CREDHUB_CLIENT` | CredHub UAA client ID     |
| `CREDHUB_SECRET` | CredHub UAA client secret |

### Optional: Tuning

| Variable              | Description                        | Default |
| --------------------- | ---------------------------------- | ------- |
| `PORT`                | HTTP server port                   | `8080`  |
| `CACHE_TTL`           | General cache TTL (seconds)        | `300`   |
| `DASHBOARD_CACHE_TTL` | Dashboard data cache TTL (seconds) | `30`    |
| `VSPHERE_CACHE_TTL`   | vSphere data cache TTL (seconds)   | `300`   |

## Deployment to Cloud Foundry

### Prerequisites

1. CF CLI installed
2. Logged into CF: `cf login`
3. BOSH credentials from Ops Manager

### Get BOSH Credentials

```bash
export OM_TARGET=https://opsmgr.example.com
export OM_USERNAME=admin
export OM_PASSWORD=<password>
export OM_SKIP_SSL_VALIDATION=true

om curl -s -p /api/v0/deployed/director/credentials/bosh_commandline_credentials
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

# Optional: vSphere credentials
cf set-env capacity-backend VSPHERE_HOST vcenter.example.com
cf set-env capacity-backend VSPHERE_USERNAME administrator@vsphere.local
cf set-env capacity-backend VSPHERE_PASSWORD <vsphere-password>
cf set-env capacity-backend VSPHERE_DATACENTER Datacenter-Name

# Restage to apply env vars
cf restage capacity-backend

# Get app URL
cf app capacity-backend
```

### Test Deployment

```bash
BACKEND_URL=$(cf app capacity-backend | grep routes: | awk '{print $2}')
curl https://$BACKEND_URL/api/v1/health
curl https://$BACKEND_URL/api/v1/dashboard
```
