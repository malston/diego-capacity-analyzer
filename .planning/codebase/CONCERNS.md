# Codebase Concerns

**Analysis Date:** 2026-02-24

## Tech Debt

**`TPSDataPoint` alias and `ChunkSizeGB` constant are dead code:**

- Issue: `TPSDataPoint = models.TPSPt` in `backend/services/scenario.go:73` is labeled "for backward compatibility" but no callers exist outside that file. `ChunkSizeGB = 4` constant (line 19) is defined but never referenced anywhere.
- Files: `backend/services/scenario.go`
- Impact: Misleading -- the alias implies an external API contract that doesn't exist. The constant implies it controls behavior, but `resolveChunkSizeMB` uses a hardcoded `4096` literal instead.
- Fix approach: Remove `TPSDataPoint` alias and the `ChunkSizeGB` constant; replace the `4096` literal in `resolveChunkSizeMB` with a named constant if desired.

**CredHub config is loaded but never used:**

- Issue: `Config` struct holds `CredHubURL`, `CredHubClient`, and `CredHubSecret` fields; `config.go` reads them from env vars. No code outside `config.go` references these fields.
- Files: `backend/config/config.go:47-50`, `backend/config/config.go:96-98`
- Impact: Dead configuration confuses operators who may set these vars expecting functionality.
- Fix approach: Either implement CredHub integration or remove the fields and env var docs until it is needed.

**`GetAppMemoryPromQL` is dead code:**

- Issue: `LogCacheClient.GetAppMemoryPromQL` in `backend/services/logcache.go:148` is fully implemented but has zero call sites in production code.
- Files: `backend/services/logcache.go:148-202`
- Impact: Untested code path that rotates with the live codebase but provides no value.
- Fix approach: Remove the method or wire it into `GetApps` to replace the current gauge-envelope approach.

**Legacy `/api/` routes registered unconditionally:**

- Issue: `main.go:199-206` automatically registers every `/api/v1/` route as a matching `/api/` path, described as "backward compatibility." There is no plan or mechanism to remove them.
- Files: `backend/main.go:198-206`
- Impact: All security middleware (CSRF, auth, rate limiting, RBAC) applies equally to legacy paths, but the legacy surface area doubles the attack surface and prevents future path restructuring.
- Fix approach: Add `LEGACY_API_DISABLED` env var support, or set a deprecation timeline documented in the openapi spec, and remove in a major version.

**`VSphereClientFromEnv` ignores the `VSPHERE_INSECURE` config setting:**

- Issue: `services/vsphere.go:504` hardcodes `Insecure: true`. The config correctly reads `VSPHERE_INSECURE` (default `false`), but `handlers/handlers.go:62-68` calls `VSphereClientFromEnv` which does not accept the insecure flag.
- Files: `backend/services/vsphere.go:497-506`, `backend/handlers/handlers.go:62-68`, `backend/config/config.go:57`
- Impact: The `VSPHERE_INSECURE` env var has no effect. TLS verification is always disabled for vSphere, regardless of operator intent.
- Fix approach: Add an `insecure bool` parameter to `VSphereClientFromEnv` and pass `cfg.VSphereInsecure` from `handlers.go`.

**`CFClient` re-authenticates on every request without token expiry check:**

- Issue: `CFClient.Authenticate` always fetches a new token (no expiry guard). Handlers call it on every non-cached request: `health.go:57`, `infrastructure.go:227`, `infrastructure.go:271`.
- Files: `backend/services/cfapi.go:47-118`, `backend/handlers/health.go:57`, `backend/handlers/infrastructure.go:227,271`
- Impact: Each dashboard load or infrastructure fetch performs an extra UAA OAuth round-trip, adding ~100-500ms latency per uncached request. Under load, this also generates unnecessary UAA token churn.
- Fix approach: Mirror the `BOSHClient` pattern -- store `tokenExpiry` and skip re-auth if token is valid (see `backend/services/boshapi.go:154-167`).

**`boshapi.go` imports legacy `log` package for socks5 proxy:**

