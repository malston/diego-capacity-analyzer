# Backend Error Messages Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace raw "Failed to fetch" errors with user-friendly messages when the backend is unreachable.

**Architecture:** A shared `apiFetch()` wrapper catches network-level `TypeError`s and throws structured `ApiConnectionError` with a user-friendly summary and expandable detail. All API callers migrate to it.

**Tech Stack:** React 18, Vitest, fetch API

**Design doc:** `docs/plans/2026-02-17-backend-error-messages-design.md`

---

### Task 1: Create `apiClient.js` with TDD

**Files:**

- Create: `frontend/src/services/apiClient.js`
- Create: `frontend/src/services/apiClient.test.js`

**Step 1: Write the failing tests**

Create `frontend/src/services/apiClient.test.js`:

```js
// ABOUTME: Unit tests for shared API client wrapper
// ABOUTME: Verifies network error classification, HTTP error handling, and JSON parsing

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { apiFetch, ApiConnectionError } from "./apiClient";

describe("apiFetch", () => {
  let originalFetch;

  beforeEach(() => {
    originalFetch = global.fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  describe("network errors", () => {
    it("throws ApiConnectionError on TypeError", async () => {
      global.fetch = vi
        .fn()
        .mockRejectedValue(new TypeError("Failed to fetch"));

      await expect(apiFetch("/api/v1/health")).rejects.toThrow(
        ApiConnectionError,
      );
    });

    it("includes user-friendly summary in message", async () => {
      global.fetch = vi
        .fn()
        .mockRejectedValue(new TypeError("Failed to fetch"));

      await expect(apiFetch("/api/v1/health")).rejects.toThrow(
        "Unable to reach the server",
      );
    });

    it("includes diagnostic detail", async () => {
      global.fetch = vi
        .fn()
        .mockRejectedValue(new TypeError("Failed to fetch"));

      try {
        await apiFetch("/api/v1/health");
      } catch (err) {
        expect(err.detail).toContain("not responding");
        expect(err.detail).toContain("CORS");
      }
    });

    it("re-throws non-TypeError errors unchanged", async () => {
      const original = new Error("some other error");
      global.fetch = vi.fn().mockRejectedValue(original);

      await expect(apiFetch("/api/v1/health")).rejects.toBe(original);
    });
  });

  describe("HTTP errors", () => {
    it("throws with server error message from JSON body", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
        statusText: "Internal Server Error",
        json: () => Promise.resolve({ error: "database connection failed" }),
      });

      await expect(apiFetch("/api/v1/dashboard")).rejects.toThrow(
        "database connection failed",
      );
    });

    it("falls back to status text when body is not JSON", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: false,
        status: 502,
        statusText: "Bad Gateway",
        json: () => Promise.reject(new Error("not json")),
      });

      await expect(apiFetch("/api/v1/dashboard")).rejects.toThrow(
        "Server error: 502 Bad Gateway",
      );
    });
  });

  describe("successful responses", () => {
    it("returns parsed JSON on success", async () => {
      const payload = { cells: [], apps: [] };
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(payload),
      });

      const result = await apiFetch("/api/v1/dashboard");
      expect(result).toEqual(payload);
    });

    it("passes url and options to fetch", async () => {
      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({}),
      });

      await apiFetch("/api/v1/test", { method: "POST", body: "{}" });

      expect(global.fetch).toHaveBeenCalledWith("/api/v1/test", {
        method: "POST",
        body: "{}",
      });
    });
  });
});
```

**Step 2: Run tests to verify they fail**

Run: `cd frontend && npx vitest run src/services/apiClient.test.js`
Expected: FAIL -- module `./apiClient` not found

**Step 3: Write minimal implementation**

Create `frontend/src/services/apiClient.js`:

