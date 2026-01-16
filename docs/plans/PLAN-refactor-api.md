# Implementation Plan: API Refactoring

**Feature:** `refactor-api`  
**Design Doc:** `docs/DESIGN-refactor-api.md`  
**Created:** 2026-01-16  
**Status:** Ready for Implementation

---

## Overview

Refactor the 586-line monolithic `handlers.go` into domain-focused files with declarative routing, composable middleware, and optional API versioning. The refactoring maintains 100% backward compatibility while improving code organization and reducing boilerplate.

**Key Outcomes:**
- Split handlers into 5 domain files (each <200 lines)
- Extract CORS into composable middleware
- Add declarative route table with Go 1.22+ method routing
- Support both `/api/` and `/api/v1/` route prefixes
- Eliminate 11 manual method checks from handlers

---

## Tasks

### Phase 1: Foundation (Low Risk)

#### Task 1.1: Create CORS Middleware
- [ ] Write tests for CORS middleware in `middleware/cors_test.go`
- [ ] Create `middleware/cors.go` with `CORS()` function
- [ ] Verify CORS headers are set correctly
- [ ] Verify OPTIONS preflight returns 200 without calling handler

**Test:**
```bash
cd backend && go test ./middleware/... -v -run TestCORS
```

**Success Criteria:**
- `TestCORS_AddsHeaders` passes - verifies Allow-Origin, Allow-Methods, Allow-Headers
- `TestCORS_HandlesPreflight` passes - verifies OPTIONS returns 200
- `TestCORS_CallsNextForNonOptions` passes - verifies GET/POST pass through

**Files:**
- Create: `backend/middleware/cors.go`
- Create: `backend/middleware/cors_test.go`

---

#### Task 1.2: Create Middleware Chain Helper
- [ ] Write tests for `Chain()` function in `middleware/chain_test.go`
- [ ] Create `middleware/chain.go` with `Chain()` function
- [ ] Verify middleware executes in correct order (first is outermost)

**Test:**
```bash
cd backend && go test ./middleware/... -v -run TestChain
```

**Success Criteria:**
- `TestChain_SingleMiddleware` passes
- `TestChain_MultipleMiddleware` passes - verifies order
- `TestChain_EmptyMiddleware` passes - returns original handler

**Files:**
- Create: `backend/middleware/chain.go`
- Create: `backend/middleware/chain_test.go`

---

#### Task 1.3: Add Response Helpers to Handler
- [ ] Write tests for `writeJSON()` and `writeError()` methods
- [ ] Add `writeJSON(w, status, data)` method to Handler
- [ ] Add `writeErrorWithDetails(w, message, details, code)` method
- [ ] Ensure existing `writeError()` package function still works

**Test:**
```bash
cd backend && go test ./handlers/... -v -run TestWriteJSON
```

**Success Criteria:**
- `TestWriteJSON_SetsContentType` passes
- `TestWriteJSON_SetsStatusCode` passes
- `TestWriteJSON_EncodesData` passes
- `TestWriteError_ReturnsErrorResponse` passes
- All existing tests still pass

**Files:**
- Modify: `backend/handlers/handlers.go` (add methods)
- Modify: `backend/handlers/handlers_test.go` (add tests)

---

#### Task 1.4: Create Route Definition Structure
- [ ] Write tests for Route struct and Routes() method
- [ ] Create `handlers/routes.go` with Route struct
- [ ] Implement `Routes()` method returning all API routes
- [ ] Add validation tests for route table integrity

**Test:**
```bash
cd backend && go test ./handlers/... -v -run TestRoutes
```

**Success Criteria:**
- `TestRoutes_AllRoutesHaveRequiredFields` passes
- `TestRoutes_NoDuplicatePaths` passes
- `TestRoutes_AllPathsStartWithApiV1` passes
- Route count matches expected (11 routes)

**Files:**
- Create: `backend/handlers/routes.go`
- Create: `backend/handlers/routes_test.go`

---

### Phase 2: Handler Split (Medium Risk)

#### Task 2.1: Extract Health Handlers
- [ ] Create `handlers/health.go` with Health and Dashboard handlers
- [ ] Update handlers to use `h.writeJSON()` instead of inline encoding
- [ ] Verify all health-related tests pass
- [ ] Keep original methods in `handlers.go` as delegating wrappers (temporary)

**Test:**
```bash
cd backend && go test ./handlers/... -v -run "TestHealth|TestDashboard"
```

**Success Criteria:**
- All existing health/dashboard tests pass unchanged
- `health.go` is under 80 lines
- Response format identical to before

**Files:**
- Create: `backend/handlers/health.go`
- Modify: `backend/handlers/handlers.go` (remove implementations, keep wrappers)

---

#### Task 2.2: Extract Infrastructure Handlers
- [ ] Create `handlers/infrastructure.go` with 6 infrastructure handlers
- [ ] Rename handlers per design doc (e.g., `HandleInfrastructure` → `GetInfrastructure`)
- [ ] Update to use `h.writeJSON()` / `h.writeError()`
- [ ] Add internal state access helpers (`getInfrastructureState`, `setInfrastructureState`)