- Issue: `backend/services/boshapi.go:13` imports `"log"` solely to pass `log.Default()` to `proxy.NewSocks5Proxy`. All other logging uses `slog`.
- Files: `backend/services/boshapi.go:13,343`
- Impact: Log output from the socks5 proxy goes to the default logger, not the structured `slog` handler, making it inconsistent with the rest of the application.
- Fix approach: If the socks5 library accepts an `io.Writer`, construct a minimal bridge. If not, document as an accepted limitation.

**`defer resp.Body.Close()` inside pagination loops:**

- Issue: In `cfapi.go:GetApps` and `GetIsolationSegments`, `defer resp.Body.Close()` is called inside a `for nextURL != ""` loop (lines 161 and 353). In Go, `defer` runs at function return, not loop iteration -- all response bodies accumulate open until the function completes.
- Files: `backend/services/cfapi.go:161`, `backend/services/cfapi.go:353`
- Impact: For large environments with many pagination pages, this holds multiple open HTTP connections and their memory until `GetApps` returns.
- Fix approach: Replace `defer resp.Body.Close()` with explicit `resp.Body.Close()` after each page decode, or extract the per-page fetch into a helper function so `defer` scopes correctly.

## Known Bugs

**Isolation segment names are hardcoded to `"isolated"` for all `p-isolation-segment-*` deployments:**

- Symptoms: All cells from any `p-isolation-segment-*` BOSH deployment show `IsolationSegment: "isolated"` regardless of actual tile configuration. Multiple isolation segment tiles all map to the same segment name.
- Files: `backend/services/boshapi.go:527-533`
- Trigger: Environments with more than one isolation segment tile.
- Workaround: None -- the cell-to-segment mapping is inaccurate for multi-segment environments.

## Security Considerations

**`InsecureSkipVerify` in `handlers/auth.go` lacks `nolint:gosec` annotations:**

- Risk: Two `&tls.Config{InsecureSkipVerify: h.cfg.CFSkipSSLValidation}` calls at lines 193 and 246 lack the `//nolint:gosec` comment present elsewhere. Static analysis tools will flag these, and the inconsistency may cause confusion about whether these are reviewed or accidental.
- Files: `backend/handlers/auth.go:193,246`, `backend/handlers/cfproxy.go:57`
- Current mitigation: The value is gated on an operator-controlled config flag, same as the annotated instances.
- Recommendations: Add `//nolint:gosec // Operator-controlled setting` to match pattern in `main.go:89`.

**CF proxy handlers (`cfproxy.go`) do not validate GUIDs from URL path parameters:**

- Risk: `CFProxyAppProcesses`, `CFProxyProcessStats`, `CFProxySpaces`, and `CFProxyIsolationSegmentByGUID` read `r.PathValue("guid")` and embed it directly into the CF API proxy URL without calling `services.ValidateGUID`. A malformed path value could craft unexpected downstream URLs.
- Files: `backend/handlers/cfproxy.go:115-121`, `backend/handlers/cfproxy.go:131-137`, `backend/handlers/cfproxy.go:147-153`, `backend/handlers/cfproxy.go:163-169`
- Current mitigation: Go 1.22 router pattern matching (`/api/v1/cf/apps/{guid}/processes`) constrains path segment characters to some degree, but does not enforce UUID format.
- Recommendations: Add `services.ValidateGUID(guid)` check before constructing the proxy path, matching the pattern already used in `cfapi.go:269` and `cfapi.go:321`.

**`VSphereClientFromEnv` hardcodes `Insecure: true` (TLS bypass always active for vSphere):**

- Risk: See tech debt item above. TLS certificate verification for vSphere is permanently disabled regardless of `VSPHERE_INSECURE` setting.
- Files: `backend/services/vsphere.go:504`, `backend/handlers/handlers.go:62-68`
- Current mitigation: vSphere connection is internal/operator-controlled.
- Recommendations: Fix `VSphereClientFromEnv` to accept the insecure flag as a parameter.

**Session storage is entirely in-memory with no persistence:**

