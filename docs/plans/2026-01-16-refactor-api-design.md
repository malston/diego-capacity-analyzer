I've created a comprehensive design document at **`docs/DESIGN-refactor-api.md`**. Here's a summary of what it covers:

## Design Document Summary

### 1. Overview
Documents the problems with the current 586-line monolithic handler file and establishes goals for improving maintainability while maintaining 100% backward compatibility.

### 2. Requirements
- **7 functional requirements** (P0-P2 prioritized) covering handler splitting, method validation, route registration, and API versioning
- **5 non-functional requirements** with measurable metrics (file size limits, test pass rates, no new dependencies)

### 3. Design
- **Architecture diagram** showing the new layered structure
- **Component designs** for:
  - `Route` struct and declarative route table
  - Handler base with standardized `writeJSON()`/`writeError()` helpers
  - Domain-specific handler files (health, infrastructure, scenario, analysis)
  - Composable CORS and logging middleware
  - Go 1.22+ ServeMux integration with method-aware routing

### 4. Implementation Approach
Four-phase rollout with rollback strategies:
1. **Foundation** — Add new middleware without changing behavior
2. **Handler Split** — Reorganize into domain files
3. **Route Registration** — Switch to declarative pattern
4. **Cleanup** — Remove deprecated code

### 5. Testing Strategy
- Preservation of existing 45K+ lines of handler tests
- New tests for CORS middleware and route table validation
- Integration tests verifying both `/api/` and `/api/v1/` routes
- Manual testing checklist for CLI and frontend

### 6. Open Questions
Five items requiring decisions, including deprecation warnings, method mismatch handling, and handler naming conventions.

The document is ready for review. Would you like me to make any adjustments or expand on any section?
