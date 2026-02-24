# Phase 3: Domain Expertise - Research

**Researched:** 2026-02-24
**Domain:** LLM system prompt engineering for TAS/Diego capacity planning domain expertise
**Confidence:** HIGH

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions

- Platform architect level expertise: N-1 redundancy, HA Admission Control, vCPU:pCPU ratios, cell sizing heuristics, isolation segment tradeoffs, AZ placement, Diego auction mechanics, Garden container lifecycle
- Encode all standard VMware/TAS heuristics: N-1 host failure tolerance, HA Admission Control at 25-33%, vCPU:pCPU ratio ceilings (4:1 safe, 8:1 risky), cell memory sizing (32-64GB typical), 80% utilization target, isolation segment overhead
- Cover both standard TAS (dedicated diego_cell VMs) and Small Footprint TAS (colocated Diego on compute VMs) -- the dashboard already detects both
- Stay at the logical capacity layer for vSphere: host memory, CPU, and HA. Do not encode DRS rules, resource pools, or storage I/O knowledge since the dashboard doesn't provide that data
- Primary audience: platform engineers making procurement cases to management. They understand TAS but need data-backed arguments for capacity review documents
- Proactively recommend actions when data shows clear issues (high utilization, N-1 violations, ratio overcommit) -- don't wait to be asked
- Frame procurement recommendations around real-world realities: lead times (8-12 week typical), budget cycles, growth planning horizons
- No persona or name -- just "the advisor." Professional and tool-like, reads like a senior engineer's capacity review notes
- Acknowledge material gaps only -- caveat when missing data is relevant to the question being asked
- If someone asks about cell sizing and vSphere is NOT CONFIGURED, that's material -- acknowledge it
- If someone asks about app memory and vSphere is missing, that's not material -- skip the caveat
- Never hallucinate data to fill gaps; state what's unknown and what can still be concluded
- Consistent format: finding + evidence + recommendation
- State the finding, cite specific numbers from the provided context, then recommend action
- Concise: 2-4 paragraphs typical, use tables for comparisons
- Always reference actual data values from context when making claims

### Claude's Discretion

- Exact system prompt length and token budget allocation
- How to organize domain knowledge sections within the prompt (topical vs priority-ordered)
- Whether to use few-shot examples in the system prompt or rely on instruction-following
- Specific wording of data gap acknowledgment patterns

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>

## Phase Requirements

| ID     | Description                                                                                                                                                              | Research Support                                                                                                                                                                                                |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| DOM-01 | System prompt encodes TAS/Diego capacity planning knowledge: N-1 redundancy, HA Admission Control, vCPU:pCPU ratios, cell sizing heuristics, isolation segment tradeoffs | Domain knowledge section with heuristic thresholds mapped to BuildContext output; [HIGH]/[CRITICAL] flags from `utilizationFlag()` and `vcpuRatioFlag()` already in context; prompt teaches interpretation      |
| DOM-02 | System prompt frames analysis in procurement terms: lead times, budget cycles, growth planning, headroom targets                                                         | Procurement framing section with concrete lead times (8-12 weeks), budget cycle awareness, growth horizon planning, and instruction to express needs as specific hardware quantities                            |
| DOM-03 | System prompt instructs LLM to acknowledge data gaps rather than hallucinate when information is missing                                                                 | Data gap handling section keyed to exact BuildContext markers: "NOT CONFIGURED", "UNAVAILABLE", "No scenario comparison has been run"; materiality rules with section-specific examples                         |
| DOM-04 | System prompt instructs LLM to reference specific data values from context when making claims                                                                            | Response structure rules requiring finding + evidence + recommendation; explicit instruction to cite cell counts, utilization percentages, and memory values from `<infrastructure_context>` when making claims |

</phase_requirements>

## Summary

Phase 3 produces a single Go string constant containing the system prompt that transforms a generic Claude model into a TAS/Diego capacity planning advisor. This phase is pure prompt engineering with a lightweight Go implementation -- no new APIs, services, or complex architecture. The system prompt is passed per-request via the existing `WithSystem` option established in Phase 1, concatenated at call time with the BuildContext output from Phase 2.

The primary technical challenge is **prompt organization** -- encoding enough domain expertise to be useful while staying within a reasonable token budget. The static prompt should be roughly 1500-2500 tokens (~6000-10000 characters). Combined with ~1000 tokens from BuildContext (verified against the existing `TestBuildContext_TokenBudget` test which caps output at 5000 chars), the total system prompt stays under 3500 tokens, leaving 196K+ tokens for conversation history in Claude's 200K context window.

