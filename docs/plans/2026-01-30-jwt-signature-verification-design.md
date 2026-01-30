# JWT Signature Verification Design

**Date:** 2026-01-30
**Issue:** #44 - Add authorization middleware to API handler chain
**Milestone:** Security & Auth Hardening

## Context

The Diego Capacity Analyzer needs to support internet-facing deployment on Cloud Foundry. The current auth middleware validates JWT structure and expiration but does not cryptographically verify signatures. This allows token forgery attacks.

Platform engineers need CLI/automation access via Bearer tokens in addition to the existing web UI session-based auth.

## Decision Summary

| Topic                | Decision                                             |
| -------------------- | ---------------------------------------------------- |
| Deployment context   | Internet-facing on Cloud Foundry                     |
| Bearer token support | Yes, for CLI/automation                              |
| Token types          | Both user tokens and client credentials              |
| JWT verification     | JWKS-based signature verification via UAA            |
| JWKS caching         | Fetch on startup + refresh on unknown key ID         |
| Library              | Go standard library only (crypto/rsa, encoding/json) |

### Deferred Items

These will be addressed in separate issues within the same milestone:

- #92: Rate limiting (requires architecture decision)
- #93: Role-based authorization (RBAC)
- #94: CSRF token protection

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Request Flow                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Browser ──cookie──► Auth Middleware ──► Session Service ──► ✓  │
│                           │                                      │
│  CLI ────Bearer────► Auth Middleware ──► JWKS Verifier ────► ✓  │
│                                               │                  │
│                                               ▼                  │
│                                      ┌───────────────┐           │
│                                      │  UAA /token_keys         │
│                                      │  (cached)     │           │
│                                      └───────────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

### Components

**New:** `services/jwks.go` - JWKS client that:

- Fetches public keys from UAA's `/token_keys` endpoint on startup
- Caches keys in memory indexed by key ID (kid)
- Refreshes cache when encountering an unknown kid (with thundering herd prevention)
- Verifies RS256/RS384/RS512 signatures

**Modified:** `middleware/auth.go` - Updated to:

- Accept JWKSClient in AuthConfig
- Use JWKS client for Bearer token signature verification
- Reject Bearer tokens if JWKS client unavailable (graceful degradation)

## JWKS Client Design

```go
type JWKSClient struct {
    uaaURL     string
    httpClient *http.Client              // With configured timeouts
    keys       map[string]*rsa.PublicKey // kid -> public key
    mu         sync.RWMutex
    sfGroup    singleflight.Group        // Prevents thundering herd on refresh
}
```

### Behaviors

1. **Initialization** - Fetch keys from `{uaaURL}/token_keys`, parse JWKS response, cache RSA public keys by kid

2. **GetKey(kid)** - Return cached key, or refresh once if unknown. Uses `singleflight.Group` to prevent concurrent refresh requests from hammering UAA.

3. **VerifyAndParse(token)** - Parse JWT header for kid and alg, validate alg is one of the allowed RSA algorithms (RS256/RS384/RS512), retrieve key, and verify the signature. **Signature is verified BEFORE expiration check** to prevent timing attacks on forged tokens.

4. **Error handling:**
   - UAA unreachable on startup (network failure/timeout): warn, start anyway (session-only mode)
   - UAA returns HTTP error (4xx/5xx): treat as unreachable, warn and continue
   - UAA unreachable during refresh: use cached keys, log error
   - Unknown kid after refresh: reject token with clear error
   - Missing kid in JWT header: reject token (require explicit key identification)

### UAA JWKS Response Format

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-1",
      "n": "base64url-encoded-modulus",
      "e": "AQAB",
      "alg": "RS256",
      "use": "sig"
    }
  ]
}
```

## Auth Middleware Changes

### Updated AuthConfig

```go
type AuthConfig struct {
    Mode             AuthMode
    SessionValidator SessionValidatorFunc
    JWKSClient       *services.JWKSClient  // NEW
}
```

### Bearer Token Flow

```go
// If JWKS client not configured, reject Bearer tokens with helpful message
if cfg.JWKSClient == nil {
    http.Error(w, "Bearer authentication unavailable, please use web UI login", http.StatusUnauthorized)
    return
}

