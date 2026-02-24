# AI Capacity Advisor (Phase 1)

## What This Is

An AI-powered conversational advisor embedded in the Diego Capacity Analyzer dashboard. The advisor is a domain expert in TAS/Diego capacity planning that can see the operator's current analysis data and have an interactive dialogue about capacity challenges, data interpretation, and procurement decisions. Phase 1 delivers chat with read-only access to infrastructure state.

## Core Value

Operators can have a conversation with a domain expert that sees their live capacity data -- turning raw metrics into actionable procurement guidance without context-switching away from the dashboard.

## Requirements

### Validated

- ✓ Go backend with CF, BOSH, vSphere, Log Cache integrations -- existing
- ✓ React frontend with capacity analysis dashboard -- existing
- ✓ BFF OAuth2 authentication with session cookies and CSRF -- existing
- ✓ RBAC with viewer/operator roles -- existing
- ✓ Scenario comparison calculator -- existing
- ✓ Infrastructure planning calculator with N-1 HA -- existing
- ✓ Bottleneck analysis and recommendations -- existing
- ✓ CLI with TUI dashboard -- existing
- ✓ Middleware chain (CORS, auth, CSRF, rate limiting, logging) -- existing
- ✓ In-memory TTL cache with background cleanup -- existing

### Active

- [ ] Pluggable LLM provider abstraction with streaming support
- [ ] Anthropic Claude provider implementation
- [ ] Context builder that serializes infrastructure/scenario state for the LLM
- [ ] Domain expertise system prompt encoding TAS/Diego capacity planning knowledge
- [ ] SSE chat endpoint (`POST /api/v1/chat`) with token-by-token streaming
- [ ] Feature gating via `AI_PROVIDER` env var; health endpoint reports `ai_configured`
- [ ] Side panel UI that slides over dashboard content
- [ ] Streaming chat display with token-by-token rendering via SSE
- [ ] Markdown rendering in assistant responses
- [ ] Starter prompts based on current data when conversation is empty
- [ ] Auto-context: panel receives updated infrastructure/scenario data automatically
- [ ] Graceful degradation: advisor works with CF-only data, flags missing BOSH/vSphere data
- [ ] Rate limiting on chat endpoint (10 req/min)

### Out of Scope

- OpenAI and OpenAI-compatible providers -- not needed until someone requests them (Phase 2+)
- Per-user API key override (BYOK) -- system key only for Phase 1
- Tool use / scenario execution via chat -- Phase 2
- Live UI sync between advisor actions and dashboard -- Phase 3
- Driving wizard inputs from chat -- Phase 3
- Conversation persistence across sessions -- Phase 3
- Push-content panel layout on wide screens -- overlay is sufficient for Phase 1

## Context

- The Diego Capacity Analyzer helps platform operators plan hardware procurement with 6-12 month lead times
- Operators currently interpret capacity metrics (N-1 utilization, HA constraints, bottleneck analysis) manually
- The backend already holds all infrastructure state in memory (dashboard cache, infrastructure state) -- the LLM context builder can read directly from these
- The existing middleware chain (auth, CSRF, rate limiting) applies to the new chat endpoint
- Issue #123 tracks this phase; #124 (Phase 2) and #125 (Phase 3) are future milestones
- Design document: https://gist.github.com/malston/9c121bc753dcd2e87e6b24ebf947a939

## Constraints

- **LLM Provider**: Anthropic Claude only for Phase 1 -- but implement behind a `ChatProvider` interface so adding providers later is straightforward
- **API Key**: System-level key in `AI_API_KEY` env var only -- no per-user key management
- **Data Sent to LLM**: Infrastructure metadata only (cluster sizes, utilization percentages, cell counts) -- no credentials, no PII
- **Streaming**: SSE (Server-Sent Events) for chat responses -- matches the existing frontend tech stack (no WebSocket dependency)
- **Auth**: Chat endpoint requires the same auth middleware as all other endpoints
- **Frontend State**: No new state management library -- use React Context or local state consistent with existing patterns

## Key Decisions

| Decision                                 | Rationale                                                                          | Outcome    |
| ---------------------------------------- | ---------------------------------------------------------------------------------- | ---------- |
| Anthropic only for Phase 1               | Simplifies implementation; add providers when needed                               | -- Pending |
| System key only (no BYOK)                | Reduces frontend complexity; BYOK adds config modal + localStorage key management  | -- Pending |
| Overlay panel (not push-content)         | Simpler CSS; push-content can be added in Phase 3 with UI sync work                | -- Pending |
| Provider abstraction interface           | Even with one provider, interface enables clean Phase 2 extension                  | -- Pending |
| Context builder reads from Handler state | Backend already caches dashboard/infrastructure state; no new data fetching needed | -- Pending |
| CF-only graceful degradation             | Many operators run without BOSH/vSphere access; advisor should still be useful     | -- Pending |

---

_Last updated: 2026-02-24 after initialization_
