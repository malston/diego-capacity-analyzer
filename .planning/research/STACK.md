# Stack Research

**Domain:** AI conversational advisor in an existing Go/React capacity planning dashboard
**Researched:** 2026-02-24
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology                                | Version        | Purpose                                                   | Why Recommended                                                                                                                                                                                                                                                               |
| ----------------------------------------- | -------------- | --------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `github.com/anthropics/anthropic-sdk-go`  | v1.26.0        | Go client for Anthropic Messages API with streaming       | Official Anthropic SDK. First-party, typed, actively maintained (multiple releases per week in Feb 2026). Provides `NewStreaming()` with event iteration, `Message.Accumulate()`, and typed event handling. Requires Go 1.22+; project is on Go 1.24.                         |
| Go `net/http` + `http.ResponseController` | Go 1.24 stdlib | SSE streaming from backend to frontend                    | No library needed. `http.NewResponseController(w)` (added Go 1.20) provides `Flush()` for SSE event delivery. Set `Content-Type: text/event-stream`, `Cache-Control: no-cache`, write `data:` lines, flush after each event. Project already uses `net/http` for all routing. |
| `streamdown`                              | 2.3.0          | Streaming-aware markdown rendering in React chat messages | Drop-in `react-markdown` replacement built for AI streaming. Handles incomplete/unterminated markdown blocks, code fences, and lists mid-stream without flicker. Peer deps: React `^18.0.0 \|\| ^19.0.0` (project is React 18.2). Tailwind CSS v3 compatible. MIT license.    |

### Supporting Libraries

| Library            | Version                   | Purpose                                                      | When to Use                                                                                                  |
| ------------------ | ------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| `@streamdown/code` | (matches streamdown)      | Syntax highlighting in markdown code blocks                  | Install alongside `streamdown` for code block rendering in assistant responses. Uses Shiki for highlighting. |
| `remark-gfm`       | (bundled with streamdown) | GitHub Flavored Markdown (tables, task lists, strikethrough) | Bundled as a streamdown dependency. Enables tables in assistant capacity analysis output.                    |
| `lucide-react`     | ^0.294.0 (existing)       | Icons for chat UI (send button, close panel, advisor icon)   | Already installed. Use existing icons; no new icon library needed.                                           |

### Development Tools

| Tool                                | Purpose                    | Notes                                                                             |
| ----------------------------------- | -------------------------- | --------------------------------------------------------------------------------- |
| `vitest` (existing)                 | Frontend test runner       | Already configured. Test SSE hook with `vi.fn()` mocking `EventSource`.           |
| `@testing-library/react` (existing) | React component testing    | Already configured. Test chat panel rendering, message display, streaming states. |
| `testify` (existing)                | Backend Go test assertions | Already configured. Test provider abstraction, context builder, SSE handler.      |

## Installation

```bash
# Backend - Go module
cd backend
go get github.com/anthropics/anthropic-sdk-go@v1.26.0

# Frontend - npm (project uses npm, not bun)
cd frontend
npm install streamdown @streamdown/code
```

No other new dependencies required. SSE streaming uses Go stdlib. Chat UI is custom components with Tailwind CSS (already installed).

## Alternatives Considered

| Recommended                   | Alternative                                        | When to Use Alternative                                                                                                                                                                                                            |
| ----------------------------- | -------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `anthropic-sdk-go` (official) | `liushuangls/go-anthropic` (community)             | Never for this project. Community SDK has fewer features, no official support, and lags behind API changes. Official SDK tracks API releases within days.                                                                          |
| `anthropic-sdk-go` (official) | `digitallysavvy/go-ai` (multi-provider)            | If you need 26+ LLM providers from a single SDK. Overkill here -- project only needs Anthropic for Phase 1 and the `ChatProvider` interface enables adding providers without a meta-SDK.                                           |
| `streamdown`                  | `react-markdown`                                   | If you only render static (non-streaming) markdown. `react-markdown` re-parses the entire document on every token update, causing O(n^2) behavior and visible flicker with incomplete markdown blocks during streaming.            |
| `streamdown`                  | Custom incremental parser (`marked` + memoization) | If you need zero dependencies. Requires implementing your own incomplete-block handling, stable-boundary detection, and memoization. Streamdown solves this out of the box.                                                        |
| Go stdlib SSE                 | `r3labs/sse` or `tmaxmax/go-sse`                   | If you need pub/sub fan-out to many clients, event replay with IDs, or named channels. The chat endpoint is 1:1 (one SSE stream per request), so a library adds unnecessary abstraction.                                           |
| Custom chat components        | `@chatscope/chat-ui-kit-react`                     | If building a general-purpose chat app with complex features (typing indicators, avatars, presence, threads). Our UI is a simple side panel with message bubbles -- a component library would fight our existing Tailwind styling. |
| Custom chat components        | `@llamaindex/chat-ui`                              | If using LlamaIndex as the LLM framework. We use Anthropic directly; LlamaIndex components assume LlamaIndex data structures.                                                                                                      |

## What NOT to Use

