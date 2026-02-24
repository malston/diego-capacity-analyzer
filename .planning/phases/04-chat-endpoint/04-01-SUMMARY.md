---
phase: 04-chat-endpoint
plan: 01
subsystem: api
tags: [sse, streaming, chat, http, go]

requires:
  - phase: 01-ai-provider
    provides: ChatProvider interface, Message/TokenEvent types, WithSystem option
  - phase: 02-context-serializer
    provides: BuildContext function, ContextInput type
  - phase: 03-domain-expertise
    provides: BuildSystemPrompt function, static system prompt
provides:
  - POST /api/v1/chat SSE streaming endpoint
  - Pre-stream JSON validation (nil provider, bad request, empty/too many messages)
  - SSE event types: token (text+seq), done (stop_reason+usage), error (code+message)
  - Chat rate limiter tier (10/min per user)
  - AI timeout config fields (AIIdleTimeoutSecs, AIMaxDurationSecs)
affects: [05-frontend-chat, 04-02-timeouts]

tech-stack:
  added: []
  patterns:
    [
      SSE streaming via http.Flusher,
      pre-stream validation then SSE mode switch,
      context snapshot at request time,
    ]

key-files:
  created:
    - backend/handlers/chat.go
    - backend/handlers/chat_test.go
  modified:
    - backend/config/config.go
    - backend/handlers/routes.go
    - backend/main.go

key-decisions:
  - "No Role restriction on chat route -- any authenticated user can chat (trivial to tighten later via route table)"
  - "LogCacheAvailable derived by checking if any app has ActualMB > 0 in dashboard cache"
  - "maxChatMessages const (50) in chat.go; maxRequestBodySize reused from infrastructure.go (package-level const)"

patterns-established:
  - "SSE handler pattern: pre-stream JSON validation, then set SSE headers, then streaming loop"
  - "writeSSEEvent helper for consistent SSE formatting with flush"
  - "buildChatSystemPrompt snapshots cache + mutex-protected state at request time"

requirements-completed: [CHAT-01, CHAT-02, CHAT-03, CHAT-04]

duration: 5min
completed: 2026-02-24
---

# Phase 4 Plan 1: Chat Endpoint Summary

**SSE streaming chat endpoint with pre-stream validation, infrastructure context snapshot, and token/done/error event streaming**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-24T21:53:45Z
- **Completed:** 2026-02-24T21:59:01Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- POST /api/v1/chat endpoint with SSE streaming, protected by auth and rate limiting middleware
- Pre-stream validation returns JSON errors for nil provider (503), bad request (400), empty/too many messages (400), invalid role/content (400)
- SSE streaming loop produces token events with 1-based sequence numbers, done events with usage stats, and error events with machine-readable codes
- System prompt includes live infrastructure context snapshotted from cache and mutex-protected state
- 10 comprehensive tests covering all validation, streaming, and context snapshot paths

## Task Commits

Each task was committed atomically:

1. **Task 1: Config, route, and rate limiter wiring** - `8a38140` (feat)
2. **Task 2: Chat handler with TDD (RED)** - `2edb576` (test)
3. **Task 2: Chat handler with TDD (GREEN)** - `a771243` (feat)

## Files Created/Modified

- `backend/handlers/chat.go` - SSE streaming chat handler with request types and writeSSEEvent helper
- `backend/handlers/chat_test.go` - 10 tests covering validation, streaming, headers, and context snapshot
- `backend/config/config.go` - AIIdleTimeoutSecs, AIMaxDurationSecs, RateLimitChat fields
- `backend/handlers/routes.go` - Chat route registration with "chat" rate limit tier
- `backend/main.go` - Chat rate limiter tier in enabled and disabled paths

## Decisions Made

- No Role restriction on chat route: any authenticated user can chat. The route table makes this trivial to tighten later (e.g., add `Role: middleware.RoleOperator`). Default authenticated access is safer for initial release.
- LogCacheAvailable derived by checking if any app has `ActualMB > 0` in the cached dashboard data, matching the heuristic discussed in research.
- Reused `maxRequestBodySize` from `infrastructure.go` (same package) rather than redeclaring it in `chat.go`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed duplicate maxRequestBodySize constant**

- **Found during:** Task 2 (Chat handler implementation)
- **Issue:** `maxRequestBodySize` was already declared in `infrastructure.go`; redeclaring it in `chat.go` caused a compilation error
- **Fix:** Removed the duplicate declaration from `chat.go` and referenced the existing package-level constant
- **Files modified:** `backend/handlers/chat.go`
- **Verification:** `go build ./...` passes
- **Committed in:** a771243 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Trivial constant deduplication. No scope creep.

## Issues Encountered

None.

## User Setup Required

None -- no external service configuration required. The new config fields (`AI_IDLE_TIMEOUT_SECS`, `AI_MAX_DURATION_SECS`, `RATE_LIMIT_CHAT`) have sensible defaults.

## Next Phase Readiness

- Chat endpoint fully operational for Plan 02 (idle timeout and max duration timers)
- Frontend (Phase 5) can consume the SSE stream once Plan 02 completes the timeout logic
- Config fields for timeouts are already wired; Plan 02 only needs to add timer logic to the streaming loop

## Self-Check: PASSED

All 6 files verified present. All 3 commits verified in git log.

---

_Phase: 04-chat-endpoint_
_Completed: 2026-02-24_