- Risk: All active sessions are lost on server restart. Users are silently logged out without warning.
- Files: `backend/services/session.go`, `backend/cache/cache.go`
- Current mitigation: Sessions use secure random IDs, httpOnly cookies, and CSRF tokens -- the implementation is correct for single-instance deployments.
- Recommendations: Document the single-instance limitation. If HA deployment is needed, replace the `cache.Cache` backend with Redis or a DB.

## Performance Bottlenecks

**`GetApps` makes 2-3 serial HTTP calls per application:**

- Problem: For each app on each pagination page, `GetApps` sequentially calls: (1) `getAppProcesses` per app, (2) `logCache.GetAppMemoryMetrics` per app, (3) `getSpaceIsolationSegment` per app. In an environment with 500 apps, this is ~1500 serial HTTP calls.
- Files: `backend/services/cfapi.go:187-243`
- Cause: No batching, no parallelism, no caching of space-to-segment mappings.
- Improvement path: Cache space-to-segment results within a single `GetApps` call (spaces repeat across apps). Parallelize Log Cache calls with bounded goroutines. Consider using CF API `include=processes` query parameter if available.

**BOSH task polling uses unconditional `time.Sleep(2 * time.Second)`:**

- Problem: `waitForTaskAndGetOutput` busy-waits with a fixed 2-second sleep per poll, for up to 60 iterations (2 minutes max). The poll loop is not context-aware -- client disconnect does not cancel the BOSH poll.
- Files: `backend/services/boshapi.go:583-615`
- Cause: `waitForTaskAndGetOutput` does not accept a context, and `time.Sleep` is not interruptible.
- Improvement path: Accept a `context.Context` parameter; replace `time.Sleep(2*time.Second)` with `select { case <-ctx.Done(): ... case <-time.After(2*time.Second): }`.

**BOSH `GetDiegoCells` does not accept or propagate context:**

- Problem: `BOSHClient.GetDiegoCells`, `getDeployments`, `getCellsForDeployment`, `authenticate`, `getUAAEndpoint`, and `waitForTaskAndGetOutput` all create `http.NewRequest` (no context). A client disconnect or timeout cannot cancel an in-flight BOSH operation.
- Files: `backend/services/boshapi.go:403-433`, all HTTP call sites in the same file
- Cause: These methods predate the context propagation refactor applied to the CF client.
- Improvement path: Add `context.Context` as the first parameter to `GetDiegoCells` and propagate through all internal methods, matching the CF client pattern.

## Fragile Areas

**`getSpaceIsolationSegment` makes 2 API calls per space without caching:**

- Files: `backend/services/cfapi.go:295-341`
- Why fragile: First fetches the space-to-segment relationship, then fetches the segment by GUID to get its name. Every app in the same space triggers these two calls again. Failures silently fall back to `"default"`, masking isolation segment misconfiguration.
- Safe modification: Add an in-function `map[string]string` cache (spaceGUID -> segmentName) passed as a parameter, or promote it to a `CFClient` field with TTL.
- Test coverage: `cfapi_test.go` tests the happy path but lacks tests for repeated-space deduplication or the fallback behavior.

**`TASCapacityAnalyzer.jsx` is a 561-line root component:**

- Files: `frontend/src/TASCapacityAnalyzer.jsx`
- Why fragile: Single file orchestrates data fetching, error handling, state management, and rendering. Changes to any concern require reading and understanding the entire file.
- Safe modification: Identify data-fetching logic and extract to a custom hook. Changes to one area risk unintended side effects in another.
- Test coverage: No direct test file for `TASCapacityAnalyzer.jsx`.

**`ScenarioResults.jsx` is a 966-line component:**

- Files: `frontend/src/components/ScenarioResults.jsx`
- Why fragile: The largest file in the frontend. Combines multiple display concerns in one component, making targeted changes difficult.
- Safe modification: Changes to chart rendering could break tabular display. Verify full scenario comparison flow after any changes.
- Test coverage: `ScenarioResults.test.jsx` exists (621 lines) but targets the full component rather than individual sub-concerns.

