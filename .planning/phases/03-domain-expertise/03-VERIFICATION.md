---
phase: 03-domain-expertise
verified: 2026-02-24T14:15:00Z
status: passed
score: 5/5 must-haves verified
re_verification: false
---

# Phase 3: Domain Expertise Verification Report

**Phase Goal:** System prompt turns the LLM from a generic chatbot into a TAS/Diego capacity planning domain expert that reasons about procurement decisions using the operator's live data
**Verified:** 2026-02-24T14:15:00Z
**Status:** passed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| #   | Truth                                                                                                                                                                        | Status   | Evidence                                                                                                                                          |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | System prompt contains TAS/Diego capacity planning heuristics (N-1, HA Admission Control, vCPU:pCPU ratios, cell sizing, isolation segments, Diego auction, Small Footprint) | VERIFIED | All terms present in `systemPrompt` const; `TestSystemPromptContainsHeuristics` passes                                                            |
| 2   | System prompt frames findings in procurement terms (lead times, budget cycles, growth planning, concrete hardware quantities)                                                | VERIFIED | `<procurement_framing>` section present; "lead times", "budget", "procurement" all confirmed; `TestSystemPromptContainsProcurementFraming` passes |
| 3   | System prompt teaches the LLM to acknowledge material data gaps using exact BuildContext markers (NOT CONFIGURED, UNAVAILABLE, No scenario comparison)                       | VERIFIED | All three exact markers present in `<data_gap_handling>` section; `TestSystemPromptContainsGapHandling` passes                                    |
| 4   | System prompt instructs the LLM to cite specific data values from infrastructure context when making claims                                                                  | VERIFIED | `<response_rules>` instructs "Cite specific numbers from the infrastructure context"; `TestSystemPromptContainsEvidenceRequirement` passes        |
| 5   | Combined system prompt (static + BuildContext) stays within token budget (static under ~2500 tokens, combined under ~3500 tokens)                                            | VERIFIED | Static prompt is 6091 chars (~1522 tokens), well under 10000-char/2500-token budget; `TestSystemPromptTokenBudget` passes                         |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact                             | Expected                                                                                       | Status   | Details                                                                                                                                                                                                                                                                                                                                                            |
| ------------------------------------ | ---------------------------------------------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `backend/services/ai/prompt.go`      | Static `systemPrompt` const and `BuildSystemPrompt` composition function                       | VERIFIED | File exists, 99 lines, contains `const systemPrompt` (6091 chars) and `func BuildSystemPrompt`, substantive XML-structured content across all 4 required sections                                                                                                                                                                                                  |
| `backend/services/ai/prompt_test.go` | Content-assertion tests validating all four DOM requirements plus token budget and composition | VERIFIED | File exists, 115 lines, 8 test functions present (TestSystemPromptContainsDomainKnowledge, TestSystemPromptContainsHeuristics, TestSystemPromptContainsProcurementFraming, TestSystemPromptContainsGapHandling, TestSystemPromptContainsEvidenceRequirement, TestSystemPromptTokenBudget, TestBuildSystemPromptIncludesContext, TestBuildSystemPromptEmptyContext) |

### Key Link Verification

| From                            | To                               | Via                                                                                    | Status               | Details                                                                                                                                                                                                                                                               |
| ------------------------------- | -------------------------------- | -------------------------------------------------------------------------------------- | -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `backend/services/ai/prompt.go` | `backend/services/ai/context.go` | `BuildSystemPrompt` wraps `BuildContext` output in `<infrastructure_context>` XML tags | WIRED                | `BuildSystemPrompt` appends `\n\n<infrastructure_context>\n` + context + `\n</infrastructure_context>` -- directly consumes `BuildContext` output string; `TestBuildSystemPromptIncludesContext` and `TestBuildSystemPromptEmptyContext` verify the wrapping behavior |
| `backend/services/ai/prompt.go` | `backend/services/ai/options.go` | Phase 4 will pass `BuildSystemPrompt` result to `WithSystem` option                    | DEFERRED (by design) | `WithSystem` function confirmed present in `options.go`. The PLAN explicitly states this link is Phase 4's responsibility. `BuildSystemPrompt` is exported and ready; not yet called in production code outside tests, which is correct at this phase boundary.       |

