# Configurable OAuth Client for Password Grants

**Issue:** [#108](https://github.com/malston/diego-capacity-analyzer/issues/108)
**Date:** 2026-02-18
**Status:** Approved

## Problem

The backend hardcodes `req.SetBasicAuth("cf", "")` for OAuth2 password and refresh grants. This forces custom scopes (e.g., `diego-analyzer.operator`) to be added to the shared `cf` client, which is bad practice since the `cf` client is used by the CF CLI and other tools.

## Design

### Config Changes (`config/config.go`)

Two fields added to `Config`:

- `OAuthClientID` -- loaded from `OAUTH_CLIENT_ID`, default `"cf"`
- `OAuthClientSecret` -- loaded from `OAUTH_CLIENT_SECRET`, default `""`

No validation needed -- both have safe defaults matching current behavior.

### Auth Changes (`handlers/auth.go`)

Replace the two `SetBasicAuth("cf", "")` calls with `SetBasicAuth(h.cfg.OAuthClientID, h.cfg.OAuthClientSecret)`:

- `refreshWithCFUAA` (line 214)
- `authenticateWithCFUAA` (line 268)

### Environment Example (`.env.example`)

Add OAuth Client section with commented-out examples.

### Test Changes (`handlers/auth_test.go`)

- Update mock UAA server to validate Basic Auth credentials
- Return 401 if client credentials don't match expected values
- Add test verifying custom client ID/secret are sent to UAA
- Default mock to `"cf"` / `""` so existing tests remain simple

### Documentation (`docs/AUTHENTICATION.md`)

Add `OAUTH_CLIENT_ID` and `OAUTH_CLIENT_SECRET` to the configuration table (already documented elsewhere in the file).

## Backward Compatibility

Defaults to `cf` client with empty secret, matching current behavior. No breaking change for existing deployments.
