# External Integrations

**Analysis Date:** 2026-02-24

## APIs & External Services

**Cloud Foundry (CF API v3):**

- Service: Cloud Foundry CAPI v3
- What it's used for: Fetches all apps, processes, isolation segments, and space relationships for capacity analysis
- SDK/Client: Custom HTTP client in `backend/services/cfapi.go` -- no external SDK
- Auth: OAuth2 password grant against CF UAA (`CF_USERNAME` / `CF_PASSWORD`)
- Endpoints used: `/v3/info`, `/v3/apps`, `/v3/apps/{guid}/processes`, `/v3/spaces/{guid}/relationships/isolation_segment`, `/v3/isolation_segments`
- Pagination: Handled automatically (100 per page)
- Env vars: `CF_API_URL`, `CF_USERNAME`, `CF_PASSWORD`, `CF_SKIP_SSL_VALIDATION`

**CF UAA (User Account and Authentication):**

- Service: Cloud Foundry UAA OAuth2 server
- What it's used for: Two purposes -- (1) backend authenticates its own service account for CF API queries; (2) BFF pattern for frontend user login via UAA password grant
- SDK/Client: Custom HTTP client in `backend/services/cfapi.go`, `backend/services/boshapi.go`, and `backend/handlers/auth.go`
- Auth endpoint: Discovered dynamically from CF API `/v3/info` response (`links.login.href`); falls back to replacing `api.` with `login.` in `CF_API_URL`
- Token endpoint: `{uaa_url}/oauth/token` (password grant and client_credentials grant)
- JWKS endpoint: `{uaa_url}/token_keys` -- used by `backend/services/jwks.go` for Bearer token signature verification
- Env vars: `OAUTH_CLIENT_ID` (default: `cf`), `OAUTH_CLIENT_SECRET`

**BOSH Director:**

- Service: BOSH (Cloud Foundry infrastructure orchestrator)
- What it's used for: Retrieves Diego cell VM list with memory/CPU vitals across all CF and isolation segment deployments
- SDK/Client: Custom HTTP client in `backend/services/boshapi.go` -- no external SDK
- Auth: OAuth2 client_credentials grant against BOSH's built-in UAA (`BOSH_CLIENT` / `BOSH_CLIENT_SECRET`)
- UAA endpoint: Discovered from BOSH Director `/info` endpoint; falls back to port 8443 on Director host
- Endpoints used: `/info`, `/deployments`, `/deployments/{name}/vms?format=full`, `/tasks/{id}`, `/tasks/{id}/output?type=result`
- Task polling: Async task pattern -- polls every 2 seconds, max 60 attempts (2 minutes)
- Response format: NDJSON (newline-delimited JSON) for task output
- Deployment filter: Only queries deployments matching `cf-*` or `p-isolation-segment*` prefix
- Env vars: `BOSH_ENVIRONMENT`, `BOSH_CLIENT`, `BOSH_CLIENT_SECRET`, `BOSH_CA_CERT`, `BOSH_DEPLOYMENT`, `BOSH_SKIP_SSL_VALIDATION`, `BOSH_ALL_PROXY`, `BOSH_SSH_KEY_ALLOWED_DIRS`

**CF Log Cache:**

- Service: Cloud Foundry Log Cache (Loggregator subsystem)
- What it's used for: Retrieves actual container memory usage metrics per app (more accurate than CF API "requested" values)
- SDK/Client: Custom HTTP client in `backend/services/logcache.go`
- Auth: Reuses CF API Bearer token from `CFClient.Authenticate()`
- URL derivation: Replaces `api.` with `log-cache.` in `CF_API_URL` (e.g., `api.sys.example.com` -> `log-cache.sys.example.com`)
- Endpoints used: `/api/v1/read/{app_guid}?envelope_types=GAUGE&limit=100` (gauge envelopes), `/api/v1/promql` (PromQL queries)
- Graceful degradation: Falls back to CF API requested memory if Log Cache is unavailable

**VMware vSphere / vCenter:**

- Service: VMware vCenter Server
- What it's used for: Discovers infrastructure topology (clusters, ESXi hosts, Diego cell VMs) for capacity planning
- SDK/Client: `github.com/vmware/govmomi` v0.52.0 (`backend/services/vsphere.go`)
- Auth: Username/password via govmomi client (`VSPHERE_USERNAME` / `VSPHERE_PASSWORD`)
- Connection: `https://{VSPHERE_HOST}/sdk` (VMware SDK endpoint)
- Diego cell detection: Via BOSH custom VM attributes and fallback to VM name pattern matching
- Env vars: `VSPHERE_HOST`, `VSPHERE_USERNAME`, `VSPHERE_PASSWORD`, `VSPHERE_DATACENTER`, `VSPHERE_INSECURE`, `VSPHERE_CACHE_TTL`
- Integration is optional -- backend runs in manual mode if not configured

## Data Storage

**Databases:**

- None -- no external database

**In-Memory Cache:**

