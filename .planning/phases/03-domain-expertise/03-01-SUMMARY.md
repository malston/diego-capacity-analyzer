---
phase: 03-domain-expertise
plan: 01
subsystem: ai
tags:
  [
    system-prompt,
    domain-knowledge,
    prompt-engineering,
    xml-tags,
    capacity-planning,
  ]

requires:
  - phase: 02-context-builder
    provides: BuildContext function and ContextInput type for live data serialization
provides:
  - Static system prompt const encoding TAS/Diego capacity planning domain expertise
  - BuildSystemPrompt composition function wrapping BuildContext output in infrastructure_context XML tags
affects: [04-chat-endpoint, 07-graceful-degradation, 08-polish]

tech-stack:
  added: []
  patterns:
    [xml-structured-prompt, const-plus-composition, content-assertion-tests]

key-files:
  created:
    - backend/services/ai/prompt.go
    - backend/services/ai/prompt_test.go
  modified: []

key-decisions:
  - "Prompt uses XML tags for section delineation per Anthropic best practices (domain_knowledge, procurement_framing, response_rules, data_gap_handling)"
  - "Measured instruction language per Claude 4.6 guidance -- no excessive MUST/ALWAYS/NEVER emphasis"
  - "Materiality-based gap handling with general rules plus examples (not exhaustive per-marker if/then rules)"
  - "Static prompt is ~3900 chars (~975 tokens), well under 10000-char budget"

patterns-established:
  - "XML-structured system prompt: domain_knowledge, procurement_framing, response_rules, data_gap_handling sections"
  - "Composition pattern: BuildSystemPrompt wraps BuildContext output in infrastructure_context XML tags"
  - "Content-assertion testing: string-contains checks validating prompt content requirements"

requirements-completed: [DOM-01, DOM-02, DOM-03, DOM-04]

duration: 2min
completed: 2026-02-24
---

# Phase 3 Plan 1: System Prompt with Domain Expertise Summary

**XML-structured system prompt encoding TAS/Diego capacity planning heuristics, procurement framing, materiality-based gap handling, and BuildSystemPrompt composition function**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-24T21:06:13Z
- **Completed:** 2026-02-24T21:08:37Z
- **Tasks:** 2 (TDD: RED + GREEN)
- **Files modified:** 2

## Accomplishments

- System prompt const with 4 XML-structured sections covering all DOM requirements: domain knowledge (N-1, HA Admission Control, vCPU:pCPU ratios, cell sizing, utilization targets, free chunks, isolation segments, Diego auction, Small Footprint TAS), procurement framing (lead times, budget cycles, concrete quantities), response rules (finding + evidence + recommendation, concise tone), and data gap handling (materiality-based, keyed to BuildContext markers)
- BuildSystemPrompt composition function that wraps BuildContext output in infrastructure_context XML tags for per-request injection
- 8 content-assertion tests validating all 4 DOM requirements plus token budget and composition behavior
- Full ai package suite: 36 tests passing (28 existing + 8 new)

## Task Commits

Each task was committed atomically:

1. **Task 1: Write failing tests for prompt content and composition** - `d8854d2` (test)
2. **Task 2: Implement system prompt const and BuildSystemPrompt function** - `5c679d9` (feat)

## Files Created/Modified

- `backend/services/ai/prompt.go` - Static systemPrompt const with XML-structured domain expertise, BuildSystemPrompt composition function
- `backend/services/ai/prompt_test.go` - 8 content-assertion tests: domain knowledge sections, heuristics, procurement framing, gap handling markers, evidence requirement, token budget, context inclusion, empty context

## Decisions Made

- Prompt uses XML tags for section delineation per Anthropic best practices. Tags: domain_knowledge, procurement_framing, response_rules, data_gap_handling, infrastructure_context (added by BuildSystemPrompt).
- Used measured instruction language per Claude 4.6 guidance. "Cite specific numbers" rather than "you MUST ALWAYS cite specific numbers."
- Materiality-based gap handling uses general rules plus 2 concrete examples (cell sizing with missing vSphere = material; app memory with missing vSphere = not material). This fits the token budget better than exhaustive per-marker rules and Claude 4.6's instruction-following is reliable.
- Static prompt is approximately 3900 characters (~975 tokens), well under the 10000-character budget. Combined with BuildContext output (~1000 tokens max), total system prompt stays under 2000 tokens.

## Deviations from Plan

None -- plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None -- no external service configuration required.

## Next Phase Readiness

- BuildSystemPrompt is ready for integration with the chat endpoint (Phase 4)
- Phase 4 will call: `ctx := ai.BuildContext(input)` then `sysPrompt := ai.BuildSystemPrompt(ctx)` then pass via `ai.WithSystem(sysPrompt)`
- No blockers for subsequent phases

## Self-Check: PASSED

- FOUND: backend/services/ai/prompt.go
- FOUND: backend/services/ai/prompt_test.go
- FOUND: commit d8854d2
- FOUND: commit 5c679d9
- FOUND: 03-01-SUMMARY.md

---

_Phase: 03-domain-expertise_
_Completed: 2026-02-24_
