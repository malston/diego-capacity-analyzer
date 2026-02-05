# CSRF Token Protection Design

**Issue:** #94
**Date:** 2026-02-05
**Status:** Approved

## Summary

Add CSRF token protection for state-changing endpoints using the double-submit cookie pattern to supplement existing SameSite cookie protection.

## Context

The application uses session cookies with `SameSite=Strict` for web UI authentication. While this provides good CSRF protection in modern browsers, gaps exist:

- Older browsers don't support SameSite
- Subdomain takeover attacks can bypass it

Bearer token requests (CLI/automation) are inherently CSRF-safe since tokens require explicit attachment.

## Design Decisions

| Decision              | Choice               | Rationale                                          |
| --------------------- | -------------------- | -------------------------------------------------- |
| Token generation      | On login             | Simplest approach, one token per session           |
| Pattern               | Double-submit cookie | Stateless validation, no additional server storage |
| Protected methods     | POST, PUT, DELETE    | Standard practice, GET should be idempotent        |
| Bearer token handling | Skip CSRF check      | Tokens aren't auto-attached like cookies           |

## Implementation

### Token Generation

Generate a cryptographically secure random token (32 bytes, base64 encoded) when a session is created. Store in session and set as readable cookie.

**Cookie properties:**

```go
http.Cookie{
    Name:     "DIEGO_CSRF",
    Value:    csrfToken,
    HttpOnly: false,  // Must be readable by JavaScript
    Secure:   true,   // HTTPS only
    SameSite: http.SameSiteStrictMode,
    Path:     "/",
    MaxAge:   3600,   // Match session cookie lifetime
}
```

### CSRF Middleware

New file: `middleware/csrf.go`

**Validation flow:**

1. If method is GET, HEAD, OPTIONS: skip check
2. If Authorization header present: skip check (Bearer token)
3. If no session cookie: skip check (not session-authenticated)
4. Get CSRF token from `X-CSRF-Token` header
5. Get CSRF token from `DIEGO_CSRF` cookie
6. If either missing or tokens don't match: return 403 Forbidden
7. Proceed to handler

**Error response:**

```json
HTTP 403 Forbidden
{"error": "CSRF token missing or invalid"}
```

### Frontend Changes

**CSRF utility** (`frontend/src/utils/csrf.js`):

```javascript
export function getCSRFToken() {
  const match = document.cookie.match(/DIEGO_CSRF=([^;]+)/);
  return match ? match[1] : null;
}
```

**API service update:**
Add `X-CSRF-Token` header to POST, PUT, DELETE requests:

```javascript
if (["POST", "PUT", "DELETE"].includes(method)) {
  const csrfToken = getCSRFToken();
  if (csrfToken) {
    headers["X-CSRF-Token"] = csrfToken;
  }
}
```

### Protected Endpoints

All POST endpoints accessed via session cookies:

- `POST /auth/logout`
- `POST /auth/refresh`
- `POST /infrastructure/manual`
- `POST /infrastructure/state`
- `POST /infrastructure/planning`
- `POST /scenario/compare`

## Testing Strategy

**Backend unit tests** (`middleware/csrf_test.go`):

- Skip check for GET requests
- Skip check for Bearer auth
- Skip check when no session cookie
- Reject missing header (403)
- Reject missing cookie (403)
- Reject token mismatch (403)
- Accept valid matching tokens

**Integration tests** (`e2e/csrf_test.go`):

- Full login flow with CSRF validation
- Rejection without valid token

## Implementation Tasks

1. **Backend: Generate CSRF token on login** - Update Session model and cookie setting
2. **Backend: CSRF middleware** - Create middleware with validation logic and tests
3. **Backend: Wire up middleware** - Add to chain, update OpenAPI spec
4. **Frontend: Add CSRF header** - Create utility, update API service
5. **Integration tests** - E2E and browser testing

## Related

- Issue #44: JWT signature verification (parent security work)
- Current protection: `SameSite=Strict` cookies