```js
// ABOUTME: Shared fetch wrapper with structured error handling
// ABOUTME: Catches network errors and throws user-friendly ApiConnectionError

/**
 * Structured error for network/connection failures.
 * `message` contains a user-friendly summary.
 * `detail` contains diagnostic information for troubleshooting.
 */
export class ApiConnectionError extends Error {
  constructor(url) {
    super("Unable to reach the server");
    this.name = "ApiConnectionError";

    let origin;
    try {
      origin = new URL(url, window.location.origin).origin;
    } catch {
      origin = "the configured backend";
    }

    this.detail =
      `The backend at ${origin} is not responding. ` +
      "Common causes: the server isn't running, a firewall is blocking " +
      "the connection, or CORS is misconfigured. Check that the backend " +
      "is running and accessible.";
  }
}

/**
 * Fetch wrapper that classifies errors into user-friendly categories.
 *
 * - Network failures (TypeError) become ApiConnectionError
 * - HTTP errors parse the JSON body for the server's error message
 * - Successful responses return parsed JSON
 *
 * @param {string} url - Request URL
 * @param {RequestInit} options - Fetch options
 * @returns {Promise<any>} Parsed JSON response
 */
export async function apiFetch(url, options) {
  let response;
  try {
    response = await fetch(url, options);
  } catch (err) {
    if (err instanceof TypeError) {
      throw new ApiConnectionError(url);
    }
    throw err;
  }

  if (!response.ok) {
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

  return response.json();
}
```

**Step 4: Run tests to verify they pass**

Run: `cd frontend && npx vitest run src/services/apiClient.test.js`
Expected: All 7 tests PASS

**Step 5: Commit**

```bash
git add frontend/src/services/apiClient.js frontend/src/services/apiClient.test.js
git commit -m "feat: add shared apiFetch wrapper with connection error handling (#110)"
```

---

### Task 2: Migrate `scenarioApi.js` to `apiFetch`

**Files:**

- Modify: `frontend/src/services/scenarioApi.js`

**Step 1: Update the module**

Replace all raw `fetch()` + `!response.ok` patterns with `apiFetch`. Import it
at the top. Each method simplifies to just calling `apiFetch` and returning the
result.

Special case: `getInfrastructureStatus()` (line 68-78) currently swallows errors
and returns a fallback `{ vsphere_configured: false, has_data: false }`. Keep
this behavior by wrapping its `apiFetch` call in a try/catch that returns the
fallback on any error.

The updated `scenarioApi.js` should look like:

```js
// ABOUTME: API client for what-if scenario analysis endpoints
// ABOUTME: Handles manual infrastructure input and scenario comparison with BFF auth

import { withCSRFToken } from "../utils/csrf";
import { apiFetch } from "./apiClient";

const API_URL = import.meta.env.VITE_API_URL || "";

export const scenarioApi = {
  async setManualInfrastructure(data) {
    return apiFetch(`${API_URL}/api/v1/infrastructure/manual`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(data),
    });
  },

  async compareScenario(input) {
    return apiFetch(`${API_URL}/api/v1/scenario/compare`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(input),
    });
  },

  async getLiveInfrastructure() {
    return apiFetch(`${API_URL}/api/v1/infrastructure`, {
      method: "GET",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
    });
  },

  async getInfrastructureStatus() {
    try {
      return await apiFetch(`${API_URL}/api/v1/infrastructure/status`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
      });
    } catch {
      return { vsphere_configured: false, has_data: false };
    }
  },

  async setInfrastructureState(state) {
    return apiFetch(`${API_URL}/api/v1/infrastructure/state`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(state),
    });
  },

  async calculatePlanning(input) {
    return apiFetch(`${API_URL}/api/v1/infrastructure/planning`, {
      method: "POST",
      headers: withCSRFToken({ "Content-Type": "application/json" }),
      credentials: "include",
      body: JSON.stringify(input),
    });
  },
};
```

**Step 2: Run existing tests to verify nothing breaks**

