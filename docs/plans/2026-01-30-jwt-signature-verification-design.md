# JWT Signature Verification Design

**Date:** 2026-01-30
**Issue:** #44 - Add authorization middleware to API handler chain
**Milestone:** Security & Auth Hardening

## Context

The Diego Capacity Analyzer needs to support internet-facing deployment on Cloud Foundry. The current auth middleware validates JWT structure and expiration but does not cryptographically verify signatures. This allows token forgery attacks.

Platform engineers need CLI/automation access via Bearer tokens in addition to the existing web UI session-based auth.

## Decision Summary

| Topic                | Decision                                     |
| -------------------- | -------------------------------------------- |
| Deployment context   | Internet-facing on Cloud Foundry             |
| Bearer token support | Yes, for CLI/automation                      |
| Token types          | Both user tokens and client credentials      |
| JWT verification     | JWKS-based signature verification via UAA    |
| JWKS caching         | Fetch on startup + refresh on unknown key ID |

### Deferred Items

These will be addressed in separate issues within the same milestone:

- Rate limiting (requires architecture decision)
- Role-based authorization (RBAC)
- CSRF token protection

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
- Refreshes cache when encountering an unknown kid
- Verifies RS256/RS384/RS512 signatures

**Modified:** `middleware/auth.go` - Updated to:

- Accept JWKSClient in AuthConfig
- Use JWKS client for Bearer token signature verification
- Reject Bearer tokens if JWKS client unavailable (graceful degradation)

## JWKS Client Design

```go
type JWKSClient struct {
    uaaURL     string
    httpClient *http.Client
    keys       map[string]*rsa.PublicKey  // kid -> public key
    mu         sync.RWMutex
}
```

### Behaviors

1. **Initialization** - Fetch keys from `{uaaURL}/token_keys`, parse JWKS response, cache RSA public keys by kid

2. **GetKey(kid)** - Return cached key, or refresh once if unknown

3. **Verify(token)** - Parse JWT header for kid, retrieve key, verify RS256 signature

4. **Error handling:**
   - UAA unreachable on startup: warn, start anyway (session-only mode)
   - UAA unreachable during refresh: use cached keys
   - Unknown kid after refresh: reject token

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
// If JWKS client not configured, reject Bearer tokens
if cfg.JWKSClient == nil {
    http.Error(w, "Bearer authentication not configured", http.StatusUnauthorized)
    return
}

// Verify signature + structure + expiry
claims, err := cfg.JWKSClient.VerifyAndParse(token)
```

### Claims Extraction

| Token Type         | Username Source   | UserID Source   |
| ------------------ | ----------------- | --------------- |
| User token         | `user_name` claim | `user_id` claim |
| Client credentials | `client_id` claim | `sub` claim     |

## Configuration

No new required environment variables. UAA URL is auto-discovered from CF API.

### Optional Variables

| Variable                | Default         | Purpose                   |
| ----------------------- | --------------- | ------------------------- |
| `UAA_URL`               | Auto-discovered | Override UAA endpoint     |
| `JWKS_REFRESH_INTERVAL` | 0 (disabled)    | Periodic refresh interval |

### Startup Sequence

1. Load config
2. Initialize cache
3. Initialize session service
4. Discover UAA URL from CF API
5. **Initialize JWKS client (fetch keys)** ← NEW
6. Configure auth middleware
7. Register routes
8. Start server

### Failure Modes

- CF API unreachable: fatal (existing behavior)
- UAA `/token_keys` unreachable: warning, continue without Bearer support
- JWKS parse error: warning, continue without Bearer support

## Testing Strategy

### Unit Tests - services/jwks.go

- Parse valid JWKS response
- Verify signature with known key pair
- Handle unknown kid (trigger refresh)
- Reject invalid signatures
- Reject expired tokens with valid signature
- Reject HS256 tokens (algorithm confusion attack)

### Unit Tests - middleware/auth.go

- Valid Bearer token passes
- Invalid signature rejected with 401
- No JWKS client: Bearer rejected, session cookies work
- Fallback from missing Bearer to session cookie

### Integration Tests - e2e/

- Mock UAA server serving JWKS endpoint
- Full auth flow with test JWT
- Key rotation simulation

### Test Fixtures

- RSA key pair for tests (in testdata/)
- Helper to mint test JWTs with configurable claims

## Security Considerations

### OWASP Compliance

| Requirement                          | Status                     |
| ------------------------------------ | -------------------------- |
| Cryptographic signature verification | Addressed in this design   |
| Token expiration validation          | Already implemented        |
| Secure token transmission (TLS)      | Platform responsibility    |
| Rate limiting                        | Deferred to separate issue |
| CSRF protection                      | Deferred to separate issue |

### Algorithm Validation

Only accept RS256, RS384, RS512. Reject symmetric algorithms (HS256) to prevent algorithm confusion attacks where an attacker uses the public key as an HMAC secret.

## Implementation Scope

1. Implement `services/jwks.go`
2. Update `middleware/auth.go` with signature verification
3. Update `main.go` to wire JWKS client
4. Support both user tokens and client credentials
5. Add unit and integration tests
6. Document auth requirements in OpenAPI spec
