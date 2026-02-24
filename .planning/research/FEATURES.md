# Feature Research

**Domain:** AI-embedded conversational advisor in an operational capacity planning dashboard
**Researched:** 2026-02-24
**Confidence:** MEDIUM -- Features are synthesized from multiple shipped products (Datadog Bits AI, New Relic AI, Grafana Assistant, Dynatrace Davis CoPilot, Power BI Copilot) and established UX patterns, but "table stakes" for this specific niche (TAS capacity planning) is partly inferred since no direct competitor embeds a chat advisor in a Diego/CF capacity tool.

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist in any AI chat panel embedded in an operational dashboard. Missing these and the advisor feels broken or amateurish.

| Feature                                   | Why Expected                                                                                                                                                                                                                                                                            | Complexity | Notes                                                                                                                                                            |
| ----------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Streaming token display**               | Every modern chat interface streams tokens as they arrive. Waiting for a full response feels broken. Datadog Bits, New Relic AI, Grafana Assistant all stream.                                                                                                                          | MEDIUM     | SSE from backend; `react-markdown` with incremental rendering. Buffer with `requestAnimationFrame` to avoid render thrashing. Already scoped in PROJECT.md.      |
| **Markdown rendering**                    | LLMs produce Markdown natively (headers, lists, bold, code blocks). Rendering as plain text looks unprofessional. All competitor advisors render Markdown.                                                                                                                              | LOW        | `react-markdown` + `react-syntax-highlighter` for code fences. Standard pattern. Already scoped.                                                                 |
| **Context awareness (sees current data)** | The entire value proposition is "advisor that sees your infrastructure." If it gives generic answers without referencing actual cell counts, utilization, or bottlenecks, it's just ChatGPT in a sidebar. Grafana Assistant and Datadog Bits both operate on the user's live telemetry. | HIGH       | Context builder must serialize dashboard/infrastructure/scenario state. Must update when data changes. This is the hardest table-stakes feature. Already scoped. |
| **Starter/suggested prompts**             | Empty chat with a blinking cursor is intimidating. Users need affordance showing what the advisor can do. Every major AI product (ChatGPT, Copilot, Grafana Assistant) ships suggested prompts.                                                                                         | LOW        | 3-5 contextual suggestions based on current data state (e.g., "What are my N-1 HA risks?" when infrastructure data is loaded). Already scoped.                   |
| **Conversation threading**                | Users expect multi-turn conversation where the advisor remembers earlier messages in the session. Single-shot Q&A feels like a search box, not an advisor.                                                                                                                              | LOW        | Send full conversation history with each request. Context window management is straightforward for Phase 1 session-length conversations.                         |
| **Clear/reset conversation**              | Users need to start fresh without reloading the page. Standard in all chat interfaces.                                                                                                                                                                                                  | LOW        | Clear local state, reset to starter prompts.                                                                                                                     |
| **Loading/thinking indicator**            | Users need visual feedback that the advisor is processing before tokens start arriving. Without it, the UI feels frozen.                                                                                                                                                                | LOW        | Animated indicator between send and first token. Disappears when streaming begins.                                                                               |
| **Error handling with retry**             | LLM API failures, rate limits, and timeouts happen. Users need clear error messages and a retry action, not silent failures or stack traces.                                                                                                                                            | LOW        | Display user-friendly error, offer "Try again" button. Map common failure modes (429, 500, timeout, network) to specific messages.                               |
| **Panel open/close toggle**               | The advisor must not permanently consume screen real estate. Operators need their dashboard visible. A toggle to open/close the panel is fundamental.                                                                                                                                   | LOW        | Already scoped as "side panel that slides over." Add keyboard shortcut (common pattern: Ctrl/Cmd+Shift+A or similar).                                            |
| **Graceful degradation messaging**        | When BOSH or vSphere data is unavailable, the advisor must explicitly state what data it lacks and what it can still help with, rather than hallucinating or failing silently. Grafana Assistant and New Relic AI both indicate data source limitations.                                | MEDIUM     | Context builder flags missing data sources. System prompt instructs LLM to acknowledge gaps. Already scoped in PROJECT.md.                                       |

### Differentiators (Competitive Advantage)

Features that set this advisor apart. Not expected in a Phase 1 MVP, but each one adds significant value and some are low enough cost to include.

