# RBAC 403 Error Handling Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Show a user-friendly permission error with actionable UAA setup guidance when RBAC denies access to operator-only endpoints.

**Architecture:** Add `ApiPermissionError` class to the shared `apiClient.js` module (paralleling existing `ApiConnectionError`). `apiFetch` throws it on 403 responses. `cfApi.js` gets matching 403 handling in its `request()` method. No component changes needed -- `ScenarioAnalyzer.jsx` already renders `err.message` + expandable `err.detail`.

**Tech Stack:** React, Vitest, existing `apiClient.js` and `cfApi.js` modules

**Design doc:** `docs/plans/2026-02-17-rbac-403-error-handling-design.md`

---

### Task 1: Add `ApiPermissionError` class and 403 detection to `apiFetch`

**Files:**

- Modify: `frontend/src/services/apiClient.test.js`
- Modify: `frontend/src/services/apiClient.js`

**Step 1: Write the failing tests**

Add a new `describe("permission errors")` block in `apiClient.test.js` after the existing `"network errors"` block. Import `ApiPermissionError` alongside `ApiConnectionError`:

```js
import { apiFetch, ApiConnectionError, ApiPermissionError } from "./apiClient";
```

Add the test block:

```js
describe("permission errors", () => {
  it("throws ApiPermissionError on 403 response", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      statusText: "Forbidden",
      json: () => Promise.resolve({ error: "Insufficient permissions" }),
    });

    await expect(apiFetch("/api/v1/infrastructure/manual")).rejects.toThrow(
      ApiPermissionError,
    );
  });

  it("includes user-friendly message", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      statusText: "Forbidden",
      json: () => Promise.resolve({ error: "Insufficient permissions" }),
    });

    await expect(apiFetch("/api/v1/infrastructure/manual")).rejects.toThrow(
      "You don't have permission to perform this action",
    );
  });

  it("includes UAA setup guidance in detail", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      statusText: "Forbidden",
      json: () => Promise.resolve({ error: "Insufficient permissions" }),
    });

    const err = await apiFetch("/api/v1/infrastructure/manual").catch((e) => e);
    expect(err).toBeInstanceOf(ApiPermissionError);
    expect(err.detail).toContain("diego-analyzer.operator");
    expect(err.detail).toContain("AUTHENTICATION.md");
  });

  it("throws ApiPermissionError even when 403 body is not JSON", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 403,
      statusText: "Forbidden",
      json: () => Promise.reject(new Error("not json")),
    });

    await expect(apiFetch("/api/v1/infrastructure/manual")).rejects.toThrow(
      ApiPermissionError,
    );
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npx vitest run src/services/apiClient.test.js`
Expected: FAIL -- `ApiPermissionError` is not exported from `./apiClient`

**Step 3: Implement `ApiPermissionError` and 403 detection**

In `apiClient.js`, add the new class after `ApiConnectionError`:

```js
/**
 * Structured error for permission/authorization failures (HTTP 403).
 * `message` contains a user-friendly summary.
 * `detail` contains guidance on resolving the permission issue.
 */
export class ApiPermissionError extends Error {
  constructor() {
    super("You don't have permission to perform this action");
    this.name = "ApiPermissionError";
    this.detail =
      "Your account lacks the required role. Ask an administrator " +
      "to add your user to the 'diego-analyzer.operator' UAA group. " +
      "See AUTHENTICATION.md for setup instructions.";
  }
}
```

In `apiFetch`, add a 403 check before the existing generic error handling. Replace the `if (!response.ok)` block:

```js
if (!response.ok) {
  if (response.status === 403) {
    throw new ApiPermissionError();
  }

  let message;
  try {
    const body = await response.json();
    message = body.error || body.description || body.message;
  } catch {
    // Response body is not JSON
  }
  throw new Error(
    message || `Server error: ${response.status} ${response.statusText}`,
  );
}
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/services/apiClient.test.js`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add frontend/src/services/apiClient.js frontend/src/services/apiClient.test.js
git commit -m "feat: add ApiPermissionError for RBAC 403 responses (#106)"
```

---

### Task 2: Add 403 handling to `cfApi.js`

**Files:**

- Modify: `frontend/src/services/cfApi.js`

**Step 1: Add `ApiPermissionError` import**

Update the existing import line in `cfApi.js`:

```js
import { ApiConnectionError, ApiPermissionError } from "./apiClient";
```

**Step 2: Add 403 check in `request()` method**

In the `request()` method, add a 403 check after the existing 401 check (around line 39):

```js
if (!response.ok) {
  if (response.status === 401) {
    throw new Error("Authentication required. Please login.");
  }
  if (response.status === 403) {
    throw new ApiPermissionError();
  }

  let errorMsg = `API Error: ${response.status}`;
  // ... rest unchanged
```

**Step 3: Run all frontend tests**

Run: `cd frontend && npx vitest run`
Expected: ALL PASS (no behavior change for existing tests)

**Step 4: Commit**

```bash
git add frontend/src/services/cfApi.js
git commit -m "feat: add 403 handling to cfApi request method (#106)"
```

---

### Task 3: Verify end-to-end behavior and clean up

**Files:**

- Remove: `docs/plans/2026-02-17-rbac-403-error-handling-design.md`
- Remove: `docs/plans/2026-02-17-rbac-403-error-handling-plan.md`

**Step 1: Run full test suite and linters**

Run: `make check`
Expected: ALL PASS

**Step 2: Remove plan files**

```bash
rm docs/plans/2026-02-17-rbac-403-error-handling-design.md
rm docs/plans/2026-02-17-rbac-403-error-handling-plan.md
```

**Step 3: Commit cleanup**

```bash
git add -u docs/plans/
git commit -m "chore: remove completed plan files"
```