**Test:**
```bash
cd backend && go test ./handlers/... -v -run "TestInfrastructure|TestManual|TestPlanning|TestApps"
```

**Success Criteria:**
- All existing infrastructure tests pass
- `infrastructure.go` is under 200 lines
- Handler renames work via method aliases (backward compatible)

**Files:**
- Create: `backend/handlers/infrastructure.go`
- Modify: `backend/handlers/handlers.go` (remove implementations)

---

#### Task 2.3: Extract Scenario Handlers
- [ ] Create `handlers/scenario.go` with CompareScenario handler
- [ ] Rename `HandleScenarioCompare` → `CompareScenario`
- [ ] Update to use response helpers

**Test:**
```bash
cd backend && go test ./handlers/... -v -run TestScenario
```

**Success Criteria:**
- All existing scenario tests pass
- `scenario.go` is under 80 lines

**Files:**
- Create: `backend/handlers/scenario.go`
- Modify: `backend/handlers/handlers.go` (remove implementation)

---

#### Task 2.4: Extract Analysis Handlers
- [ ] Create `handlers/analysis.go` with AnalyzeBottleneck and GetRecommendations
- [ ] Rename handlers per design doc
- [ ] Update to use response helpers

**Test:**
```bash
cd backend && go test ./handlers/... -v -run "TestBottleneck|TestRecommendations"
```

**Success Criteria:**
- All existing analysis tests pass
- `analysis.go` is under 100 lines

**Files:**
- Create: `backend/handlers/analysis.go`
- Modify: `backend/handlers/handlers.go` (remove implementations)

---

#### Task 2.5: Consolidate Handler Base
- [ ] Move Handler struct, NewHandler, and helpers to clean `handler.go`
- [ ] Remove delegating wrappers from old `handlers.go`
- [ ] Delete empty `handlers.go` or rename to preserve git history
- [ ] Verify all tests pass with new file structure

**Test:**
```bash
cd backend && go test ./handlers/... -v
```

**Success Criteria:**
- All 1579 lines of existing tests pass
- No file exceeds 200 lines (verify with `wc -l`)
- `handlers.go` removed or minimal

**Files:**
- Create: `backend/handlers/handler.go` (base struct and helpers)
- Delete/Rename: `backend/handlers/handlers.go`

---

### Phase 3: Route Registration (Medium Risk)

#### Task 3.1: Update main.go to Use Route Table
- [ ] Write integration test for route registration
- [ ] Modify `main.go` to iterate over `h.Routes()`
- [ ] Use `middleware.Chain()` for middleware composition
- [ ] Use Go 1.22+ `"METHOD /path"` pattern for ServeMux

**Test:**
```bash
cd backend && go test ./... -v -run TestRouteRegistration
```

**Success Criteria:**
- All routes respond correctly
- Method validation happens at router level (405 for wrong method)
- Middleware chain applied to all routes

**Files:**
- Modify: `backend/main.go`
- Create: `backend/main_test.go` (integration tests)

---

#### Task 3.2: Add Legacy Route Compatibility
- [ ] Register both `/api/v1/` and `/api/` patterns for each route
- [ ] Add integration tests verifying both patterns work
- [ ] Verify CLI commands work with legacy routes

**Test:**
```bash
cd backend && go test ./... -v -run TestLegacyRoutes
# Manual: cd cli && go run . health
```

**Success Criteria:**
- `/api/health` and `/api/v1/health` return identical responses
- All legacy routes work
- CLI commands succeed

**Files:**
- Modify: `backend/main.go`
- Modify: `backend/main_test.go`

---

#### Task 3.3: Remove Manual Method Checks from Handlers
- [ ] Remove `if r.Method != ...` checks from all handlers (11 instances)
- [ ] Rely on Go 1.22+ ServeMux method routing
- [ ] Verify 405 responses for wrong methods

**Test:**
```bash
cd backend && go test ./handlers/... -v
# Verify: grep -r "r.Method !=" backend/handlers/ returns nothing
```

**Success Criteria:**
- No manual method checks in handler files
- Wrong method requests return 405 Method Not Allowed
- All tests pass

**Files:**
- Modify: `backend/handlers/infrastructure.go`
- Modify: `backend/handlers/scenario.go`
- Modify: `backend/handlers/analysis.go`

---

#### Task 3.4: Remove EnableCORS from Handler
- [ ] Delete `EnableCORS()` method from Handler struct
- [ ] Update any remaining references to use `middleware.CORS()`
- [ ] Verify CORS still works via middleware chain

**Test:**
```bash
cd backend && go test ./... -v
# Verify: grep -r "EnableCORS" backend/ returns nothing
```

**Success Criteria:**
- `EnableCORS` method removed
- CORS headers still present on all API responses
- OPTIONS preflight works

**Files:**
- Modify: `backend/handlers/handler.go`
- Modify: `backend/main.go`

---

### Phase 4: Cleanup and Documentation (Low Risk)

