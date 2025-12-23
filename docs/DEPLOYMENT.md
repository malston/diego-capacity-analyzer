# Deployment Guide

Complete guide for deploying the TAS Capacity Analyzer to Cloud Foundry.

## Prerequisites

### Required Tools

- **CF CLI** - Cloud Foundry command-line interface (v8+)

  ```bash
  cf version
  ```

- **om CLI** - Ops Manager CLI for retrieving BOSH credentials

  ```bash
  om version
  ```

- **Node.js & npm** - For building the frontend (v18+)

  ```bash
  node --version
  npm --version
  ```

### Required Access

- **Cloud Foundry:** Admin user credentials for target foundation
- **Ops Manager:** Admin access to retrieve BOSH Director credentials
- **Network Access:** Backend app must be able to reach:
  - CF API endpoint
  - BOSH Director (typically internal IP like 10.0.0.6)

### Login to Cloud Foundry

```bash
cf login -a https://api.sys.your-domain.com -u admin
# Enter password when prompted

# Target your org and space
cf target -o system -s system
```

---

## Step 1: Get BOSH Credentials from Ops Manager

The backend service needs BOSH Director credentials to query Diego cell metrics. Use the `om` CLI to retrieve these from Ops Manager.

### Configure om CLI Environment

```bash
export OM_TARGET=https://opsmgr.your-domain.com
export OM_USERNAME=admin
export OM_PASSWORD=<your-ops-manager-password>
export OM_SKIP_SSL_VALIDATION=true
```

### Retrieve BOSH Credentials

```bash
om curl -s -p /api/v0/deployed/director/credentials/bosh_commandline_credentials
```

This returns JSON with credentials. Extract the values:

```json
{
  "credential": "export BOSH_CLIENT=ops_manager\nexport BOSH_CLIENT_SECRET=abc123...\nexport BOSH_CA_CERT='-----BEGIN CERTIFICATE-----\n...'\nexport BOSH_ENVIRONMENT=10.0.0.6 bosh"
}
```

### Extract Individual Values

```bash
# Get BOSH client ID (typically "ops_manager")
BOSH_CLIENT=$(om curl -s -p /api/v0/deployed/director/credentials/bosh_commandline_credentials | jq -r '.credential' | awk '{print $1}' | cut -d= -f2)

# Get BOSH client secret
BOSH_CLIENT_SECRET=$(om curl -s -p /api/v0/deployed/director/credentials/bosh_commandline_credentials | jq -r '.credential' | awk '{print $2}' | cut -d= -f2)

# Get BOSH CA certificate (the active one -- in case its been rotated)
om certificate-authorities -f json | jq -r '.[] | select(.active==true) | .cert_pem' > bosh-ca.crt

# Get BOSH Director IP
BOSH_ENVIRONMENT=$(om curl -s -p /api/v0/deployed/director/credentials/bosh_commandline_credentials | jq -r '.credential' | awk '{print $4}' | cut -d= -f2)
```

### Get BOSH Deployment Name

You'll need the CF deployment name. Connect to BOSH Director to find it:

```bash
# Export credentials
export BOSH_CLIENT=$BOSH_CLIENT
export BOSH_CLIENT_SECRET=$BOSH_CLIENT_SECRET
export BOSH_CA_CERT=$(cat bosh-ca.crt)
export BOSH_ENVIRONMENT=https://$BOSH_ENVIRONMENT:25555

# List deployments
bosh deployments

# Look for deployment starting with "cf-" (e.g., "cf-abc123def456")
bosh deployments --json | jq -r '.Tables[0].Rows[] | select(.name | startswith("cf-")) | .name'
```

Save the deployment name:

```bash
export BOSH_DEPLOYMENT=$(bosh deployments --json | jq -r '.Tables[0].Rows[] | select(.name | startswith("cf-")) | .name')
```

---

## Step 2: Deploy Backend Service

The backend is a Go application that provides the REST API for capacity analysis.

### Update Backend Manifest

