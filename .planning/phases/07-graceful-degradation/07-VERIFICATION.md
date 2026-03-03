---
phase: 07-graceful-degradation
verified: 2026-03-03T20:55:00Z
status: human_needed
score: 9/9 must-haves verified
human_verification:
  - test: "Open chat panel in a live CF-only environment (no BOSH/vSphere)"
    expected: "Amber banner reads 'BOSH and vSphere data unavailable'; starter prompts show CF-only options (Review app distribution, Analyze memory allocation, etc.) instead of BOSH-dependent ones"
    why_human: "Banner and prompt filtering depend on a live health endpoint returning real data_sources values; cannot assert correct runtime behavior purely from code inspection"
  - test: "Open chat panel in an environment where BOSH is configured but vSphere is not"
    expected: "Banner reads 'vSphere data unavailable'; BOSH-dependent prompts (Assess current capacity, Plan for growth, Review cell sizing) appear; 'Check HA readiness' is absent because it requires both bosh and vsphere"
    why_human: "Combination behavior (banner text, prompt set) requires a real environment to confirm end-to-end"
  - test: "Open chat panel when all sources are available (BOSH + vSphere both configured)"
    expected: "No banner appears; all four original BOSH/vSphere prompts appear"
    why_human: "Absence of banner when all sources present cannot be confirmed without a live endpoint"
  - test: "Click a CF-only starter prompt (e.g., 'Review app distribution') in CF-only mode"
    expected: "The full question is sent and the advisor responds acknowledging it only has CF-level data, not BOSH cell metrics"
    why_human: "LLM response behavior depends on system prompt instructions and live inference -- cannot verify programmatically"
---

# Phase 7: Graceful Degradation -- Verification Report

**Phase Goal:** The advisor works meaningfully with CF-only data, explicitly communicates what it cannot analyze when BOSH or vSphere data is missing, and adapts its UI accordingly
**Verified:** 2026-03-03T20:55:00Z
**Status:** human_needed (all automated checks pass; 4 items require live environment confirmation)
**Re-verification:** No -- initial verification

---

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Health endpoint returns `data_sources` object with `bosh`, `vsphere`, `log_cache` boolean fields | VERIFIED | `health.go:43-47` returns `map[string]bool{"bosh":..., "vsphere":..., "log_cache":...}` |
| 2 | `data_sources.bosh` is true when BOSH client is configured, false otherwise | VERIFIED | `health.go:44`: `"bosh": h.boshClient != nil`; confirmed by 9-subtest table-driven test `TestHealthHandler_DataSources` |
| 3 | `data_sources.vsphere` is true when vSphere config is complete, false otherwise | VERIFIED | `health.go:45`: `"vsphere": h.cfg != nil && h.cfg.VSphereConfigured()`; nil guard prevents panic when Handler has no cfg |
| 4 | `data_sources.log_cache` reflects actual app memory data in cache | VERIFIED | `health.go:31-41`: iterates `dashboard.Apps` for any `ActualMB > 0`; covered by 3 dedicated subtests |
| 5 | Existing health response fields are unchanged | VERIFIED | `health.go:16-24`: `cf_api`, `bosh_api`, `ai_configured`, `cache_status` all present; existing tests `TestHealthHandler`, `TestHealthHandler_WithBOSH`, `TestHealthHandler_AIConfiguredTrue/False` all pass |
| 6 | Banner appears listing missing BOSH and/or vSphere sources | VERIFIED | `ChatMessages.jsx:82-97` exports `DataSourceBanner`; builds `missing` array from `!dataSources.bosh` and `!dataSources.vsphere`; renders `"${missing.join(' and ')} data unavailable"` |
| 7 | Banner does not appear when all sources are available or data not yet loaded | VERIFIED | `DataSourceBanner` returns `null` when `!dataSources` (null) and when `missing.length === 0`; 5 tests in `describe("DataSourceBanner")` confirm |
| 8 | Starter prompts adapt to available data sources | VERIFIED | `ChatMessages.jsx:68-80`: `getAvailablePrompts(dataSources)` filters `ALL_PROMPTS` by `requires` field against available source set; 7 test cases in `describe("ChatMessages - Adaptive starter prompts")` confirm |
| 9 | At least 3 prompts always appear regardless of data source availability | VERIFIED | CF-only mode yields 4 `requires:["cf"]` prompts; `getAvailablePrompts` defaults to first 4; test "always renders at least 3 starter prompt chips" passes with `dataSources={bosh:false,vsphere:false,log_cache:false}` |

