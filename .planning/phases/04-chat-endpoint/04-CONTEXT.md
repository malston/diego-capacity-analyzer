# Phase 4: Chat Endpoint - Context

**Gathered:** 2026-02-24
**Status:** Ready for planning

<domain>
## Phase Boundary

SSE streaming endpoint (`POST /api/v1/chat`) that connects the frontend to the AI provider. Accepts conversation messages as JSON, returns streaming SSE token events. Protected by existing auth, CSRF, and rate-limiting middleware. Includes idle timeout and max duration to prevent hanging streams. Frontend chat panel is Phase 5.

</domain>

<decisions>
## Implementation Decisions

### SSE Event Format

- Forward tokens individually (token-by-token), matching the provider's native granularity
- Three event types: `token`, `done`, `error`
- Token event payload: `{"text": "...", "seq": N}` -- includes sequence number for ordering guarantees
- Done event payload: `{"stop_reason": "...", "usage": {"input_tokens": N, "output_tokens": N}}` -- includes token usage stats
- Error event payload: `{"code": "provider_error|timeout|rate_limit", "message": "..."}` -- machine-readable code plus human message
- Error event is the terminal event on failure (no done event follows an error)

### Request Body Contract

- POST body: `{"messages": [{"role": "user|assistant", "content": "..."}]}`
- Messages use role + content only, no metadata fields
- Maximum 50 messages per request; reject with 400 Bad Request if exceeded
- No optional LLM parameters (temperature, max tokens) in the request body -- server controls all provider settings
- Context (infrastructure state) is snapshotted server-side at request time via BuildContext + BuildSystemPrompt; frontend does not send or know about context

### Mid-Stream Error Behavior

- On provider failure after streaming starts: send `error` event with code and message, then close the stream
- Error event only, no `done` event on failure -- avoids ambiguity about response completeness
- Error codes distinguish failure types: `provider_error`, `timeout`, `rate_limit` -- enables error-specific UX in Phase 6
- Mid-stream errors logged at `slog.Warn` level with request context (user, conversation length)

### Timeout and Cancellation

- Idle timeout: 30 seconds with no token arrival triggers timeout error event and stream close
- Max total duration: 5 minutes wall clock, even if tokens are still arriving
- Client disconnect (browser closes connection): cancel upstream Anthropic API call immediately via context cancellation -- stops token generation, frees resources
- Timeout values configurable via environment variables: `AI_IDLE_TIMEOUT_SECS` (default 30), `AI_MAX_DURATION_SECS` (default 300)

### Claude's Discretion

- Handler code structure and internal wiring
- How to integrate with existing middleware chain (auth, CSRF, rate limiting)
- Go concurrency patterns for SSE streaming
- Test architecture and strategy
- Pre-stream validation ordering

</decisions>

<specifics>
## Specific Ideas

- Pre-stream failures (auth, rate limit, missing provider, bad request) return structured JSON errors with standard HTTP status codes -- SSE streaming only begins after all preconditions pass (per CHAT-04)
- Rate limit is 10 requests per minute per user (per CHAT-03)
- Reuse existing `middleware.RateLimit` with a "chat" tier
- Route registration follows the existing declarative pattern in `handlers/routes.go`

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

_Phase: 04-chat-endpoint_
_Context gathered: 2026-02-24_