| Feature                                    | Value Proposition                                                                                                                                                                                                                                                                   | Complexity | Notes                                                                                                                                                                                                                      |
| ------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Domain-expert system prompt**            | Generic LLM assistants can't interpret N-1 HA constraints, isolation segment density, or Diego cell sizing tradeoffs. A deeply encoded system prompt that understands TAS capacity planning turns the advisor from "chatbot" into "consultant." This is the primary differentiator. | MEDIUM     | Requires encoding significant domain knowledge: HA formulas, cell sizing heuristics, procurement lead time awareness, isolation segment tradeoffs. Not just a prompt -- it's domain expertise engineering. Already scoped. |
| **Data-grounded responses with citations** | When the advisor says "your production segment is at 78% memory utilization," it should reference the actual data point. Prevents hallucination and builds operator trust. Datadog Bits AI cites sources. New Relic AI attributes to specific metrics.                              | MEDIUM     | Context builder includes labeled data sections. System prompt instructs citing specific values. Frontend could render referenced values distinctly (bold or highlighted).                                                  |
| **Procurement-oriented framing**           | Most infrastructure AI assistants optimize for troubleshooting. This advisor interprets capacity data for 6-12 month hardware procurement decisions -- a unique angle. No competitor does this for TAS.                                                                             | LOW        | Primarily system prompt engineering. Teach the LLM about procurement lead times, budget cycles, growth planning, headroom targets. Low implementation cost, high differentiation.                                          |
| **Contextual starter prompts**             | Instead of static suggestions, generate prompts based on current data state: "You have 3 cells at >90% memory -- want to explore expansion options?" when hot spots exist, vs. "Your capacity looks healthy -- want to plan for next quarter?" when things are green.               | MEDIUM     | Requires logic to inspect current data and select relevant prompt sets. More valuable than static prompts but more complex.                                                                                                |
| **Copy response to clipboard**             | Operators need to paste advisor analysis into procurement requests, Slack messages, or tickets. One-click copy of a response (formatted or plain text).                                                                                                                             | LOW        | Standard clipboard API. Include copy button on each assistant message. High utility, trivial to build.                                                                                                                     |
| **Response feedback (thumbs up/down)**     | Captures quality signal for future improvement. Microsoft Copilot Studio, ChatGPT, and Grafana all include this. Even without a backend feedback store in Phase 1, logging feedback events enables later analysis.                                                                  | LOW        | Two buttons per response. Log to `slog` in Phase 1. Can add persistence later. Low cost, enables iteration.                                                                                                                |
| **Keyboard shortcut to toggle panel**      | Power users (SREs, platform engineers) prefer keyboard navigation. Cmd/Ctrl+K or similar to open advisor and focus input.                                                                                                                                                           | LOW        | Standard `useEffect` keydown listener.                                                                                                                                                                                     |
| **Conversation token/length awareness**    | When conversation gets long, proactively warn the user that context may be getting truncated or suggest starting fresh. Prevents degraded responses without explanation.                                                                                                            | LOW        | Track approximate token count client-side. Show subtle indicator when approaching limits.                                                                                                                                  |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems, especially for a Phase 1 read-only advisor.

