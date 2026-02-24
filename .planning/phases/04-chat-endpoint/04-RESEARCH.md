# Phase 4: Chat Endpoint - Research

**Researched:** 2026-02-24
**Domain:** Go HTTP SSE streaming, Anthropic SDK integration, middleware composition
**Confidence:** HIGH

## Summary

Phase 4 connects the existing `ChatProvider` (Phase 1), context builder (Phase 2), and system prompt (Phase 3) to a single HTTP endpoint: `POST /api/v1/chat`. The handler accepts a JSON conversation, builds context from cached dashboard/infrastructure state, calls the Anthropic provider, and streams SSE token events back to the client.

The Go standard library (`net/http`) provides everything needed for SSE. No third-party SSE libraries are required. The `http.Flusher` interface on `ResponseWriter` enables per-event flushing, and `http.ResponseController` (Go 1.20+, project uses Go 1.24) allows per-request write deadline management for long-lived streams. The existing middleware chain (auth, CSRF, rate limit) already produces JSON errors, which naturally satisfies CHAT-04's pre-stream error requirement -- middleware rejects before the handler runs.

**Primary recommendation:** Build a single `Chat` handler method on the existing `Handler` struct that validates the request, snapshots context, starts the provider stream, then loops over the token channel writing SSE events. Use `context.WithCancel` wrapping the request context for client-disconnect propagation, and `time.AfterFunc`/timer reset for idle timeout. No new libraries needed.

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions

**SSE Event Format:**

- Forward tokens individually (token-by-token), matching the provider's native granularity
- Three event types: `token`, `done`, `error`
- Token event payload: `{"text": "...", "seq": N}` -- includes sequence number for ordering guarantees
- Done event payload: `{"stop_reason": "...", "usage": {"input_tokens": N, "output_tokens": N}}` -- includes token usage stats
- Error event payload: `{"code": "provider_error|timeout|rate_limit", "message": "..."}` -- machine-readable code plus human message
- Error event is the terminal event on failure (no done event follows an error)

**Request Body Contract:**

- POST body: `{"messages": [{"role": "user|assistant", "content": "..."}]}`
- Messages use role + content only, no metadata fields
- Maximum 50 messages per request; reject with 400 Bad Request if exceeded
- No optional LLM parameters (temperature, max tokens) in the request body -- server controls all provider settings
- Context (infrastructure state) is snapshotted server-side at request time via BuildContext + BuildSystemPrompt; frontend does not send or know about context

**Mid-Stream Error Behavior:**

- On provider failure after streaming starts: send `error` event with code and message, then close the stream
- Error event only, no `done` event on failure -- avoids ambiguity about response completeness
- Error codes distinguish failure types: `provider_error`, `timeout`, `rate_limit` -- enables error-specific UX in Phase 6
- Mid-stream errors logged at `slog.Warn` level with request context (user, conversation length)

**Timeout and Cancellation:**

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

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>

## Phase Requirements

| ID      | Description                                                                                        | Research Support                                                                                                                                           |
| ------- | -------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CHAT-01 | `POST /api/v1/chat` accepts JSON with conversation messages and returns SSE stream of token events | SSE via `http.Flusher`, event format from locked decisions, handler pattern documented below                                                               |
| CHAT-02 | Chat endpoint requires authentication (same middleware as all other endpoints)                     | Existing `middleware.Auth` + route registration with `Public: false` (default) handles this automatically                                                  |
| CHAT-03 | Chat endpoint is rate-limited to 10 requests per minute per user                                   | New "chat" tier in rate limiter map with limit=10, keyed by `middleware.UserOrIP`                                                                          |
| CHAT-04 | Chat endpoint returns structured JSON errors for pre-stream failures                               | Middleware chain (auth, CSRF, rate limit) already returns JSON errors; handler validates request body and provider availability before writing SSE headers |
| CHAT-05 | Chat endpoint includes idle timeout detection                                                      | `time.Timer` reset on each token; `time.AfterFunc` or select-based pattern in the streaming loop                                                           |

</phase_requirements>

## Standard Stack

### Core

| Library            | Version        | Purpose                                | Why Standard                                                        |
| ------------------ | -------------- | -------------------------------------- | ------------------------------------------------------------------- |
| `net/http`         | Go 1.24 stdlib | SSE streaming via `http.Flusher`       | No third-party needed; `ResponseWriter` supports `Flush()` natively |
| `encoding/json`    | Go 1.24 stdlib | Request parsing and SSE event payloads | Already used throughout codebase                                    |
| `log/slog`         | Go 1.24 stdlib | Structured logging                     | Already the project's logging standard                              |
| `anthropic-sdk-go` | v1.26.0        | Upstream LLM streaming                 | Already integrated in `services/ai/anthropic.go`                    |