Run: `cd frontend && npx vitest run`
Expected: All existing tests PASS (ScenarioAnalyzer tests mock `scenarioApi`
directly, so they don't exercise the real fetch path)

**Step 3: Commit**

```bash
git add frontend/src/services/scenarioApi.js
git commit -m "refactor: migrate scenarioApi to shared apiFetch wrapper (#110)"
```

---

### Task 3: Migrate `cfApi.js` to `apiFetch`

**Files:**

- Modify: `frontend/src/services/cfApi.js`

**Step 1: Update the module**

In the `request()` method, replace `fetch()` with `apiFetch()`. Since `apiFetch`
already handles `!response.ok` by throwing with the server's error message, and
`cfApi.request()` has its own 401 handling and custom error parsing, we need to
be careful here.

The approach: use `apiFetch` only for the network error detection. Keep the
existing HTTP error handling by not relying on `apiFetch`'s `!response.ok` path.

Actually, looking more closely, `cfApi.request()` has a special 401 check and
custom error parsing that differs from `apiFetch`'s generic handler. The cleanest
approach is to import `ApiConnectionError` and add a catch for `TypeError` in
`request()` rather than replacing `fetch` with `apiFetch`:

```js
import { ApiConnectionError } from "./apiClient";

// In the request() method, wrap the fetch call:
async request(endpoint, options = {}) {
    const proxyPath = this.mapToProxyPath(endpoint);

    let response;
    try {
      response = await fetch(`${API_URL}${proxyPath}`, {
        ...options,
        headers: {
          "Content-Type": "application/json",
          ...options.headers,
        },
        credentials: "include",
      });
    } catch (err) {
      if (err instanceof TypeError) {
        throw new ApiConnectionError(`${API_URL}${proxyPath}`);
      }
      throw err;
    }

    if (!response.ok) {
      if (response.status === 401) {
        throw new Error("Authentication required. Please login.");
      }

      let errorMsg = `API Error: ${response.status}`;
      try {
        const error = await response.json();
        errorMsg = error.description || error.error || errorMsg;
      } catch {
        // Use default error message if JSON parsing fails
      }
      throw new Error(errorMsg);
    }

    return await response.json();
  }
```

Also add the same catch in `getInfo()` (line 211-222) since it does a raw
`fetch()` outside `request()`:

```js
async getInfo() {
    try {
      let response;
      try {
        response = await fetch(`${API_URL}/api/v1/health`, {
          credentials: "include",
        });
      } catch (err) {
        if (err instanceof TypeError) {
          throw new ApiConnectionError(`${API_URL}/api/v1/health`);
        }
        throw err;
      }
      return await response.json();
    } catch (error) {
      console.error("Error fetching CF info:", error);
      throw error;
    }
  }
```

**Step 2: Run tests to verify nothing breaks**

Run: `cd frontend && npx vitest run`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add frontend/src/services/cfApi.js
git commit -m "refactor: add connection error handling to cfApi (#110)"
```

---

### Task 4: Migrate `TASCapacityAnalyzer.jsx` to `apiFetch`

**Files:**

- Modify: `frontend/src/TASCapacityAnalyzer.jsx`

**Step 1: Update `loadCFData()` (line 51-84)**

Import `apiFetch` and replace the raw `fetch()` call. Since this function already
catches errors and sets state, `apiFetch` slots in directly:

```js
import { apiFetch } from "./services/apiClient";

// In loadCFData():
const loadCFData = async () => {
  setLoading(true);
  setError(null);

  try {
    const apiURL = import.meta.env.VITE_API_URL || "";
    const dashboardData = await apiFetch(`${apiURL}/api/v1/dashboard`, {
      headers: { "Content-Type": "application/json" },
      credentials: "include",
    });

    setData({
      cells: dashboardData.cells,
      apps: dashboardData.apps,
    });

    setUseMockData(false);
    setLastRefresh(new Date(dashboardData.metadata.timestamp));
  } catch (err) {
    console.error("Error loading data:", err);
    setError(err.message);
    setData(mockData);
    setUseMockData(true);
  } finally {
    setLoading(false);
  }
};
```

**Step 2: Update error display (line 304-341)**

Store the error detail alongside the message. Change state from a string to
storing both pieces. The catch block becomes:

```js
    } catch (err) {
      console.error("Error loading data:", err);
      setError(err.message);
      setErrorDetail(err.detail || null);
      setData(mockData);
      setUseMockData(true);
    }
```

Add the new state variable near line 46:

```js
const [errorDetail, setErrorDetail] = useState(null);
```

Update the error banner JSX (line 304-341) to replace the CORS-specific block
with a generic expandable detail:

```jsx
{
  activeTab === "dashboard" && error && (
    <div
      className="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded-lg flex items-start gap-3"
      role="alert"
    >
      <AlertTriangle
        className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5"
        aria-hidden="true"
      />
      <div className="flex-1">
        <p className="text-red-300 text-sm font-semibold">{error}</p>
        <p className="text-red-400/60 text-xs mt-1">
          Falling back to mock data.
        </p>
        {errorDetail && (
          <details className="mt-3">
            <summary className="text-xs text-slate-400 cursor-pointer hover:text-slate-300">
              Troubleshooting details
            </summary>
            <p className="mt-2 p-3 bg-slate-900/50 rounded text-xs text-slate-400">
              {errorDetail}
            </p>
          </details>
        )}
      </div>
    </div>
  );
}
```

Also clear `errorDetail` where `error` is cleared (in `loadCFData` at the start
and in `toggleDataSource`):

- Line 53: add `setErrorDetail(null);` after `setError(null);`
- Line 123: add `setErrorDetail(null);` after `setError(null);`

**Step 3: Run tests**

Run: `cd frontend && npx vitest run`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add frontend/src/TASCapacityAnalyzer.jsx
git commit -m "feat: show friendly error with expandable detail on dashboard (#110)"
```

