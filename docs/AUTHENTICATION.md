# Authentication

## Overview

Diego Capacity Analyzer uses a Backend-For-Frontend (BFF) OAuth pattern. The frontend never handles OAuth tokens directly. Instead:

1. The backend authenticates with CF UAA on behalf of the user
2. Tokens are stored in server-side sessions (never exposed to JavaScript)
3. The browser receives httpOnly session cookies
4. CSRF protection uses a double-submit cookie pattern

```text
Browser ──── httpOnly cookies ────> Backend ──── OAuth tokens ────> CF UAA
       <──── session + CSRF ───────        <──── access/refresh ───
```

## Configuration

Authentication-related environment variables:

| Variable                 | Default    | Description                                  |
| ------------------------ | ---------- | -------------------------------------------- |
| `AUTH_MODE`              | `optional` | `disabled`, `optional`, or `required`        |
| `COOKIE_SECURE`          | `true`     | Set `false` for local dev (HTTP without TLS) |
| `CORS_ALLOWED_ORIGINS`   | (empty)    | Comma-separated list of allowed origins      |
| `CF_API_URL`             | (required) | Cloud Foundry API URL                        |
| `CF_USERNAME`            | (required) | CF admin username for backend API access     |
| `CF_PASSWORD`            | (required) | CF admin password                            |
| `CF_SKIP_SSL_VALIDATION` | `false`    | Skip TLS verification for CF/UAA endpoints   |
| `OAUTH_CLIENT_ID`        | `cf`       | OAuth client ID for UAA password grants      |
| `OAUTH_CLIENT_SECRET`    | (empty)    | OAuth client secret                          |

## How Authentication Works

### OAuth2 Flow (BFF)

1. User submits credentials to the frontend login form
2. Frontend POSTs to `/api/v1/auth/login` with `{ username, password }`
3. Backend discovers the UAA endpoint from CF API (`/v3/info`)
4. Backend exchanges credentials with CF UAA using the OAuth2 password grant
5. Backend stores access token, refresh token, and scopes in a server-side session
6. Backend sets an httpOnly session cookie (`DIEGO_SESSION`) and a CSRF cookie (`DIEGO_CSRF`)
7. Frontend receives a success response containing only the username -- no tokens

### Session Management

| Cookie          | Flags                                   | Purpose                                      |
| --------------- | --------------------------------------- | -------------------------------------------- |
| `DIEGO_SESSION` | `HttpOnly`, `Secure`, `SameSite=Strict` | Session identifier (opaque, 32 random bytes) |
| `DIEGO_CSRF`    | `Secure`, `SameSite=Lax`                | CSRF token readable by JavaScript            |

- Sessions are stored in the backend's in-memory cache with a TTL matching the token lifetime (plus a 10-minute buffer for refresh)
- The browser sends cookies automatically on every request (`credentials: "include"`)
- Session IDs and CSRF tokens are cryptographically random (32 bytes, base64url-encoded)

### Token Refresh

- The backend checks whether the access token expires within 5 minutes
- `POST /api/v1/auth/refresh` triggers a `refresh_token` grant with CF UAA
- The session is updated with the new access and refresh tokens; cookies remain unchanged
- If the refresh fails, the session is deleted and the user must log in again

### CSRF Protection

The backend enforces CSRF protection using the double-submit cookie pattern:

- On login, the backend sets a `DIEGO_CSRF` cookie with a random token
- For state-changing requests (`POST`, `PUT`, `DELETE`), the `DIEGO_CSRF` cookie value must match the `X-CSRF-Token` request header
- Comparison uses constant-time comparison (`crypto/subtle`) to prevent timing attacks
- CSRF validation is skipped for:
  - Safe methods: `GET`, `HEAD`, `OPTIONS`
  - Bearer token authentication (via `Authorization` header)
  - Requests without a session cookie (not session-authenticated)

## Auth Endpoints

All auth endpoints use the `/api/v1/auth/` prefix.

| Endpoint               | Method | Description                       | Rate Limit |
| ---------------------- | ------ | --------------------------------- | ---------- |
| `/api/v1/auth/login`   | `POST` | Authenticate and create session   | 5/min      |
| `/api/v1/auth/logout`  | `POST` | Destroy session and clear cookies | 5/min      |
| `/api/v1/auth/me`      | `GET`  | Check authentication status       | None       |
| `/api/v1/auth/refresh` | `POST` | Refresh access token              | 10/min     |

**Login request:**

```json
{ "username": "admin", "password": "..." }
```

**Login response (success):**

```json
{ "success": true, "username": "admin", "user_id": "..." }
```

**Login response (failure):**

```json
{ "success": false, "error": "Invalid credentials" }
```

**Me response (authenticated):**

```json
{ "authenticated": true, "username": "admin", "user_id": "..." }
```

**Me response (not authenticated):**

```json
{ "authenticated": false }
```

## Role-Based Access Control (RBAC)

The backend enforces role-based authorization on API endpoints. Roles are derived from CF UAA JWT scopes.