### Supporting

| Library                   | Version         | Purpose                                                | When to Use                                                           |
| ------------------------- | --------------- | ------------------------------------------------------ | --------------------------------------------------------------------- |
| `http.ResponseController` | Go 1.20+ stdlib | Per-request write deadline for long-lived SSE          | Use to disable/extend server `WriteTimeout` for streaming connections |
| `context`                 | Go 1.24 stdlib  | Cancellation propagation (client disconnect, timeouts) | Wrap `r.Context()` with cancel; pass to `ChatProvider.Chat()`         |
| `time`                    | Go 1.24 stdlib  | Idle timer, max duration timer                         | `time.NewTimer` for idle, `time.AfterFunc` for max duration           |

### Alternatives Considered

| Instead of            | Could Use                       | Tradeoff                                                                                             |
| --------------------- | ------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Raw `http.Flusher`    | `github.com/tmaxmax/go-sse`     | Adds dependency for a simple pattern; project has no complex SSE needs (single endpoint, no pub/sub) |
| `time.Timer` for idle | `context.WithTimeout` per token | Timer reset is cleaner than creating new contexts per token                                          |

**Installation:**
No new dependencies required. Everything uses Go standard library + existing `anthropic-sdk-go`.

## Architecture Patterns

### Recommended Project Structure

```
handlers/
├── chat.go              # Chat SSE handler + request/response types
├── chat_test.go         # Unit tests for chat handler
```

One new file in `handlers/`. The handler follows the same pattern as other handlers: a method on `*Handler` that accesses `chatProvider`, `cache`, `infrastructureState`, and `cfg`.

### Pattern 1: Pre-Stream Validation Then SSE

**What:** Validate everything that can fail (auth, CSRF, rate limit, request body, provider availability) while the response is still in JSON mode. Only after all preconditions pass, set SSE headers and begin streaming. Once SSE headers are written, errors must be sent as SSE `error` events.

**When to use:** Always -- this is the core pattern for CHAT-04.

**Example:**

```go
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
    // Phase 1: Pre-stream validation (JSON errors)
    if h.chatProvider == nil {
        h.writeError(w, "AI advisor not configured", http.StatusServiceUnavailable)
        return
    }

    var req ChatRequest
    if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBodySize)).Decode(&req); err != nil {
        h.writeError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if len(req.Messages) == 0 {
        h.writeError(w, "Messages required", http.StatusBadRequest)
        return
    }
    if len(req.Messages) > 50 {
        h.writeError(w, "Maximum 50 messages per request", http.StatusBadRequest)
        return
    }

    // Phase 2: Build context snapshot
    systemPrompt := h.buildChatSystemPrompt()

    // Phase 3: Set SSE headers and begin streaming
    flusher, ok := w.(http.Flusher)
    if !ok {
        h.writeError(w, "Streaming not supported", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")

    // Disable write deadline for streaming (Go 1.20+)
    rc := http.NewResponseController(w)
    rc.SetWriteDeadline(time.Time{})

    // ... streaming loop ...
}
```

### Pattern 2: Idle Timer Reset in Streaming Loop

**What:** Use a `time.Timer` that resets on each token arrival. If the timer fires, send a timeout error event and close. A separate max-duration timer fires once and cannot be reset.

**When to use:** CHAT-05 idle timeout + max duration.

**Example:**

```go
ctx, cancel := context.WithCancel(r.Context())
defer cancel()

// Max duration timer
maxDuration := time.AfterFunc(h.cfg.AIMaxDurationSecs * time.Second, func() {
    cancel()
})
defer maxDuration.Stop()

idleTimeout := time.Duration(h.cfg.AIIdleTimeoutSecs) * time.Second
idleTimer := time.NewTimer(idleTimeout)
defer idleTimer.Stop()

messages := toProviderMessages(req.Messages)
tokenCh := h.chatProvider.Chat(ctx, messages, ai.WithSystem(systemPrompt))

var seq int
for {
    select {
    case event, ok := <-tokenCh:
        if !ok {
            return // channel closed without done event (shouldn't happen)
        }
        if !idleTimer.Stop() {
            select {
            case <-idleTimer.C:
            default:
            }
        }
        idleTimer.Reset(idleTimeout)

        // Handle event...

    case <-idleTimer.C:
        writeSSEEvent(w, flusher, "error", ErrorPayload{
            Code:    "timeout",
            Message: "No response from AI provider within timeout window",
        })
        return

    case <-ctx.Done():
        // Client disconnect or max duration
        if ctx.Err() == context.Canceled {
            // Could be client disconnect or max duration cancel
            // If max duration, send error event
            // If client disconnect, just return (no one to send to)
        }
        return
    }
}
```

