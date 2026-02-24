# Phase 1: Provider Foundation - Research

**Researched:** 2026-02-24
**Domain:** LLM provider abstraction in Go with streaming Anthropic integration
**Confidence:** HIGH

## Summary

Phase 1 delivers a pluggable `ChatProvider` Go interface backed by a concrete Anthropic Claude implementation using the official `anthropic-sdk-go` SDK (v1.26.0). The SDK is mature (HIGH source reputation, 94.3 benchmark score), provides first-class streaming via `ssestream.Stream[T]`, and handles authentication via `ANTHROPIC_API_KEY` environment variable natively. The streaming pattern maps cleanly to a Go channel primitive as specified in CONTEXT.md -- the Anthropic provider goroutine reads from `stream.Next()` and writes `TokenEvent` values into a channel the caller reads.

The existing codebase already follows the pattern this phase needs: optional services (BOSH, vSphere) are nil when unconfigured, the health endpoint reports their status, and configuration loads from environment variables via `config.Load()`. The AI provider follows this exact pattern -- `nil` when `AI_PROVIDER` is unset, a real client when configured.

**Primary recommendation:** Use `anthropic-sdk-go` v1.26.0 with functional options on `Chat()`, deliver tokens via `<-chan TokenEvent`, and place the provider under `backend/services/ai/` as a self-contained package with its own interface and Anthropic implementation.

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions

- Simple message type: `Message{Role, Content}` -- no metadata, timestamps, or token counts on messages
- Functional options pattern for provider-specific parameters (e.g., `WithMaxTokens(4096)`, `WithTemperature(0.3)`). Extensible without breaking the interface
- Provider accepts `context.Context` and handles cancellation (stops Anthropic stream, closes channel on ctx.Done())
- Go channel primitive: `Chat` returns `<-chan TokenEvent`. Provider fills channel in a goroutine, closes when done
- Provider handles context cancellation -- caller does not need to manage cleanup
- Track token usage (input/output counts) from day one via `slog`. Anthropic SDK provides usage data in stream responses
- Log usage per request for cost visibility from the start

### Claude's Discretion

