# Roadmap: AI Capacity Advisor (Phase 1)

## Overview

Deliver a conversational AI advisor embedded in the Diego Capacity Analyzer dashboard. The backend streaming pipeline is the critical path -- provider abstraction, context serialization, domain prompt, and SSE endpoint must all exist before any frontend work is visible. The frontend then delivers the chat panel in two stages (core rendering, then UX refinements). Graceful degradation and polish round out the release. Every phase delivers a coherent, independently testable capability.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Provider Foundation** - ChatProvider interface, Anthropic implementation, config, and feature gating (completed 2026-02-24)
- [ ] **Phase 2: Context Builder** - Serialize infrastructure/dashboard/scenario state for the LLM without leaking credentials
- [ ] **Phase 3: Domain Expertise** - System prompt encoding TAS/Diego capacity planning knowledge and procurement framing
- [ ] **Phase 4: Chat Endpoint** - SSE streaming endpoint with auth, rate limiting, error handling, and timeout protection
- [ ] **Phase 5: Chat Panel Core** - Side panel with streaming Markdown display and multi-turn conversation threading
- [ ] **Phase 6: Chat Panel UX** - Loading states, error handling, conversation reset, and starter prompts
- [ ] **Phase 7: Graceful Degradation** - CF-only operation, data gap messaging, and adaptive starter prompts
- [ ] **Phase 8: Polish** - Copy to clipboard, response feedback, and procurement-oriented system prompt tuning

## Phase Details

### Phase 1: Provider Foundation

**Goal**: Backend has a pluggable LLM provider abstraction with a working Anthropic implementation, configured via environment variables, that streams token-by-token responses
**Depends on**: Nothing (first phase)
**Requirements**: PROV-01, PROV-02, PROV-03, PROV-04
**Success Criteria** (what must be TRUE):

1. A Go interface (`ChatProvider`) exists that accepts conversation messages and returns a streaming channel of tokens, decoupled from any specific LLM vendor
2. Anthropic Claude provider streams token-by-token responses using the official `anthropic-sdk-go` SDK when given valid API credentials
3. Setting `AI_PROVIDER=anthropic` and `AI_API_KEY` at startup initializes the provider; missing or invalid config produces a clear startup log
4. When `AI_PROVIDER` is unset, the health endpoint reports `ai_configured: false` and no provider is initialized
   **Plans**: 3 plans

Plans:

- [ ] 01-01-PLAN.md -- ChatProvider interface, domain types, and functional options (TDD)
- [ ] 01-02-PLAN.md -- Anthropic provider implementation with streaming (TDD)
- [ ] 01-03-PLAN.md -- Config loading, startup wiring, health endpoint, feature gating

### Phase 2: Context Builder

**Goal**: A pure function serializes the handler's in-memory infrastructure state into human-readable annotated text for the LLM, reading only model types and never touching credentials or making API calls
**Depends on**: Phase 1
**Requirements**: CTX-01, CTX-02, CTX-03, CTX-04, CTX-05
**Success Criteria** (what must be TRUE):

1. Context builder produces annotated text containing cell counts, memory utilization, isolation segments, and app counts from dashboard state
2. Context builder includes vSphere infrastructure data (clusters, hosts, VMs) when available, and omits the section cleanly when unavailable
3. Context builder includes scenario comparison results when a scenario has been run
4. Context builder inserts explicit markers for missing data sources (e.g., "BOSH data: unavailable") that the LLM can reference in its responses
5. Serialized context contains zero credential values -- enforced by a unit test that checks output against known credential field values
   **Plans**: TBD

Plans:

- [ ] 02-01: TBD
- [ ] 02-02: TBD

### Phase 3: Domain Expertise

**Goal**: System prompt turns the LLM from a generic chatbot into a TAS/Diego capacity planning domain expert that reasons about procurement decisions using the operator's live data
**Depends on**: Phase 2
**Requirements**: DOM-01, DOM-02, DOM-03, DOM-04
**Success Criteria** (what must be TRUE):

1. System prompt encodes TAS/Diego capacity planning knowledge including N-1 redundancy, HA Admission Control, vCPU:pCPU ratios, cell sizing heuristics, and isolation segment tradeoffs
2. System prompt frames analysis in procurement terms: lead times, budget cycles, growth planning, and headroom targets
3. When context contains missing-data markers, the LLM acknowledges gaps rather than hallucinating (verified by manual testing with incomplete context)
4. LLM references specific data values from the provided context when making claims (verified by manual testing with known infrastructure state)
   **Plans**: TBD

Plans:

- [ ] 03-01: TBD

### Phase 4: Chat Endpoint

**Goal**: Operators can send conversation messages to `POST /api/v1/chat` and receive streaming SSE token responses, protected by auth, CSRF, and rate limiting
**Depends on**: Phase 3
**Requirements**: CHAT-01, CHAT-02, CHAT-03, CHAT-04, CHAT-05
**Success Criteria** (what must be TRUE):

