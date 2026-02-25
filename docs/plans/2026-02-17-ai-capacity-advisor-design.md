# AI Capacity Advisor -- Design Document

**Date:** 2026-02-17
**Status:** Approved
**Phase:** 1 of 3 (Chat + Read-Only)

## Problem

Platform operators use the Diego Capacity Analyzer to plan hardware procurement with 6-12 month lead times. The tool produces comprehensive metrics (N-1 utilization, HA constraints, bottleneck analysis, sizing recommendations), but operators must interpret the data themselves. There is no conversational interface to help operators reason about what the numbers mean, identify gaps in the data, or think through procurement timing.

## Solution

An AI-powered conversational advisor embedded in the capacity planning UI. The advisor is a domain expert in TAS/Diego capacity planning that can see the operator's current analysis data and have an interactive dialogue about capacity challenges, data interpretation, and procurement decisions.

## Design Decisions

| Decision      | Choice                             | Rationale                                                                             |
| ------------- | ---------------------------------- | ------------------------------------------------------------------------------------- |
| Agent type    | Domain expert (not code-aware)     | Operators need capacity planning advice, not source code inspection                   |
| LLM backend   | Pluggable provider                 | Enterprise operators have different vendor relationships and compliance requirements  |
| Architecture  | Backend-proxied chat               | Backend already holds infrastructure/scenario state; natural path to Phase 2 tool use |
| Data access   | Read-only (Phase 1)                | Get the conversational foundation solid before adding tool use                        |
| UI placement  | Side panel / drawer                | Operator sees capacity data and chat simultaneously                                   |
| API key model | System default + per-user override | Flexibility without requiring BYOK for every operator                                 |

## Phase Roadmap

| Phase           | Scope                        | Agent Capabilities                                                    |
| --------------- | ---------------------------- | --------------------------------------------------------------------- |
| 1 (this design) | Chat + read-only             | See current data, explain metrics, identify gaps, suggest what to try |
| 2 (future)      | Chat + scenario execution    | Run scenario comparisons, calculate planning, analyze bottlenecks     |
| 3 (future)      | Full tool use + live UI sync | Drive wizard inputs, update main UI, persist conversations            |

---

## Backend Architecture

### Provider Abstraction

New package: `backend/ai/`

```
ai/
  provider.go        # ChatProvider interface, Message types, config
  anthropic.go       # Anthropic Claude implementation
  openai.go          # OpenAI implementation (also covers Azure OpenAI)
  openai_compat.go   # Generic OpenAI-compatible (Ollama, vLLM, etc.)
  context.go         # Builds LLM context from infrastructure/scenario data
```

#### ChatProvider Interface

```go
type ChatProvider interface {
    StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
}

type ChatRequest struct {
    SystemPrompt string
    Messages     []Message
    MaxTokens    int
}

type Message struct {
    Role    string // "user" or "assistant"
    Content string
}

type StreamEvent struct {
    Type    string // "text", "error", "done"
    Content string
}
```

#### Provider Selection

Provider is selected via environment variable. Each provider implementation handles its own SDK/HTTP client:

- `anthropic` -- Anthropic Messages API with streaming
- `openai` -- OpenAI Chat Completions API with streaming
- `openai-compat` -- Any OpenAI-compatible endpoint (Ollama, vLLM, Azure OpenAI) using a configurable base URL

### Chat Endpoint

New handler: `handlers/chat.go`

```
POST /api/v1/chat
  Request:  { "messages": [...], "api_key": "sk-..." (optional) }
  Response: SSE stream (text/event-stream)
    data: {"type":"text","content":"Based on your current..."}
    data: {"type":"done"}
```

Handler flow:

1. Read current `InfrastructureState` and most recent `ScenarioComparison` from handler's in-memory state
2. Call `ai.BuildContext()` to format data as structured text for the LLM
3. Prepend the domain expertise system prompt
4. If `api_key` is provided in the request body, use it instead of the system default
5. Stream the response back via SSE using Go's `http.Flusher`

### Context Building

`ai/context.go` serializes the current application state into annotated text the LLM can reason about. This is not a raw JSON dump -- it structures data with interpretive annotations:

**Infrastructure summary:**

- Cluster count, host count, cell count
- Per-cluster: hosts, memory, CPU, HA config, cell configuration
- Aggregated totals: memory, N-1 memory, HA-usable memory, vCPUs

**Scenario results (when available):**

- Current vs proposed utilization for each resource
- Constraint analysis (HA vs N-1, which is more restrictive)
- Active warnings with severity
- Delta summary (capacity change, utilization change)

**Contextual annotations:**