Edit `backend/manifest.yml` and update the placeholder values:

```yaml
---
applications:
- name: capacity-backend
  memory: 256M
  instances: 1
  buildpacks:
  - go_buildpack
  env:
    CF_API_URL: https://api.sys.your-domain.com     # Update this
    BOSH_ENVIRONMENT: https://10.0.0.6:25555        # Update this
    BOSH_DEPLOYMENT: cf-abc123def456                # Update this
    CACHE_TTL: 300
```

### Push Backend App

```bash
cd backend

# Push the app (will use manifest.yml)
cf push

# App will start but won't function until credentials are set
```

### Set Sensitive Credentials

Set credentials as environment variables (not in manifest for security):

```bash
# CF API credentials
cf set-env capacity-backend CF_USERNAME admin
cf set-env capacity-backend CF_PASSWORD <your-cf-admin-password>

# BOSH Director credentials (from Step 1)
cf set-env capacity-backend BOSH_CLIENT $BOSH_CLIENT
cf set-env capacity-backend BOSH_CLIENT_SECRET $BOSH_CLIENT_SECRET
cf set-env capacity-backend BOSH_CA_CERT "$(cat bosh-ca.crt)"
```

### Restage to Apply Credentials

```bash
cf restage capacity-backend
```

### Verify Backend Deployment

```bash
# Get the backend app route
cf app capacity-backend

# Test health endpoint
BACKEND_URL=$(cf app capacity-backend | grep routes: | awk '{print $2}')
curl https://$BACKEND_URL/api/health

# Expected response:
# {
#   "cf_api": "ok",
#   "bosh_api": "ok",
#   "cache_status": {
#     "cells_cached": false,
#     "apps_cached": false
#   }
# }

# Test dashboard endpoint
curl https://$BACKEND_URL/api/dashboard | jq .

# Should return JSON with cells, apps, segments, and metadata
```

---

## Step 3: Deploy Frontend Application

The frontend is a React single-page application that consumes the backend API.

### Configure Frontend Environment

Create `.env` file in the `frontend/` directory with the backend URL:

```bash
cd frontend

# Use the backend URL from Step 2
BACKEND_URL=$(cf app capacity-backend | grep routes: | awk '{print $2}')
echo "VITE_API_URL=https://$BACKEND_URL" > .env
```

### Build Frontend

```bash
# Install dependencies (if not already done)
npm install

# Build production bundle
npm run build

# This creates the dist/ directory with static assets
```

### Update Frontend Manifest (Optional)

The `frontend/manifest.yml` is already configured, but you can verify it points to the correct backend:

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
    VITE_API_URL: https://capacity-backend.apps.your-domain.com
```

### Push Frontend App

```bash
cf push
```

### Verify Frontend Deployment

```bash
# Get the frontend app route
cf app capacity-ui

# Open in browser
UI_URL=$(cf app capacity-ui | grep routes: | awk '{print $2}')
echo "Open this URL in your browser: https://$UI_URL"
```

---

## Step 4: Access the Application

### Open the Dashboard

Navigate to the frontend URL in your browser:

```
https://capacity-ui.apps.your-domain.com
```

You should see:

- Diego cell capacity metrics
- Application memory usage
- Isolation segment filtering
- What-if scenario modeling
- Right-sizing recommendations

### Verify Data is Loading

1. **Check for Real Data**: The dashboard should show actual Diego cells from your foundation (not mock data)
2. **Check Metadata**: Look at the metadata section to verify:
   - `bosh_available: true`
   - Recent timestamp
   - Cache status

3. **Test Filtering**: Use the isolation segment dropdown to filter cells by segment

---

## Troubleshooting

### Backend Won't Start

**Symptom**: `cf app capacity-backend` shows crashed status

**Check logs**:

```bash
cf logs capacity-backend --recent
```

**Common issues**:

- Missing required environment variables (CF_API_URL, CF_USERNAME, CF_PASSWORD)
- Invalid CF credentials
- Go buildpack compilation errors

**Solution**:

```bash
# Verify all environment variables are set
cf env capacity-backend

