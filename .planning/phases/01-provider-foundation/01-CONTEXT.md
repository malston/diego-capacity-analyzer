# Phase 1: Provider Foundation - Context

**Gathered:** 2026-02-24
**Status:** Ready for planning

<domain>
## Phase Boundary

Pluggable LLM provider abstraction (`ChatProvider` interface) with a working Anthropic Claude implementation that streams token-by-token responses configured via environment variables. Feature gating disables the advisor cleanly when unconfigured. This phase delivers backend infrastructure only -- no HTTP endpoint, no UI, no domain prompt.

</domain>

<decisions>
## Implementation Decisions

### Interface contract

- Simple message type: `Message{Role, Content}` -- no metadata, timestamps, or token counts on messages
- Functional options pattern for provider-specific parameters (e.g., `WithMaxTokens(4096)`, `WithTemperature(0.3)`). Extensible without breaking the interface
- Provider accepts `context.Context` and handles cancellation (stops Anthropic stream, closes channel on ctx.Done())

### Streaming delivery

- Go channel primitive: `Chat` returns `<-chan TokenEvent`. Provider fills channel in a goroutine, closes when done
- Provider handles context cancellation -- caller does not need to manage cleanup

### Model & parameters

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

</decisions>

<specifics>
## Specific Ideas

- Functional options pattern chosen specifically for clean extensibility when Phase 2+ adds OpenAI and other providers
- Token usage logging enables cost conversations with operators from the first release -- important for an AI feature that has per-request cost

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

_Phase: 01-provider-foundation_
_Context gathered: 2026-02-24_
