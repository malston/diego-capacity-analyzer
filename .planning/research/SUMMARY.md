# Project Research Summary

**Project:** AI Capacity Advisor -- Diego Capacity Analyzer
**Domain:** AI conversational advisor embedded in a Go/React operational capacity planning dashboard
**Researched:** 2026-02-24
**Confidence:** HIGH

## Executive Summary

The AI Capacity Advisor is a domain-specific conversational feature added to an existing Go/React dashboard. The core premise -- an LLM that sees live infrastructure state and answers operator questions about TAS/Diego capacity -- is well-validated by industry parallels (Datadog Bits AI, Grafana Assistant, New Relic AI), but this project occupies a differentiated niche: procurement-oriented capacity planning for a specific platform (TAS/Diego) rather than broad-platform observability. The recommended approach is a lightweight SSE streaming pipeline using the official Anthropic Go SDK (`anthropic-sdk-go` v1.26.0), Go stdlib SSE, and Vercel `streamdown` for streaming Markdown on the frontend -- all integrating cleanly into the existing middleware chain, handler pattern, and configuration system without touching existing code paths.

The highest-value design decision is the context builder: a pure function that serializes the handler's in-memory infrastructure state into human-readable text for the LLM. This bridge between live infrastructure data and the LLM determines whether the advisor answers like a domain expert or like a generic chatbot. The domain-expert system prompt -- encoding TAS/Diego capacity planning heuristics, N-1 HA formulas, and procurement framing -- is the primary competitive differentiator that no existing tool offers for this specific domain.

The key risks are almost entirely in Phase 1 implementation: SSE streaming has seven concrete pitfalls that are all avoidable with known patterns. The most dangerous are credential leakage into LLM context (security), streaming timeouts causing goroutine leaks (reliability), and React re-render storms during token streaming (UX). All three have well-understood mitigations. The most operationally significant is the context builder's strict requirement to accept only model types, never the Handler struct or Config -- enforced by a unit test checking for absence of credential field values in the serialized context.

## Key Findings

### Recommended Stack

The technology choices are minimal and fit cleanly into the existing project. The backend adds only one new Go dependency (`anthropic-sdk-go`); SSE streaming uses Go stdlib's `http.ResponseController`. The frontend adds `streamdown` (streaming-aware Markdown renderer) and `@streamdown/code` (syntax highlighting). No state management library, no WebSocket library, no chat UI kit -- the existing Tailwind CSS, `lucide-react`, and React Context pattern cover everything needed. See [STACK.md](.planning/research/STACK.md) for full details.

**Core technologies:**

- `anthropic-sdk-go` v1.26.0: Anthropic Messages API with streaming -- official SDK, Go 1.22+ (project is 1.24), multiple releases per week as of Feb 2026
- Go `net/http` + `http.ResponseController`: SSE streaming from backend -- stdlib only, `http.NewResponseController` available since Go 1.20
- `streamdown` 2.3.0: Streaming Markdown rendering in React -- built specifically for incremental LLM output; avoids `react-markdown`'s O(n^2) re-parse and flicker on partial markdown blocks
- `fetch()` + `ReadableStream`: Frontend SSE consumption -- `EventSource` cannot POST, so `fetch` with manual SSE line parsing is the correct and industry-standard pattern (used by ChatGPT, Claude.ai, etc.)

### Expected Features

The MVP must cover all table-stakes features expected from any modern AI chat panel; the differentiator is domain depth. See [FEATURES.md](.planning/research/FEATURES.md) for full prioritization matrix and competitor analysis.

**Must have (table stakes -- Phase 1 launch):**

- SSE streaming with token-by-token display -- every modern chat interface streams; waiting for full response feels broken
- Streaming Markdown rendering -- LLMs produce Markdown natively; plain text looks unprofessional
- Context awareness with live infrastructure data -- without this, the advisor is just ChatGPT in a sidebar
- Domain-expert system prompt (TAS/Diego capacity planning knowledge) -- the primary differentiator
- Side panel with open/close toggle -- advisor must not permanently consume screen real estate
- Static starter prompts -- empty chat with blinking cursor is intimidating
- Conversation threading (session-scoped) -- multi-turn dialogue expected
- Loading indicator + error handling with retry -- basic resilience
- Graceful degradation when BOSH/vSphere unavailable -- works with CF-only data, explicitly flags gaps
- Rate limiting on chat endpoint + feature gating via `AI_PROVIDER` env var

