# Rate Limiting Middleware Design

**Issue:** #92
**Date:** 2026-02-10
**Status:** Approved

## Summary

Add per-endpoint rate limiting to protect against brute force, credential stuffing, and abuse of expensive endpoints on internet-facing Cloud Foundry deployments.

## Context

With JWT authentication (#44) and CSRF protection (#94) in place, rate limiting is the next defense layer. The app is single-instance with in-memory session storage (`sync.Map`-based cache), so the rate limiting solution should match this architecture.

## Design Decisions

| Decision          | Choice                  | Rationale                                                         |
| ----------------- | ----------------------- | ----------------------------------------------------------------- |
| Storage           | In-process (sync.Mutex) | Matches existing in-memory session architecture, no external deps |
| Algorithm         | Fixed-window counter    | Simple, well-understood, adequate for 5-100 req/min limits        |
| Key strategy      | Per-endpoint tier       | Different endpoints have different abuse profiles                 |
| Disable mechanism | Environment variable    | Operators can disable during debugging or load testing            |

**Why not sliding window:** Fixed window has a known 2x burst at window boundaries. At 5 req/min for auth endpoints, a worst-case burst of 10 requests across a boundary is acceptable. Sliding window adds complexity for no practical benefit at these rates.

**Limitation:** Counters reset on app restart, same as sessions. Acceptable for CF deployments where restarts are infrequent and the primary goal is abuse prevention, not precise metering.

## Rate Limit Tiers

| Tier      | Limit   | Key Strategy                | Endpoints                              |
| --------- | ------- | --------------------------- | -------------------------------------- |
| `auth`    | 5/min   | Client IP (X-Forwarded-For) | login, logout                          |
| `refresh` | 10/min  | Session cookie ID           | refresh                                |
| `write`   | 10/min  | User ID (fallback: IP)      | POST infrastructure/_, POST scenario/_ |
| (default) | 100/min | User ID (fallback: IP)      | All other endpoints                    |
| `none`    | exempt  | --                          | health, openapi, auth/me               |

All limits configurable via environment variables. Entire system disableable via `RATE_LIMIT_ENABLED=false`.

## Implementation

### Core RateLimiter

```go
type RateLimiter struct {
    mu      sync.Mutex
    windows map[string]*counter
    limit   int
    window  time.Duration
}

func (rl *RateLimiter) Allow(key string) (allowed bool, retryAfter time.Duration)
```

### Key Extraction Functions

- `ClientIP(r)` -- parses X-Forwarded-For leftmost IP, falls back to RemoteAddr
- `SessionKey(r)` -- reads DIEGO_SESSION cookie value
- `UserOrIP(r)` -- reads UserClaims from context, falls back to ClientIP

Key prefixes (`user:`, `ip:`, `session:`) prevent collisions between strategies.

### Middleware Factory

```go
func RateLimit(limiter *RateLimiter, keyFunc func(*http.Request) string) func(http.HandlerFunc) http.HandlerFunc
```

- Nil limiter: pass-through (disabled mode)
- Empty key from keyFunc: pass-through (unidentifiable client)
- Over limit: 429 response with JSON body and `Retry-After` header

### Error Response

```json
HTTP 429 Too Many Requests
Retry-After: 45

{"error": "Rate limit exceeded", "retry_after": 45}
```

### Middleware Chain Position

- Public routes: `CORS -> CSRF -> RateLimit -> LogRequest -> Handler`
- Protected routes: `CORS -> CSRF -> Auth -> RateLimit -> LogRequest -> Handler`
- Exempt routes (`none` tier): no rate limit middleware in chain

Auth runs before rate limiting on protected routes so `UserOrIP` can extract user identity from context.

## Configuration

Environment variables with defaults:

| Variable           | Default | Description                    |
| ------------------ | ------- | ------------------------------ |
| RATE_LIMIT_ENABLED | true    | Enable/disable rate limiting   |
| RATE_LIMIT_AUTH    | 5       | Auth endpoint limit per minute |
| RATE_LIMIT_REFRESH | 10      | Refresh endpoint limit/min     |
| RATE_LIMIT_WRITE   | 10      | Write endpoint limit/min       |
| RATE_LIMIT_DEFAULT | 100     | Default endpoint limit/min     |

## Testing Strategy

**Unit tests** (`middleware/ratelimit_test.go`):

- Allow requests within limit
- Reject requests over limit with correct retryAfter
- Separate keys get independent quotas
- Window reset after expiry
- Concurrent access safety
- Key extraction functions (ClientIP, SessionKey, UserOrIP)
- Middleware factory (nil limiter, empty key, 429 response)

**E2E tests** (`e2e/ratelimit_test.go`):

- Auth endpoint rate limiting (5 OK, 6th returns 429)
- Exempt endpoints not rate limited
- Disabled mode passes all requests
- Separate IP keys get separate quotas

## Related

- Issue #44: JWT signature verification
- Issue #94: CSRF protection
- Milestone: Security & Auth Hardening