**Note on deferred link:** `BuildSystemPrompt` is only referenced in `prompt_test.go` in production code. This is expected -- Phase 4 (Chat Endpoint) is the phase that wires `BuildSystemPrompt` into request handling via `WithSystem`. The Phase 3 deliverable is to make `BuildSystemPrompt` available and correct, which it is.

### Requirements Coverage

| Requirement | Source Plan   | Description                                                                                                                                                              | Status    | Evidence                                                                                                                                                                                                                                                                                                             |
| ----------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| DOM-01      | 03-01-PLAN.md | System prompt encodes TAS/Diego capacity planning knowledge: N-1 redundancy, HA Admission Control, vCPU:pCPU ratios, cell sizing heuristics, isolation segment tradeoffs | SATISFIED | `<domain_knowledge>` section contains: N-1 redundancy section, HA Admission Control with 25-33% figures, vCPU:pCPU ratios (4:1 safe, 8:1 risky), cell sizing (32-64GB), isolation segment minimum (4 cells), Diego auction mechanics, Small Footprint TAS. `TestSystemPromptContainsHeuristics` validates all terms. |
| DOM-02      | 03-01-PLAN.md | System prompt frames analysis in procurement terms: lead times, budget cycles, growth planning, headroom targets                                                         | SATISFIED | `<procurement_framing>` section states "8-12 weeks from order to rack-ready", quarterly/annual budget cycles, 6-12 month growth horizons, concrete quantity recommendations. `TestSystemPromptContainsProcurementFraming` validates.                                                                                 |
| DOM-03      | 03-01-PLAN.md | System prompt instructs the LLM to acknowledge data gaps rather than hallucinate when information is missing                                                             | SATISFIED | `<data_gap_handling>` section contains all three exact BuildContext markers ("NOT CONFIGURED", "UNAVAILABLE", "No scenario comparison has been run"), materiality rules, and prohibition on inventing data. `TestSystemPromptContainsGapHandling` validates.                                                         |
| DOM-04      | 03-01-PLAN.md | System prompt instructs the LLM to reference specific data values from context when making claims                                                                        | SATISFIED | `<response_rules>` explicitly states "Cite specific numbers from the infrastructure context (cell counts, utilization percentages, memory values, host counts)". `TestSystemPromptContainsEvidenceRequirement` validates.                                                                                            |

No orphaned requirements -- all four DOM-01 through DOM-04 requirements are claimed by 03-01-PLAN.md and verified above.

### Anti-Patterns Found

| File   | Line | Pattern | Severity | Impact |
| ------ | ---- | ------- | -------- | ------ |
| (none) | --   | --      | --       | --     |

No TODO, FIXME, placeholder, empty return, or stub patterns found in `prompt.go` or `prompt_test.go`.

### Human Verification Required

#### 1. LLM Does Not Hallucinate With Incomplete Context

**Test:** Send a chat message asking about cell sizing in an environment where vSphere shows "NOT CONFIGURED". Confirm the LLM acknowledges the gap rather than inventing host specs.
**Expected:** Response mentions that physical host constraints cannot be evaluated because vSphere is not configured.
**Why human:** The system prompt's data gap handling instructions can only be validated by observing actual LLM behavior, not by static string checks.

#### 2. LLM Cites Actual Context Values

**Test:** Send a chat message against a context with known values (e.g., "6 cells, 80% utilization"). Confirm the response references "6 cells" and "80%" rather than generic figures.
**Expected:** LLM response contains the specific numbers from the provided BuildContext output.
**Why human:** Instruction-following quality requires runtime observation of LLM behavior.

#### 3. Procurement Framing Tone

**Test:** Ask "do I need more capacity?" and confirm the response reads like senior engineer capacity review notes -- direct, data-driven, no conversational preamble.
**Expected:** Response states finding, cites numbers, recommends action. No "Great question!" or "I'd be happy to help."
**Why human:** Tone quality is subjective and requires a human to judge.

### Gaps Summary

No gaps. All five must-have truths are verified, both artifacts are substantive and correctly implemented, the key link within Phase 3's scope is wired, and all four DOM requirements are satisfied with implementation evidence. The full ai package test suite (36 tests) passes with no failures.

---

_Verified: 2026-02-24T14:15:00Z_
_Verifier: Claude (gsd-verifier)_