### Pattern 3: Context Snapshot for System Prompt

**What:** Read cached dashboard data and mutex-protected infrastructure state at request time, build the system prompt once, and pass it to the provider. No re-reading during the stream.

**When to use:** Every chat request. Already established pattern from Phase 2/3.

**Example:**

```go
func (h *Handler) buildChatSystemPrompt() string {
    input := ai.ContextInput{
        BOSHConfigured:    h.boshClient != nil,
        VSphereConfigured: h.cfg.VSphereConfigured(),
    }

    if cached, found := h.cache.Get("dashboard:all"); found {
        if dashboard, ok := cached.(models.DashboardResponse); ok {
            input.Dashboard = &dashboard
            input.LogCacheAvailable = /* derive from dashboard data */
        }
    }

    h.infraMutex.RLock()
    input.Infra = h.infrastructureState
    h.infraMutex.RUnlock()

    context := ai.BuildContext(input)
    return ai.BuildSystemPrompt(context)
}
```

### Pattern 4: SSE Event Writing Helper

**What:** A small helper function that formats SSE events per the spec (`event: type\ndata: json\n\n`) and flushes immediately.

**When to use:** Every SSE write in the streaming loop.

**Example:**

```go
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventType string, data any) error {
    payload, err := json.Marshal(data)
    if err != nil {
        return err
    }
    fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, payload)
    flusher.Flush()
    return nil
}
```

### Anti-Patterns to Avoid

- **Writing SSE headers before validation:** Once `Content-Type: text/event-stream` is written, you cannot return JSON errors. Validate everything first.
- **Blocking on channel read without timeout:** Without the idle timer select, a hung provider connection blocks the goroutine indefinitely.
- **Forgetting to call `Flush()`:** Without explicit flush, the ResponseWriter buffers output and the client sees nothing until the buffer fills or the handler returns.
- **Using `http.TimeoutHandler`:** This wraps the ResponseWriter and breaks `http.Flusher`. Use `http.ResponseController.SetWriteDeadline(time.Time{})` instead.
- **Not propagating context cancellation:** If the client disconnects but the provider goroutine keeps running, you waste resources and money on API calls. The request context (`r.Context()`) is canceled on client disconnect; pass it through to `ChatProvider.Chat()`.

## Don't Hand-Roll

| Problem                             | Don't Build                       | Use Instead                                             | Why                                                                                          |
| ----------------------------------- | --------------------------------- | ------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| SSE format                          | Custom string building            | Small `writeSSEEvent` helper                            | SSE has a specific format (`event:\ndata:\n\n`) with subtleties around newlines and encoding |
| Write deadline management           | Global server timeout adjustments | `http.ResponseController.SetWriteDeadline(time.Time{})` | Per-request control without affecting other endpoints                                        |
| Rate limiting                       | New rate limiter for chat         | Existing `middleware.RateLimit` with new "chat" tier    | Already tested, handles key extraction, Retry-After headers                                  |
| Auth enforcement                    | Custom auth checks in handler     | Existing `middleware.Auth` via route registration       | Route `Public: false` (default) enforces auth automatically                                  |
| JSON errors for pre-stream failures | Custom error writing              | Existing `h.writeError()`                               | Consistent error format across all endpoints                                                 |

**Key insight:** The existing middleware chain handles CHAT-02, CHAT-03, and most of CHAT-04 without any code in the handler. The handler only needs to check `chatProvider != nil` and validate the request body before switching to SSE mode.

## Common Pitfalls

### Pitfall 1: ResponseWriter Wrapping Breaks Flusher