#### Task 4.1: Verify File Size Constraints
- [ ] Run `wc -l` on all handler files
- [ ] Ensure no file exceeds 200 lines
- [ ] Refactor if any file is too large

**Test:**
```bash
wc -l backend/handlers/*.go
# All files should be < 200 lines
```

**Success Criteria:**
- `handler.go` < 100 lines
- `health.go` < 80 lines
- `infrastructure.go` < 200 lines
- `scenario.go` < 80 lines
- `analysis.go` < 100 lines
- `routes.go` < 80 lines

**Files:**
- Potentially modify any handler file exceeding limits

---

#### Task 4.2: Add OPTIONS Handler for CORS Preflight
- [ ] Add catch-all OPTIONS handler for `/api/` prefix
- [ ] Test CORS preflight from browser or curl

**Test:**
```bash
curl -X OPTIONS http://localhost:8080/api/v1/health -v
# Should return 200 with CORS headers
```

**Success Criteria:**
- OPTIONS requests to any `/api/` path return 200
- CORS headers present in response

**Files:**
- Modify: `backend/main.go`

---

#### Task 4.3: Final Integration Test Suite
- [ ] Run full test suite
- [ ] Run manual testing checklist from design doc
- [ ] Verify code coverage >= 80%

**Test:**
```bash
cd backend && go test ./... -v -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
```

**Success Criteria:**
- All tests pass (including existing 45K+ lines)
- Coverage >= 80% on handlers package
- Manual tests pass (CLI, frontend if available)

**Files:**
- No changes (verification only)

---

#### Task 4.4: Update API Documentation
- [ ] Update `docs/API.md` to document `/api/v1/` routes
- [ ] Add note about legacy `/api/` route support
- [ ] Document new handler naming convention

**Test:**
- Review documentation for accuracy

**Success Criteria:**
- All endpoints documented with both patterns
- Request/response examples accurate

**Files:**
- Modify: `docs/API.md`

---

## Dependencies

```
Phase 1 (Foundation):
  Task 1.1 ─┐
  Task 1.2 ─┼─> Task 1.4 (routes can use middleware types)
  Task 1.3 ─┘

Phase 2 (Handler Split):
  Task 1.3 ─> Task 2.1 ─┐
              Task 2.2 ─┼─> Task 2.5 (consolidate after all extractions)
              Task 2.3 ─┤
              Task 2.4 ─┘

Phase 3 (Route Registration):
  Task 1.4 ─┐
  Task 2.5 ─┼─> Task 3.1 ─> Task 3.2 ─> Task 3.3 ─> Task 3.4
  Task 1.1 ─┘

Phase 4 (Cleanup):
  Task 3.4 ─> Task 4.1 ─> Task 4.2 ─> Task 4.3 ─> Task 4.4
```

**Parallel Opportunities:**
- Tasks 1.1, 1.2, 1.3 can be done in parallel
- Tasks 2.1, 2.2, 2.3, 2.4 can be done in parallel (after 1.3)

---

## Verification

### Automated Verification
```bash
# Run full test suite
cd backend && go test ./... -v

# Check file sizes
wc -l backend/handlers/*.go | grep -v total

# Verify no manual method checks remain
grep -r "r.Method !=" backend/handlers/ && echo "FAIL: Manual method checks found" || echo "PASS"

# Verify EnableCORS removed
grep -r "EnableCORS" backend/ && echo "FAIL: EnableCORS still exists" || echo "PASS"

# Check coverage
go test ./handlers/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep handlers
```

### Manual Verification Checklist
- [ ] `curl http://localhost:8080/api/health` returns health status
- [ ] `curl http://localhost:8080/api/v1/health` returns identical response
- [ ] `curl -X POST http://localhost:8080/api/health` returns 405
- [ ] `curl -X OPTIONS http://localhost:8080/api/health` returns 200 with CORS headers
- [ ] CLI: `cd cli && go run . health` succeeds
- [ ] CLI: `cd cli && go run . status` succeeds
- [ ] Frontend loads dashboard (if available)

### Rollback Procedures
- **Phase 1:** Delete new middleware files, revert handler.go changes
- **Phase 2:** `git checkout backend/handlers/handlers.go`, delete new files
- **Phase 3:** Revert main.go to explicit `http.HandleFunc()` calls
- **Phase 4:** N/A (documentation only)

---

## Estimated Effort

| Phase | Tasks | Estimated Time |
|-------|-------|----------------|
| Phase 1: Foundation | 4 | 2-3 hours |
| Phase 2: Handler Split | 5 | 3-4 hours |
| Phase 3: Route Registration | 4 | 2-3 hours |
| Phase 4: Cleanup | 4 | 1-2 hours |
| **Total** | **17** | **8-12 hours** |

---

## Open Decisions (from Design Doc)

| # | Question | Recommendation |
|---|----------|----------------|
| 1 | Deprecation warning header on `/api/` routes? | Defer - add later if migration needed |
| 2 | 405 response with `Allow` header? | Yes - Go 1.22 ServeMux does this automatically |
| 4 | Handler naming consistency? | Yes - adopt proposed names (GetX, SetX, CompareX) |