- Type: Custom TTL-based in-process cache using `sync.Map`
- Implementation: `backend/cache/cache.go`
- Stores: Dashboard data, session data, vSphere infrastructure data
- TTL configuration:
  - General cache: `CACHE_TTL` env var (default: 300 seconds)
  - Dashboard/BOSH data: `DASHBOARD_CACHE_TTL` env var (default: 30 seconds)
  - vSphere data: `VSPHERE_CACHE_TTL` env var (default: 300 seconds)
  - Sessions: TTL matched to OAuth token expiry + 10 minutes
- Cleanup: Background goroutine runs every 1 minute

**File Storage:**

- Local filesystem only -- for SSH private keys (when using BOSH SSH proxy)
- SSH key path is validated against an allowlist of directories (`backend/services/boshapi.go`)

## Authentication & Identity

**Auth Provider:**

- CF UAA (Cloud Foundry User Account and Authentication)
- Implementation: BFF (Backend-For-Frontend) OAuth2 pattern
  - Frontend never receives tokens; only gets httpOnly session cookies
  - Backend holds tokens in server-side sessions (`backend/services/session.go`)
  - Session ID: 32 cryptographically random bytes, base64url encoded
  - CSRF token: 32 cryptographically random bytes, stored in session and sent as non-httpOnly cookie
- Session cookies: `DIEGO_SESSION` (httpOnly, Secure, SameSite=Strict), `DIEGO_CSRF` (Secure, SameSite=Lax)
- CSRF protection: Double-submit cookie pattern using `X-CSRF-Token` header vs `DIEGO_CSRF` cookie (`backend/middleware/csrf.go`)
- Bearer token support: JWT verification using RSA PKCS1v15 signature with JWKS key rotation (`backend/services/jwks.go`)
- RBAC: Two roles resolved from CF scopes -- `operator` (`cloud_controller.admin`, `network.admin`, `diego-analyzer.operator`) and `viewer` (`cloud_controller.read`, `doppler.firehose`, `cloud_controller.global_auditor`)
- Auth modes: `disabled` (dev), `optional` (default), `required`
- RBAC enforcement: `backend/middleware/rbac.go`, role requirement declared per-route in `backend/handlers/routes.go`

## Monitoring & Observability

**Error Tracking:**

- None -- no external error tracking service (e.g., Sentry)

**Logs:**

- Structured logging via Go standard library `log/slog` (`backend/logger/logger.go`)
- Format: `text` or `json` (configurable via `LOG_FORMAT` env var)
- Level: `debug` | `info` | `warn` | `error` (configurable via `LOG_LEVEL` env var)
- Output: stdout only
- Frontend: `console.error()` for auth failures in `frontend/src/services/cfAuth.js`

**Metrics:**

- None -- no metrics collection (no Prometheus, Datadog, etc.)

## CI/CD & Deployment

**Hosting:**

- Cloud Foundry / Tanzu Application Service (primary target)
- CF manifests: `backend/manifest.yml`, `frontend/manifest.yml`

**CI Pipeline:**

- Not detected in repository (no `.github/workflows/`, `.circleci/`, `.travis.yml`, etc.)

## Ops Manager Integration

**Ops Manager (VMware Tanzu Ops Manager):**

- Used only for credential extraction via `generate-env.sh` script
- Calls `om staged-director-config --no-redact` to extract BOSH and vSphere credentials
- Env vars: `OM_TARGET`, `OM_USERNAME`, `OM_PASSWORD`, `OM_CLIENT_ID`, `OM_CLIENT_SECRET`, `OM_SKIP_SSL_VALIDATION`, `OM_PRIVATE_KEY`, `OM_CONNECT_TIMEOUT`
- Not a runtime dependency -- only used during environment setup

## Environment Configuration

**Required env vars:**

- `CF_API_URL` - Cloud Foundry API URL (e.g., `https://api.sys.example.com`)
- `CF_USERNAME` - CF admin username
- `CF_PASSWORD` - CF admin password

**Optional env vars:**

- `BOSH_ENVIRONMENT` - BOSH Director URL (enables Diego cell metrics)
- `BOSH_CLIENT` - BOSH UAA client ID
- `BOSH_CLIENT_SECRET` - BOSH UAA client secret
- `BOSH_CA_CERT` - BOSH Director CA certificate (PEM)
- `BOSH_ALL_PROXY` - SSH+SOCKS5 proxy URL for non-routable BOSH networks
- `VSPHERE_HOST` - vCenter hostname (enables infrastructure discovery)
- `VSPHERE_USERNAME` - vCenter username
- `VSPHERE_PASSWORD` - vCenter password
- `VSPHERE_DATACENTER` - vCenter datacenter name
- `CORS_ALLOWED_ORIGINS` - Comma-separated allowed CORS origins
- `OAUTH_CLIENT_ID` - UAA OAuth client ID (default: `cf`)
- `OAUTH_CLIENT_SECRET` - UAA OAuth client secret

**Secrets location:**

- `.env` file at project root (gitignored)
- Generated via `generate-env.sh` from Ops Manager, or set manually from `.env.example`

## Webhooks & Callbacks

**Incoming:**

- None -- no webhook endpoints

**Outgoing:**

- None -- no outgoing webhooks; all external calls are request/response

---

_Integration audit: 2026-02-24_