**What goes wrong:** Middleware that wraps `http.ResponseWriter` (e.g., response loggers, gzip middleware) can break the `http.Flusher` type assertion.
**Why it happens:** Wrapped writers don't always implement `Flusher`.
**How to avoid:** The project's middleware does not wrap ResponseWriter (verified by reading all middleware). The `http.Flusher` assertion will succeed with Go's default ResponseWriter. If middleware is added later that wraps the writer, the `flusher, ok := w.(http.Flusher)` check will catch it.
**Warning signs:** SSE handler returns 500 "Streaming not supported" in test or production.

### Pitfall 2: Timer Drain Race in Select

**What goes wrong:** `time.Timer.Stop()` returns false if the timer already fired, but the channel still has a value. If you don't drain the channel, the next `select` iteration may immediately trigger the timeout case.
**Why it happens:** Go timer semantics require draining after a failed `Stop()`.
**How to avoid:** Use the standard drain pattern:

```go
if !idleTimer.Stop() {
    select {
    case <-idleTimer.C:
    default:
    }
}
idleTimer.Reset(idleTimeout)
```

**Warning signs:** Spurious timeout errors immediately after receiving a token.

### Pitfall 3: Writing to Closed Connection

**What goes wrong:** After the client disconnects, writing to the ResponseWriter returns an error that is often ignored, potentially causing panics or goroutine leaks.
**Why it happens:** The streaming loop continues after `r.Context()` is canceled but before the select catches it.
**How to avoid:** Check `ctx.Done()` in the select alongside the token channel. Ignore write errors in the client-disconnect path (there is no one to send to). Use `defer cancel()` to ensure the provider context is canceled.
**Warning signs:** "broken pipe" or "connection reset" errors in logs.

### Pitfall 4: Buffering by Reverse Proxies

**What goes wrong:** Nginx, CloudFlare, or gorouter buffers SSE responses, causing tokens to arrive in batches instead of individually.
**Why it happens:** Reverse proxies buffer by default for performance.
**How to avoid:** Set `X-Accel-Buffering: no` header (Nginx) and `Cache-Control: no-cache`. The gorouter (CF) does not buffer streaming responses when these headers are set.
**Warning signs:** Client receives tokens in bursts instead of one-by-one.

### Pitfall 5: Max Duration vs Idle Timeout Interaction

**What goes wrong:** Max duration fires via `context.Cancel()` but the handler tries to send an error event after the context is already canceled, resulting in the error event never reaching the client.
**Why it happens:** The max duration timer cancels the context, but the write to the ResponseWriter may fail because the context is done.
**How to avoid:** Distinguish between max-duration cancel and client-disconnect cancel. For max duration, send the error event before returning. Use a separate flag or channel for max duration rather than relying solely on context cancellation.
**Warning signs:** Stream ends abruptly without an error event when max duration is reached.

### Pitfall 6: Sequence Number Off-by-One

**What goes wrong:** If sequence numbers start at 0 but the client expects 1, or vice versa, ordering logic breaks.
**Why it happens:** Convention mismatch between backend and frontend.
**How to avoid:** Start at `seq: 1` (1-based) so the first token is unambiguously "first". Document the convention. The frontend (Phase 5) will consume this.
**Warning signs:** Frontend shows tokens in wrong order or drops the first token.

## Code Examples

### Chat Request/Response Types

```go
// ChatRequest is the POST body for /api/v1/chat.
type ChatRequest struct {
    Messages []ChatMessage `json:"messages"`
}

// ChatMessage represents a single conversation turn.
type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

// TokenPayload is the SSE data for "token" events.
type TokenPayload struct {
    Text string `json:"text"`
    Seq  int    `json:"seq"`
}

// DonePayload is the SSE data for "done" events.
type DonePayload struct {
    StopReason string     `json:"stop_reason"`
    Usage      UsageStats `json:"usage"`
}

// UsageStats reports token consumption.
type UsageStats struct {
    InputTokens  int64 `json:"input_tokens"`
    OutputTokens int64 `json:"output_tokens"`
}

// ErrorPayload is the SSE data for "error" events.
type ErrorPayload struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

### Route Registration

```go
// In handlers/routes.go Routes() method:
{Method: http.MethodPost, Path: "/api/v1/chat", Handler: h.Chat, RateLimit: "chat", Role: middleware.RoleOperator},
```

### Rate Limiter Configuration

```go
// In main.go rate limiter map:
"chat": middleware.RateLimit(middleware.NewRateLimiter(cfg.RateLimitChat, window), middleware.UserOrIP),
```

With `RATE_LIMIT_CHAT` defaulting to 10 in `config.go`, and validation range 1-10000.

### Config Additions

```go
// In config.Config struct:
AIIdleTimeoutSecs int // AI streaming idle timeout (default: 30)
AIMaxDurationSecs int // AI streaming max duration (default: 300)
RateLimitChat     int // Requests per minute for chat endpoint (default: 10)

