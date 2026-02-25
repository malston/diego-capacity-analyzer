# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-24)

**Core value:** Operators can have a conversation with a domain expert that sees their live capacity data -- turning raw metrics into actionable procurement guidance.
**Current focus:** Phase 5: Chat Panel Core (Plan 01 complete, Plan 02 next)

## Current Position

Phase: 5 of 8 (Chat Panel Core) -- Executing
Plan: 1 of 2 in current phase
Status: Executing
Last activity: 2026-02-24 -- Plan 05-01 complete (chat data transport layer)

Progress: [██████░░░░] 55%

## Performance Metrics

**Velocity:**

- Total plans completed: 10
- Average duration: 4 min
- Total execution time: 39 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
| ----- | ----- | ----- | -------- |
| 01    | 3     | 9 min | 3 min    |
| 02    | 2     | 6 min | 3 min    |
| 03    | 1     | 2 min | 2 min    |
| 04    | 2     | 9 min | 4.5 min  |
| 04.1  | 1     | 9 min | 9 min    |
| 05    | 1     | 4 min | 4 min    |

**Recent Trend:**

- Last 5 plans: 04-01 (5 min), 04-02 (4 min), 04.1-01 (9 min), 05-01 (4 min)
- Trend: stable

_Updated after each plan completion_

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Roadmap: 8 phases derived from 33 requirements (comprehensive depth); backend pipeline phases 1-4 before frontend phases 5-8
- 01-01: System prompt passed per-request via WithSystem option (not construction time)
- 01-01: Temperature as \*float64 pointer to distinguish unset from zero
- 01-01: ChatConfig zero-value defaults; provider implementations resolve actual defaults
- 01-02: Default model updated to Claude Sonnet 4.5 (plan's 3.7 Sonnet reached EOL Feb 19 2026)
- 01-02: SDK client stored as value type (anthropic.Client), not pointer
- 01-02: System prompt not defaulted in resolveConfig -- per-request only
- 01-03: AI provider initialized after Handler construction via setter (matching SetSessionService pattern)
- 01-03: Startup validation: exit on unknown AI_PROVIDER or missing AI_API_KEY
- 01-03: Nil-based feature gating: chatProvider != nil determines AI availability
- 02-01: Top-N apps capped at 10 (fits token budget with room; const for easy tuning)
- 02-01: Segment sort: shared first, then alphabetical (predictable output ordering)
- 02-01: vCPU ratio flag at >4:1 [HIGH], >8:1 [CRITICAL] (matches models.CPURiskLevel thresholds)
- 02-01: CF API status checks both Apps and Cells length (either suffices to confirm connectivity)
- 02-02: All 4 edge-case tests passed immediately -- Plan 02-01 handles credential safety, aggregation, markers, and token budget
- 02-02: Credential safety uses belt-and-suspenders: compile-time type constraint + runtime sentinel scan
- 02-02: InfrastructureState.Name not rendered in output -- only cluster names appear
- 03-01: Prompt uses XML tags for section delineation per Anthropic best practices (domain_knowledge, procurement_framing, response_rules, data_gap_handling)
- 03-01: Measured instruction language per Claude 4.6 guidance -- no excessive MUST/ALWAYS/NEVER emphasis
- 03-01: Materiality-based gap handling with general rules plus examples (not exhaustive per-marker if/then rules)
- 03-01: Static prompt is ~3900 chars (~975 tokens), well under 10000-char budget
- 04-01: No Role restriction on chat route -- any authenticated user can chat (trivial to tighten later)
- 04-01: LogCacheAvailable derived by checking if any app has ActualMB > 0 in cached dashboard
- 04-01: maxRequestBodySize reused from infrastructure.go (package-level const, not redeclared)
- 04-02: Config fields are int seconds (minimum 1s granularity); tests use 1-second timeouts for reasonable speed
- 04-02: maxDurationExceeded channel (not atomic.Bool) distinguishes max-duration from client disconnect in ctx.Done case
- 04-02: Test helper newChatTestHandler sets production-default timeout values (30s idle, 300s max) to avoid zero-value timer issues
- 04.1-01: sync.RWMutex for per-user scenario map (matches existing infraMutex pattern; typed map with read/write distinction)
- 04.1-01: Auth check first in Chat(), before nil provider check -- unauthenticated users get 401, not 503
- 04.1-01: Scenario stored per-user keyed on claims.Username; not stored for anonymous users
- 04.1-01: Skipped adding AI_MODEL to health endpoint (keeps changes minimal; health reports availability, not config)
- 05-01: withCSRFToken reused from existing csrf.js utility (consistent with apiClient.js pattern)
- 05-01: Async generator pattern for streamChat enables natural for-await consumption in hook
- 05-01: Functional state updates in token handler prevent stale closure issues during streaming

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-02-24
Stopped at: Completed 05-01-PLAN.md
Resume file: .planning/phases/05-chat-panel-core/05-02-PLAN.md