The secondary challenge is **data gap handling**. The prompt must teach the LLM to distinguish material gaps from immaterial ones based on the specific question being asked. BuildContext emits five distinct markers across four sections (Data Sources, Infrastructure, Diego Cells, Apps, Scenario Comparison). The prompt must map each marker to materiality rules.

**Primary recommendation:** Implement the system prompt as a Go `const` in a dedicated file (`backend/services/ai/prompt.go`), organized using XML tags per Anthropic's best practices. Use topical section ordering (domain knowledge first, then procurement framing, then response rules, then gap handling, then infrastructure context last). Skip few-shot examples -- Claude 4.6's instruction-following is reliable for structured advisory responses, and the token budget is better spent on domain knowledge. Use measured language in instructions -- Claude 4.6 is more proactive by default and overly aggressive "you MUST" phrasing may cause over-triggering.

## Standard Stack

### Core

| Library             | Version    | Purpose                                          | Why Standard                                                               |
| ------------------- | ---------- | ------------------------------------------------ | -------------------------------------------------------------------------- |
| Go standard library | 1.23+      | String constant definition, string concatenation | System prompt is a const; no templating needed                             |
| `ai.WithSystem()`   | (existing) | Per-request system prompt injection              | Phase 1 established this pattern; system prompt passed as option to Chat() |

### Supporting

| Library             | Version    | Purpose                     | When to Use                                             |
| ------------------- | ---------- | --------------------------- | ------------------------------------------------------- |
| `ai.BuildContext()` | (existing) | Generates live data context | Called at request time, concatenated with system prompt |

### Alternatives Considered

| Instead of      | Could Use                      | Tradeoff                                                                                                          |
| --------------- | ------------------------------ | ----------------------------------------------------------------------------------------------------------------- |
| Go const string | Template file (embed)          | Templates add runtime complexity; const is simpler and the prompt has no dynamic sections outside of BuildContext |
| Go const string | `text/template` with variables | Over-engineering; the system prompt is static domain knowledge, not parameterized                                 |

**Installation:**
No new dependencies. This phase uses only existing code.

## Architecture Patterns

### Recommended File Structure

```
backend/services/ai/
├── prompt.go          # System prompt const + BuildSystemPrompt() function
├── prompt_test.go     # Tests for prompt composition and content validation
├── provider.go        # ChatProvider interface (existing)
├── options.go         # WithSystem, etc. (existing)
├── anthropic.go       # Anthropic implementation (existing)
├── context.go         # BuildContext (existing)
└── context_test.go    # Context tests (existing)
```

### Pattern 1: System Prompt as Go Const with Composition Function

**What:** Define the domain knowledge as a `const` string. Provide a `BuildSystemPrompt(contextMarkdown string) string` function that concatenates the static prompt with the dynamic BuildContext output.
**When to use:** Always -- this is the only pattern needed for this phase.
**Why:** The system prompt has two parts: (1) static domain expertise/instructions that never change, and (2) dynamic infrastructure context that changes per request. Separating them enables testing each independently. The composition function wraps the context in an `<infrastructure_context>` XML tag so Claude can parse the boundary between instructions and data.

```go
// prompt.go
package ai

const systemPrompt = `...static domain knowledge and rules...`

// BuildSystemPrompt combines static domain expertise with live infrastructure context.
func BuildSystemPrompt(context string) string {
    return systemPrompt + "\n\n<infrastructure_context>\n" + context + "\n</infrastructure_context>"
}
```

Integration point (Phase 4 will call this):

```go
ctx := ai.BuildContext(input)
sysPrompt := ai.BuildSystemPrompt(ctx)
ch := provider.Chat(reqCtx, messages, ai.WithSystem(sysPrompt))
```

This matches the existing pattern: `AnthropicProvider.Chat()` reads `cfg.System` and passes it to `params.System` as a `TextBlockParam` (see `anthropic.go` lines 51-53).

### Pattern 2: XML-Structured System Prompt Sections

**What:** Use XML tags to delineate sections within the system prompt.
**When to use:** For all multi-section system prompts with Claude models.
**Why:** Anthropic's prompt engineering best practices explicitly state: "XML tags help Claude parse complex prompts unambiguously, especially when your prompt mixes instructions, context, examples, and variable inputs." Use consistent, descriptive tag names. Recommended tags for this prompt:

- `<domain_knowledge>` -- capacity planning heuristics
- `<procurement_framing>` -- budget/lead time guidance
- `<response_rules>` -- output structure and tone
- `<data_gap_handling>` -- missing data behavior
- `<infrastructure_context>` -- live data (added by BuildSystemPrompt)

