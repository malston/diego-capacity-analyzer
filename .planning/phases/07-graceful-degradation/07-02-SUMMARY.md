---
phase: 07-graceful-degradation
plan: 02
subsystem: ui
tags: [chat-panel, data-sources, graceful-degradation, react, adaptive-prompts]

# Dependency graph
requires:
  - phase: 07-graceful-degradation
    provides: data_sources object in health endpoint (bosh, vsphere, log_cache booleans)
  - phase: 06-chat-panel-ux
    provides: starter prompts and chat panel UI components
provides:
  - DataSourceBanner component showing missing BOSH/vSphere sources
  - Adaptive starter prompts filtered by data source availability
  - Health fetch on panel open for data source status
affects: [08-polish]

# Tech tracking
tech-stack:
  added: []
  patterns: [data-source-aware-ui-filtering, health-driven-feature-gating]

key-files:
  created: []
  modified:
    - frontend/src/components/chat/ChatMessages.jsx
    - frontend/src/components/chat/ChatPanel.jsx
    - frontend/src/components/chat/ChatPanel.test.jsx

key-decisions:
  - "ALL_PROMPTS tagged with requires field (bosh, vsphere, cf) for declarative filtering"
  - "getAvailablePrompts returns original BOSH-dependent prompts when dataSources is null (fallback before health loads)"
  - "DataSourceBanner excludes log_cache from banner text (internal detail, not operator-actionable)"
  - "Health fetched on [isOpen] dep (not mount) for resilience if parent rendering strategy changes"

patterns-established:
  - "Tagged prompt data with requires field for declarative source-aware filtering"
  - "Health-driven feature gating: fetch /api/v1/health on panel open, gate UI on data_sources"

requirements-completed: [DEG-03]

# Metrics
duration: 5min
completed: 2026-03-03
---

# Phase 7 Plan 2: Data Source Banner and Adaptive Prompts Summary

**Amber info banner and source-filtered starter prompts adapt the chat panel UX to CF-only environments without BOSH or vSphere**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-03T19:30:00Z
- **Completed:** 2026-03-03T19:42:21Z
- **Tasks:** 2 (1 auto TDD + 1 human-verify checkpoint)
- **Files modified:** 3

## Accomplishments
- DataSourceBanner component renders persistent amber info banner listing missing BOSH and/or vSphere sources
- ALL_PROMPTS array tagged with `requires` field enables declarative prompt filtering by data source
- getAvailablePrompts function filters starter prompts to only show questions answerable with available data
- ChatPanel fetches /api/v1/health on each isOpen=true transition, passes data_sources to children
- CF-only fallback prompts (app distribution, memory allocation, isolation segments, app density) appear when BOSH/vSphere unavailable
- 16 new test cases covering banner visibility, prompt filtering, health fetch, and edge cases

## Task Commits

Each task was committed atomically:

1. **Task 1: Add tagged prompts, filtering function, and data source banner**
   - `d6b2e83` (test) - Failing tests for data source banner and adaptive prompts
   - `d94b096` (feat) - Implementation passing all tests

2. **Task 2: Verify graceful degradation UX** - Human-verify checkpoint, approved by operator

## Files Created/Modified
- `frontend/src/components/chat/ChatMessages.jsx` - Added ALL_PROMPTS with requires tags, getAvailablePrompts filter, DataSourceBanner component
- `frontend/src/components/chat/ChatPanel.jsx` - Added health fetch on isOpen transition, passes dataSources prop to children
- `frontend/src/components/chat/ChatPanel.test.jsx` - 16 new tests for banner visibility, prompt filtering, health fetch, and edge cases

## Decisions Made
- ALL_PROMPTS tagged with requires field (bosh, vsphere, cf) for declarative filtering rather than imperative conditionals
- getAvailablePrompts returns original BOSH-dependent prompts when dataSources is null (fallback before health loads, matching pre-Phase-7 behavior)
- DataSourceBanner excludes log_cache from banner text -- it's an internal detail, not operator-actionable
- Health fetched on [isOpen] dependency (not just mount) for resilience if parent rendering strategy changes from conditional render to CSS slide

## Deviations from Plan

None -- plan executed exactly as written.

## Issues Encountered
None

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness
- Phase 7 (Graceful Degradation) is fully complete
- Ready for Phase 8 (Polish): copy to clipboard, response feedback, procurement-oriented prompt tuning
- All data source awareness infrastructure is in place for future features

## Self-Check: PASSED

All files verified present. All commits verified in git log.

---
*Phase: 07-graceful-degradation*
*Completed: 2026-03-03*
