# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-24)

**Core value:** Operators can have a conversation with a domain expert that sees their live capacity data -- turning raw metrics into actionable procurement guidance.
**Current focus:** Phase 4: Chat Endpoint

## Current Position

Phase: 4 of 8 (Chat Endpoint)
Plan: 2 of 2 in current phase
Status: In Progress
Last activity: 2026-02-24 -- Completed 04-01-PLAN.md (SSE streaming chat endpoint)

Progress: [████░░░░░░] 35%

## Performance Metrics

**Velocity:**

- Total plans completed: 7
- Average duration: 3 min
- Total execution time: 22 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
| ----- | ----- | ----- | -------- |
| 01    | 3     | 9 min | 3 min    |
| 02    | 2     | 6 min | 3 min    |
| 03    | 1     | 2 min | 2 min    |
| 04    | 1     | 5 min | 5 min    |

**Recent Trend:**

- Last 5 plans: 02-01 (3 min), 02-02 (3 min), 03-01 (2 min), 04-01 (5 min)
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

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-02-24
Stopped at: Completed 04-01-PLAN.md
Resume file: .planning/phases/04-chat-endpoint/04-01-SUMMARY.md