---

### Task 5: Update `ScenarioAnalyzer.jsx` error display

**Files:**

- Modify: `frontend/src/components/ScenarioAnalyzer.jsx`

**Step 1: Update error state handling**

The component stores `err.message` as a string in `setError(err.message)` at
lines 148 and 463. To preserve the detail, store the whole error object or add a
separate state variable. The simplest approach: add `errorDetail` state and
capture `err.detail` alongside.

Add near line 64:

```js
const [errorDetail, setErrorDetail] = useState(null);
```

Update the two catch blocks:

- Line 146-149: add `setErrorDetail(err.detail || null);`
- Line 462-464: add `setErrorDetail(err.detail || null);`

Clear `errorDetail` where `error` is cleared (at the start of
`handleDataLoaded` and `handleRunAnalysis`).

**Step 2: Update error banner JSX (line 777-781)**

Replace:

```jsx
{
  error && (
    <div className="bg-red-900/20 border border-red-800 rounded-lg p-4 text-red-300">
      Error: {error}
    </div>
  );
}
```

With:

```jsx
{
  error && (
    <div className="bg-red-900/20 border border-red-800 rounded-lg p-4 text-red-300">
      <p className="font-semibold text-sm">{error}</p>
      {errorDetail && (
        <details className="mt-2">
          <summary className="text-xs text-red-400/60 cursor-pointer hover:text-red-300">
            Troubleshooting details
          </summary>
          <p className="mt-2 text-xs text-red-400/80">{errorDetail}</p>
        </details>
      )}
    </div>
  );
}
```

**Step 3: Run tests**

Run: `cd frontend && npx vitest run`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add frontend/src/components/ScenarioAnalyzer.jsx
git commit -m "feat: show friendly error with expandable detail in scenario analyzer (#110)"
```

---

### Task 6: Add integration test for error display

**Files:**

- Modify: `frontend/src/components/ScenarioAnalyzer.test.jsx`

**Step 1: Write the failing test**

Add a new describe block to `ScenarioAnalyzer.test.jsx`:

```jsx
describe("error display", () => {
  it("shows friendly error message when backend is unreachable", async () => {
    scenarioApi.setManualInfrastructure.mockRejectedValue(
      Object.assign(new Error("Unable to reach the server"), {
        detail: "The backend at http://localhost:8080 is not responding.",
      }),
    );
    mockLocalStorage.getItem.mockReturnValue(
      JSON.stringify({
        name: "Test",
        clusters: [
          { diego_cell_count: 5, diego_cell_memory_gb: 64, diego_cell_cpu: 8 },
        ],
      }),
    );

    renderWithProviders(<ScenarioAnalyzer />);

    await waitFor(() => {
      expect(
        screen.getByText("Unable to reach the server"),
      ).toBeInTheDocument();
    });

    // Detail should be in a collapsed details element
    expect(screen.getByText(/Troubleshooting details/)).toBeInTheDocument();
  });
});
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/components/ScenarioAnalyzer.test.jsx`
Expected: FAIL -- "Unable to reach the server" not found (old code shows
"Error: Unable to reach the server")

Note: If the test passes already (because our Task 5 changes are in place),
that's fine -- it confirms the feature works. If the test was written before
Task 5, run it first to verify it fails, then apply Task 5 changes.

**Step 3: Verify it passes after Task 5 changes**

Run: `cd frontend && npx vitest run src/components/ScenarioAnalyzer.test.jsx`
Expected: PASS

**Step 4: Commit**

```bash
git add frontend/src/components/ScenarioAnalyzer.test.jsx
git commit -m "test: add integration test for friendly error display (#110)"
```

---

### Task 7: Run full test suite and verify

**Step 1: Run all frontend tests**

Run: `cd frontend && npx vitest run`
Expected: All tests PASS with no regressions

**Step 2: Run linters**

Run: `make lint` (from project root)
Expected: No lint errors

**Step 3: Manual smoke test (optional)**

Start frontend without backend:

```bash
cd frontend && npx vite --port 3000
```

Navigate to both Dashboard and Capacity Planning tabs. Verify that error
messages say "Unable to reach the server" with an expandable details section,
rather than "Failed to fetch".