**Should have (add after validation -- v1.x):**

- Contextual starter prompts (data-state aware, not static)
- Data-grounded responses with citations of specific values
- Copy response to clipboard (workflow integration for procurement requests)
- Response feedback (thumbs up/down) logging
- Procurement-oriented framing sharpened in system prompt
- Token/length awareness indicator for long conversations

**Defer (v2+):**

- Tool use / scenario execution via chat (requires action confirmation UI, safety controls)
- Additional LLM providers via existing ChatProvider interface
- Conversation persistence across sessions (requires storage layer; stale context is a real UX risk)
- Per-user API keys (BYOK)

### Architecture Approach

The advisor follows the same patterns as existing optional integrations (BOSH, vSphere): gated by env var, initialized at startup, nil-checked in the handler, nil when unconfigured. New backend components live in `services/ai/` (provider interface + Anthropic implementation), `services/advisor.go` (orchestration), `services/context_builder.go` (state serialization), and `handlers/chat.go` (SSE endpoint). Frontend components live in a self-contained `components/advisor/` directory with a new `ChatContext` following the existing `AuthContext`/`ToastContext` pattern. The chat endpoint plugs into the existing middleware chain (CORS, CSRF, Auth, RateLimit, LogRequest) without modification. See [ARCHITECTURE.md](.planning/research/ARCHITECTURE.md) for build order and detailed data flow diagrams.

**Major components:**

1. **ChatProvider interface + AnthropicProvider** -- decouples LLM mechanics from business logic; returns a `<-chan Token` so the handler writes SSE frames without knowing which LLM is behind it
2. **ContextBuilder** -- pure function (no I/O); accepts only `*models.InfrastructureState` and `*models.DashboardResponse`; produces human-readable aggregate text (NOT raw JSON, which wastes context window tokens on formatting noise)
3. **Advisor service** -- orchestrates context building + system prompt assembly + provider call; sits at the same layer as `scenario.go` and `planning.go`
4. **Chat handler (`POST /api/v1/chat`)** -- SSE endpoint; builds context snapshot from handler state, calls advisor, streams tokens to client via `http.Flusher`; stateless (frontend owns conversation history)
5. **ChatContext (React)** -- conversation state via React Context; wraps `AdvisorPanel`, `MessageList`, `MessageBubble`, `StarterPrompts`, `ChatInput`
6. **advisorApi service** -- `fetch()` + `ReadableStream` SSE consumer; includes CSRF token from existing `withCSRFToken()` utility

### Critical Pitfalls

Seven critical pitfalls are documented in [PITFALLS.md](.planning/research/PITFALLS.md). All apply to Phase 1. The top five by impact:

1. **Credential leakage into LLM context** -- Context builder must accept only model types, never `*Handler` or `*config.Config`. Enforce with a unit test that checks the serialized context string contains no values from credential fields. Recovery is high-cost (rotate all leaked credentials); prevention is a single architectural constraint.

2. **Anthropic streaming hang (goroutine leak)** -- `http.Client.Timeout` does not protect mid-stream stalls. Use `context.WithTimeout` per-request plus an idle timeout (30-60s between events). Without this, stalled streams hold goroutines forever; under load this exhausts memory.

3. **React re-render storm during token streaming** -- SSE events fire outside React's event system; `setState` per token triggers 30-80 re-renders/second. Buffer tokens in `useRef`, flush to state at ~60fps via `requestAnimationFrame`. Use `React.memo` on individual messages. Profile with a 500-token response before declaring done.

4. **CSRF middleware blocking SSE POST** -- The existing CSRF middleware returns a JSON 403 if `X-CSRF-Token` is missing on POST. The streaming fetch client must include this header via the existing `withCSRFToken()` utility. The frontend SSE parser must check `response.ok` before calling `response.body.getReader()`, or it will receive JSON and attempt to parse it as SSE events.

5. **EventSource vs fetch ReadableStream** -- `EventSource` is GET-only; chat is POST (conversation history in body). Use `fetch()` with manual SSE line parsing. This is the industry standard (ChatGPT, Claude.ai) but not the textbook SSE pattern -- teams frequently reach for `EventSource` first.

## Implications for Roadmap

Based on combined research, the natural phase structure is determined by two factors: (1) the backend streaming pipeline is the critical path that must exist before any frontend work is visible, and (2) the context builder is the highest-risk component and needs standalone development and testing before it connects to the LLM.