# Look for CF_API_URL, CF_USERNAME, CF_PASSWORD, BOSH_* variables

# If missing, set them and restage
cf set-env capacity-backend <VARIABLE_NAME> <value>
cf restage capacity-backend
```

### BOSH Connection Fails

**Symptom**: Backend runs but `bosh_api: "not_configured"` or errors in logs about BOSH

**Check logs**:

```bash
cf logs capacity-backend --recent | grep -i bosh
```

**Common issues**:

- BOSH Director IP not accessible from CF network
- Incorrect BOSH credentials
- Missing or invalid CA certificate
- Wrong BOSH deployment name

**Solution**:

```bash
# Verify BOSH credentials
cf env capacity-backend | grep BOSH

# Test BOSH connectivity (SSH into backend app)
cf ssh capacity-backend
$ curl -k https://10.0.0.6:25555/info

# If connection fails, BOSH IP may not be routable from CF
# The backend will run in "degraded mode" (apps only, no cell metrics)
```

**Note**: The backend gracefully handles BOSH unavailability and runs in degraded mode, showing app data only.

### Frontend Shows Mock Data

**Symptom**: Dashboard displays, but metadata shows `cached: false` and generic data

**Check browser console**:

- Open Developer Tools (F12)
- Look for CORS errors or fetch failures

**Common issues**:

- VITE_API_URL points to wrong backend URL
- Backend is down or unreachable
- CORS not enabled (should be enabled by default)

**Solution**:

```bash
# Verify backend is running
cf app capacity-backend

# Check frontend environment
cf env capacity-ui | grep VITE_API_URL

# If wrong, rebuild frontend with correct .env
cd frontend
echo "VITE_API_URL=https://<correct-backend-url>" > .env
npm run build
cf push
```

### CORS Errors in Browser

**Symptom**: Browser console shows `Access-Control-Allow-Origin` errors

**Check**: Backend CORS headers should be enabled by default in the `handlers.go` file.

**Verify CORS**:

```bash
BACKEND_URL=$(cf app capacity-backend | grep routes: | awk '{print $2}')
curl -i -X OPTIONS https://$BACKEND_URL/api/dashboard

# Should see:
# Access-Control-Allow-Origin: *
# Access-Control-Allow-Methods: GET, OPTIONS
```

**Solution**: CORS is enabled by default. If not working, check backend logs for errors.

### High Memory Usage

**Symptom**: Backend app restarts frequently due to memory limits

**Check**:

```bash
cf app capacity-backend  # Look at memory usage
```

**Solution**: Increase memory allocation in `backend/manifest.yml`:

```yaml
memory: 512M  # Increase from 256M
```

Then:

```bash
cf push
```

### Slow Dashboard Load

**Symptom**: Dashboard takes 30+ seconds to load

**Check**: Cache may not be working, or CF/BOSH APIs are slow

**Verify cache**:

```bash
# First request (cache miss)
time curl https://$BACKEND_URL/api/dashboard

# Second request (should be cached, much faster)
time curl https://$BACKEND_URL/api/dashboard
```

**Solution**:

- First load will be slow (fetching from CF + BOSH)
- Subsequent loads within 5 minutes should be instant (cached)
- Increase `CACHE_TTL` if needed: `cf set-env capacity-backend CACHE_TTL 600`

---

## Updating the Application

### Update Backend Code

```bash
cd backend
# Make your code changes
git commit -m "Your changes"

# Redeploy
cf push

# No need to restage unless env vars changed
```

### Update Frontend Code

```bash
cd frontend
# Make your code changes
git commit -m "Your changes"

# Rebuild and redeploy
npm run build
cf push
```

---

## Uninstalling

### Delete Applications

```bash
# Delete frontend
cf delete capacity-ui -f

# Delete backend
cf delete capacity-backend -f

# Verify deletion
cf apps | grep capacity
```

### Clean Up Local Files

```bash
# Remove BOSH CA certificate
rm bosh-ca.crt