- System prompt: whether passed per-request or set at construction time (consider Phase 2/3 needs where context changes per request)
- Health check: whether ChatProvider exposes a Ping/Validate method or health is inferred from provider-is-non-nil
- Event shape: what TokenEvent carries (text-only vs structured with stop reason, usage, error)
- Return timing: whether Chat returns immediately or blocks until first token/error (consider Phase 4's need to distinguish pre-stream errors from mid-stream errors)
- Model selection: default Claude model, balancing reasoning capability against cost for capacity planning domain
- Parameter defaults: temperature, max output tokens -- tuned for deterministic infrastructure advice
- Token budget: whether to enforce a local input token cap or rely on API limits

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>

## Phase Requirements

| ID      | Description                                                                                                                    | Research Support                                                                                                                                                                                                |
| ------- | ------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| PROV-01 | Backend exposes a `ChatProvider` interface with streaming support that decouples LLM interaction from HTTP handling            | Interface design with `Chat(ctx, []Message, ...Option) <-chan TokenEvent` using functional options pattern; Go channel primitive for streaming decouples provider from transport                                |
| PROV-02 | Anthropic Claude provider implementation streams token-by-token responses using the official `anthropic-sdk-go` SDK            | SDK v1.26.0 provides `Messages.NewStreaming()` returning `ssestream.Stream[MessageStreamEventUnion]`; `ContentBlockDeltaEvent` with `TextDelta` delivers individual tokens; `Message.Accumulate()` tracks usage |
| PROV-03 | Provider is configured via `AI_PROVIDER` and `AI_API_KEY` environment variables with validation at startup                     | Follows existing `config.Load()` pattern; SDK's `option.WithAPIKey()` accepts key programmatically; validation checks provider name + key presence at startup                                                   |
| PROV-04 | When `AI_PROVIDER` is unset, the advisor feature is completely disabled and the health endpoint reports `ai_configured: false` | Follows existing nil-service pattern (BOSH, vSphere); health endpoint adds `ai_configured` field keyed on provider-is-non-nil                                                                                   |

</phase_requirements>

## Standard Stack

### Core

| Library            | Version | Purpose                                    | Why Standard                                                                                                                               |
| ------------------ | ------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `anthropic-sdk-go` | v1.26.0 | Anthropic Claude API client with streaming | Official Anthropic SDK; HIGH source reputation; typed Go API with `ssestream.Stream[T]` for SSE streaming; handles auth, retries, timeouts |

### Supporting

| Library    | Version          | Purpose                            | When to Use                                                          |
| ---------- | ---------------- | ---------------------------------- | -------------------------------------------------------------------- |
| `log/slog` | stdlib (Go 1.24) | Structured logging for token usage | Already used project-wide; log input/output token counts per request |
| `context`  | stdlib           | Cancellation propagation           | Already used project-wide; pass to SDK and use for channel cleanup   |

### Alternatives Considered

| Instead of               | Could Use                       | Tradeoff                                                                                                            |
| ------------------------ | ------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `anthropic-sdk-go`       | Raw HTTP + SSE parsing          | Full control but must handle auth, retries, SSE parsing, union types, error mapping manually                        |
| Go channel for streaming | `io.Reader` / callback function | Channel is idiomatic Go concurrency; `io.Reader` works but callers must buffer; callbacks couple caller to provider |
| Functional options       | Config struct parameter         | Config struct requires version bumps when adding fields; functional options are additive and zero-value-safe        |

**Installation:**

```bash
cd backend && go get github.com/anthropics/anthropic-sdk-go@v1.26.0
```

## Architecture Patterns

### Recommended Project Structure

```
backend/services/ai/
├── provider.go        # ChatProvider interface, Message, TokenEvent, Option types
├── anthropic.go       # Anthropic implementation of ChatProvider
├── anthropic_test.go  # Unit tests (mocked HTTP or SDK interface)
├── options.go         # Functional option constructors (WithMaxTokens, WithTemperature, etc.)
└── options_test.go    # Option tests
```

Rationale: A dedicated `ai` sub-package under `services/` keeps the provider abstraction self-contained. The interface, types, and options live in `provider.go` (the package's public contract). The Anthropic implementation lives in `anthropic.go`. Future providers (OpenAI, Ollama) add files without modifying the interface.

### Pattern 1: ChatProvider Interface with Functional Options

**What:** A Go interface that accepts conversation messages and returns a streaming channel, with provider-specific behavior controlled by functional options.

**When to use:** Any LLM interaction from the backend.

**Example:**

```go
// provider.go

// Message represents a single conversation turn.
type Message struct {
    Role    string // "user", "assistant", "system"
    Content string
}

// TokenEvent carries a streaming token or terminal signal.
type TokenEvent struct {
    Text       string // Token text (empty on final event)
    Done       bool   // True on the final event
    StopReason string // Why generation stopped (only on Done)
    Usage      *Usage // Token counts (only on Done)
    Err        error  // Non-nil if stream failed
}

// Usage tracks token consumption for cost monitoring.
type Usage struct {
    InputTokens  int64
    OutputTokens int64
}

// ChatProvider streams LLM responses token by token.
type ChatProvider interface {
    Chat(ctx context.Context, messages []Message, opts ...Option) <-chan TokenEvent
}

// Option configures a Chat request.
type Option func(*ChatConfig)

// ChatConfig holds resolved options for a single request.
type ChatConfig struct {
    MaxTokens   int64
    Temperature *float64
    System      string
}
```

### Pattern 2: Anthropic Provider Streaming via Goroutine

**What:** The Anthropic implementation spawns a goroutine that reads from the SDK's `ssestream.Stream` and writes `TokenEvent` values to a channel, handling context cancellation.

**When to use:** When `AI_PROVIDER=anthropic`.

**Example:**

```go
// anthropic.go

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message, opts ...Option) <-chan TokenEvent {
    cfg := p.resolveConfig(opts)
    ch := make(chan TokenEvent, 1) // buffered to allow final event without blocking

    go func() {
        defer close(ch)

        params := p.buildParams(messages, cfg)
        stream := p.client.Messages.NewStreaming(ctx, params)

        var accumulated anthropic.Message
        for stream.Next() {
            event := stream.Current()
            if err := accumulated.Accumulate(event); err != nil {
                p.send(ctx, ch, TokenEvent{Err: err, Done: true})
                return
            }

            switch variant := event.AsAny().(type) {
            case anthropic.ContentBlockDeltaEvent:
                switch delta := variant.Delta.AsAny().(type) {
                case anthropic.TextDelta:
                    p.send(ctx, ch, TokenEvent{Text: delta.Text})
                }
            }
        }

        if err := stream.Err(); err != nil {
            p.send(ctx, ch, TokenEvent{Err: err, Done: true})
            return
        }

        // Final event with usage data
        usage := &Usage{
            InputTokens:  accumulated.Usage.InputTokens,
            OutputTokens: accumulated.Usage.OutputTokens,
        }
        slog.Info("chat completed",
            "input_tokens", usage.InputTokens,
            "output_tokens", usage.OutputTokens,
            "model", string(p.model),
        )
        p.send(ctx, ch, TokenEvent{
            Done:       true,
            StopReason: string(accumulated.StopReason),
            Usage:      usage,
        })
    }()

    return ch
}

// send writes to ch respecting context cancellation.
func (p *AnthropicProvider) send(ctx context.Context, ch chan<- TokenEvent, event TokenEvent) {
    select {
    case ch <- event:
    case <-ctx.Done():
    }
}
```

### Pattern 3: Feature Gating via Nil Provider

**What:** When `AI_PROVIDER` is unset, no provider is constructed. Health endpoint checks for nil.

**When to use:** Startup configuration and health checks.

**Example:**

```go
// In config.Load() or main.go initialization:
aiProvider := os.Getenv("AI_PROVIDER")
aiAPIKey := os.Getenv("AI_API_KEY")

if aiProvider == "" {
    slog.Info("AI provider not configured, advisor feature disabled")
    // chatProvider remains nil
} else if aiProvider == "anthropic" {
    if aiAPIKey == "" {
        slog.Error("AI_API_KEY required when AI_PROVIDER is set", "provider", aiProvider)
        os.Exit(1)
    }
    chatProvider = ai.NewAnthropicProvider(aiAPIKey, ai.DefaultConfig())
    slog.Info("AI provider initialized", "provider", aiProvider)
} else {
    slog.Error("Unknown AI_PROVIDER value", "provider", aiProvider)
    os.Exit(1)
}

// Health endpoint:
resp["ai_configured"] = chatProvider != nil
```

### Anti-Patterns to Avoid

- **Leaking SDK types through the interface:** The `ChatProvider` interface must not expose `anthropic.MessageParam` or any SDK-specific types. Callers only see `Message`, `TokenEvent`, and `Option`.
- **Unbuffered channel with deferred close:** If the channel is unbuffered and the goroutine tries to send a final `Done` event after the caller stops reading, the goroutine leaks. Use a buffered channel (size 1) or always respect `ctx.Done()` on sends.
- **Blocking Chat return on first token:** `Chat` should return the channel immediately. The goroutine fills it asynchronously. Pre-stream errors (invalid config, auth failure) are delivered as a `TokenEvent{Err: ..., Done: true}` on the channel, not as a second return value -- this keeps the interface simple and uniform.
- **Accumulating full response text in the provider:** The provider sends tokens as they arrive. If a caller needs the full text, the caller accumulates it. The provider only accumulates the SDK's `Message` struct to extract usage at the end.

## Don't Hand-Roll

| Problem                     | Don't Build                       | Use Instead                                  | Why                                                                                  |
| --------------------------- | --------------------------------- | -------------------------------------------- | ------------------------------------------------------------------------------------ |
| SSE stream parsing          | Custom SSE reader                 | `anthropic-sdk-go` `Messages.NewStreaming()` | SDK handles SSE framing, event typing, reconnection, and error mapping               |
| API authentication          | Manual OAuth/key header injection | `option.WithAPIKey()`                        | SDK manages auth headers, reads `ANTHROPIC_API_KEY` env var by default               |
| API error classification    | Status code switch statements     | `*anthropic.Error` with `errors.As()`        | SDK provides typed errors with `StatusCode`, request/response dumps                  |
| Request timeout calculation | Manual timeout logic              | SDK auto-calculates from `MaxTokens`         | SDK sets appropriate timeouts based on model and token limits                        |
| Union type deserialization  | Manual JSON parsing of SSE events | SDK's `AsAny()` type switch pattern          | SDK handles all event variants (`ContentBlockDeltaEvent`, `MessageDeltaEvent`, etc.) |

**Key insight:** The `anthropic-sdk-go` SDK handles all Anthropic API complexity -- SSE parsing, event union types, authentication, error typing, and timeout calculation. Building custom solutions for any of these adds maintenance burden with no benefit.

## Common Pitfalls

### Pitfall 1: Goroutine Leak on Context Cancellation

**What goes wrong:** Provider goroutine blocks on `ch <- event` after caller's context is cancelled and caller stops reading the channel.
**Why it happens:** If the channel is unbuffered or full, the send blocks forever when no one reads.
**How to avoid:** Always use `select { case ch <- event: case <-ctx.Done(): }` for every channel send. Buffer the channel (size 1 minimum) so the final Done event can be sent without blocking even if the caller has already stopped reading.
**Warning signs:** Goroutine count grows over time under load; leaked goroutines visible in pprof.

### Pitfall 2: Not Closing the SDK Stream on Cancellation

**What goes wrong:** When context is cancelled, the goroutine exits but the SDK's HTTP response body is never closed, leaking connections.
**Why it happens:** `stream.Next()` returns false when context is cancelled, but `stream.Close()` is never called.
**How to avoid:** Use `defer stream.Close()` after creating the stream, or explicitly call it on exit paths. The SDK's `ssestream.Stream` implements `Close() error`.
**Warning signs:** HTTP connection pool exhaustion under cancellation load; "too many open files" errors.

### Pitfall 3: Exposing SDK Types in the Interface

**What goes wrong:** The `ChatProvider` interface or `TokenEvent` references `anthropic.TextDelta` or other SDK types, coupling all callers to the Anthropic SDK.
**Why it happens:** Convenient to pass SDK types directly rather than mapping to domain types.
**How to avoid:** Define domain types (`Message`, `TokenEvent`, `Usage`) in the provider package. Map SDK types to domain types in the Anthropic implementation only.
**Warning signs:** Import of `anthropic-sdk-go` in packages that shouldn't know about Anthropic.

### Pitfall 4: Forgetting the Final Done Event

**What goes wrong:** Caller reads tokens but never knows the stream is finished because no terminal event is sent.
**Why it happens:** Channel close is the only signal, but callers may not check for channel closure correctly in a `select`.
**How to avoid:** Send a final `TokenEvent{Done: true, Usage: ..., StopReason: ...}` before closing the channel. Callers can check `event.Done` explicitly. Channel closure is the backup signal.
**Warning signs:** Caller hangs waiting for more tokens; SSE endpoint never sends the `[DONE]` event.

### Pitfall 5: Mishandling the System Prompt Type

**What goes wrong:** System prompt is passed as a string but the SDK expects `[]anthropic.TextBlockParam`.
**Why it happens:** The SDK's `System` field on `MessageNewParams` is `[]TextBlockParam`, not `string`.
**How to avoid:** Convert the system prompt string to `[]anthropic.TextBlockParam{{Text: systemPrompt}}` when building params.
**Warning signs:** Compilation error or empty system prompt in API requests.

## Code Examples

Verified patterns from official sources:

### Creating the Anthropic Client

```go
// Source: anthropic-sdk-go README
import (
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
)

client := anthropic.NewClient(
    option.WithAPIKey(apiKey), // Explicit key, not env var
)
```

### Streaming with Token Extraction

```go
// Source: anthropic-sdk-go README - streaming example
stream := client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
    Model:     anthropic.ModelClaude3_7SonnetLatest,
    MaxTokens: 4096,
    System: []anthropic.TextBlockParam{
        {Text: systemPrompt},
    },
    Messages: sdkMessages,
})
defer stream.Close()

var accumulated anthropic.Message
for stream.Next() {
    event := stream.Current()
    accumulated.Accumulate(event)

    switch variant := event.AsAny().(type) {
    case anthropic.ContentBlockDeltaEvent:
        switch delta := variant.Delta.AsAny().(type) {
        case anthropic.TextDelta:
            // delta.Text contains the token string
        }
    }
}
// After loop: accumulated.Usage.InputTokens, accumulated.Usage.OutputTokens
```

### Building Messages from Domain Types

```go
// Convert domain Message slice to SDK MessageParam slice
func toSDKMessages(msgs []Message) []anthropic.MessageParam {
    sdkMsgs := make([]anthropic.MessageParam, 0, len(msgs))
    for _, m := range msgs {
        switch m.Role {
        case "user":
            sdkMsgs = append(sdkMsgs, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
        case "assistant":
            sdkMsgs = append(sdkMsgs, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
        }
    }
    return sdkMsgs
}
```

### Error Handling

```go
// Source: anthropic-sdk-go README - error handling
import "errors"

if err := stream.Err(); err != nil {
    var apiErr *anthropic.Error
    if errors.As(err, &apiErr) {
        slog.Error("Anthropic API error",
            "status", apiErr.StatusCode,
            "message", err.Error(),
        )
        // StatusCode 401 = invalid key, 429 = rate limited, 529 = overloaded
    }
}
```

## Discretion Recommendations

Based on research, here are recommendations for areas left to Claude's discretion:

### System Prompt: Per-Request (RECOMMENDED)

Pass the system prompt as a functional option (`WithSystem(string)`) rather than setting it at construction time. Phases 2 and 3 build context that changes per request -- a construction-time system prompt would require creating a new provider instance for each context change, which is wasteful. The SDK's `System` field on `MessageNewParams` is per-request by design.

### Health Check: Provider-is-Non-Nil (RECOMMENDED)

Infer health from whether the provider was successfully created (`chatProvider != nil`). A `Ping()`/`Validate()` method would need to make an API call (billing cost, latency, and the API has no free ping endpoint). The existing codebase uses the nil-check pattern for BOSH and vSphere. Keep it consistent.

### TokenEvent Shape: Structured with Stop Reason, Usage, and Error (RECOMMENDED)

`TokenEvent` should carry `Text`, `Done`, `StopReason`, `Usage`, and `Err` fields. Text-only events would require a separate error channel or sentinel values, complicating the caller. The Done event doubles as the usage delivery mechanism for slog. The Err field enables Phase 4's HTTP endpoint to distinguish error types (auth failure vs mid-stream error) without parsing channel close reasons.

### Return Timing: Return Channel Immediately (RECOMMENDED)

`Chat` returns `<-chan TokenEvent` immediately. The goroutine fills it asynchronously. Pre-stream errors (bad config, immediate auth rejection) are delivered as `TokenEvent{Err: ..., Done: true}` on the channel. This keeps the interface uniform -- callers always read from a channel, never check two return paths. Phase 4's SSE endpoint benefits because it can immediately start writing SSE headers, then stream events as they arrive.

### Model Selection: Claude 3.7 Sonnet (RECOMMENDED)

Use `anthropic.ModelClaude3_7SonnetLatest` as the default. Sonnet balances reasoning capability against cost -- it handles technical analysis well without the Opus price point. For capacity planning advice (N-1 calculations, procurement guidance), Sonnet's reasoning is sufficient. Allow override via `WithModel(string)` option for future flexibility. Note: newer models (Claude Sonnet 4, etc.) may exist by deployment time; the `Latest` suffix in the constant ensures the latest Sonnet 3.7 variant.

### Parameter Defaults: Low Temperature, 4096 Max Tokens (RECOMMENDED)

- **Temperature:** `0.3` -- capacity planning advice should be deterministic and consistent. Lower temperature reduces variability in infrastructure recommendations.
- **Max output tokens:** `4096` -- sufficient for detailed capacity analysis responses; not so large that it generates unnecessarily long answers.
- Both should be overridable via `WithTemperature()` and `WithMaxTokens()` options.

### Token Budget: Rely on API Limits (RECOMMENDED)

Do not enforce a local input token cap in Phase 1. The Anthropic API enforces its own context window limits and returns a clear error when exceeded. Phase 2's context builder will need to manage input size, but that is explicitly out of scope. Log token usage via slog so operators can monitor costs and set alerts externally.

## State of the Art

| Old Approach                                 | Current Approach                              | When Changed | Impact                                                                              |
| -------------------------------------------- | --------------------------------------------- | ------------ | ----------------------------------------------------------------------------------- |
| Community Go SDKs (liushuangls/go-anthropic) | Official `anthropic-sdk-go` from Anthropic    | 2024         | First-party support, typed unions, streaming, maintained by Anthropic               |
| Raw SSE parsing                              | SDK's `ssestream.Stream[T]` with typed events | 2024         | No manual SSE parsing needed; `AsAny()` type switch handles all event variants      |
| `anthropic.F()` field helpers                | Direct value assignment with `omitzero`       | SDK v1.x     | Params use Go zero-value semantics; no wrapper functions needed for required fields |

**Deprecated/outdated:**

- Third-party Go SDKs (liushuangls/go-anthropic, 3JoB/anthropic-sdk-go): Superseded by the official SDK
- `anthropic.F()` / `anthropic.Int()` wrapper pattern: Replaced by direct assignment in newer SDK versions; only `anthropic.Float()` and `anthropic.String()` remain for optional pointer fields

## Open Questions

1. **Exact model constant for Claude Sonnet 4**
   - What we know: `anthropic.ModelClaude3_7SonnetLatest` exists and is verified. `ModelClaudeSonnet4_5_20250929` appears in search results.
   - What's unclear: The exact set of model constants available in SDK v1.26.0, and whether a Claude Sonnet 4 constant exists with a `Latest` variant.
   - Recommendation: Use `ModelClaude3_7SonnetLatest` as default. If a newer model is preferred at implementation time, verify the constant exists by checking the SDK source or `go doc`.

2. **`stream.Close()` behavior on context cancellation**
   - What we know: `ssestream.Stream` has a `Close() error` method.
   - What's unclear: Whether `Close()` is implicitly called when context is cancelled and `Next()` returns false, or must be called explicitly.
   - Recommendation: Always `defer stream.Close()` after creation for safety. Even if the SDK handles it internally, explicit close is defensive and costs nothing.

## Sources

### Primary (HIGH confidence)

- `/anthropics/anthropic-sdk-go` via Context7 - streaming patterns, client creation, system prompts, multi-turn conversations, error handling, event types
- [anthropic-sdk-go GitHub README](https://github.com/anthropics/anthropic-sdk-go) - SDK v1.26.0, Go 1.22+ requirement, installation, option package, helper functions, `Message.Accumulate()`, `Message.Usage`
- [pkg.go.dev anthropic-sdk-go](https://pkg.go.dev/github.com/anthropics/anthropic-sdk-go) - API documentation, type definitions

### Secondary (MEDIUM confidence)

- [Anthropic Streaming Messages API docs](https://docs.anthropic.com/en/api/messages-streaming) - SSE event structure, `message_start` (input_tokens), `message_delta` (output_tokens), event ordering
- Existing codebase patterns (`config/config.go`, `handlers/handlers.go`, `main.go`) - env var loading, nil-service pattern, health endpoint structure

### Tertiary (LOW confidence)

- Model constant names beyond `ModelClaude3_7SonnetLatest` -- inferred from search results, needs verification at implementation time

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - Official SDK verified via Context7 and GitHub; v1.26.0 confirmed; streaming API documented with code examples
- Architecture: HIGH - Patterns derived from SDK documentation and existing codebase conventions; channel-based streaming is idiomatic Go
- Pitfalls: HIGH - Goroutine leak and cancellation patterns are well-documented Go concurrency concerns; SDK-specific pitfalls (system prompt type, stream close) verified against SDK source

**Research date:** 2026-02-24
**Valid until:** 2026-03-24 (SDK is stable; check for version updates before implementation)
