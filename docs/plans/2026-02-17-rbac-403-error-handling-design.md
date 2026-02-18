# RBAC 403 Error Handling Design

**Issue:** #106 -- Frontend shows raw JSON parse error when RBAC denies infrastructure endpoints
**Date:** 2026-02-17

## Problem

Users without the `operator` role see "Insufficient permissions" (or a JSON parse error if the backend regresses) when loading Capacity Planning. No actionable guidance is provided.

This is the default experience for all users until UAA groups are created and the user is added to the `diego-analyzer.operator` group.

## Current State

- **Backend:** Already fixed (commit `fd77e71`). `rbac.go` uses `writeJSONError()`, returning `{"error": "Insufficient permissions", "code": 403}`.
- **Frontend `apiFetch`:** Handles non-JSON responses defensively (falls back to generic status text). No 403-specific handling.
- **Frontend `cfApi.js`:** Has 401-specific handling but not 403.
- **ScenarioAnalyzer.jsx:** Error banner already supports `err.message` + expandable `err.detail`.

## Solution

Add `ApiPermissionError` to `apiClient.js`, thrown on any 403 response. Follows the existing `ApiConnectionError` pattern.

### Changes

**`frontend/src/services/apiClient.js`**

- Add `ApiPermissionError` class with user-friendly `message` and `detail` containing UAA group setup guidance
- `apiFetch` checks `response.status === 403` before the generic error path and throws `ApiPermissionError`

**`frontend/src/services/apiClient.test.js`**

- Verify `apiFetch` throws `ApiPermissionError` on 403
- Verify `message` and `detail` properties

**`frontend/src/services/cfApi.js`**

- Import `ApiPermissionError`
- Add 403 handling alongside existing 401 handling in `request()`

### Unchanged

- Backend -- already returns JSON from RBAC middleware
- Error banner UI -- already supports `detail` field
- Other API services using `apiFetch` -- inherit the fix automatically

## Error Message

**User-facing (short):** "You don't have permission to perform this action"

**Expandable detail:** "Your account lacks the required role. Ask an administrator to add your user to the 'diego-analyzer.operator' UAA group. See AUTHENTICATION.md for setup instructions."
