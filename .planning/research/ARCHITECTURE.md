# Architecture Research

**Domain:** AI conversational advisor embedded in a Go/React capacity planning dashboard
**Researched:** 2026-02-24
**Confidence:** HIGH

## System Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                           Frontend (React 18)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐   │
│  │ AuthContext   │  │ ToastContext  │  │ ChatContext              │   │
│  └──────┬───────┘  └──────┬───────┘  │  messages[]              │   │
│         │                 │          │  streaming: bool          │   │
│  ┌──────┴─────────────────┴──────┐   │  sendMessage()           │   │
│  │      TASCapacityAnalyzer      │   │  infrastructureSnapshot  │   │
│  │  (dashboard state, data)      │   └────────────┬─────────────┘   │
│  └──────────────┬────────────────┘                │                 │
│                 │                    ┌─────────────┴──────────┐      │
│                 │                    │    AdvisorPanel         │      │
│                 │                    │  (slide-over side panel)│      │
│                 │                    └─────────────┬──────────┘      │
│                 │                                  │                 │
│  ┌──────────────┴──────────────────────────────────┴─────────┐      │
│  │              apiClient / advisorApi (fetch + EventSource) │      │
│  └────────────────────────────┬───────────────────────────────┘      │
└───────────────────────────────┼──────────────────────────────────────┘
                                │ HTTP (REST + SSE)