### Phase 1: Backend Streaming Pipeline

**Rationale:** Everything else depends on this. Provider interface, Anthropic implementation, context builder, advisor service, and chat handler must all exist before any streaming works in the UI. These are independent of each other (except dependency order) and can be developed in sequence with full unit test coverage at each step.
**Delivers:** A working `POST /api/v1/chat` SSE endpoint, verified with integration tests, that accepts conversation history, builds context from live infrastructure state, and streams Anthropic tokens.
**Addresses:** SSE chat endpoint, context awareness, domain-expert system prompt, graceful degradation, rate limiting, feature gating
**Avoids:** Credential leakage (context builder type constraint), streaming hangs (context timeout + idle timeout), CSRF rejection (route wired identically to other POST routes)
**Build order within phase:** Config/models -> Provider interface -> Anthropic provider -> Context builder -> Advisor service -> Chat handler + route registration

### Phase 2: Frontend Chat Panel

**Rationale:** Frontend work can start after the endpoint exists (or against a mock). The UI layer is entirely self-contained (new `components/advisor/` directory, new `ChatContext`). Integration with the existing `TASCapacityAnalyzer` is the last and riskiest step (touches existing components).
**Delivers:** A functional side panel with streaming Markdown display, starter prompts, conversation threading, error handling, and loading states -- wired to the backend endpoint.
**Uses:** `streamdown` 2.3.0 for streaming Markdown, `fetch()` + `ReadableStream` SSE consumption, existing `lucide-react` icons and Tailwind CSS
**Implements:** `advisorApi` service, `ChatContext`, `AdvisorPanel` and sub-components, integration into `App.jsx` + `TASCapacityAnalyzer`
**Avoids:** Re-render storm (token buffering via `useRef` + `requestAnimationFrame`), CSRF rejection (include `withCSRFToken()` in streaming fetch), EventSource mistake (use `fetch` + `ReadableStream`)

### Phase 3: Polish and Validation Features

**Rationale:** Once the core advisor is functional, add the v1.x differentiators that improve trust and workflow integration. These are all independent features that can be sequenced by value.
**Delivers:** Contextual starter prompts, data-grounded citations, copy-to-clipboard, feedback logging, procurement framing in system prompt, token/length awareness
**Note:** This phase validates whether the concept is worth investing in Phase 4 (tool use). Ship Phase 3, collect feedback, then decide.

### Phase 4: Tool Use (Future)

**Rationale:** Deferred deliberately. Tool use (advisor executes scenarios, changes infrastructure state) requires action confirmation UI, safety controls, error recovery, and testing of every tool path. Phase 1-3 must be shipped and validated first.
**Delivers:** Advisor can trigger scenario comparisons, run planning calculations, and surface results in the dashboard
**Note:** The `ChatProvider` interface already accommodates tool use patterns; no architectural changes needed.

### Phase Ordering Rationale

- Backend before frontend: the streaming pipeline is the critical path; nothing is visible until the endpoint works
- Context builder as standalone component: highest-risk piece, needs independent unit tests before connecting to the LLM
- Provider interface before Anthropic implementation: keeps the handler decoupled for when Phase 4 adds providers
- Polish after core: contextual prompts require inspecting data state (more complex); static prompts work for launch
- Tool use last: safety-critical, requires validated read-only advisor first

### Research Flags

Phases with well-documented patterns (deeper research not needed):

- **Phase 1 (backend pipeline):** Patterns fully documented in ARCHITECTURE.md and STACK.md with working code examples. Build order is clear. `anthropic-sdk-go` is well-documented.
- **Phase 2 (frontend panel):** Standard React patterns; `streamdown` is well-documented. Token buffering pattern is documented in PITFALLS.md.

Phases that may benefit from targeted research:

- **Phase 1 (context builder):** The exact serialization format for the LLM (what fields, what level of detail, how much context budget to allocate) needs empirical validation -- test with real infrastructure data and real LLM responses. Research documented the anti-patterns (no raw JSON, no per-app detail in Phase 1) but optimal format requires iteration.
- **Phase 3 (system prompt engineering):** Domain-expert system prompt quality determines advisor usefulness. TAS/Diego capacity planning heuristics need to be encoded correctly. This is iteration work, not research, but plan for multiple rounds of prompt tuning with real operator feedback.

## Confidence Assessment