# Remove frontend build artifacts
cd frontend
rm -rf dist/ node_modules/

# Remove backend build artifacts
cd ../backend
rm -f capacity-backend
```

---

## Security Considerations

### Credentials Management

**Current approach**: Environment variables set via `cf set-env`

**Production recommendation**: Use CredHub for credential storage

```bash
# Store in CredHub
credhub set -n /cf/capacity-backend/cf-password -t password -w <password>

# Bind to app
cf bind-service capacity-backend <credhub-service-instance>

# Update app to read from CredHub (requires code changes)
```

### Network Security

- **BOSH Access**: Backend needs network access to BOSH Director (typically internal IP)
- **Recommendation**: Deploy backend to a space with appropriate ASGs (Application Security Groups)

### User Authentication

**Current state**: No authentication on frontend

**Production recommendation**: Add authentication via:

- OAuth2/OIDC integration
- CF SSO tile
- Custom authentication layer

---

## Advanced Configuration

### Adjust Cache TTL

Default: 5 minutes (300 seconds)

```bash
# Increase to 10 minutes
cf set-env capacity-backend CACHE_TTL 600
cf restage capacity-backend
```

### Scale Backend

```bash
# Scale to 2 instances for high availability
cf scale capacity-backend -i 2
```

**Note**: In-memory cache is per-instance. Consider Redis for shared cache if scaling beyond 2 instances.

### Custom Domains

```bash
# Map custom domain to frontend
cf map-route capacity-ui your-domain.com --hostname capacity

# Map custom domain to backend
cf map-route capacity-backend your-domain.com --hostname capacity-api
```

---

## Monitoring and Observability

### View Application Logs

```bash
# Tail backend logs
cf logs capacity-backend

# View recent logs
cf logs capacity-backend --recent

# Tail frontend logs
cf logs capacity-ui
```

### Check Application Health

```bash
# Backend health
curl https://$BACKEND_URL/api/health

# Application status
cf app capacity-backend
cf app capacity-ui
```

### Common Log Messages

**Backend**:

```
Starting Diego Capacity Analyzer Backend
CF API: https://api.sys.example.com
BOSH: https://10.0.0.6:25555
Cache TTL: 5m0s
Server listening on :8080
```

**Expected in logs**:

- `Fetching fresh data` - Cache miss, querying APIs
- `Serving from cache` - Cache hit, returning cached data
- `BOSH API error (degraded mode)` - BOSH unavailable, running without cell metrics

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Browser                              │
│                     (User Interface)                        │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTPS
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                 Frontend (React SPA)                        │
│                  capacity-ui (CF App)                       │
│                  staticfile_buildpack                       │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTPS /api/*
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                Backend (Go HTTP Service)                    │
│               capacity-backend (CF App)                     │
│                    go_buildpack                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              In-Memory Cache                         │   │
│  │              (5 minute TTL)                          │   │
│  └──────────────────────────────────────────────────────┘   │
└────────┬──────────────────────────────────┬─────────────────┘
         │ HTTPS                            │ HTTPS
         ▼                                  ▼
┌─────────────────────┐          ┌─────────────────────────┐
│    CF API v3        │          │    BOSH Director        │
│  (Apps, Segments)   │          │   (Diego Cell VMs)      │
└─────────────────────┘          └─────────────────────────┘
```

**Data Flow**:

1. User opens frontend in browser
2. Frontend fetches `/api/dashboard` from backend
3. Backend checks cache (5min TTL)
4. On cache miss:
   - Query CF API for apps and isolation segments
   - Query BOSH Director for Diego cell VM metrics
   - Combine data and cache result
5. Return unified JSON to frontend
6. Frontend renders visualizations

---

## Support

For issues or questions:

- Check application logs: `cf logs <app-name> --recent`
- Review this troubleshooting guide
- Check backend health: `curl https://$BACKEND_URL/api/health`
- Contact your platform team or open an issue in the repository
