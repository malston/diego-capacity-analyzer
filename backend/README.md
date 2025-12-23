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