**Score:** 9/9 truths verified

---

### Required Artifacts

| Artifact | Provides | Level 1: Exists | Level 2: Substantive | Level 3: Wired | Status |
|----------|----------|-----------------|----------------------|-----------------|--------|
| `backend/handlers/health.go` | `data_sources` object in health response | Yes (51 lines) | Yes -- derives bosh/vsphere/log_cache from live handler state | Yes -- registered via `h.Routes()` in `handlers/routes.go`, served at `/api/v1/health` | VERIFIED |
| `backend/handlers/handlers_test.go` | Tests for all data source configurations | Yes (1000+ lines) | Yes -- `TestHealthHandler_DataSources` with 9 table-driven subtests covering all combinations | Yes -- runs via `go test ./handlers/` | VERIFIED |
| `frontend/src/components/chat/ChatMessages.jsx` | `ALL_PROMPTS` with `requires` tags, `getAvailablePrompts`, `DataSourceBanner` | Yes (199 lines) | Yes -- 8 prompts tagged, filter function, banner component all present | Yes -- `getAvailablePrompts` called in empty state render at line 158; `DataSourceBanner` exported and used by ChatPanel | VERIFIED |
| `frontend/src/components/chat/ChatPanel.jsx` | Health fetch on `isOpen` transition, passes `dataSources` to children | Yes (136 lines) | Yes -- `useEffect([isOpen])` at lines 24-41 fetches `/api/v1/health` | Yes -- `dataSources` state passed to both `DataSourceBanner` (line 117) and `ChatMessages` (line 126) | VERIFIED |
| `frontend/src/components/chat/ChatPanel.test.jsx` | Tests for banner, prompt filtering, health fetch | Yes (879 lines) | Yes -- 16 new tests across 3 new describe blocks | Yes -- all 51 tests in file pass | VERIFIED |

---

### Key Link Verification

| From | To | Via | Status | Evidence |
|------|----|-----|--------|----------|
| `backend/handlers/health.go` | `backend/config/config.go` | `h.cfg.VSphereConfigured()` | WIRED | `health.go:45` calls `h.cfg.VSphereConfigured()`; function defined at `config.go:72` |
| `backend/handlers/health.go` | `backend/cache/cache.go` | `h.cache.Get("dashboard:all")` | WIRED | `health.go:32` calls `h.cache.Get("dashboard:all")`; result cast to `models.DashboardResponse` |
| `frontend/src/components/chat/ChatPanel.jsx` | `/api/v1/health` | `fetch` on `isOpen` transition (useEffect `[isOpen]` dep) | WIRED | `ChatPanel.jsx:29`: `fetch(\`${apiURL}/api/v1/health\`, {credentials:"include"})`; test "re-fetches on subsequent isOpen=true transitions" confirms re-fetch behavior |
| `frontend/src/components/chat/ChatPanel.jsx` | `frontend/src/components/chat/ChatMessages.jsx` | `dataSources` prop | WIRED | `ChatPanel.jsx:126`: `dataSources={dataSources}`; `ChatMessages.jsx:120` accepts `dataSources` in props; `ChatPanel.jsx:8` imports `DataSourceBanner` from `ChatMessages` |
| `frontend/src/components/chat/ChatMessages.jsx` | `ALL_PROMPTS` data | `getAvailablePrompts` filter function | WIRED | `ChatMessages.jsx:158`: `getAvailablePrompts(dataSources).map(...)` in empty state render; filter reads `ALL_PROMPTS` array at lines 17-66 |

