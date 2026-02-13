# RBAC Design: Role-Based Authorization for API Endpoints

**Issue:** #93
**Milestone:** Security & Auth Hardening
**Date:** 2026-02-11

## Summary

Add role-based access control to differentiate viewers (read-only) from
operators (read + mutate). Roles are derived from OAuth2 JWT `scope`
claims issued by CF UAA (or another IdP). UAA groups are one mechanism
for granting those scopes to users; the application itself looks only at
scopes present in the token.

## Role Model

Two application roles mapped from OAuth scopes (typically granted via
UAA groups):

| Role     | UAA Group                 | JWT Scope                 | Capabilities                                           |
| -------- | ------------------------- | ------------------------- | ------------------------------------------------------ |
| viewer   | `diego-analyzer.viewer`   | `diego-analyzer.viewer`   | Read all data, run calculations                        |
| operator | `diego-analyzer.operator` | `diego-analyzer.operator` | Everything viewer can do + mutate infrastructure state |

Operator is a superset of viewer. Users need only one of the role scopes
(usually via a single group assignment in UAA).

### Role Resolution Rules

- Token has `diego-analyzer.operator` scope: operator
- Token has `diego-analyzer.viewer` scope: viewer
- Token has both: operator (highest privilege wins)
- Token has neither: viewer (safe default for authenticated users without the role scopes)
- No token, auth disabled: no RBAC checks
- No token, auth optional: viewer

A standalone `ResolveRole(scopes []string) string` function is the single
source of truth for both Bearer token and session cookie auth paths.

## Authorization Matrix

| Endpoint                                 | Viewer | Operator | Rule             |
| ---------------------------------------- | ------ | -------- | ---------------- |
| All `GET` endpoints                      | yes    | yes      | read             |
| `POST /scenario/compare`                 | yes    | yes      | calculation      |
| `POST /infrastructure/planning`          | yes    | yes      | calculation      |
| `POST /infrastructure/manual`            | no     | yes      | mutates state    |
| `POST /infrastructure/state`             | no     | yes      | mutates state    |
| Public endpoints (health, auth, openapi) | n/a    | n/a      | no auth required |

The rule is: **viewers can read and calculate, operators can mutate state**.

## Auth Mode Behavior

| Auth Mode | RBAC Behavior                                                                   |
| --------- | ------------------------------------------------------------------------------- |
| disabled  | No RBAC checks. All requests pass through.                                      |
| optional  | Anonymous requests treated as viewer. Authenticated users get token-based role. |
| required  | Must authenticate. Role from token scopes.                                      |

## Implementation

### JWT Scope Extraction

**`services/jwks.go`:**

- Add `Scope []string` to `jwtClaimsForVerification`
- Add `Scopes []string` to `JWTClaims`, populated from parsed token

**`middleware/auth.go`:**

- Add `Scopes []string` and `Role string` to `UserClaims`
- Auth middleware maps `JWTClaims.Scopes` to role via `ResolveRole()`

**Session path:**

- Store scopes in session at login time (login handler already receives UAA token)
- Session validator populates `UserClaims.Role` from stored scopes

### RequireRole Middleware

```go
func RequireRole(role string) func(http.HandlerFunc) http.HandlerFunc
```

- Extracts `UserClaims` from request context (set by Auth middleware upstream)
- Claims present, role sufficient: pass through
- Claims present, role insufficient: 403 Forbidden
- No claims (anonymous in optional mode): viewer default applies
- Slots into middleware chain: `CORS -> CSRF -> Auth -> RequireRole -> RateLimit -> LogRequest -> Handler`

### Route Table

The `Route` struct gains a `Role string` field. Only state-mutating endpoints
set it:

```go
{Method: http.MethodPost, Path: "/api/v1/infrastructure/manual", Handler: h.SetManualInfrastructure, Role: "operator"}
{Method: http.MethodPost, Path: "/api/v1/infrastructure/state",  Handler: h.SetInfrastructureState,  Role: "operator"}
```

The `main.go` middleware chain builder adds `RequireRole(route.Role)` only when
`route.Role` is non-empty and auth mode is not disabled.

## Testing

### Unit: Role Resolution (`middleware/auth_test.go`)

- `diego-analyzer.operator` scope resolves to operator
- `diego-analyzer.viewer` scope resolves to viewer
- Both scopes resolves to operator
- Neither scope resolves to viewer
- Empty scope list resolves to viewer

### Unit: RequireRole Middleware (`middleware/rbac_test.go`)

- Operator claims + RequireRole("operator"): 200
- Viewer claims + RequireRole("operator"): 403
- Viewer claims + RequireRole("viewer"): 200
- No claims (anonymous) + RequireRole("viewer"): 200
- No claims (anonymous) + RequireRole("operator"): 403

### E2E: Endpoint Authorization (`e2e/rbac_test.go`)

- Operator token POSTs to `/infrastructure/manual`: 200
- Viewer token POSTs to `/infrastructure/manual`: 403
- Viewer token POSTs to `/scenario/compare`: 200
- Viewer token GETs `/dashboard`: 200
- Auth disabled, no token POSTs to `/infrastructure/manual`: 200

E2E tests construct JWTs with known scopes using test keys, consistent with
existing auth E2E tests.

## Deployment

Operators create UAA groups and assign users:

```bash
# Create groups (one-time setup)
uaac group add diego-analyzer.viewer
uaac group add diego-analyzer.operator

# Assign users
uaac member add diego-analyzer.operator admin
uaac member add diego-analyzer.viewer readonly-user
```

The UAA client used for OAuth must include `diego-analyzer.viewer` and
`diego-analyzer.operator` in its configured `scope` so UAA includes these
scopes in issued tokens.

No new environment variables are required. RBAC rides on top of the existing
`AUTH_MODE` setting. If UAA groups have not been created, all authenticated
users default to viewer -- a safe degradation that does not break existing
deployments.

## Files to Change

| File                      | Change                                                    |
| ------------------------- | --------------------------------------------------------- |
| `services/jwks.go`        | Add `Scope` to JWT claims structs                         |
| `middleware/auth.go`      | Add `Scopes`, `Role` to `UserClaims`; add `ResolveRole()` |
| `middleware/rbac.go`      | `RequireRole` middleware (new file)                       |
| `handlers/routes.go`      | Add `Role` field to `Route` struct; set on 2 endpoints    |
| `main.go`                 | Wire `RequireRole` into middleware chain                  |
| `handlers/auth.go`        | Store scopes in session at login                          |
| `middleware/auth_test.go` | Role resolution tests                                     |
| `middleware/rbac_test.go` | RequireRole middleware tests (new file)                   |
| `e2e/rbac_test.go`        | Endpoint authorization E2E tests (new file)               |
| `docs/AUTHENTICATION.md`  | Document RBAC setup and UAA group requirements            |