// Verify signature first (prevents timing attacks), then structure, then expiry
claims, err := cfg.JWKSClient.VerifyAndParse(token)
```

### Claims Extraction

| Token Type         | Username Source   | UserID Source   |
| ------------------ | ----------------- | --------------- |
| User token         | `user_name` claim | `user_id` claim |
| Client credentials | `client_id` claim | `sub` claim     |

### JWT Claims Validation

The following claims are validated:

| Claim                      | Required | Validation                                            |
| -------------------------- | -------- | ----------------------------------------------------- |
| `exp`                      | Yes      | Token not expired                                     |
| `nbf`                      | No       | If present, current time >= nbf (RFC 7519 compliance) |
| `user_name` OR `client_id` | Yes      | At least one must be present                          |

**Optional enhancements (future):**

- `aud` (audience) validation to prevent token reuse from other apps
- `iss` (issuer) validation to prevent tokens from other UAA instances

## Configuration

No new required environment variables. UAA URL is auto-discovered from CF API.

### Optional Variables

| Variable                | Default         | Purpose                                                |
| ----------------------- | --------------- | ------------------------------------------------------ |
| `UAA_URL`               | Auto-discovered | Override UAA endpoint                                  |
| `JWKS_REFRESH_INTERVAL` | 24h             | Periodic refresh interval (recommended for production) |

### HTTP Client Configuration

The JWKS client uses configured timeouts:

| Operation       | Timeout    | Rationale                        |
| --------------- | ---------- | -------------------------------- |
| Startup fetch   | 30 seconds | Allow for slow UAA on cold start |
| Runtime refresh | 10 seconds | Fail fast, use cached keys       |

### Startup Sequence

1. Load config
2. Initialize cache
3. Initialize session service
4. Discover UAA URL from CF API (`/v3/info` → `links.login.href`)
5. **Initialize JWKS client (fetch keys)** ← NEW
6. Configure auth middleware
7. Register routes
8. Start server

### Failure Modes

- CF API unreachable: fatal (existing behavior)
- UAA `/token_keys` unreachable (network/HTTP error): warning, continue without Bearer support
- JWKS parse error: warning, continue without Bearer support

## Testing Strategy

### Unit Tests - services/jwks.go

- Parse valid JWKS response
- Parse JWKS with multiple keys
- Skip non-RSA keys in JWKS
- Verify signature with known key pair
- Handle unknown kid (trigger refresh)
- Reject invalid signatures
- Reject expired tokens (with valid signature)
- Reject tokens with future nbf (not yet valid)
- Reject HS256 tokens (algorithm confusion attack)
- Reject tokens with missing kid header
- Concurrent refresh requests (thundering herd prevention)
- Network timeout during refresh

### Unit Tests - middleware/auth.go

- Valid Bearer token passes
- Valid client credentials token passes
- Invalid signature rejected with 401
- No JWKS client: Bearer rejected with helpful message, session cookies work
- Fallback from missing Bearer to session cookie

### Integration Tests - e2e/

- Mock UAA server serving JWKS endpoint
- Full auth flow with test JWT (user token)
- Full auth flow with test JWT (client credentials)
- Key rotation simulation
- UAA returns HTTP 500 (graceful degradation)

### Test Fixtures

- RSA key pair for tests (in testdata/)
- Helper to mint test JWTs with configurable claims

## Security Considerations

### OWASP Compliance

| Requirement                          | Status                   |
| ------------------------------------ | ------------------------ |
| Cryptographic signature verification | Addressed in this design |
| Token expiration validation          | Already implemented      |
| Secure token transmission (TLS)      | Platform responsibility  |
| Rate limiting                        | Deferred to #92          |
| CSRF protection                      | Deferred to #94          |

### Algorithm Validation

Only accept RS256, RS384, RS512. Reject symmetric algorithms (HS256) to prevent algorithm confusion attacks where an attacker uses the public key as an HMAC secret.

Additionally, validate that the JWT's `alg` header matches the key's algorithm to prevent key substitution attacks.

### Verification Order

**Critical:** Signature verification MUST happen BEFORE expiration check. This prevents timing-based attacks where an attacker can determine if a forged token's structure is valid by observing whether they get "expired" vs "invalid signature" errors.

## Implementation Scope

1. Implement `services/jwks.go` with singleflight for concurrent refresh
2. Update `middleware/auth.go` with signature verification (signature before expiration)
3. Update `main.go` to wire JWKS client with proper timeouts
4. Support both user tokens and client credentials
5. Validate nbf claim when present
6. Add unit and integration tests
7. Document auth requirements in OpenAPI spec