┌───────────────────────────────┼──────────────────────────────────────┐
│                           Backend (Go)                               │
│  ┌────────────────────────────┴──────────────────────────────────┐   │
│  │  Middleware Chain (CORS → CSRF → Auth → RateLimit → Log)      │   │
│  └────────────┬──────────────────────────────────────────────────┘   │
│               │                                                      │
│  ┌────────────┴──────────┐   ┌───────────────────┐                  │
│  │  handlers/chat.go     │   │  handlers/*.go     │                  │
│  │  POST /api/v1/chat    │   │  (existing routes) │                  │
│  └────────────┬──────────┘   └───────────────────┘                  │
│               │                                                      │
│  ┌────────────┴──────────┐                                          │
│  │  services/advisor.go  │ Advisor (orchestrates context + provider) │
│  └───┬────────────┬──────┘                                          │
│      │            │                                                  │
│  ┌───┴──────┐  ┌──┴──────────────┐                                  │
│  │ Context  │  │ ChatProvider     │  (interface)                     │
│  │ Builder  │  │  ├─ anthropic.go │  (Anthropic implementation)      │
│  └───┬──────┘  └──┬──────────────┘                                  │
│      │            │                                                  │
│      │ reads      │ streams                                          │
│      ▼            ▼                                                  │
│  ┌──────────┐  ┌──────────────┐                                     │
│  │ Handler  │  │ Anthropic    │                                      │
│  │ State    │  │ Messages API │                                      │
│  │ (cache,  │  │ (external)   │                                      │
│  │  infra)  │  └──────────────┘                                     │
│  └──────────┘                                                        │
└──────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component                    | Responsibility                                                            | Implementation                                                                                               |
| ---------------------------- | ------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
| **ChatProvider** (interface) | Send messages to an LLM and stream token-by-token responses               | Go interface with `StreamMessage(ctx, messages, system) -> TokenStream`                                      |
| **Anthropic Provider**       | Anthropic Claude implementation of ChatProvider                           | Uses `anthropics/anthropic-sdk-go` SDK; wraps `client.Messages.NewStreaming()`                               |
| **Context Builder**          | Serialize current infrastructure/dashboard state into LLM-consumable text | Reads from `Handler.cache` and `Handler.infrastructureState`; produces structured text for the system prompt |
| **Advisor** (service)        | Orchestrate context building, system prompt assembly, and provider call   | Holds references to context builder and provider; called by chat handler                                     |
| **Chat Handler**             | HTTP endpoint that accepts user message, streams SSE response             | `POST /api/v1/chat`; writes SSE frames via `http.Flusher`                                                    |
| **ChatContext** (React)      | Frontend state for conversation messages, streaming status                | React Context wrapping message array, send/abort functions                                                   |
| **AdvisorPanel**             | Side panel UI rendering chat messages with markdown                       | Slide-over panel triggered from dashboard header                                                             |
| **advisorApi**               | Frontend service for chat HTTP calls and SSE consumption                  | Uses `fetch` for POST, reads response body as SSE stream via `ReadableStream`                                |

## Recommended Project Structure

### Backend additions

```
backend/
├── services/
│   ├── advisor.go             # Advisor service (orchestration)
│   ├── advisor_test.go
│   ├── context_builder.go     # Infrastructure state serializer
│   ├── context_builder_test.go
│   └── ai/
│       ├── provider.go        # ChatProvider interface + types
│       ├── anthropic.go       # Anthropic implementation
│       └── anthropic_test.go
├── handlers/
│   ├── chat.go                # POST /api/v1/chat SSE handler
│   └── chat_test.go
├── config/
│   └── config.go              # Add AI_PROVIDER, AI_API_KEY, AI_MODEL fields
└── models/
    └── chat.go                # ChatRequest, ChatMessage types
```

### Frontend additions

```
frontend/src/
├── contexts/
│   └── ChatContext.jsx         # Chat state management
├── components/
│   └── advisor/
│       ├── AdvisorPanel.jsx    # Slide-over panel container
│       ├── MessageList.jsx     # Scrollable message list
│       ├── MessageBubble.jsx   # Individual message with markdown rendering
│       ├── ChatInput.jsx       # Text input with submit
│       └── StarterPrompts.jsx  # Contextual starter suggestions
├── services/
│   └── advisorApi.js           # Chat API + SSE stream reader
└── utils/
    └── sseParser.js            # Parse SSE text/event-stream lines (if needed)
```

### Structure Rationale

- **`services/ai/`:** Isolates LLM provider concerns from domain logic. The interface lives alongside its implementations so adding a provider means adding one file, not touching existing code.
- **`services/advisor.go`:** Sits at the same layer as existing calculators (`scenario.go`, `planning.go`). It coordinates the context builder and provider -- it is not a handler.
- **`services/context_builder.go`:** Separate from advisor because the context serialization logic will grow as more data sources are added in later phases. Keeping it in its own file prevents advisor.go from bloating.
- **`components/advisor/`:** Groups all chat UI components. The panel is a self-contained feature overlay, not woven into existing dashboard components.
- **`contexts/ChatContext.jsx`:** Matches the existing pattern (`AuthContext`, `ToastContext`). Provides chat state to the panel without prop drilling from `TASCapacityAnalyzer`.

## Architectural Patterns

### Pattern 1: Provider Interface with Streaming Return

**What:** Define a `ChatProvider` interface that returns a channel-based token stream, letting the handler write SSE frames without knowing which LLM is behind it.

**When to use:** Any time the backend sends tokens to the frontend incrementally.

**Trade-offs:** The interface adds one level of indirection. For Phase 1 with a single provider this is marginal overhead, but it prevents the handler from coupling to Anthropic types -- worth it because Phase 2 adds providers.

**Example:**

```go
// services/ai/provider.go

type Message struct {
    Role    string `json:"role"`    // "user" or "assistant"
    Content string `json:"content"`
}

type Token struct {
    Text string // empty on final/error tokens
    Err  error  // non-nil signals stream error
    Done bool   // true signals stream complete
}

type ChatProvider interface {
    // StreamMessage sends messages to the LLM and returns a channel of tokens.
    // The caller must drain the channel. The provider closes the channel when done.
    // Cancelling ctx stops the upstream LLM call.
    StreamMessage(ctx context.Context, system string, messages []Message) (<-chan Token, error)
}
```

**Why channels:** Go channels map naturally to SSE's sequential token delivery. The handler reads from the channel in a `for range` loop and writes each token as an SSE `data:` frame. When the channel closes, the handler sends the final SSE event and returns. This avoids callback-style APIs and keeps the handler simple.

### Pattern 2: Context Builder Reads Existing Handler State

**What:** The context builder reads directly from the `Handler`'s in-memory cache and infrastructure state (both already thread-safe) rather than making HTTP calls to the backend's own API or duplicating data fetching.

**When to use:** Whenever the LLM needs to see current dashboard/infrastructure data.

**Trade-offs:** Tight coupling to Handler internals. Mitigated by the builder accepting explicit parameters (cache contents, infrastructure state) rather than holding a reference to the Handler itself.

**Example:**

```go
// services/context_builder.go

type ContextBuilder struct{}

type ContextSnapshot struct {
    DashboardData       *models.DashboardResponse
    InfrastructureState *models.InfrastructureState
}

// BuildSystemContext serializes infrastructure state into a text block
// suitable for inclusion in the LLM system prompt.
func (cb *ContextBuilder) BuildSystemContext(snap ContextSnapshot) string {
    // Renders structured text: cell counts, memory utilization,
    // HA status, bottleneck analysis, etc.
    // Returns empty string sections for nil data (graceful degradation).
}
```

The handler assembles the `ContextSnapshot` from its own fields before passing it to the advisor service. This keeps the builder a pure function with no I/O.

### Pattern 3: SSE via POST with Streaming Response Body

**What:** The chat endpoint uses `POST` (to send the request body containing conversation history) but responds with `Content-Type: text/event-stream`. The handler writes SSE frames by flushing the `http.ResponseWriter` after each token.

**When to use:** Chat endpoints where the request has a JSON body but the response is a token stream.

**Trade-offs:** `POST` with SSE is not the "textbook" SSE pattern (which uses `GET` + `EventSource`). However, `EventSource` cannot send a request body, and encoding conversation history in query parameters is impractical. The frontend uses `fetch()` + `ReadableStream` instead of `EventSource`, which is standard practice for LLM chat APIs (OpenAI, Anthropic, and most chat UIs use this pattern).

**Example (handler):**

```go
// handlers/chat.go

func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        h.writeError(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    // Parse request body
    var req models.ChatRequest
    // ... decode JSON ...

    // Build context snapshot from current state
    snap := services.ContextSnapshot{
        DashboardData:       h.getCachedDashboard(),
        InfrastructureState: h.getInfrastructureState(),
    }

    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering

    // Stream tokens
    tokens, err := h.advisor.Stream(r.Context(), snap, req.Messages)
    if err != nil {
        // Error before streaming started -- can still write JSON error
        h.writeError(w, "failed to start chat", http.StatusBadGateway)
        return
    }

    for token := range tokens {
        if token.Err != nil {
            fmt.Fprintf(w, "event: error\ndata: %s\n\n", token.Err.Error())
            flusher.Flush()
            break
        }
        fmt.Fprintf(w, "data: %s\n\n", token.Text)
        flusher.Flush()
    }

    fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
    flusher.Flush()
}
```

**Example (frontend):**

```javascript
// services/advisorApi.js

export async function streamChat(messages, onToken, onDone, onError) {
  const response = await fetch("/api/v1/chat", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...withCSRFToken(),
    },
    credentials: "include",
    body: JSON.stringify({ messages }),
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    const text = decoder.decode(value, { stream: true });
    // Parse SSE lines and call onToken for each data: frame
  }
}
```

### Pattern 4: Feature Gating via Configuration

**What:** The AI advisor is gated by `AI_PROVIDER` env var. When unset, the chat route is not registered and the health endpoint reports `"ai_configured": false`. No frontend code path reaches the chat panel if the health check says AI is not available.

**When to use:** Optional features that depend on external service credentials.

**Trade-offs:** Adds a conditional branch in `main.go` route registration and a conditional check in the frontend. This is simpler than a feature flag system and matches the existing pattern (`BOSHEnvironment != ""` gates BOSH, `VSphereConfigured()` gates vSphere).

## Data Flow

### Chat Request Flow

```
[User types message in AdvisorPanel]
    │
    ▼
[ChatContext.sendMessage()]
    │ Sets streaming=true, appends user message
    │
    ▼
[advisorApi.streamChat(messages)]
    │ POST /api/v1/chat {messages: [...]}
    │ Headers: Content-Type, X-CSRF-Token, Cookie
    │
    ▼
[Middleware Chain]
    │ CORS → CSRF validates (POST) → Auth validates session → RateLimit (chat tier) → Log
    │
    ▼
[handlers/chat.go]
    │ 1. Parse ChatRequest (messages array)
    │ 2. Read cached dashboard data from h.cache.Get("dashboard:all")
    │ 3. Read infrastructure state from h.infrastructureState (RLock)
    │ 4. Build ContextSnapshot
    │ 5. Set SSE response headers
    │
    ▼
[services/advisor.go - Stream()]
    │ 1. Call contextBuilder.BuildSystemContext(snapshot)
    │ 2. Assemble system prompt = domain expertise + context
    │ 3. Call provider.StreamMessage(ctx, systemPrompt, messages)
    │ 4. Return token channel to handler
    │
    ▼
[services/ai/anthropic.go - StreamMessage()]
    │ 1. Map messages to anthropic.MessageParam slice
    │ 2. Call client.Messages.NewStreaming() with system prompt
    │ 3. Goroutine: iterate stream events, send TextDelta to channel
    │ 4. Close channel when stream ends or ctx cancelled
    │
    ▼
[handlers/chat.go - write loop]
    │ for token := range tokens:
    │   fmt.Fprintf(w, "data: %s\n\n", token.Text)
    │   flusher.Flush()
    │
    ▼
[Frontend ReadableStream reader]
    │ Parse SSE frames, call onToken callback
    │
    ▼
[ChatContext]
    │ Append token text to last assistant message
    │ On "done" event: set streaming=false
    │
    ▼
[AdvisorPanel re-renders with updated message]
```

### Context Assembly Flow

```
[Handler receives chat request]
    │
    ├── h.cache.Get("dashboard:all")
    │       → DashboardResponse (cells, apps, segments, metadata)
    │       → May be nil (cache expired or never fetched)
    │
    ├── h.infrastructureState (RLock)
    │       → InfrastructureState (clusters, utilization, HA status)
    │       → May be nil (no infrastructure loaded yet)
    │
    ▼
[ContextBuilder.BuildSystemContext()]
    │
    │ Produces structured text like:
    │
    │ ## Current Infrastructure
    │ - Source: vsphere
    │ - Clusters: 2 (total 12 hosts, 1536 GB memory)
    │ - Diego Cells: 24 (total 384 GB cell memory)
    │ - HA Status: ok (survives 1 host failure per cluster)
    │ - Memory Utilization: 72.3%
    │
    │ ## Application Workload
    │ - Running Apps: 847
    │ - Total App Memory: 245 GB allocated
    │ - Isolation Segments: 3
    │
    │ ## Data Availability
    │ - BOSH data: available (real cell vitals)
    │ - vSphere data: available (infrastructure from vcenter)
    │ - Note: [if data missing, say what's missing and why]
    │
    ▼
[Advisor prepends domain system prompt]
    │
    │ "You are a TAS/Diego capacity planning expert.
    │  You help operators interpret capacity metrics,
    │  plan hardware procurement, and optimize density.
    │  ...
    │  [context block from builder]"
    │
    ▼
[Full system prompt sent to ChatProvider]
```

### Frontend State Flow

```
[App.jsx]
    │
    ├── AuthProvider (existing)
    ├── ToastProvider (existing)
    └── ChatProvider (new)
            │
            ├── messages: [{role, content, timestamp}]
            ├── streaming: boolean
            ├── error: string|null
            ├── sendMessage(text): async
            ├── clearMessages(): void
            └── abortStream(): void (via AbortController)
            │
            ▼
        [TASCapacityAnalyzer]
            │
            ├── (existing dashboard state: data, loading, etc.)
            │
            ├── Passes infrastructure snapshot to ChatContext
            │   when advisor panel opens or data refreshes
            │
            └── [AdvisorPanel]  (reads from ChatContext)
                    │
                    ├── MessageList → MessageBubble (markdown rendered)
                    ├── StarterPrompts (shown when messages empty)
                    └── ChatInput (text field + send button)
```

## Integration with Existing Architecture

### Middleware Chain (no changes)

The chat endpoint plugs into the existing middleware chain identically to other POST endpoints:

```go
// In routes.go
{Method: http.MethodPost, Path: "/api/v1/chat", Handler: h.Chat, RateLimit: "chat"}
```

- **CORS:** Applied (same as all routes)
- **CSRF:** Validates `X-CSRF-Token` header (POST requires it for session-authenticated requests)
- **Auth:** Required (not marked `Public`)
- **RateLimit:** Uses a dedicated "chat" tier (10 req/min as specified in PROJECT.md)
- **LogRequest:** Applied

A "chat" rate limit tier needs to be added to the rate limiter map in `main.go`, following the existing pattern.

### Handler Integration

The `Handler` struct gains one new field:

```go
type Handler struct {
    // ... existing fields ...
    advisor  *services.Advisor  // nil when AI_PROVIDER not set
}
```

The chat handler checks `h.advisor != nil` before proceeding. If nil, returns 503 Service Unavailable.

The handler exposes two reader methods for the context builder:

```go
func (h *Handler) getCachedDashboard() *models.DashboardResponse {
    if cached, found := h.cache.Get("dashboard:all"); found {
        resp := cached.(models.DashboardResponse)
        return &resp
    }
    return nil
}

func (h *Handler) getInfrastructureState() *models.InfrastructureState {
    h.infraMutex.RLock()
    defer h.infraMutex.RUnlock()
    if h.infrastructureState == nil {
        return nil
    }
    copy := *h.infrastructureState
    return &copy
}
```

These read-only accessors use the existing synchronization primitives (`cache` is thread-safe, `infraMutex` is already used for read/write).

### Config Integration

New fields in `config.Config`:

```go
// AI Advisor (optional)
AIProvider string // "anthropic" or "" (disabled)
AIAPIKey   string // API key for the configured provider
AIModel    string // Model name (default varies by provider)
```

Loaded from `AI_PROVIDER`, `AI_API_KEY`, `AI_MODEL` env vars. When `AI_PROVIDER` is empty, no AI components are initialized -- matching the BOSH/vSphere optional pattern.

### Health Endpoint Extension

Add `"ai_configured"` field to health response:

```go
resp["ai_configured"] = h.advisor != nil
```

Frontend reads this to decide whether to show the advisor button.

## Anti-Patterns

### Anti-Pattern 1: Sending Full JSON State to the LLM

**What people do:** Serialize the entire `InfrastructureState` or `DashboardResponse` as JSON and stuff it into the system prompt.

**Why it's wrong:** Raw JSON wastes tokens with field names, nesting, and values the LLM does not need (timestamps, cache flags, GUIDs). A 24-cell cluster with 800 apps produces ~50KB of JSON -- a substantial fraction of the context window used on formatting noise.

**Do this instead:** The context builder should produce human-readable summary text: aggregate metrics, key ratios, and notable conditions. Include per-cluster breakdowns but not per-app detail unless the user asks about a specific app (Phase 2 tool use).

### Anti-Pattern 2: Advisor Fetching Its Own Data via HTTP

**What people do:** Have the advisor service call the backend's own `/api/v1/dashboard` endpoint to get data.

**Why it's wrong:** Adds an HTTP roundtrip to localhost, bypasses auth (or requires service-to-service auth), and creates a circular dependency. The data is already in memory.

**Do this instead:** Pass data explicitly from the handler to the advisor. The handler already has access to cache and infrastructure state through its struct fields.

### Anti-Pattern 3: Using EventSource on the Frontend

**What people do:** Use the browser `EventSource` API for SSE streaming.

**Why it's wrong:** `EventSource` only supports GET requests. Chat requires sending a JSON body with conversation history. Workarounds (encoding messages in URL query params) are fragile, have URL length limits, and expose conversation content in server logs.

**Do this instead:** Use `fetch()` with a `ReadableStream` reader. This is the same pattern used by ChatGPT, Claude, and every major LLM chat UI. The trade-off is slightly more parsing code on the frontend, but it's well-understood.

### Anti-Pattern 4: Storing Conversation in Backend State

**What people do:** Store conversation history on the backend in a session or database.

**Why it's wrong for Phase 1:** Adds state management complexity (cleanup, storage limits, session affinity) when the frontend already holds messages in React state. The conversation is short-lived and does not need to survive page refreshes in Phase 1.

**Do this instead:** Frontend owns message history. Each POST to `/api/v1/chat` sends the full message array. Backend is stateless for conversations. If persistence is needed in Phase 3, add it then.

## Build Order (Dependencies)

The following build order reflects component dependencies:

```
Phase 1: Backend Foundation
    1. Config: Add AI_PROVIDER, AI_API_KEY, AI_MODEL to config.go
    2. Models: Add ChatRequest, ChatMessage types (models/chat.go)
    3. Provider interface: Define ChatProvider + Token types (services/ai/provider.go)
    4. Anthropic provider: Implement streaming (services/ai/anthropic.go)
       └── Depends on: provider interface, config (API key)
    5. Context builder: Serialize state to text (services/context_builder.go)
       └── Depends on: models (DashboardResponse, InfrastructureState)
    6. Advisor service: Orchestrate context + provider (services/advisor.go)
       └── Depends on: context builder, provider interface
    7. Chat handler + route: SSE endpoint (handlers/chat.go, routes.go)
       └── Depends on: advisor service, handler state accessors

Phase 2: Frontend
    8. advisorApi service: fetch + SSE stream reader (services/advisorApi.js)
       └── Depends on: chat endpoint being available
    9. ChatContext: State management (contexts/ChatContext.jsx)
       └── Depends on: advisorApi
   10. AdvisorPanel + sub-components: UI (components/advisor/*.jsx)
       └── Depends on: ChatContext
   11. Integration: Wire panel into TASCapacityAnalyzer + App.jsx
       └── Depends on: AdvisorPanel, health endpoint ai_configured flag

Phase 3: Polish
   12. Starter prompts: Generate from current data
   13. Graceful degradation: Handle missing BOSH/vSphere in context builder
   14. Rate limit tier: Add "chat" tier to main.go
   15. Feature gating: Conditional route registration + health flag
```

**Build order rationale:**

- Config and models come first because everything depends on types.
- Provider interface before implementation because the advisor depends on the interface, not the concrete type.
- Context builder and provider can be built in parallel (no dependency between them).
- The advisor wires them together.
- The handler is last on the backend because it depends on the advisor.
- Frontend starts after the endpoint exists (or can be developed against a mock).
- Integration is last because it touches existing components.

## Sources

- [Anthropic Go SDK - official repository](https://github.com/anthropics/anthropic-sdk-go) (Context7, HIGH confidence) -- streaming API, message construction, SDK types
- [Go SSE implementation patterns](https://packagemain.tech/p/implementing-server-sent-events-in-go) (WebSearch, MEDIUM confidence) -- `http.Flusher`, header patterns, timeout handling
- [FreeCodeCamp - SSE in Go](https://www.freecodecamp.org/news/how-to-implement-server-sent-events-in-go/) (WebSearch, MEDIUM confidence) -- `X-Accel-Buffering` header for reverse proxy compatibility
- Existing codebase: `backend/handlers/handlers.go`, `backend/handlers/routes.go`, `backend/main.go`, `backend/config/config.go`, `backend/middleware/chain.go`, `backend/middleware/csrf.go`, `frontend/src/contexts/AuthContext.jsx`, `frontend/src/services/apiClient.js` (HIGH confidence) -- all integration points verified by reading source

---

_Architecture research for: AI conversational advisor in Diego Capacity Analyzer_
_Researched: 2026-02-24_