| Avoid                                              | Why                                                                                                                                                                                                                                                                               | Use Instead                                                                |
| -------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| WebSocket for chat streaming                       | SSE is simpler, unidirectional (server-to-client), works with existing HTTP middleware (auth, CSRF, rate limiting), and the chat endpoint only needs server push. WebSocket requires separate connection management, auth handshake, and doesn't go through the middleware chain. | SSE via `text/event-stream` response                                       |
| `react-markdown` for streaming content             | Re-parses entire document on each token. Causes flickering code blocks, half-rendered bold/italic, and O(n^2) parse time during streaming. Well-documented problem in AI chat applications.                                                                                       | `streamdown` (streaming-aware, incremental rendering)                      |
| Vercel AI SDK (`ai` / `@ai-sdk/react`)             | Brings its own backend abstraction (`streamText`, `generateText`) that conflicts with our Go backend. Designed for Next.js/Node.js backends. Would require rewriting the backend streaming layer to match its protocol or running a Node.js proxy.                                | Direct SSE from Go backend + `streamdown` (standalone, no AI SDK coupling) |
| State management libraries (Redux, Zustand, Jotai) | Project uses React Context and local state. Chat state is component-local (messages array, streaming flag, input text). No cross-component state sharing needed beyond what Context already provides. Adding a state library for one feature violates YAGNI.                      | React `useState` + `useRef` in a custom `useAdvisor` hook                  |
| `EventSource` API for SSE client                   | `EventSource` only supports GET requests. Chat endpoint is POST (sends message history in body). `EventSource` also lacks custom headers (needed for auth Bearer token and CSRF token).                                                                                           | `fetch()` with `ReadableStream` reader, parsing SSE `data:` lines manually |

## Stack Patterns

**SSE streaming pattern (Go backend):**

- Chat handler receives POST with messages array
- Creates Anthropic streaming request via SDK
- Sets SSE headers, creates `http.ResponseController`
- Iterates SDK stream events, writes `data: {json}\n\n` lines, calls `rc.Flush()` after each
- Writes `data: [DONE]\n\n` sentinel on completion
- Context cancellation handles client disconnect

**SSE consumption pattern (React frontend):**

- `useAdvisor` hook calls `fetch()` with POST, reads response body as `ReadableStream`
- Parses SSE lines from stream chunks (`TextDecoder` + line splitting)
- Appends text deltas to current message via `setState`
- `streamdown` component re-renders incrementally as content grows
- `isAnimating` prop shows typing indicator caret during streaming

**Provider abstraction pattern (Go backend):**

- `ChatProvider` interface with `StreamChat(ctx, request) (<-chan StreamEvent, error)`
- `AnthropicProvider` implements interface using `anthropic-sdk-go`
- Handler receives provider via dependency injection
- Adding a provider = implement interface + add factory in config

## Version Compatibility

| Package                    | Compatible With              | Notes                                                                                                                       |
| -------------------------- | ---------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `anthropic-sdk-go` v1.26.0 | Go 1.22+                     | Project uses Go 1.24. SDK uses Go modules. No conflict with existing `govmomi`, `socks5-proxy`, or `godotenv` dependencies. |
| `streamdown` 2.3.0         | React `^18.0.0 \|\| ^19.0.0` | Project uses React 18.2.0. Compatible.                                                                                      |
| `streamdown` 2.3.0         | Tailwind CSS v3 and v4       | Project uses Tailwind 3.3.6. Compatible. Add `@source` directive or configure content paths for streamdown dist files.      |
| `@streamdown/code`         | `streamdown` 2.x             | Version tracks streamdown. Install together.                                                                                |
| `http.ResponseController`  | Go 1.20+                     | Available in Go 1.24 stdlib. No import needed beyond `net/http`.                                                            |

## Sources

- [anthropics/anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) -- Official Anthropic Go SDK, v1.26.0 release (Feb 19, 2026). **HIGH confidence** (Context7 + official GitHub).
- [pkg.go.dev/anthropic-sdk-go](https://pkg.go.dev/github.com/anthropics/anthropic-sdk-go) -- Go package documentation, version verification. **HIGH confidence**.
- [vercel/streamdown](https://github.com/vercel/streamdown) -- Streaming markdown renderer, v2.3.0, peer deps verified from package.json. **HIGH confidence** (Context7 + official GitHub).
- [streamdown.ai/docs](https://streamdown.ai/docs) -- Official docs confirming Tailwind v3/v4 compatibility, standalone usage without AI SDK. **MEDIUM confidence** (official docs, not independently verified for v3 claim).
- [Alex Edwards: http.ResponseController](https://www.alexedwards.net/blog/how-to-use-the-http-responsecontroller-type) -- Confirms `http.NewResponseController` added in Go 1.20 for per-request flush control. **HIGH confidence** (official Go release notes corroborate).
- [Go 1.20 Release Notes](https://go.dev/doc/go1.20) -- `http.ResponseController` addition confirmed. **HIGH confidence**.
- [HN: Flash of Incomplete Markdown](https://news.ycombinator.com/item?id=44182941) -- Community discussion of `react-markdown` streaming problems, validating `streamdown` as the solution. **MEDIUM confidence** (community consensus, not official source).
- [remarkjs/react-markdown](https://github.com/remarkjs/react-markdown) -- v10.1.0, React peer dep verification. **HIGH confidence** (Context7).
- [Vercel changelog: Streamdown introduction](https://vercel.com/changelog/introducing-streamdown) -- Confirms standalone usage, no AI SDK coupling required. **MEDIUM confidence**.

---

_Stack research for: AI Capacity Advisor (Phase 1)_
_Researched: 2026-02-24_