| Area         | Confidence | Notes                                                                                                                                                                                                                                                                                           |
| ------------ | ---------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Stack        | HIGH       | Official Anthropic SDK verified via Context7 + GitHub. `streamdown` verified from official repo + docs. Go stdlib SSE verified from release notes. Version compatibility fully checked.                                                                                                         |
| Features     | MEDIUM     | Table-stakes features synthesized from 4 major competitor platforms (Datadog, New Relic, Grafana, Dynatrace). No direct competitor in TAS/Diego capacity planning space, so "table stakes for this niche" is partly inferred. Core features are well-grounded; differentiators are opinionated. |
| Architecture | HIGH       | All integration points verified by reading the existing codebase source. Patterns match existing handler/service/config conventions. Build order validated against component dependencies.                                                                                                      |
| Pitfalls     | HIGH       | Critical pitfalls verified against official Anthropic SDK docs, SSE spec, existing codebase middleware. Specific references to existing files (`csrf.go`, `apiClient.js`, `csrf.js`) confirmed by direct inspection.                                                                            |

**Overall confidence:** HIGH

### Gaps to Address

- **Optimal context format for LLM:** Research identifies what NOT to include (raw JSON, per-app detail) but the ideal context structure for the specific combination of TAS metrics + Claude Sonnet needs empirical validation during implementation. Plan for iteration on the context builder format.
- **`streamdown` Tailwind v3 compatibility:** Documented as compatible, but the specific configuration (whether `@source` directive or `content` path is needed for streamdown's dist files) is "MEDIUM confidence" in STACK.md. Verify during frontend setup.
- **Context window budget for large environments:** The actual token cost of the infrastructure context for large environments (500+ apps, multiple isolation segments) is unknown until measured. The pitfalls document recommends token counting and a sliding window -- plan for this to be a real implementation task, not a quick add.

## Sources

### Primary (HIGH confidence)

- [anthropics/anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) -- SDK usage, streaming API, version verification
- [pkg.go.dev/anthropic-sdk-go](https://pkg.go.dev/github.com/anthropics/anthropic-sdk-go) -- Go package documentation
- [Go 1.20 Release Notes](https://go.dev/doc/go1.20) -- `http.ResponseController` availability
- [Anthropic API streaming documentation](https://platform.claude.ai/docs/en/build-with-claude/streaming) -- streaming patterns, error handling
- [Anthropic API rate limits and error codes](https://platform.claude.ai/docs/en/api/rate-limits) -- 429/529 handling
- Existing codebase: `backend/handlers/handlers.go`, `backend/handlers/routes.go`, `backend/main.go`, `backend/config/config.go`, `backend/middleware/csrf.go`, `frontend/src/services/apiClient.js`, `frontend/src/utils/csrf.js` -- all integration points verified by direct inspection

### Secondary (MEDIUM confidence)

- [vercel/streamdown](https://github.com/vercel/streamdown) -- streaming Markdown renderer, v2.3.0
- [streamdown.ai/docs](https://streamdown.ai/docs) -- Tailwind v3/v4 compatibility claim
- [Datadog Bits AI](https://www.datadoghq.com/product/ai/bits-ai-sre/), [New Relic AI](https://docs.newrelic.com/docs/agentic-ai/new-relic-ai/), [Grafana Assistant](https://grafana.com/blog/2025/05/07/llm-grafana-assistant/), [Dynatrace Davis CoPilot](https://www.dynatrace.com/news/blog/announcing-general-availability-of-davis-copilot-your-new-ai-assistant/) -- competitor feature analysis
- [SSE via POST with Fetch ReadableStream](https://medium.com/@david.richards.tech/sse-server-sent-events-using-a-post-request-without-eventsource-1c0bd6f14425) -- streaming pattern validation
- [React streaming re-render performance](https://www.sitepoint.com/streaming-backends-react-controlling-re-render-chaos/) -- token buffering patterns
- [OWASP LLM01:2025 Prompt Injection](https://genai.owasp.org/llmrisk/llm01-prompt-injection/) -- security considerations

### Tertiary (community, MEDIUM-LOW confidence)

- [HN: Flash of Incomplete Markdown](https://news.ycombinator.com/item?id=44182941) -- validates `react-markdown` streaming problems
- [Claude Code streaming hang issue #25979](https://github.com/anthropics/claude-code/issues/25979) -- confirms idle timeout is a real production issue
- [Context window management strategies](https://redis.io/blog/context-window-overflow/) -- sliding window patterns

---

_Research completed: 2026-02-24_
_Ready for roadmap: yes_
