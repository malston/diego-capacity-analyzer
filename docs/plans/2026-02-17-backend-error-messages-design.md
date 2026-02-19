# Improve Error Messages When Backend is Unavailable

**Issue:** [#110](https://github.com/malston/diego-capacity-analyzer/issues/110)
**Date:** 2026-02-17
**Status:** Approved

## Problem

When the backend server is unreachable, the frontend displays a raw browser error:
"Error: Failed to fetch". Users cannot tell whether the server is down,
misconfigured, or blocked by CORS.

## Approach

Create a shared `apiFetch()` wrapper that catches network-level `TypeError`s and
throws a structured `ApiConnectionError` with a user-friendly summary and
expandable diagnostic detail. Migrate all API callers to use it.

Browsers intentionally make network errors and CORS errors indistinguishable
(both produce `TypeError: Failed to fetch`), so we use a single combined message
that covers both possibilities rather than attempting heuristic detection.

## Design

### 1. `apiFetch` wrapper (`frontend/src/services/apiClient.js`)

Exports `apiFetch(url, options)`:

1. Calls `fetch(url, options)` inside a try/catch.
2. If a `TypeError` is caught (network-level failure), throws an
   `ApiConnectionError` with:
   - `message` (summary): "Unable to reach the server"
   - `detail`: "The backend at {origin} is not responding. Common causes: the
     server isn't running, a firewall is blocking the connection, or CORS is
     misconfigured. Check that the backend is running and accessible."
3. If `!response.ok`, parses the JSON error body and throws with the server's
   error message (preserving current behavior). Falls back to status text if
   the body isn't JSON.
4. Returns parsed JSON on success.

`ApiConnectionError` extends `Error` so existing catch blocks continue to work.
Components that want expandable detail can check `error.detail`.

### 2. Caller migration

**`scenarioApi.js`:** All 6 methods replace raw `fetch()` with `apiFetch()`.
This eliminates the duplicated `if (!response.ok)` + JSON error parsing pattern
in each method.

**`cfApi.js`:** The `request()` method replaces raw `fetch()` with `apiFetch()`.
Existing HTTP error handling (401 check, JSON error parsing) stays as-is.

**`TASCapacityAnalyzer.jsx`:** The `loadCFData()` function replaces its raw
`fetch()` with `apiFetch()`.

### 3. UI error display

Error banners show the friendly summary by default. When `error.detail` is
present, a `<details>`/`<summary>` element lets the user expand diagnostic info.

**`ScenarioAnalyzer.jsx`:** Error banner updated from `Error: {error}` to show
summary with expandable detail.

**`TASCapacityAnalyzer.jsx`:** Dashboard error banner updated similarly. The
existing CORS-specific hint block is removed since the detail message covers
CORS as a possible cause.

### 4. Testing

**`apiClient.test.js` (unit tests):**

- Network error (TypeError) produces `ApiConnectionError` with expected summary
  and detail
- HTTP error (e.g. 500) parses JSON body and throws with server's message
- HTTP error with non-JSON body falls back to status text
- Successful response returns parsed JSON

**Updated tests for existing modules:**

- `scenarioApi.js` tests updated to reflect wrapper behavior
- `ScenarioAnalyzer.jsx` tests verify friendly summary and expandable detail
- `cfApi.js` and `TASCapacityAnalyzer.jsx` tests adjusted for changed error
  message strings

## Files Changed

- `frontend/src/services/apiClient.js` (new)
- `frontend/src/services/apiClient.test.js` (new)
- `frontend/src/services/scenarioApi.js` (modified)
- `frontend/src/services/cfApi.js` (modified)
- `frontend/src/TASCapacityAnalyzer.jsx` (modified)
- `frontend/src/components/ScenarioAnalyzer.jsx` (modified)
- Existing test files for modified modules (modified)