**`scenario.go` is 929 lines with complex calculation logic:**

- Files: `backend/services/scenario.go`
- Why fragile: Contains the core math for capacity analysis. Tested by `scenario_test.go` (2111 lines), but the logic is dense and changes may have non-obvious numeric effects.
- Safe modification: Understand all constants (`DefaultMemoryOverheadPct`, `DefaultDiskOverheadPct`, `MinChunkSizeMB`) before touching calculation code.
- Test coverage: Good test coverage exists, but the test file size suggests complexity.

## Scaling Limits

**In-memory session store:**

- Current capacity: Bounded by server process memory. Sessions accumulate until TTL expiry (token lifetime + 10 minutes buffer).
- Limit: Does not support horizontal scaling (multiple backend instances). Sessions are lost on process restart.
- Scaling path: Replace `cache.Cache` session storage with a shared store (Redis, PostgreSQL, etc.) if multi-instance deployment is needed.

**In-memory cache (`cache.Cache`):**

- Current capacity: Unbounded -- no maximum entry count or total memory cap.
- Limit: Under high load with many distinct cache keys, memory usage grows until GC pressure becomes significant.
- Scaling path: Add an entry count limit with LRU eviction, or replace with Redis.

**BOSH task polling blocks a goroutine per dashboard refresh:**

- Current capacity: Each uncached `/api/v1/dashboard` call that reaches BOSH blocks a goroutine for up to 2 minutes.
- Limit: Under concurrent load, connection pool exhaustion and goroutine accumulation are possible.
- Scaling path: Implement a background BOSH data refresh goroutine on a schedule instead of on-demand polling.

## Dependencies at Risk

**`github.com/cloudfoundry/socks5-proxy` requires legacy `log.Logger`:**

- Risk: Forces import of the legacy `log` package in `boshapi.go`, creating an inconsistency with structured `slog` logging used everywhere else.
- Impact: BOSH proxy log messages appear outside the structured log pipeline, making them hard to correlate with request traces.
- Migration plan: If the library adds an `io.Writer` or `slog.Handler` interface, migrate. Until then, accept the inconsistency or wrap.

## Test Coverage Gaps

**BOSH API context propagation is not tested:**

- What's not tested: Behavior when the HTTP request context is cancelled mid-BOSH-poll (task polling loop, `waitForTaskAndGetOutput`).
- Files: `backend/services/boshapi.go:583-615`, `backend/services/boshapi_test.go`
- Risk: Client disconnect during a BOSH operation will not cancel the in-flight calls, leaking goroutines and connections.
- Priority: Medium

**CF proxy GUID injection is not tested:**

- What's not tested: What happens when `r.PathValue("guid")` contains non-UUID characters. No test exercises a path like `../admin` or `<script>`.
- Files: `backend/handlers/cfproxy.go`, `backend/handlers/cfproxy_test.go`
- Risk: A malformed GUID could construct an unexpected downstream CF API URL.
- Priority: High

**`enrichWithCFAppData` context cancellation path is not tested:**

- What's not tested: The check `if err := ctx.Err(); err != nil` at `infrastructure.go:267` -- no test cancels the context before CF enrichment.
- Files: `backend/handlers/infrastructure.go:261-305`
- Risk: The early-exit context check may not fire correctly; untested divergence between the check and the actual cancellation behavior.
- Priority: Low

**`TASCapacityAnalyzer.jsx` has no unit tests:**

- What's not tested: The root component that orchestrates all data loading and top-level state.
- Files: `frontend/src/TASCapacityAnalyzer.jsx`
- Risk: Regressions in data loading sequence or error handling go undetected until manual QA.
- Priority: Medium

**Isolation segment assignment for multi-segment BOSH deployments is not tested:**

- What's not tested: Behavior when multiple `p-isolation-segment-*` deployments exist. All are currently hardcoded to `"isolated"`.
- Files: `backend/services/boshapi.go:527-533`
- Risk: Multi-segment environments silently receive incorrect segment assignments.
- Priority: High

---

_Concerns audit: 2026-02-24_