- Threshold proximity ("N-1 utilization is 82%, 3% below the 85% warning threshold")
- Risk indicators ("vCPU:pCPU ratio of 6.2:1 is in the moderate range")
- Missing data flags ("CPU analysis unavailable: physical core count not provided")

Context truncation: If serialized context exceeds a configurable threshold, `BuildContext()` produces a summary view (aggregate stats only) rather than per-cluster detail.

---

## Domain Expertise System Prompt

The system prompt encodes the agent's identity and TAS/Diego capacity planning knowledge. It lives as an embedded file or string constant in `ai/context.go`.

### Content Areas

**Identity:** Capacity planning advisor helping platform operators with hardware procurement decisions on 6-12 month timelines.

**TAS/Diego domain knowledge:**

- Diego cell memory allocation, Garden overhead, staging chunks
- N-1 redundancy and its operational significance
- HA Admission Control vs N-1 constraint comparison
- vCPU:pCPU ratio implications by range (conservative/moderate/aggressive)
- Blast radius math for cell failure scenarios
- Isolation segment capacity considerations
- Common operator mistakes

**Metric interpretation:**

- Healthy ranges for each metric
- Warning escalation guidance
- Raw utilization vs N-1-adjusted utilization distinction

**Procurement reasoning:**

- Growth projection methodology given current utilization trends
- Lead time math for 6-12 month procurement cycles
- Cell sizing trade-offs (many small vs fewer large)
- Scale up vs scale out vs add hosts decision framework

**Known limitations the agent must disclose:**

- CPU analysis requires operator-provided physical core counts
- TPS model is experimental with generic benchmarks
- Disk overhead is minimal in the current model
- Mixed cell sizes across clusters may produce imprecise results
- Not covered: network bandwidth, storage IOPS, control plane overhead growth, seasonal workload patterns

---

## Frontend Design

### Side Panel Layout

The advisor panel slides in from the right side of the screen when toggled.

- **Width:** ~400px
- **Behavior:** Overlay on screens < 1400px, push-content on wider screens
- **Toggle:** "AI Advisor" button in the ScenarioAnalyzer header area
- **Visibility:** Button hidden entirely when `ai_configured: false` from health endpoint

### Panel Structure

```
+----------------------------------+
|  AI Capacity Advisor         [X] |
|----------------------------------|
|                                  |
|  [Assistant message bubble]      |
|                                  |
|  [User message bubble]           |
|                                  |
|  [Assistant response streaming]  |
|                                  |
|----------------------------------|
| [Message input...          ] [>] |
|----------------------------------|
| Provider: Anthropic   [Settings] |
+----------------------------------+
```

### Components

```
frontend/src/components/advisor/
  AdvisorPanel.jsx      # Side panel container, open/close state
  AdvisorChat.jsx       # Message list, auto-scroll, streaming display
  AdvisorMessage.jsx    # Single message bubble (user or assistant)
  AdvisorInput.jsx      # Text input with send button
  AdvisorConfig.jsx     # Provider/key configuration modal
  useAdvisor.js         # Hook: conversation state, SSE connection
```

### Key Behaviors

- **Auto-context:** When the panel opens or infrastructure/scenario data changes, the agent receives updated context automatically.
- **Streaming:** Assistant responses render token-by-token via SSE.
- **Markdown:** Agent responses support markdown formatting.
- **Session persistence:** Conversation persists in browser memory within a session. Not persisted across page reloads.
- **Starter prompts:** When conversation is empty, show 2-3 suggested questions based on current data:
  - "What does my N-1 utilization mean?"
  - "Am I at risk of capacity issues?"
  - "What should I order for next quarter?"

### Integration

`AdvisorPanel` is rendered as a sibling to the main content in `ScenarioAnalyzer.jsx`. It receives `infrastructureState` and `scenarioResults` as props or via shared context.

---

## Configuration

### Environment Variables

```bash
# AI Advisor (optional -- feature disabled when AI_PROVIDER is unset)
AI_PROVIDER=anthropic          # anthropic | openai | openai-compat
AI_API_KEY=                    # system-level API key (optional if BYOK)
AI_MODEL=                      # model name (defaults per provider)
AI_BASE_URL=                   # for openai-compat only
AI_MAX_TOKENS=4096             # max response tokens
```

**Feature gating:** If `AI_PROVIDER` is not set, the feature is disabled. The `/api/v1/health` endpoint reports `ai_configured: true/false` (same pattern as `vsphere_configured`). The frontend hides the AI Advisor button when `ai_configured` is false.

**Default models per provider:**

- `anthropic`: `claude-sonnet-4-20250514`
- `openai`: `gpt-4o`
- `openai-compat`: must be specified via `AI_MODEL`

