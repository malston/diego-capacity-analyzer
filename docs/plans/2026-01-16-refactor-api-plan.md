I've created a detailed implementation plan at **`docs/plans/PLAN-refactor-api.md`**. Here's a summary:

## Plan Summary

### Structure
**17 tasks across 4 phases**, estimated 8-12 hours total:

| Phase | Focus | Tasks | Risk |
|-------|-------|-------|------|
| **Phase 1** | Foundation | 4 | Low |
| **Phase 2** | Handler Split | 5 | Medium |
| **Phase 3** | Route Registration | 4 | Medium |
| **Phase 4** | Cleanup | 4 | Low |

### Key Tasks

**Phase 1 - Foundation:**
1. Create CORS middleware (`middleware/cors.go`)
2. Create middleware chain helper (`middleware/chain.go`)
3. Add response helpers to Handler (`writeJSON`, `writeError`)
4. Create Route definition structure (`handlers/routes.go`)

**Phase 2 - Handler Split:**
1. Extract health handlers → `health.go` (~80 lines)
2. Extract infrastructure handlers → `infrastructure.go` (~200 lines)
3. Extract scenario handlers → `scenario.go` (~80 lines)
4. Extract analysis handlers → `analysis.go` (~100 lines)
5. Consolidate handler base → `handler.go`

**Phase 3 - Route Registration:**
1. Update `main.go` to use route table with Go 1.22+ patterns
2. Add legacy `/api/` route compatibility
3. Remove 11 manual method checks from handlers
4. Remove `EnableCORS` from Handler struct

**Phase 4 - Cleanup:**
1. Verify file size constraints (<200 lines each)
2. Add OPTIONS handler for CORS preflight
3. Run final integration test suite
4. Update API documentation

### Each Task Includes
- Clear success criteria
- Test commands to verify
- Files to create/modify
- TDD approach (test first)

### Parallelization Opportunities
- Tasks 1.1, 1.2, 1.3 can run in parallel
- Tasks 2.1-2.4 can run in parallel after 1.3 completes

Would you like me to start implementing any specific task?