Source: Anthropic prompt engineering best practices (https://platform.claude.com/docs/en/docs/build-with-claude/prompt-engineering/claude-4-best-practices)

### Pattern 3: Context-at-End Ordering

**What:** Place the infrastructure data context at the end of the system prompt, after all instructions and domain knowledge.
**When to use:** When the system prompt contains both instructions and data payloads.
**Why:** Anthropic's guidance recommends placing longform data at the top of _user messages_ with queries at the end. For system prompts, the convention is different: role definition first, then domain knowledge/rules, then data context. This ordering puts instructions near the role definition where they anchor behavior, and data near the user message where it anchors responses. The query (user message) naturally follows the system prompt, so infrastructure data immediately precedes the first question.

### Pattern 4: Measured Instruction Language for Claude 4.6

**What:** Use clear, direct instructions without excessive emphasis markers.
**When to use:** For all Claude 4.6 system prompts.
**Why:** Anthropic's migration guide for Claude 4.6 explicitly warns: "If your prompts were designed to reduce undertriggering on tools or skills, these models may now overtrigger. The fix is to dial back any aggressive language. Where you might have said 'CRITICAL: You MUST use this tool when...', you can use more normal prompting like 'Use this tool when...'." This applies to our response rules -- use "cite specific numbers" not "you MUST ALWAYS cite specific numbers."

Source: Anthropic prompt engineering best practices, "Tune anti-laziness prompting" section.

### Anti-Patterns to Avoid

- **Overly long system prompts:** Encode heuristics and rules, not encyclopedic knowledge. Target 1500-2500 tokens for the static portion. The prompt teaches interpretation, not TAS administration.
- **Few-shot examples for advisory roles:** Claude 4.6's instruction-following is sufficient for structured advisory output. Few-shot examples consume tokens better spent on domain knowledge and conversation history.
- **Hardcoded data values in the system prompt:** All infrastructure-specific numbers come from BuildContext. The system prompt teaches interpretation rules, not specific data. Exception: heuristic thresholds (80%, 4:1, etc.) are domain knowledge, not environment data.
- **Persona-heavy prompting:** Per user decision, no persona or character. The role definition should be one sentence establishing the advisory function.
- **Aggressive emphasis language:** Claude 4.6 is more proactive by default. Excessive "CRITICAL", "ALWAYS", "NEVER", "you MUST" phrasing can cause over-triggering. Use measured language.

## Don't Hand-Roll

| Problem                             | Don't Build                       | Use Instead                                            | Why                                                                                                                        |
| ----------------------------------- | --------------------------------- | ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| System prompt per-request injection | Custom prompt middleware          | `ai.WithSystem()` from Phase 1                         | Already built, tested, and integrated with Anthropic provider                                                              |
| Infrastructure data serialization   | Re-serialize data for the prompt  | `ai.BuildContext()` from Phase 2                       | Already handles section ordering, threshold flags, missing-data markers, credential safety                                 |
| Token counting for prompt budget    | Custom tokenizer                  | Character count heuristic (~4 chars/token for English) | Exact token counts not needed; the prompt is well under limits and the heuristic is sufficient for budget validation tests |
| Context composition                 | Manual string building in handler | `BuildSystemPrompt()` function                         | Encapsulates the XML wrapper and separator; keeps composition logic testable and out of handler code                       |

**Key insight:** Phase 3 produces no new infrastructure. It produces a carefully crafted string constant and a thin composition function, then validates that the prompt content meets requirements through content-assertion tests.

## Common Pitfalls

### Pitfall 1: Prompt Bloat

**What goes wrong:** System prompt grows to 5000+ tokens trying to cover every edge case, crowding out conversation history in the context window.
**Why it happens:** Temptation to add "just one more rule" for each potential scenario.
**How to avoid:** Set a hard budget (2500 tokens / ~10000 characters max for the static prompt). The BuildContext adds ~1000 tokens (verified: `TestBuildContext_TokenBudget` caps realistic output at 5000 chars). Total system prompt should be under 3500 tokens, leaving 196K+ tokens for conversation.
**Warning signs:** Prompt exceeds 10,000 characters. Token budget test fails.

### Pitfall 2: Hallucination-Prone Gap Handling

**What goes wrong:** The LLM invents plausible-sounding capacity numbers when data is missing rather than acknowledging the gap.
**Why it happens:** The system prompt says "acknowledge gaps" but doesn't specify HOW to distinguish material vs immaterial gaps, or what to do instead of making something up.
**How to avoid:** The prompt must explicitly list all BuildContext data source markers and map them to materiality rules:

| BuildContext Marker                       | Section        | Material When                                        | Not Material When                               |
| ----------------------------------------- | -------------- | ---------------------------------------------------- | ----------------------------------------------- |
| `BOSH: NOT CONFIGURED` / `UNAVAILABLE`    | Data Sources   | Question involves cell metrics, vitals               | Question is about app-level memory              |
| `vSphere: NOT CONFIGURED` / `UNAVAILABLE` | Data Sources   | Question involves host sizing, HA, physical capacity | Question is about app memory or cell allocation |
| `vSphere data: NOT CONFIGURED`            | Infrastructure | Question involves clusters, hosts, N-1               | Question is about app counts or cell counts     |
| `Cell data: UNAVAILABLE`                  | Diego Cells    | Question involves cell capacity or utilization       | Question is about infrastructure hosts          |
| `No scenario comparison has been run`     | Scenario       | Question involves what-if or proposed changes        | Question is about current state                 |

The prompt must also explicitly forbid inventing numbers: "Never invent or estimate data values not present in the context."

**Warning signs:** Testing with partial data produces responses with specific numbers that don't appear in the context.

### Pitfall 3: Generic Capacity Advice

**What goes wrong:** The advisor gives textbook advice ("consider adding more capacity") instead of specific recommendations tied to the operator's actual data.
**Why it happens:** Domain knowledge in the prompt is too abstract -- rules without connection to the specific metrics available in BuildContext.
**How to avoid:** The prompt must bridge domain heuristics to BuildContext sections. The domain knowledge should reference the same terminology used in the context output. For example, the prompt says "utilization above 80% [HIGH]" and BuildContext emits `85.4% [HIGH]` -- the LLM can now connect the rule to the data. The response rules must require citing specific numbers from the `<infrastructure_context>` section.
**Warning signs:** Advisor responses that could apply to any TAS deployment without referencing specific numbers from the operator's environment.

### Pitfall 4: Tone Mismatch

**What goes wrong:** Advisor sounds like a chatbot ("Great question! Let me help you with that!") instead of a senior engineer.
**Why it happens:** Default LLM behavior without explicit tone guidance.
**How to avoid:** Claude 4.6 is naturally more concise and direct than previous models (Anthropic docs: "More direct and grounded: Provides fact-based progress reports rather than self-celebratory updates"). Light tone guidance is sufficient: "Write like a senior engineer's capacity review notes -- direct, data-driven, and actionable. Do not use conversational filler." Heavy negative instructions ("NEVER say Great question") are unnecessary and may over-trigger.
**Warning signs:** Responses starting with "Great question" or "I'd be happy to help."

### Pitfall 5: Ignoring Threshold Flags

**What goes wrong:** The advisor restates data without interpreting the [HIGH] and [CRITICAL] flags that BuildContext already inserts.
**Why it happens:** The prompt doesn't explain what these inline annotations mean.
**How to avoid:** The prompt must define what [HIGH] and [CRITICAL] flags mean, matching the exact thresholds used by `utilizationFlag()` and `vcpuRatioFlag()` in context.go:

- Utilization: `>80%` = [HIGH], `>90%` = [CRITICAL] (from `utilizationFlag()`)
- vCPU:pCPU: `>4:1` = [HIGH], `>8:1` = [CRITICAL] (from `vcpuRatioFlag()`)

When these flags appear in the infrastructure context, the advisor should prioritize discussing them and recommend corrective action.
**Warning signs:** Advisor mentions utilization numbers but doesn't flag high-utilization scenarios as urgent.

### Pitfall 6: Missing "Free Chunks" Concept

**What goes wrong:** The advisor talks about aggregate remaining memory without explaining that app placement can fail even when aggregate memory is available, because no single cell has enough contiguous capacity.
**Why it happens:** The prompt encodes aggregate utilization rules but not the per-cell placement constraint.
**How to avoid:** The domain knowledge section must explain the "free chunks" concept from VMware capacity management guidance: Diego places apps on individual cells, so what matters is not just total remaining memory but whether any single cell has enough remaining capacity for the next app push. If the largest app requires 4 GB and no cell has 4 GB free, placement fails even at 60% aggregate utilization.
**Warning signs:** Advisor says "you have 20% free capacity" without noting that capacity may be fragmented across cells.

## Code Examples

### System Prompt Structure (Recommended)

```go
// ABOUTME: Static system prompt encoding TAS/Diego capacity planning domain expertise
// ABOUTME: Combined with BuildContext output at request time via BuildSystemPrompt()

package ai

const systemPrompt = `You are a TAS/Diego capacity planning advisor. You analyze live infrastructure data and provide actionable procurement guidance for platform engineering teams.

<domain_knowledge>
## Capacity Planning Heuristics

### N-1 Redundancy
Every cluster must survive the loss of its largest host without app impact. After removing one host, the remaining hosts must have enough memory and CPU to run all current VMs. A cluster that cannot survive N-1 is at-risk and requires immediate attention.

### HA Admission Control
vSphere HA reserves a percentage of cluster resources (typically 25-33%) for failover. This reservation reduces usable capacity below the raw total. The "HA-usable" memory in the infrastructure context reflects this reservation. When evaluating capacity, use HA-usable memory, not raw totals.

### vCPU:pCPU Ratios
- 4:1 or below: safe for production workloads
- 4:1 to 8:1: elevated risk; CPU contention likely under sustained load
- Above 8:1: aggressive overcommit; performance degradation probable
The infrastructure context flags ratios above 4:1 as [HIGH] and above 8:1 as [CRITICAL].

### Cell Sizing
- Typical production cells: 32-64 GB memory, 4-8 vCPUs
- Larger cells (64 GB) reduce management overhead but increase blast radius per cell failure
- Smaller cells (32 GB) improve fault isolation but increase cell count and management overhead

### Utilization Targets
- Below 70%: healthy headroom for growth and failure absorption
- 70-80%: acceptable but plan procurement now (hardware lead times are 8-12 weeks)
- Above 80% [HIGH]: capacity constrained; procurement is urgent
- Above 90% [CRITICAL]: immediate risk of app placement failures
The infrastructure context flags utilization above 80% as [HIGH] and above 90% as [CRITICAL].

### Free Chunks and Placement
Diego places apps on individual cells, not across them. What matters is not just aggregate remaining memory but whether any single cell has enough contiguous capacity for the next app push. If the largest app requires 4 GB and no cell has 4 GB free, placement fails even at low aggregate utilization. When the Apps section shows large apps and Diego Cells shows high per-segment utilization, warn about fragmentation risk.

### Isolation Segments
Each isolation segment has independent cell pools. A segment with only 2-3 cells has poor fault tolerance -- losing one cell shifts a large percentage of workload to the remaining cells. Minimum recommended: 4 cells per segment for meaningful N-1 tolerance.

### Diego Auction Mechanics
Diego distributes app instances via an auction. Cells bid based on remaining capacity, and the auction spreads instances across AZs first, then across cells within each AZ. When no cell has sufficient capacity, the auctioneer carries unplaced work to the next batch rather than failing immediately, but persistent placement failures indicate a capacity shortage.

### Small Footprint TAS
Small Footprint deployments colocate Diego on compute VMs alongside routers, brains, and other platform components. Cell capacity is reduced because compute VMs share resources. Capacity analysis must account for this shared overhead -- the memory available for app instances is less than the total VM memory.
</domain_knowledge>

<procurement_framing>
## Procurement Context

Frame capacity findings in procurement terms:
- Hardware lead times are typically 8-12 weeks from order to rack-ready
- Budget requests often align with quarterly or annual cycles
- Growth projections should cover 6-12 months to account for procurement lag
- When utilization exceeds 80%, procurement should already be in progress
- Express capacity needs in concrete terms: "N additional hosts at X GB each" or "N additional Diego cells at X GB"
- When recommending procurement, state concrete quantities, not vague guidance
- Consider both horizontal scaling (more cells or hosts) and vertical scaling (larger cells) and recommend the more appropriate option based on the constraint
</procurement_framing>

<response_rules>
## Response Structure

For each finding:
1. State the finding clearly
2. Cite specific numbers from the infrastructure context (cell counts, utilization percentages, memory values, host counts)
3. Recommend a specific action

Keep responses concise: 2-4 paragraphs typical. Use tables for comparisons when presenting multi-resource or multi-segment data.

Write like a senior engineer's capacity review notes -- direct, data-driven, and actionable. Do not use conversational filler or preambles.

When the infrastructure context contains [HIGH] or [CRITICAL] flags, prioritize discussing them and recommend corrective action. [HIGH] means the metric is approaching a dangerous threshold. [CRITICAL] means immediate action is required.

When a scenario comparison is present, analyze the delta between current and proposed configurations. Call out whether the proposed change adequately addresses the identified constraints.
</response_rules>

<data_gap_handling>
## Handling Missing Data

The infrastructure context marks missing data sources with specific markers:
- "NOT CONFIGURED" -- the data source is not set up in this environment
- "UNAVAILABLE" -- the data source is configured but currently unreachable
- "No scenario comparison has been run" -- no what-if analysis has been performed yet

Rules for acknowledging gaps:
- Only mention a gap when it is material to the question being asked
- If someone asks about cell sizing and vSphere data shows NOT CONFIGURED, acknowledge that physical host constraints cannot be evaluated
- If someone asks about app memory usage and vSphere is missing, proceed without mentioning vSphere -- it is not relevant to that question
- Never invent or estimate data values that are not present in the context
- When a gap is material, state: (1) what data is missing, (2) what analysis cannot be performed, and (3) what conclusions can still be drawn from available data
</data_gap_handling>

The following section contains live infrastructure data for the operator's environment. Reference these specific values when making claims.`

// BuildSystemPrompt combines static domain expertise with live infrastructure context.
func BuildSystemPrompt(context string) string {
	return systemPrompt + "\n\n<infrastructure_context>\n" + context + "\n</infrastructure_context>"
}
```

### Prompt Composition and Content Validation Tests

```go
// ABOUTME: Tests for system prompt content requirements and composition function
// ABOUTME: Validates domain knowledge coverage, prompt budget, and context integration

package ai

import (
	"strings"
	"testing"
)

func TestSystemPromptContainsDomainKnowledge(t *testing.T) {
	requiredSections := []string{
		"<domain_knowledge>",
		"</domain_knowledge>",
		"<procurement_framing>",
		"</procurement_framing>",
		"<response_rules>",
		"</response_rules>",
		"<data_gap_handling>",
		"</data_gap_handling>",
	}
	for _, section := range requiredSections {
		if !strings.Contains(systemPrompt, section) {
			t.Errorf("system prompt missing required section tag: %s", section)
		}
	}
}

func TestSystemPromptContainsHeuristics(t *testing.T) {
	// DOM-01: Must encode these specific capacity planning concepts
	heuristics := []string{
		"N-1",
		"HA Admission Control",
		"vCPU:pCPU",
		"isolation segment",
		"cell sizing" ,  // case-insensitive check may be needed
		"Diego",
	}
	lower := strings.ToLower(systemPrompt)
	for _, h := range heuristics {
		if !strings.Contains(lower, strings.ToLower(h)) {
			t.Errorf("system prompt missing required heuristic: %s", h)
		}
	}
}

func TestSystemPromptContainsProcurementFraming(t *testing.T) {
	// DOM-02: Must frame in procurement terms
	terms := []string{
		"lead time",
		"budget",
		"procurement",
	}
	lower := strings.ToLower(systemPrompt)
	for _, term := range terms {
		if !strings.Contains(lower, term) {
			t.Errorf("system prompt missing procurement term: %s", term)
		}
	}
}

func TestSystemPromptContainsGapHandling(t *testing.T) {
	// DOM-03: Must reference the exact BuildContext markers
	markers := []string{
		"NOT CONFIGURED",
		"UNAVAILABLE",
		"No scenario comparison has been run",
	}
	for _, marker := range markers {
		if !strings.Contains(systemPrompt, marker) {
			t.Errorf("system prompt missing data gap marker: %s", marker)
		}
	}
}

func TestSystemPromptContainsEvidenceRequirement(t *testing.T) {
	// DOM-04: Must instruct citing specific data values
	lower := strings.ToLower(systemPrompt)
	if !strings.Contains(lower, "cite") && !strings.Contains(lower, "reference") {
		t.Error("system prompt missing instruction to cite/reference data values from context")
	}
}

func TestSystemPromptTokenBudget(t *testing.T) {
	// Static prompt should be under ~10000 chars (~2500 tokens at ~4 chars/token)
	const maxChars = 10000
	if len(systemPrompt) > maxChars {
		t.Errorf("system prompt is %d chars (~%d tokens), exceeds budget of %d chars (~%d tokens)",
			len(systemPrompt), len(systemPrompt)/4, maxChars, maxChars/4)
	}
}

func TestBuildSystemPromptIncludesContext(t *testing.T) {
	ctx := "## Diego Cells\n**shared**: 6 cells, 196608 MB total"
	result := BuildSystemPrompt(ctx)

	if !strings.Contains(result, "<infrastructure_context>") {
		t.Error("composed prompt missing opening infrastructure_context tag")
	}
	if !strings.Contains(result, "</infrastructure_context>") {
		t.Error("composed prompt missing closing infrastructure_context tag")
	}
	if !strings.Contains(result, ctx) {
		t.Error("composed prompt missing context data")
	}
	if !strings.Contains(result, "N-1") {
		t.Error("composed prompt missing domain knowledge from static portion")
	}
}

func TestBuildSystemPromptEmptyContext(t *testing.T) {
	result := BuildSystemPrompt("")
	if !strings.Contains(result, "<infrastructure_context>") {
		t.Error("composed prompt missing infrastructure_context tag even with empty context")
	}
	// The static prompt should still be fully present
	if !strings.Contains(result, "<domain_knowledge>") {
		t.Error("composed prompt missing domain_knowledge section")
	}
}
```

### BuildContext Output Format Reference

The system prompt's domain knowledge and gap handling rules are designed to work with the exact output format of `BuildContext()`. Here is a representative output for a full-data scenario (from `fullDataInput()` in context_test.go):

```
## Data Sources
- CF API: available
- BOSH: available
- vSphere: available
- Log Cache: available

## Infrastructure
Physical hosts and clusters backing Diego cells.

**cluster-a**: 4 hosts, 512 GB memory, 384 GB HA-usable, HA: ok (survives 1 host failure(s))
- Host memory utilization: 65.0%
- vCPU:pCPU ratio: 3.5:1

**cluster-b**: 3 hosts, 384 GB memory, 256 GB HA-usable, HA: ok (survives 1 host failure(s))
- Host memory utilization: 70.0%
- vCPU:pCPU ratio: 4.0:1

**Totals**: 7 hosts, 896 GB memory, HA: ok

## Diego Cells
Diego cell capacity grouped by isolation segment.

**shared**: 3 cells, 98304 MB total, 60000 MB allocated (61.0%)
**iso-seg-1**: 3 cells, 98304 MB total, 57000 MB allocated (58.0%)

**Totals**: 6 cells, 196608 MB memory, 59.6% utilization

## Apps
Top applications by memory allocation.

- big-app-1: 4 instances, 2048 MB requested, 1500 MB actual
- big-app-2: 3 instances, 1024 MB requested, 800 MB actual
...

## Scenario Comparison
Current vs proposed capacity changes.

| Metric | Current | Proposed | Delta |
|--------|---------|----------|-------|
| Cells | 6 | 8 | +2 |
| ...
```

And for a partial-data scenario (no vSphere, no scenario):

```
## Data Sources
- CF API: available
- BOSH: available
- vSphere: NOT CONFIGURED
- Log Cache: available

## Infrastructure
vSphere data: NOT CONFIGURED

## Diego Cells
...

## Scenario Comparison
No scenario comparison has been run.
```

This mapping between BuildContext output and system prompt rules is critical -- the prompt references the same terminology, markers, and structure that BuildContext produces.

## State of the Art

| Old Approach                                     | Current Approach                                        | When Changed      | Impact                                                                                     |
| ------------------------------------------------ | ------------------------------------------------------- | ----------------- | ------------------------------------------------------------------------------------------ |
| Generic "you are a helpful assistant" roles      | Domain-specific system prompts with structured XML tags | 2024-2025         | Much better domain adherence and instruction following                                     |
| Few-shot examples for output format              | Explicit format instructions with XML tags              | Claude 3.5+ 2024  | Saves tokens, equally reliable for structured responses                                    |
| Prefilled assistant responses for format control | Direct instructions only                                | Claude 4.6 (2026) | Prefills on last assistant turn deprecated in Claude 4.6; instruction-following sufficient |
| Heavy "you MUST" emphasis language               | Measured, direct instructions                           | Claude 4.6 (2026) | Claude 4.6 is more proactive; aggressive language causes over-triggering                   |
| Verbose summaries after tool calls               | Concise, direct responses by default                    | Claude 4.6 (2026) | Claude 4.6 is naturally less verbose; lighter tone guidance suffices                       |

**Deprecated/outdated:**

- Prefilled assistant responses: No longer supported on last assistant turn in Claude 4.6 models. Use explicit instructions instead.
- Heavy anti-laziness prompting: Claude 4.6 is more proactive by default. Instructions like "you MUST ALWAYS" that were needed for older models may now cause over-triggering. Use normal, measured language.

## Open Questions

1. **Materiality rules: explicit vs general guidance**
   - What we know: The prompt must distinguish material vs immaterial gaps. The markers are well-defined. The materiality table in Pitfall 2 maps each marker to its relevance conditions.
   - What's unclear: Whether encoding explicit if/then rules for each marker (verbose but unambiguous) or providing general guidance with 2-3 examples (shorter, relies on LLM judgment) works better in practice.
   - Recommendation: Start with general guidance plus 2-3 examples (as shown in the code example). This fits the token budget better and Claude 4.6's instruction-following is reliable. The planner should include a manual testing step where the implementer verifies gap handling with 2-3 test prompts against partial data contexts. If the general rules produce poor results, escalate to explicit per-marker rules.

2. **Token budget validation heuristic**
   - What we know: The ~4 chars/token heuristic is approximate. Anthropic does not expose a public tokenizer for Claude models.
   - What's unclear: Whether 4 chars/token or 5 chars/token is more accurate for mixed English/technical content.
   - Recommendation: Use 4 chars/token as the conservative estimate for budget tests. The existing `TestBuildContext_TokenBudget` uses ~5 chars/token for context output. The system prompt has more prose (closer to 4), while context has more numbers and formatting (closer to 5). The actual budget has large margins (3500 tokens out of 200K), so precision does not matter.

3. **Prompt wording for "free chunks" fragmentation warning**
   - What we know: BuildContext shows per-segment utilization and app sizes. The prompt should teach the LLM to warn about fragmentation risk.
   - What's unclear: BuildContext does not emit per-cell remaining memory -- only per-segment aggregates. The LLM can infer fragmentation risk from (high segment utilization + large apps) but cannot calculate exact free chunks.
   - Recommendation: Include the free chunks concept in domain knowledge but frame it as a risk indicator rather than a calculation: "When utilization is high and the Apps section shows large app sizes, warn that individual cells may not have enough contiguous capacity for new deployments, even if aggregate utilization appears manageable."

## Sources

### Primary (HIGH confidence)

- Anthropic prompt engineering best practices (https://platform.claude.com/docs/en/docs/build-with-claude/prompt-engineering/claude-4-best-practices) -- XML tag structuring, role assignment, output formatting, Claude 4.6 migration guide, measured instruction language, prefill deprecation. Verified 2026-02-24.
- Existing codebase: `backend/services/ai/` -- provider.go (ChatProvider interface, Chat method), options.go (WithSystem option), context.go (BuildContext function, ContextInput struct, threshold flag functions), anthropic.go (system prompt injection via TextBlockParam), context_test.go (token budget test, output format verification). All files read and verified.

### Secondary (MEDIUM confidence)

- VMware Tanzu capacity management blog (https://blogs.vmware.com/tanzu/keep-your-app-platform-in-a-happy-state-an-operators-guide-to-capacity-management-on-pivotal-cloud-foundry/) -- 35% capacity reserve for AZ failure, free chunks monitoring concept, CapacityRemainingMemory/Disk metrics. Verified 2026-02-24.
- VMware Tanzu key scaling indicators (https://docs.vmware.com/en/VMware-Tanzu-Application-Service/6.0/tas-for-vms/monitoring-key-cap-scaling.html) -- Diego cell capacity metrics: remaining memory, remaining disk, remaining container capacity. Three key scaling indicators. Max 250 cells per deployment. Verified 2026-02-24 (redirects to Broadcom/Elastic Application Runtime docs).
- vSphere HA Admission Control (https://techdocs.broadcom.com/us/en/vmware-cis/vsphere/vsphere/8-0/vsphere-availability/creating-and-using-vsphere-ha-clusters/vsphere-ha-admission-control.html) -- three failover capacity methods (percentage, slot, dedicated hosts), minimum 3 hosts required, performance reduction threshold. Verified 2026-02-24.
- Cloud Foundry Diego Auction documentation (https://docs.cloudfoundry.org/concepts/diego/diego-auction.html) -- auction placement priorities: stack compatibility, AZ distribution, cell-level spreading, load balancing. Unplaced work carried to next batch. Verified 2026-02-24.

### Tertiary (LOW confidence)

- VMware redundancy blog (https://blogs.vmware.com/tanzu/redundancy-free-capacity-and-example-calculations/) -- 50% headroom recommendation (specific to GemFire, not Diego, but reserve-for-failure principle applies)
- Diego cell tuning guide (https://gerg.github.io/cf-onboarding/oss/projects/diego_cell_tuning) -- community resource for cell tuning parameters

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH -- this phase uses only existing infrastructure (Go const, WithSystem, BuildContext); no new dependencies. Verified by reading all existing source files.
- Architecture: HIGH -- the pattern (static prompt const + composition function + content-assertion tests) is straightforward and well-supported by the existing codebase. File placement follows the existing `backend/services/ai/` convention.
- Domain knowledge content: HIGH (upgraded from MEDIUM) -- TAS/Diego heuristics verified against VMware/Broadcom official documentation (capacity management blog, key scaling indicators, HA Admission Control docs) and cross-referenced against thresholds already encoded in the codebase (`utilizationFlag()` at 80%/90%, `vcpuRatioFlag()` at 4:1/8:1, `CPURiskLevel()` at 4.0/8.0, `CalculateHAHostFailures()` N-1 logic). The codebase and official docs agree on thresholds.
- Prompt engineering patterns: HIGH -- verified against current Anthropic docs at platform.claude.com including Claude 4.6 specific guidance (measured language, prefill deprecation, natural conciseness).
- Pitfalls: HIGH -- based on Anthropic's published guidance, direct observation of common LLM system prompt issues, and specific analysis of how BuildContext output interacts with prompt instructions.

**Research date:** 2026-02-24
**Valid until:** 2026-03-24 (stable domain -- capacity planning heuristics and prompt engineering patterns change slowly)