// In config.Load():
AIIdleTimeoutSecs: getEnvInt("AI_IDLE_TIMEOUT_SECS", 30),
AIMaxDurationSecs: getEnvInt("AI_MAX_DURATION_SECS", 300),
RateLimitChat:     getEnvInt("RATE_LIMIT_CHAT", 10),
```

## State of the Art

| Old Approach                    | Current Approach                                 | When Changed                 | Impact                                                      |
| ------------------------------- | ------------------------------------------------ | ---------------------------- | ----------------------------------------------------------- |
| `http.CloseNotifier`            | `r.Context()` for client disconnect detection    | Go 1.8+ (deprecated in 1.11) | Use `r.Context().Done()` instead of CloseNotifier           |
| Global `WriteTimeout` only      | `http.ResponseController.SetWriteDeadline()`     | Go 1.20                      | Per-request deadline override without global config changes |
| Type-assert `http.Flusher` only | `http.ResponseController.Flush()` also available | Go 1.20                      | Either works; `Flusher` type assertion is still idiomatic   |

**Deprecated/outdated:**

- `http.CloseNotifier`: Deprecated since Go 1.11. Use `r.Context()` instead. The project already uses contexts correctly.

## Open Questions

1. **Differentiate max-duration cancel from client disconnect**
   - What we know: Both result in `ctx.Done()` firing. The max-duration `AfterFunc` calls `cancel()`, and client disconnect triggers `r.Context()` cancellation.
   - What's unclear: Whether `context.Cause(ctx)` (Go 1.20+) can be used here, or if a simple boolean flag set by the max-duration callback is cleaner.
   - Recommendation: Use a dedicated `maxDurationExceeded` atomic bool or channel. When it fires, send the error event first, then cancel. This avoids relying on context cause introspection. Simple and explicit.

2. **Role requirement for chat endpoint**
   - What we know: CHAT-02 says "same middleware as all other endpoints." Most non-public endpoints default to any authenticated user. The CONTEXT.md says operators can send messages.
   - What's unclear: Whether `viewer` role should also be able to chat, or only `operator`.
   - Recommendation: Use `Role: middleware.RoleOperator` since the CONTEXT.md phase boundary says "Operators can send conversation messages." The route table makes this trivial to change later.

3. **LogCacheAvailable derivation**
   - What we know: The `ContextInput.LogCacheAvailable` flag needs to be set. The dashboard response doesn't have an explicit Log Cache status field.
   - What's unclear: How to derive Log Cache availability from the cached dashboard data.
   - Recommendation: Check if any app in the dashboard has `ActualMB > 0` -- this indicates Log Cache metrics were available. This is the same heuristic the context builder uses.

## Sources

### Primary (HIGH confidence)

- `/anthropics/anthropic-sdk-go` Context7 - streaming API, message construction, context cancellation
- Go standard library `net/http` documentation - `http.Flusher`, `http.ResponseController`, SSE headers
- Codebase inspection of `backend/handlers/`, `backend/middleware/`, `backend/services/ai/`, `backend/config/` - existing patterns

### Secondary (MEDIUM confidence)

- [Go SSE handler patterns](https://thoughtbot.com/blog/writing-a-server-sent-events-server-in-go) - Thoughtbot SSE guide
- [Go ResponseController usage](https://www.alexedwards.net/blog/how-to-use-the-http-responsecontroller-type) - Alex Edwards blog on per-request deadlines
- [OneUptime SSE streaming in Go (2026)](https://oneuptime.com/blog/post/2026-01-25-server-sent-events-streaming-go/view) - Current SSE patterns

### Tertiary (LOW confidence)

- None -- all findings verified with primary or secondary sources

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - Uses only Go stdlib and existing project dependencies, all verified
- Architecture: HIGH - Follows established codebase patterns (handler methods, middleware chain, route table), SSE is well-documented in Go
- Pitfalls: HIGH - Timer drain, flusher wrapping, and proxy buffering are well-known Go SSE pitfalls confirmed by multiple sources

**Research date:** 2026-02-24
**Valid until:** 2026-03-24 (stable domain, Go stdlib, no fast-moving dependencies)