> **Before you start:** Capacity Planning requires the `operator` role to load and manage infrastructure data. Without completing the [UAA Group Setup](#uaa-group-setup) below, all users -- including `admin` -- default to the `viewer` role and will receive a "403 Forbidden" error when accessing Capacity Planning features.

### Roles

| Role         | Description                                                                                                  |
| ------------ | ------------------------------------------------------------------------------------------------------------ |
| **viewer**   | Read-only access to dashboards, metrics, and calculations. Default for all authenticated users.              |
| **operator** | Full access including state-mutating operations (manual infrastructure input, infrastructure state changes). |

Operator inherits all viewer permissions.

### UAA Scope Mapping

| UAA Scope                 | Application Role |
| ------------------------- | ---------------- |
| `diego-analyzer.operator` | operator         |
| `diego-analyzer.viewer`   | viewer           |
| (no matching scope)       | viewer (default) |

If a token contains both scopes, operator takes precedence.

### Protected Endpoints

Only two endpoints require the operator role:

| Endpoint                        | Method | Required Role |
| ------------------------------- | ------ | ------------- |
| `/api/v1/infrastructure/manual` | POST   | operator      |
| `/api/v1/infrastructure/state`  | POST   | operator      |

All other authenticated endpoints are accessible to any role (viewer or operator).

### UAA Group Setup

To configure RBAC, create the UAA groups, assign users, and create a dedicated OAuth client. All three steps are required.

**Automated setup (recommended):**

```bash
# Set Ops Manager credentials
export OM_TARGET=opsman.example.com
export OM_USERNAME=admin
export OM_PASSWORD=<password>

# Run setup script with the UAA usernames to grant access
./setup-uaa.sh admin operator-user
```

The script creates the groups, assigns users, creates the OAuth client, and prints the `OAUTH_CLIENT_ID` / `OAUTH_CLIENT_SECRET` values to add to your `.env` file. Run `./setup-uaa.sh --help` for all options.

**Manual setup:**

If you prefer to run the steps manually, follow Steps 1-3 below.

**Step 1: Get the UAA admin client secret**

From Ops Manager:

```bash
# Set Ops Manager credentials
export OM_TARGET=https://opsman.example.com
export OM_USERNAME=admin
export OM_PASSWORD=<password>

# Retrieve the UAA admin client secret
om -t "$OM_TARGET" -u "$OM_USERNAME" -p "$OM_PASSWORD" -k \
  credentials -p cf -c .uaa.admin_client_credentials
```

**Step 2: Authenticate with UAA and create groups**

```bash
# Target your UAA instance
uaac target https://uaa.sys.example.com --skip-ssl-validation

# Authenticate as UAA admin client
uaac token client get admin -s <admin-client-secret>

# Create the groups
uaac group add diego-analyzer.viewer
uaac group add diego-analyzer.operator

# Assign users to roles
uaac member add diego-analyzer.viewer <username>
uaac member add diego-analyzer.operator <username>
```

**Step 3: Create a dedicated OAuth client**

Do NOT modify the `cf` or `admin` clients -- they are shared system clients used by the CF CLI and other tools. Instead, create a dedicated client for the application:

```bash
uaac client add diego-analyzer \
  --name "Diego Capacity Analyzer" \
  --scope "openid diego-analyzer.operator diego-analyzer.viewer" \
  --authorized_grant_types "password,refresh_token" \
  --access_token_validity 7200 \
  --refresh_token_validity 1209600 \
  --secret <client-secret>
```

Then configure the backend to use this client via environment variables:

```bash
OAUTH_CLIENT_ID=diego-analyzer
OAUTH_CLIENT_SECRET=<client-secret>
```

After creating the client, users must log out and log back in to receive a token with the new scopes.

The application authorizes based on the JWT `scope` claim, not UAA group membership directly. For scopes to appear in issued tokens, **both** conditions must be met:

1. **UAA groups exist** and the user is a member (Step 2)
2. **The OAuth client includes these scopes** in its allowed scope list (Step 3)

If groups exist but scopes don't appear in tokens, the client configuration is the most likely cause.

### Auth Mode Behavior

RBAC enforcement depends on the `AUTH_MODE` environment variable:

| AUTH_MODE  | RBAC Behavior                                                                         |
| ---------- | ------------------------------------------------------------------------------------- |
| `disabled` | RBAC is bypassed; all requests pass through                                           |
| `optional` | Anonymous requests are treated as viewer; authenticated users get their resolved role |
| `required` | All requests must authenticate; role is resolved from token scopes                    |

### Default Behavior

If the UAA groups (`diego-analyzer.viewer`, `diego-analyzer.operator`) are not created, all authenticated users default to the **viewer** role. This means:

- All read-only endpoints work as before
- The two operator endpoints (`infrastructure/manual` and `infrastructure/state`) return **403 Forbidden**
- To enable operator access, create the groups and assign users as shown above

### Session Role Lifecycle

User roles are resolved from the JWT `scope` claim at login and on each token refresh. If a user's UAA group membership changes (e.g., operator access is revoked or granted), the new role takes effect after the next token refresh or re-login.

To force an immediate role change for a user, invalidate their session (restart the backend or wait for session expiry).

## Frontend Integration

### AuthContext

The `AuthProvider` component (`contexts/AuthContext.jsx`) wraps the app and provides authentication state via the `useAuth()` hook:

```javascript
const { isAuthenticated, user, loading, error, login, logout } = useAuth();
```

On mount, `AuthProvider` calls `GET /api/v1/auth/me` to check for an existing session.

### CSRF in Frontend

`utils/csrf.js` provides two functions for CSRF token handling:

- `getCSRFToken()` -- reads the `DIEGO_CSRF` cookie value
- `withCSRFToken(headers)` -- returns a copy of `headers` with `X-CSRF-Token` added

All state-changing API calls (POST, PUT, DELETE) must include the CSRF token:

```javascript
fetch("/api/v1/infrastructure/manual", {
  method: "POST",
  credentials: "include",
  headers: withCSRFToken({ "Content-Type": "application/json" }),
  body: JSON.stringify(data),
});
```

## Auth Modes

| Mode       | Unauthenticated Requests    | Authenticated Requests              |
| ---------- | --------------------------- | ----------------------------------- |
| `disabled` | Allowed, no auth checks     | Auth headers ignored                |
| `optional` | Allowed as anonymous viewer | Validated; role resolved from token |
| `required` | Rejected with 401           | Validated; role resolved from token |

In `optional` mode, if a token is present but invalid, the request is rejected (not treated as anonymous).

## Troubleshooting

**CORS errors:** Set `CORS_ALLOWED_ORIGINS` to include your frontend's origin (e.g., `http://localhost:5173` for local dev).

**401 Unauthorized:** Check that `CF_USERNAME` and `CF_PASSWORD` are correct and that `AUTH_MODE` is not set to `disabled` when authentication is expected.

**403 Forbidden:** The user's role lacks permission. Check the [RBAC section](#role-based-access-control-rbac) for required roles and UAA group setup.

**CSRF validation failures:** Ensure the `DIEGO_CSRF` cookie is present in the browser and that the `X-CSRF-Token` header is included on POST/PUT/DELETE requests. Use `withCSRFToken()` from `utils/csrf.js`.

**Login works locally but cookies not sent:** Set `COOKIE_SECURE=false` when running over HTTP (local dev without TLS).

**Verifying OAuth client credentials:** To confirm your dedicated OAuth client is configured correctly, check two things:

1. The backend logs the configured client at startup:

   ```text
   INFO Auth mode configured mode=required oauth_client=diego-analyzer
   ```

2. Test the credentials directly against UAA with a password grant:

   ```bash
   curl -sk -X POST "https://login.sys.example.com/oauth/token" \
     -u "$OAUTH_CLIENT_ID:$OAUTH_CLIENT_SECRET" \
     -d "grant_type=password&username=$CF_USERNAME&password=$CF_PASSWORD" \
     | jq .
   ```

   A successful response includes an `access_token` with `diego-analyzer.operator` and `diego-analyzer.viewer` in the `scope` field. If the client credentials are wrong, UAA returns `"error": "unauthorized"` with `"Bad client credentials"`.

   The login URL is derived from your CF API URL by replacing `api.` with `login.` (e.g., `https://api.sys.example.com` becomes `https://login.sys.example.com`).

## CI / Automation

For pipelines and scripts that call the API, create a dedicated UAA service account instead of using human credentials. This avoids embedding personal passwords in CI secrets and gives the automation its own audit trail.

**Step 1: Create the service account**

```bash
# Authenticate with UAA (see UAA Group Setup above for getting the admin secret)
uaac target https://uaa.sys.example.com --skip-ssl-validation
uaac token client get admin -s <admin-client-secret>

# Create a service account user
uaac user add ci-pipeline -p <service-account-password> --emails ci-pipeline@example.com

# Grant the appropriate role
uaac member add diego-analyzer.operator ci-pipeline
uaac member add diego-analyzer.viewer ci-pipeline
```

**Step 2: Use in your pipeline**

```bash
# Get a token (valid for 2 hours with default client settings)
export OAUTH_TOKEN=$(curl -sk -X POST "https://login.sys.example.com/oauth/token" \
  -u "$OAUTH_CLIENT_ID:$OAUTH_CLIENT_SECRET" \
  -d "grant_type=password&username=ci-pipeline&password=$CI_SERVICE_ACCOUNT_PASSWORD" \
  | jq -r '.access_token')

# Call the API
curl -s http://your-backend:8080/api/v1/dashboard \
  -H "Authorization: Bearer $OAUTH_TOKEN" | jq .
```

Store `OAUTH_CLIENT_ID`, `OAUTH_CLIENT_SECRET`, and `CI_SERVICE_ACCOUNT_PASSWORD` in your pipeline's secrets manager. The token can be reused for the duration of the pipeline run (2-hour lifetime).

## Security Properties

- OAuth tokens are stored server-side and never exposed to JavaScript
- Session cookies use `HttpOnly`, `Secure`, and `SameSite=Strict` flags
- CSRF tokens use a double-submit cookie pattern with constant-time comparison
- Auth endpoints are rate-limited (login/logout: 5/min, refresh: 10/min)
- Session IDs and CSRF tokens are 32 bytes of cryptographic randomness
