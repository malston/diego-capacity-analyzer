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

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_MODE` | `optional` | `disabled`, `optional`, or `required` |
| `COOKIE_SECURE` | `true` | Set `false` for local dev (HTTP without TLS) |
| `CORS_ALLOWED_ORIGINS` | (empty) | Comma-separated list of allowed origins |
| `CF_API_URL` | (required) | Cloud Foundry API URL |
| `CF_USERNAME` | (required) | CF admin username for backend API access |
| `CF_PASSWORD` | (required) | CF admin password |
| `CF_SKIP_SSL_VALIDATION` | `false` | Skip TLS verification for CF/UAA endpoints |

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

| Cookie | Flags | Purpose |
|--------|-------|---------|
| `DIEGO_SESSION` | `HttpOnly`, `Secure`, `SameSite=Strict` | Session identifier (opaque, 32 random bytes) |
| `DIEGO_CSRF` | `Secure`, `SameSite=Lax` | CSRF token readable by JavaScript |

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

| Endpoint | Method | Description | Rate Limit |
|----------|--------|-------------|------------|
| `/api/v1/auth/login` | `POST` | Authenticate and create session | 5/min |
| `/api/v1/auth/logout` | `POST` | Destroy session and clear cookies | 5/min |
| `/api/v1/auth/me` | `GET` | Check authentication status | None |
| `/api/v1/auth/refresh` | `POST` | Refresh access token | 10/min |

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

### Roles

| Role | Description |
|------|-------------|
| **viewer** | Read-only access to dashboards, metrics, and calculations. Default for all authenticated users. |
| **operator** | Full access including state-mutating operations (manual infrastructure input, infrastructure state changes). |

Operator inherits all viewer permissions.

### UAA Scope Mapping

| UAA Scope | Application Role |
|-----------|-----------------|
| `diego-analyzer.operator` | operator |
| `diego-analyzer.viewer` | viewer |
| (no matching scope) | viewer (default) |

If a token contains both scopes, operator takes precedence.

### Protected Endpoints

Only two endpoints require the operator role:

| Endpoint | Method | Required Role |
|----------|--------|---------------|
| `/api/v1/infrastructure/manual` | POST | operator |
| `/api/v1/infrastructure/state` | POST | operator |

All other authenticated endpoints are accessible to any role (viewer or operator).

### UAA Group Setup

To configure RBAC, create the UAA groups and assign users:

```bash
# Target your UAA instance
uaac target https://login.sys.example.com --skip-ssl-validation

# Authenticate as admin
uaac token client get admin -s <admin-client-secret>

# Create the groups
uaac group add diego-analyzer.viewer
uaac group add diego-analyzer.operator

# Assign users to roles
uaac member add diego-analyzer.viewer <username>
uaac member add diego-analyzer.operator <username>
```

The application authorizes based on the JWT `scope` claim, not UAA group membership directly. For scopes to appear in issued tokens, two conditions must be met:

1. **UAA groups exist** and the user is a member (commands above)
2. **The OAuth client includes these scopes** in its allowed scope list

For the default `cf` client, UAA group membership is automatically reflected in token scopes. For custom OAuth clients, you must explicitly add `diego-analyzer.viewer` and `diego-analyzer.operator` to the client's `scope` and `authorities` configuration. If groups exist but scopes don't appear in tokens, the client configuration is the most likely cause.

### Auth Mode Behavior

RBAC enforcement depends on the `AUTH_MODE` environment variable:

| AUTH_MODE | RBAC Behavior |
|-----------|---------------|
| `disabled` | RBAC is bypassed; all requests pass through |
| `optional` | Anonymous requests are treated as viewer; authenticated users get their resolved role |
| `required` | All requests must authenticate; role is resolved from token scopes |

### Default Behavior

If the UAA groups (`diego-analyzer.viewer`, `diego-analyzer.operator`) are not created, all authenticated users default to the **viewer** role. This means:

- All read-only endpoints work as before
- The two operator endpoints (`infrastructure/manual` and `infrastructure/state`) return **403 Forbidden**
- To enable operator access, create the groups and assign users as shown above

### Session Role Lifecycle

User roles are resolved at login time from the JWT `scope` claim and persist for the session lifetime. If a user's UAA group membership changes (e.g., operator access is revoked or granted), the change does not take effect until the user logs out and logs back in. Token refresh updates the access and refresh tokens but does not re-resolve roles from the refreshed token's scopes.

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

| Mode | Unauthenticated Requests | Authenticated Requests |
|------|--------------------------|------------------------|
| `disabled` | Allowed, no auth checks | Auth headers ignored |
| `optional` | Allowed as anonymous viewer | Validated; role resolved from token |
| `required` | Rejected with 401 | Validated; role resolved from token |

In `optional` mode, if a token is present but invalid, the request is rejected (not treated as anonymous).

## Troubleshooting

**CORS errors:** Set `CORS_ALLOWED_ORIGINS` to include your frontend's origin (e.g., `http://localhost:5173` for local dev).

**401 Unauthorized:** Check that `CF_USERNAME` and `CF_PASSWORD` are correct and that `AUTH_MODE` is not set to `disabled` when authentication is expected.

**403 Forbidden:** The user's role lacks permission. Check the [RBAC section](#role-based-access-control-rbac) for required roles and UAA group setup.

**CSRF validation failures:** Ensure the `DIEGO_CSRF` cookie is present in the browser and that the `X-CSRF-Token` header is included on POST/PUT/DELETE requests. Use `withCSRFToken()` from `utils/csrf.js`.

**Login works locally but cookies not sent:** Set `COOKIE_SECURE=false` when running over HTTP (local dev without TLS).

## Security Properties

- OAuth tokens are stored server-side and never exposed to JavaScript
- Session cookies use `HttpOnly`, `Secure`, and `SameSite=Strict` flags
- CSRF tokens use a double-submit cookie pattern with constant-time comparison
- Auth endpoints are rate-limited (login/logout: 5/min, refresh: 10/min)
- Session IDs and CSRF tokens are 32 bytes of cryptographic randomness