| Feature                                      | Why Requested                                                                                                                                           | Why Problematic                                                                                                                                                                                                                                                                   | Alternative                                                                                                                                                                         |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Tool use / action execution**              | Users will want the advisor to "run that scenario" or "add 4 cells and show me the result." Natural desire for the advisor to do things, not just talk. | Phase 1 is explicitly read-only. Adding tool use requires safety controls, action confirmation UI, error recovery for failed actions, and testing of every tool path. Premature tool use risks dangerous mutations (changing infrastructure state) and massively increases scope. | Advisor can describe what the user should do ("Try adding 4 cells in the scenario wizard with 32GB RAM each") with specific guidance. Phase 2 adds tool use with proper guardrails. |
| **Conversation persistence across sessions** | Users expect to resume where they left off.                                                                                                             | Requires a persistence layer (database or file), session-to-conversation mapping, conversation list UI, and decisions about TTL/cleanup. Infrastructure state changes between sessions make old conversations potentially misleading.                                             | Each session starts fresh with current data. The advisor's value is interpreting _current_ state, not historical conversations. Phase 3 if validated.                               |
| **Per-user API keys (BYOK)**                 | Power users want to use their own API keys to avoid rate limits or use preferred models.                                                                | Adds key management UI, localStorage security concerns, key validation flows, and per-user billing complexity. Splits the user experience (some users get better models).                                                                                                         | System key with rate limiting. Revisit if rate limits become a real user pain point.                                                                                                |
| **Multi-model selection**                    | Users want to pick between Claude, GPT-4, Gemini, etc.                                                                                                  | Each model has different context windows, capabilities, pricing, and failure modes. Testing and prompt engineering multiply per model. Provider abstraction interface exists for Phase 2, but exposing model selection in Phase 1 creates support burden.                         | Ship with Claude only behind the provider interface. Add providers in Phase 2 when there's user demand.                                                                             |
| **Voice input**                              | "Talk to the advisor" feels futuristic and natural.                                                                                                     | Voice processing adds speech-to-text dependency, microphone permissions, noise handling, and a completely different input modality to test. Operators in data centers may not have quiet environments.                                                                            | Text input only. Voice is a Phase 3+ exploration if there's demand.                                                                                                                 |
| **Real-time data push to advisor**           | Auto-refresh advisor context whenever dashboard data changes in real time.                                                                              | Creates race conditions (advisor responding about stale context while new data arrives), unnecessary LLM API costs (re-sending context on every refresh), and confusing UX (advisor's answer references data that just changed).                                                  | Update context when user sends a message (pull model). Advisor always sees current data at query time, not between queries.                                                         |
| **Chart/visualization generation**           | Advisor generates inline charts to illustrate capacity projections.                                                                                     | Requires a charting library integration within the chat panel, LLM-to-chart-spec translation, and handling of malformed chart specs. The dashboard already has charts.                                                                                                            | Advisor references existing dashboard visualizations ("look at the memory utilization chart in the dashboard") and describes data in tables within Markdown.                        |
| **Autonomous alerts/proactive messages**     | Advisor proactively warns about capacity issues without being asked.                                                                                    | Push notifications from an AI feel intrusive. Requires background processing, notification system, and decisions about frequency/relevance thresholds. The existing dashboard already surfaces bottlenecks and recommendations.                                                   | Starter prompts surface the most urgent issue. The advisor is reactive (user-initiated) for Phase 1.                                                                                |

## Feature Dependencies

```
[Streaming token display]
    requires [SSE chat endpoint]
        requires [LLM provider abstraction]
            requires [Anthropic provider implementation]

[Context awareness]
    requires [Context builder]
        requires [Access to Handler state / cache]

[Starter prompts]
    enhances [Context awareness] (contextual prompts need data inspection)

[Markdown rendering]
    requires [Streaming token display] (must render incrementally as tokens arrive)

[Graceful degradation messaging]
    requires [Context builder] (must know which data sources are available)

[Domain-expert system prompt]
    enhances [Context awareness] (prompt interprets the context data)

[Data-grounded responses]
    requires [Context awareness] (needs labeled data sections to cite)
    enhances [Domain-expert system prompt] (prompt instructs citation behavior)

[Response feedback]
    independent (can be added at any time)

[Copy response]
    independent (can be added at any time)

[Procurement-oriented framing]
    enhances [Domain-expert system prompt] (encoded in prompt, no code dependency)
```

### Dependency Notes

- **Streaming requires the full backend chain:** SSE endpoint, provider abstraction, and Anthropic implementation must all be in place before any streaming works in the UI. This is the critical path.
- **Context awareness is the highest-risk dependency:** The context builder is the bridge between existing infrastructure state and the LLM. Getting the serialization right (what to include, how to structure it, what to omit) determines whether the advisor is useful or generic.
- **Starter prompts have an optional enhancement:** Static prompts work without data inspection, but contextual prompts (the differentiator) require inspecting current state. Ship static first, upgrade to contextual.
- **Markdown rendering during streaming is non-trivial:** Partial Markdown (e.g., half a code block) can cause rendering glitches. Buffer strategy needed.
- **Independent features (feedback, copy) can be added at any point** without blocking the critical path.

## MVP Definition

### Launch With (v1 -- Phase 1)

Minimum viable advisor -- what's needed to validate the concept that an AI chat panel adds value to the capacity planning workflow.

- [ ] **SSE chat endpoint with Anthropic provider** -- the plumbing that makes everything else possible
- [ ] **Context builder reading Handler state** -- without this, the advisor is just a generic chatbot
- [ ] **Domain-expert system prompt** -- the encoded TAS/Diego capacity planning knowledge
- [ ] **Streaming token display** -- table stakes UX
- [ ] **Markdown rendering** -- table stakes UX
- [ ] **Side panel with open/close toggle** -- the container for the advisor
- [ ] **Static starter prompts** -- onboarding affordance
- [ ] **Conversation threading (session-scoped)** -- multi-turn dialogue
- [ ] **Clear/reset conversation** -- basic conversation management
- [ ] **Loading indicator** -- feedback during processing
- [ ] **Error handling with retry** -- resilience
- [ ] **Graceful degradation messaging** -- works with CF-only data
- [ ] **Rate limiting on chat endpoint** -- operational safety (already scoped at 10 req/min)
- [ ] **Feature gating via `AI_PROVIDER` env var** -- advisor only appears when configured

### Add After Validation (v1.x)

Features to add once the core advisor is working and operators confirm it's useful.

- [ ] **Contextual starter prompts** -- upgrade from static; triggered by data state
- [ ] **Data-grounded responses with citations** -- improved trust and accuracy
- [ ] **Copy response to clipboard** -- workflow integration
- [ ] **Response feedback (thumbs up/down)** -- quality signal collection
- [ ] **Keyboard shortcut for panel toggle** -- power user efficiency
- [ ] **Procurement-oriented framing in system prompt** -- sharpen the unique angle
- [ ] **Token/length awareness indicator** -- prevent degraded long conversations

### Future Consideration (v2+)

Features to defer until the advisor concept is validated and Phase 2+ planning begins.

- [ ] **Tool use / scenario execution via chat** -- Phase 2; requires action confirmation UI and safety controls
- [ ] **Additional LLM providers (OpenAI, etc.)** -- Phase 2; provider interface already exists
- [ ] **Conversation persistence** -- Phase 3; requires storage layer
- [ ] **Live UI sync (advisor actions reflected in dashboard)** -- Phase 3
- [ ] **Per-user API keys (BYOK)** -- Phase 2+ if rate limits are a real problem
- [ ] **Push-content panel layout** -- Phase 3; overlay is sufficient for validation

## Feature Prioritization Matrix

| Feature                                | User Value | Implementation Cost | Priority |
| -------------------------------------- | ---------- | ------------------- | -------- |
| SSE chat endpoint + Anthropic provider | HIGH       | HIGH                | P1       |
| Context builder (Handler state)        | HIGH       | HIGH                | P1       |
| Domain-expert system prompt            | HIGH       | MEDIUM              | P1       |
| Streaming token display                | HIGH       | MEDIUM              | P1       |
| Markdown rendering                     | HIGH       | LOW                 | P1       |
| Side panel with toggle                 | HIGH       | MEDIUM              | P1       |
| Static starter prompts                 | MEDIUM     | LOW                 | P1       |
| Conversation threading                 | HIGH       | LOW                 | P1       |
| Clear/reset conversation               | MEDIUM     | LOW                 | P1       |
| Loading indicator                      | MEDIUM     | LOW                 | P1       |
| Error handling with retry              | HIGH       | LOW                 | P1       |
| Graceful degradation messaging         | HIGH       | MEDIUM              | P1       |
| Rate limiting (chat endpoint)          | HIGH       | LOW                 | P1       |
| Feature gating (`AI_PROVIDER`)         | MEDIUM     | LOW                 | P1       |
| Contextual starter prompts             | MEDIUM     | MEDIUM              | P2       |
| Data-grounded citations                | MEDIUM     | MEDIUM              | P2       |
| Copy response to clipboard             | MEDIUM     | LOW                 | P2       |
| Response feedback                      | LOW        | LOW                 | P2       |
| Keyboard shortcut                      | LOW        | LOW                 | P2       |
| Procurement framing (prompt)           | MEDIUM     | LOW                 | P2       |
| Token/length awareness                 | LOW        | LOW                 | P2       |
| Tool use / action execution            | HIGH       | HIGH                | P3       |
| Additional LLM providers               | MEDIUM     | MEDIUM              | P3       |
| Conversation persistence               | MEDIUM     | HIGH                | P3       |

**Priority key:**

- P1: Must have for launch (Phase 1 MVP)
- P2: Should have, add after core validation
- P3: Future phases, defer until concept is proven

## Competitor Feature Analysis

| Feature                   | Datadog Bits AI                         | New Relic AI                          | Grafana Assistant              | Dynatrace Davis CoPilot         | Our Approach (Phase 1)                                                     |
| ------------------------- | --------------------------------------- | ------------------------------------- | ------------------------------ | ------------------------------- | -------------------------------------------------------------------------- |
| Natural language querying | Full telemetry query (logs, APM, infra) | NRQL generation from natural language | DQL generation from prompts    | DQL from natural language       | Read-only interpretation of pre-fetched capacity data; no query generation |
| Streaming responses       | Yes                                     | Yes                                   | Yes                            | Yes                             | Yes -- SSE with token-by-token display                                     |
| Context awareness         | Full Datadog environment context        | Full New Relic telemetry context      | Full Grafana Cloud data access | Full Dynatrace topology context | Dashboard + infrastructure + scenario state from Handler cache             |
| Tool use / actions        | Yes (restart services, reboot, etc.)    | Limited (run queries)                 | Multi-step investigations      | Root cause remediation          | No -- read-only Phase 1; Phase 2 adds tool use                             |
| Domain specialization     | SRE / incident response                 | Observability / troubleshooting       | Observability / dashboarding   | AIOps / root cause analysis     | TAS/Diego capacity planning + procurement guidance                         |
| Starter prompts           | Yes                                     | "Ask AI" contextual button            | Yes                            | Yes                             | Yes -- static in v1, contextual in v1.x                                    |
| Response citations        | Rich widgets with data links            | Dashboard references                  | Query results embedded         | Topology references             | Labeled data values in v1.x                                                |
| Feedback mechanism        | Implicit (usage analytics)              | Thumbs up/down                        | Usage analytics                | Analytics dashboard             | Thumbs up/down logging in v1.x                                             |
| Graceful degradation      | N/A (full platform)                     | N/A (full platform)                   | N/A (full platform)            | N/A (full platform)             | Explicit -- works with CF-only, flags missing BOSH/vSphere                 |
| Pricing model             | Add-on to Datadog plans                 | Advanced Compute pricing              | Included in Cloud              | Platform license                | System API key; no per-user cost in Phase 1                                |

**Key competitive insight:** All four competitors are full-platform observability tools with AI assistants bolted on. Our advisor is purpose-built for a specific workflow (capacity planning for procurement) in a specific ecosystem (TAS/Diego). The competitors are broad; we go deep on one problem. This domain focus is the primary differentiator.

## Sources

- [Datadog Bits AI SRE](https://www.datadoghq.com/product/ai/bits-ai-sre/) -- Features and capabilities of Datadog's embedded AI assistant
- [Datadog Bits AI Documentation](https://docs.datadoghq.com/bits_ai/chat_with_bits_ai/) -- Chat interface documentation
- [New Relic AI Documentation](https://docs.newrelic.com/docs/agentic-ai/new-relic-ai/) -- NRAI capabilities and interaction methods
- [Now GA: New Relic AI](https://newrelic.com/blog/ai/nrai-agentic-ga) -- GA announcement with feature details
- [Grafana Assistant Introduction](https://grafana.com/blog/2025/05/07/llm-grafana-assistant/) -- Context-aware LLM agent in Grafana Cloud
- [Grafana AI Tools for Observability](https://grafana.com/products/cloud/ai-tools-for-observability/) -- AI-driven capacity planning features
- [Dynatrace Davis CoPilot GA](https://www.dynatrace.com/news/blog/announcing-general-availability-of-davis-copilot-your-new-ai-assistant/) -- CoPilot features and natural language querying
- [Dynatrace Davis AI Dashboarding](https://www.dynatrace.com/news/blog/better-dashboarding-with-dynatrace-davis-ai/) -- Dashboard integration patterns
- [AI Chat Interfaces in Enterprise Decision Platforms: 2026 Trends](https://lumitech.co/insights/ai-chat-interfaces-in-enterprise-decision-platforms) -- Industry trends for embedded AI chat
- [Where Should AI Sit in Your UI?](https://uxdesign.cc/where-should-ai-sit-in-your-ui-1710a258390e) -- UI placement patterns for embedded AI (side panel, inline, overlay)
- [Rethinking LLM Interfaces](https://ericmjl.github.io/blog/2025/6/14/rethinking-llm-interfaces-from-chatbots-to-contextual-applications/) -- Beyond chatbots to contextual applications
- [Integrating Markdown in Streaming Chat](https://athrael.net/blog/building-an-ai-chat-assistant/add-markdown-to-streaming-chat) -- Markdown rendering during SSE streaming
- [Microsoft Copilot Studio Feedback](https://learn.microsoft.com/en-us/power-platform/release-plan/2025wave1/microsoft-copilot-studio/collect-thumbs-up-or-down-feedback-comments-agents) -- Thumbs up/down feedback patterns
- [Vercel Streamdown](https://vercel.com/changelog/introducing-streamdown) -- Open source Markdown renderer for AI streaming
- [Designing for Agentic AI UX Patterns](https://www.smashingmagazine.com/2026/02/designing-agentic-ai-practical-ux-patterns/) -- Autonomy controls and phased rollout
- [AI Copilot Product Trends Q1 2026](https://www.harshal-patil.com/post/ai-copilot-product-trends-2026q1) -- Current copilot design trends

---

_Feature research for: AI-embedded conversational advisor in operational capacity planning dashboard_
_Researched: 2026-02-24_
