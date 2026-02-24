---
status: complete
phase: 03-domain-expertise
source: 03-01-SUMMARY.md
started: 2026-02-24T20:30:00Z
updated: 2026-02-24T20:35:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Domain Knowledge Coverage

expected: System prompt contains TAS/Diego capacity planning heuristics: N-1 redundancy, HA Admission Control (25-33%), vCPU:pCPU ratios (4:1/8:1 thresholds), cell sizing (32-64 GB), utilization targets (80%/90%), free chunks/placement, isolation segments (min 4 cells), Diego auction mechanics, and Small Footprint TAS.
result: pass

### 2. Procurement Framing

expected: System prompt frames findings in procurement terms: 8-12 week hardware lead times, quarterly/annual budget cycles, 6-12 month growth projections, concrete quantity recommendations ("N additional hosts at X GB"), and urgency tied to utilization thresholds.
result: pass

### 3. Data Gap Handling

expected: System prompt instructs the LLM to handle missing data using BuildContext markers (NOT CONFIGURED, UNAVAILABLE, "No scenario comparison has been run"). Material gaps are acknowledged with what's missing, what can't be analyzed, and what conclusions remain possible. Immaterial gaps are not mentioned. Data is never invented.
result: pass

### 4. Evidence-Based Responses

expected: System prompt requires the LLM to cite specific numbers from infrastructure context (cell counts, utilization percentages, memory values, host counts) and follow a finding + evidence + recommendation structure. Responses should be 2-4 paragraphs, use tables for comparisons, and prioritize [HIGH]/[CRITICAL] flags.
result: pass

### 5. BuildSystemPrompt Composition

expected: Running `go test ./services/ai/ -run TestBuildSystemPrompt -v` from backend/ passes. BuildSystemPrompt wraps any context string in `<infrastructure_context>` XML tags and prepends the static system prompt. Empty context produces valid output with empty infrastructure_context tags.
result: pass

### 6. Full Test Suite Regression

expected: Running `go test ./services/ai/ -v` from backend/ passes all 36 tests (28 existing from Phases 1-2 plus 8 new prompt tests) with no failures.
result: pass

## Summary

total: 6
passed: 6
issues: 0
pending: 0
skipped: 0

## Gaps

[none]
