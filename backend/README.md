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