1. `POST /api/v1/chat` accepts JSON with conversation messages and returns an SSE stream of token events that can be consumed by a standard SSE client
2. Unauthenticated requests receive a JSON 401 error (not an SSE stream)
3. Requests exceeding 10/min per user receive a JSON 429 error with retry-after guidance
4. Pre-stream failures (auth, rate limit, missing provider, bad request) return structured JSON errors; SSE streaming only begins after all preconditions pass
5. Streaming does not hang indefinitely -- idle timeout terminates the stream if no tokens arrive within a configured window
   **Plans**: TBD

Plans:

- [ ] 04-01: TBD
- [ ] 04-02: TBD

### Phase 5: Chat Panel Core

**Goal**: Operators see a side panel in the dashboard with streaming Markdown display and can have multi-turn conversations with the advisor
**Depends on**: Phase 4
**Requirements**: UI-01, UI-02, UI-03, UI-04, UI-05
**Success Criteria** (what must be TRUE):

1. A side panel slides in from the right as an overlay, usable on all screen sizes, without disrupting the underlying dashboard layout
2. Panel toggle button appears in the dashboard header only when the health endpoint reports `ai_configured: true`; button is absent otherwise
3. Tokens stream into the panel as they arrive with smooth rendering -- no visible flicker or re-render storms during a 500+ token response
4. Assistant messages render Markdown correctly: headers, lists, bold, code blocks, and tables all display with proper formatting
5. Conversation maintains multi-turn context within the session -- the operator can ask follow-up questions that reference prior messages
   **Plans**: TBD

Plans:

- [ ] 05-01: TBD
- [ ] 05-02: TBD

### Phase 6: Chat Panel UX

**Goal**: The chat panel handles all edge cases gracefully -- loading states, errors, conversation management, and empty-state guidance
**Depends on**: Phase 5
**Requirements**: UI-06, UI-07, UI-08, UI-09
**Success Criteria** (what must be TRUE):

1. Operator can clear/reset the conversation to start a fresh dialogue without reloading the page
2. A loading/thinking indicator appears between sending a message and receiving the first token
3. LLM API failures, rate limit errors, timeouts, and network errors display user-friendly messages with a "Try again" action
4. When conversation is empty, static starter prompts appear suggesting common capacity planning questions the operator can click to ask
   **Plans**: TBD

Plans:

- [ ] 06-01: TBD

### Phase 7: Graceful Degradation

**Goal**: The advisor works meaningfully with CF-only data, explicitly communicates what it cannot analyze when BOSH or vSphere data is missing, and adapts its UI accordingly
**Depends on**: Phase 6
**Requirements**: DEG-01, DEG-02, DEG-03
**Success Criteria** (what must be TRUE):

1. With only CF data available (no BOSH, no vSphere), the advisor answers capacity questions using app counts, process stats, and memory allocations
2. When data sources are missing, the advisor explicitly tells the operator which sources are unavailable and what analysis it cannot perform (e.g., "I don't have BOSH cell vitals, so I can't assess actual memory utilization vs allocated")
3. Starter prompts adapt to available data -- vSphere-dependent questions do not appear when vSphere is unconfigured; BOSH-dependent questions do not appear when BOSH is unavailable
   **Plans**: TBD

Plans:

- [ ] 07-01: TBD

### Phase 8: Polish

**Goal**: Quality-of-life features that improve trust, workflow integration, and domain specificity of the advisor
**Depends on**: Phase 7
**Requirements**: POL-01, POL-02, POL-03
**Success Criteria** (what must be TRUE):

1. Operator can copy any assistant response to the clipboard with a single click (for pasting into procurement requests or tickets)
2. Operator can provide thumbs up/down feedback on any assistant response, and feedback is logged to the backend via `slog`
3. System prompt includes sharpened procurement-oriented framing so the advisor interprets capacity data specifically in terms of hardware procurement decisions, budget justification, and lead time planning
   **Plans**: TBD

Plans:

- [ ] 08-01: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8

| Phase                   | Plans Complete | Status      | Completed |
| ----------------------- | -------------- | ----------- | --------- |
| 1. Provider Foundation  | 2/3            | Complete    | 2026-02-24 |
| 2. Context Builder      | 0/0            | Not started | -         |
| 3. Domain Expertise     | 0/0            | Not started | -         |
| 4. Chat Endpoint        | 0/0            | Not started | -         |
| 5. Chat Panel Core      | 0/0            | Not started | -         |
| 6. Chat Panel UX        | 0/0            | Not started | -         |
| 7. Graceful Degradation | 0/0            | Not started | -         |
| 8. Polish               | 0/0            | Not started | -         |