---

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| DEG-01 | 07-01, 07-02 | Advisor works with CF-only data when BOSH and vSphere are unavailable | SATISFIED | CF-only prompts (`requires:["cf"]`) appear when BOSH/vSphere absent; system prompt context builder (prior phases) already flags missing sources via `CTX-04`; health endpoint correctly reflects degraded state |
| DEG-02 | 07-01, 07-02 | Advisor explicitly tells the operator which data sources are missing and what analysis it cannot perform | SATISFIED | `DataSourceBanner` displays "BOSH data unavailable", "vSphere data unavailable", or "BOSH and vSphere data unavailable" as persistent amber info banner; `data_sources` in health response lets frontend communicate this clearly |
| DEG-03 | 07-02 | Starter prompts adapt to available data (do not suggest vSphere-dependent questions when vSphere is unconfigured) | SATISFIED | `getAvailablePrompts` filters `ALL_PROMPTS` by `requires` field; "Check HA readiness" (`requires:["bosh","vsphere"]`) absent when vsphere false; verified by 7 test cases |

No orphaned requirements found. All three DEG requirements are claimed by plans and verified implemented.

---

### Anti-Patterns Found

| File | Pattern | Severity | Notes |
|------|---------|----------|-------|
| `ChatMessages.jsx:83,89` | `return null` | Info (not a stub) | Guard clauses in `DataSourceBanner` for null dataSources and empty missing array -- correct React pattern, not a stub |

No blockers or warnings found.

---

### Human Verification Required

The automated test suite (51 frontend tests + 13 handler tests) passes cleanly. The following items cannot be verified programmatically and require a live environment:

#### 1. CF-Only Banner and Prompt Adaptation

**Test:** Start the backend with only `CF_API_URL`, `CF_USERNAME`, `CF_PASSWORD`, `AI_PROVIDER`, and `AI_API_KEY` set (no BOSH, no vSphere). Open the chat panel.
**Expected:** Amber banner reads "BOSH and vSphere data unavailable". Starter prompts show CF-only questions (Review app distribution, Analyze memory allocation, Check isolation segments, Assess app density) instead of the default BOSH-dependent ones.
**Why human:** Banner visibility and correct prompt filtering depend on the live health endpoint returning `data_sources.bosh = false` and `data_sources.vsphere = false`.

#### 2. Partial Degradation (BOSH Only)

**Test:** Start the backend with BOSH configured (`BOSH_ENVIRONMENT`, `BOSH_CLIENT`, `BOSH_CLIENT_SECRET`) but no vSphere. Open the chat panel.
**Expected:** Banner reads "vSphere data unavailable" (BOSH is present, only vSphere missing). BOSH-dependent prompts (Assess current capacity, Plan for growth, Review cell sizing) appear. "Check HA readiness" does not appear.
**Why human:** Combination behavior (banner text, exact prompt set) requires a real BOSH connection for `data_sources.bosh = true`.

#### 3. No Banner When All Sources Available

**Test:** Start the backend with BOSH and vSphere both fully configured. Open the chat panel.
**Expected:** No amber banner appears. All four original prompts appear (Assess current capacity, Plan for growth, Review cell sizing, Check HA readiness).
**Why human:** Absence of the banner when `data_sources` has all-true values cannot be confirmed without a live environment.

#### 4. CF-Only Advisor Response Quality

**Test:** In CF-only mode, click the "Review app distribution" starter prompt.
**Expected:** The advisor responds and acknowledges it is working with CF-level data only. It does not pretend to have BOSH cell metrics or vSphere infrastructure data.
**Why human:** LLM response behavior depends on the system prompt instructions (CTX-04, DOM-03) and live inference -- cannot verify programmatically.

---

## Gaps Summary

No gaps. All automated must-haves are verified. The four human-verification items are standard end-to-end UX and LLM behavior checks that pass automated testing but require live confirmation.

---

_Verified: 2026-03-03T20:55:00Z_
_Verifier: Claude (gsd-verifier)_
