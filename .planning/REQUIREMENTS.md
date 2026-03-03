# Requirements: AI Capacity Advisor (Phase 1)

**Defined:** 2026-02-24
**Core Value:** Operators can have a conversation with a domain expert that sees their live capacity data -- turning raw metrics into actionable procurement guidance.

## v1 Requirements

Requirements for Phase 1 release. Each maps to roadmap phases.

### Provider Infrastructure

- [x] **PROV-01**: Backend exposes a `ChatProvider` interface with streaming support that decouples LLM interaction from HTTP handling
- [x] **PROV-02**: Anthropic Claude provider implementation streams token-by-token responses using the official `anthropic-sdk-go` SDK
- [x] **PROV-03**: Provider is configured via `AI_PROVIDER` and `AI_API_KEY` environment variables with validation at startup
- [x] **PROV-04**: When `AI_PROVIDER` is unset, the advisor feature is completely disabled and the health endpoint reports `ai_configured: false`

### Context Builder

- [x] **CTX-01**: Context builder serializes current dashboard state (cell counts, memory utilization, isolation segments, app counts) into annotated text for the LLM
- [x] **CTX-02**: Context builder serializes infrastructure state (clusters, hosts, VMs) when vSphere data is available
- [x] **CTX-03**: Context builder serializes scenario comparison results when a scenario has been run
- [x] **CTX-04**: Context builder flags missing data sources (BOSH unavailable, vSphere unconfigured) with explicit markers the LLM can reference
- [x] **CTX-05**: Context builder reads from existing Handler state (cache and mutex-protected infrastructure state) without making additional API calls

### Domain Expertise

- [x] **DOM-01**: System prompt encodes TAS/Diego capacity planning knowledge: N-1 redundancy, HA Admission Control, vCPU:pCPU ratios, cell sizing heuristics, isolation segment tradeoffs
- [x] **DOM-02**: System prompt frames analysis in procurement terms: lead times, budget cycles, growth planning, headroom targets
- [x] **DOM-03**: System prompt instructs the LLM to acknowledge data gaps rather than hallucinate when information is missing
- [x] **DOM-04**: System prompt instructs the LLM to reference specific data values from context when making claims

### Chat Endpoint

- [x] **CHAT-01**: `POST /api/v1/chat` accepts a JSON body with conversation messages and returns an SSE stream of token events
- [x] **CHAT-02**: Chat endpoint requires authentication (same middleware as all other endpoints)
- [x] **CHAT-03**: Chat endpoint is rate-limited to 10 requests per minute per user
- [x] **CHAT-04**: Chat endpoint returns structured JSON errors (not SSE) for pre-stream failures (auth, rate limit, missing provider, missing API key)
- [x] **CHAT-05**: Chat endpoint includes idle timeout detection so streaming does not hang indefinitely on provider failure

### Chat Panel UI

- [x] **UI-01**: Side panel slides in from the right as an overlay on all screen sizes
- [x] **UI-02**: Panel toggle button appears in the dashboard header only when `ai_configured` is true
- [x] **UI-03**: Streaming chat displays tokens as they arrive with smooth rendering (no re-render storms)
- [x] **UI-04**: Assistant messages render Markdown (headers, lists, bold, code blocks, tables)
- [x] **UI-05**: Conversation maintains multi-turn threading within the session (full history sent with each request)
- [x] **UI-06**: User can clear/reset the conversation to start fresh
- [x] **UI-07**: Loading/thinking indicator appears between sending a message and receiving the first token
- [x] **UI-08**: Error messages display user-friendly text with a "Try again" action for LLM API failures, rate limits, timeouts, and network errors
- [x] **UI-09**: Static starter prompts appear when conversation is empty, suggesting questions based on common capacity planning concerns

### Graceful Degradation

- [x] **DEG-01**: Advisor works with CF-only data when BOSH and vSphere are unavailable
- [x] **DEG-02**: Advisor explicitly tells the operator which data sources are missing and what analysis it cannot perform
- [ ] **DEG-03**: Starter prompts adapt to available data (do not suggest vSphere-dependent questions when vSphere is unconfigured)

### Polish