---

## Error Handling

| Scenario                         | Backend behavior                       | Frontend display                                    |
| -------------------------------- | -------------------------------------- | --------------------------------------------------- |
| No AI provider configured        | Chat endpoint returns 404              | Button hidden                                       |
| No API key (system or user)      | Chat endpoint returns 400              | "Configure an API key to use the advisor"           |
| LLM API error (rate limit, auth) | Stream error event                     | Error message inline in chat                        |
| No infrastructure loaded         | Chat endpoint returns 400              | "Load infrastructure data first" with chat disabled |
| SSE connection dropped           | Stream ends                            | "Connection lost. Send a new message to reconnect." |
| Context exceeds model limit      | `BuildContext()` produces summary view | Transparent to user                                 |

No automatic retries. The operator can re-send a message to retry.

---

## Security

- **System API key** stays in backend environment only. Never sent to frontend.
- **User API key** stored in localStorage. Sent per-request via `X-AI-API-Key` header. Backend uses it for that request only, does not log or persist it.
- **Auth required:** Chat endpoint uses the same Auth middleware as all other endpoints. No anonymous access.
- **Data sent to LLM:** Infrastructure metadata (cluster sizes, utilization percentages, cell counts). No credentials, no PII. This should be documented in user-facing help text.
- **Rate limiting:** `/api/v1/chat` gets the existing per-endpoint rate limiter with a conservative limit (10 requests/minute) to prevent runaway API costs.

---

## Testing Strategy

### Backend Tests

- `ai/context_test.go` -- `BuildContext()` with various infrastructure states: empty, single cluster, multi-cluster, with/without scenario results, context truncation
- `ai/anthropic_test.go`, `openai_test.go`, `openai_compat_test.go` -- Provider tests against mock HTTP servers. Verify request formatting, streaming parse, error handling per provider.
- `handlers/chat_test.go` -- Auth required, 400 when no infrastructure loaded, SSE streaming format, per-request API key override, rate limiting
- `e2e/chat_test.go` -- Full flow with mock LLM server: load infrastructure, send chat message, receive streamed response

### Frontend Tests

- `AdvisorPanel.test.jsx` -- Panel open/close, hidden when AI not configured
- `AdvisorChat.test.jsx` -- Message rendering, streaming display, auto-scroll
- `AdvisorConfig.test.jsx` -- Provider selection, key storage in localStorage
- `useAdvisor.test.js` -- SSE connection, message state management, error handling
- Integration: Advisor panel receives updated context when infrastructure state changes

---

## Files Changed

### New Files

| File                                                 | Purpose                                              |
| ---------------------------------------------------- | ---------------------------------------------------- |
| `backend/ai/provider.go`                             | ChatProvider interface, Message types, configuration |
| `backend/ai/anthropic.go`                            | Anthropic Claude provider implementation             |
| `backend/ai/openai.go`                               | OpenAI provider implementation                       |
| `backend/ai/openai_compat.go`                        | Generic OpenAI-compatible provider                   |
| `backend/ai/context.go`                              | Context builder, domain expertise system prompt      |
| `backend/ai/*_test.go`                               | Tests for all ai package files                       |
| `backend/handlers/chat.go`                           | Chat SSE endpoint handler                            |
| `backend/handlers/chat_test.go`                      | Chat handler tests                                   |
| `frontend/src/components/advisor/AdvisorPanel.jsx`   | Side panel container                                 |
| `frontend/src/components/advisor/AdvisorChat.jsx`    | Chat message list                                    |
| `frontend/src/components/advisor/AdvisorMessage.jsx` | Message bubble component                             |
| `frontend/src/components/advisor/AdvisorInput.jsx`   | Text input component                                 |
| `frontend/src/components/advisor/AdvisorConfig.jsx`  | Provider/key config                                  |
| `frontend/src/components/advisor/useAdvisor.js`      | Chat state management hook                           |
| `frontend/src/components/advisor/*.test.jsx`         | Frontend tests                                       |

### Modified Files

| File                                           | Change                                      |
| ---------------------------------------------- | ------------------------------------------- |
| `backend/main.go`                              | Initialize AI provider, register chat route |
| `backend/handlers/handlers.go`                 | Add AI provider field to Handler struct     |
| `backend/handlers/routes.go`                   | Add `/api/v1/chat` route                    |
| `backend/handlers/health.go`                   | Add `ai_configured` to status response      |
| `frontend/src/components/ScenarioAnalyzer.jsx` | Add AdvisorPanel, toggle button             |
| `frontend/src/services/scenarioApi.js`         | Add chat SSE connection helper              |
| `.env.example`                                 | Add AI configuration variables              |