- [ ] **POL-01**: User can copy any assistant response to clipboard with a single click
- [ ] **POL-02**: User can provide thumbs up/down feedback on any assistant response (logged to backend via `slog`)
- [ ] **POL-03**: System prompt includes procurement-oriented framing so the advisor interprets capacity data in terms of hardware procurement decisions

## v2 Requirements

Deferred to Phase 2 (Scenario Execution via Chat).

- **TOOL-01**: Advisor can execute scenario comparisons via LLM tool use
- **TOOL-02**: Advisor can run planning calculations via LLM tool use
- **TOOL-03**: Advisor can analyze bottlenecks via LLM tool use
- **TOOL-04**: All tool-invoked operations remain read-only or idempotent
- **MULTI-01**: OpenAI provider implementation
- **MULTI-02**: OpenAI-compatible provider implementation (Ollama, vLLM, Azure OpenAI)
- **BYOK-01**: Per-user API key override via request header
- **BYOK-02**: Frontend configuration modal for provider/key stored in localStorage

## Out of Scope

| Feature                                         | Reason                                                               |
| ----------------------------------------------- | -------------------------------------------------------------------- |
| Tool use / action execution                     | Phase 2 -- requires safety controls, confirmation UI, error recovery |
| Conversation persistence across sessions        | Phase 3 -- requires storage layer; stale context risk                |
| Live UI sync (advisor actions update dashboard) | Phase 3 -- requires bidirectional state management                   |
| Driving wizard inputs from chat                 | Phase 3 -- tightly coupled to UI sync                                |
| Push-content panel layout on wide screens       | Overlay sufficient for validation; revisit in Phase 3                |
| Additional LLM providers (OpenAI, etc.)         | Ship with Claude only; add when requested                            |
| Per-user API keys (BYOK)                        | System key only; add if rate limits become a pain point              |
| Voice input                                     | Text-only; no demand signal for voice                                |
| Chart generation in chat                        | Dashboard already has charts; advisor references them                |
| Autonomous alerts / proactive messages          | Advisor is reactive (user-initiated) for Phase 1                     |
| Real-time data push to advisor                  | Context updates on each message send (pull model)                    |

## Traceability

| Requirement | Phase   | Status  |
| ----------- | ------- | ------- |
| PROV-01     | Phase 1 | Complete |
| PROV-02     | Phase 1 | Complete |
| PROV-03     | Phase 1 | Complete |
| PROV-04     | Phase 1 | Complete |
| CTX-01      | Phase 2 | Complete |
| CTX-02      | Phase 2 | Complete |
| CTX-03      | Phase 2 | Complete |
| CTX-04      | Phase 2 | Complete |
| CTX-05      | Phase 2 | Complete |
| DOM-01      | Phase 3 | Complete |
| DOM-02      | Phase 3 | Complete |
| DOM-03      | Phase 3 | Complete |
| DOM-04      | Phase 3 | Complete |
| CHAT-01     | Phase 4 | Complete |
| CHAT-02     | Phase 4 | Complete |
| CHAT-03     | Phase 4 | Complete |
| CHAT-04     | Phase 4 | Complete |
| CHAT-05     | Phase 4 | Complete |
| UI-01       | Phase 5 | Complete |
| UI-02       | Phase 5 | Complete |
| UI-03       | Phase 5 | Complete |
| UI-04       | Phase 5 | Complete |
| UI-05       | Phase 5 | Complete |
| UI-06       | Phase 6 | Complete |
| UI-07       | Phase 6 | Complete |
| UI-08       | Phase 6 | Complete |
| UI-09       | Phase 6 | Complete |
| DEG-01      | Phase 7 | Complete |
| DEG-02      | Phase 7 | Complete |
| DEG-03      | Phase 7 | Pending |
| POL-01      | Phase 8 | Pending |
| POL-02      | Phase 8 | Pending |
| POL-03      | Phase 8 | Pending |

**Coverage:**

- v1 requirements: 33 total
- Mapped to phases: 33
- Unmapped: 0

---

_Requirements defined: 2026-02-24_
_Last updated: 2026-02-24 after roadmap creation_
