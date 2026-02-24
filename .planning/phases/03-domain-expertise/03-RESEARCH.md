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

| ID     | Description                                                                                                                                                              | Research Support                                                                                                                                                                         |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| DOM-01 | System prompt encodes TAS/Diego capacity planning knowledge: N-1 redundancy, HA Admission Control, vCPU:pCPU ratios, cell sizing heuristics, isolation segment tradeoffs | Domain knowledge section of system prompt with heuristic thresholds; threshold flags in BuildContext ([HIGH], [CRITICAL]) already signal these; prompt teaches the LLM to interpret them |
| DOM-02 | System prompt frames analysis in procurement terms: lead times, budget cycles, growth planning, headroom targets                                                         | Procurement framing section of system prompt with specific guidance on lead times, budget justification language, and growth horizon planning                                            |
| DOM-03 | System prompt instructs LLM to acknowledge data gaps rather than hallucinate when information is missing                                                                 | Data gap handling section of system prompt keyed to BuildContext markers (NOT CONFIGURED, UNAVAILABLE); materiality-based caveat rules                                                   |
| DOM-04 | System prompt instructs LLM to reference specific data values from context when making claims                                                                            | Response structure rules in system prompt requiring finding + evidence + recommendation pattern; explicit instruction to cite numbers from context                                       |

</phase_requirements>

## Summary

Phase 3 produces a single Go string constant containing the system prompt that transforms a generic Claude model into a TAS/Diego capacity planning advisor. This phase is pure prompt engineering with a lightweight Go implementation -- no new APIs, services, or complex architecture. The system prompt is passed per-request via the existing `WithSystem` option established in Phase 1, concatenated at call time with the BuildContext output from Phase 2.

The primary technical challenge is prompt organization -- encoding enough domain expertise to be useful while staying within a reasonable token budget (the system prompt itself should be roughly 1500-2500 tokens, combined with ~1000 tokens from BuildContext, leaving ample room for conversation in the model's context window). The secondary challenge is getting the data gap handling right -- the prompt must teach the LLM to distinguish material gaps from immaterial ones based on the specific question being asked.

**Primary recommendation:** Implement the system prompt as a Go `const` in a dedicated file (`backend/services/ai/prompt.go`), organized using XML tags per Anthropic's best practices, with domain knowledge sections ordered by priority of use. Skip few-shot examples in favor of explicit instruction-following rules -- the prompt is too small to waste tokens on examples, and Claude models follow structured instructions reliably.

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
**Why:** The system prompt has two parts: (1) static domain expertise/instructions that never change, and (2) dynamic infrastructure context that changes per request. Separating them enables testing each independently.

```go
// prompt.go

const systemPrompt = `You are a TAS/Diego capacity planning advisor...
<domain_knowledge>
...
</domain_knowledge>

<response_rules>
...
</response_rules>

<data_gap_handling>
...
</data_gap_handling>`

// BuildSystemPrompt combines static domain expertise with live infrastructure context.
func BuildSystemPrompt(context string) string {
    return systemPrompt + "\n\n<infrastructure_context>\n" + context + "\n</infrastructure_context>"
}
```

Integration at chat endpoint (Phase 4 will call this):

```go
ctx := ai.BuildContext(input)
sysPrompt := ai.BuildSystemPrompt(ctx)
ch := provider.Chat(reqCtx, messages, ai.WithSystem(sysPrompt))
```

### Pattern 2: XML-Structured System Prompt Sections

**What:** Use XML tags to delineate sections within the system prompt per Anthropic's official guidance.
**When to use:** For all multi-section system prompts with Claude models.
**Why:** Anthropic's documentation explicitly recommends XML tags for complex prompts. They reduce misinterpretation and help Claude parse sections unambiguously. Tags like `<domain_knowledge>`, `<response_rules>`, `<data_gap_handling>`, and `<infrastructure_context>` make the prompt self-documenting.

Source: Anthropic prompt engineering best practices (https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering/claude-4-best-practices) -- "XML tags help Claude parse complex prompts unambiguously, especially when your prompt mixes instructions, context, examples, and variable inputs."

### Pattern 3: Context-at-End Ordering

**What:** Place the infrastructure data context at the end of the system prompt, after instructions and domain knowledge.
**When to use:** When the system prompt contains both instructions and large data payloads.
**Why:** Anthropic's documentation recommends placing longform data at the top of prompts, but this guidance is for user messages with queries at the end. For system prompts, the convention is: role definition first, then domain knowledge/rules, then data context. The query (user message) naturally comes after the system prompt. This ordering puts instructions near the role definition where they anchor behavior, and data near the query where it anchors responses.

### Anti-Patterns to Avoid

- **Overly long system prompts:** The prompt should encode heuristics and rules, not encyclopedic knowledge. Target 1500-2500 tokens for the static portion.
- **Few-shot examples in system prompts for advisory roles:** For this use case, Claude's instruction-following is sufficient. Few-shot examples consume tokens that are better spent on domain knowledge and user conversation history. Reserve few-shot for cases where output format is non-obvious.
- **Hardcoded data values in the system prompt:** All infrastructure-specific numbers come from BuildContext. The system prompt teaches interpretation rules, not specific data.
- **Persona-heavy prompting:** Per user decision, no persona or character. Just "the advisor" -- professional and tool-like.

## Don't Hand-Roll

| Problem                             | Don't Build                      | Use Instead                                            | Why                                                                                                                        |
| ----------------------------------- | -------------------------------- | ------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| System prompt per-request injection | Custom prompt middleware         | `ai.WithSystem()` from Phase 1                         | Already built, tested, and integrated with Anthropic provider                                                              |
| Infrastructure data serialization   | Re-serialize data for the prompt | `ai.BuildContext()` from Phase 2                       | Already handles section ordering, threshold flags, missing-data markers, credential safety                                 |
| Token counting for prompt budget    | Custom tokenizer                 | Character count heuristic (~4 chars/token for English) | Exact token counts not needed; the prompt is well under limits and the heuristic is sufficient for budget validation tests |

**Key insight:** Phase 3 produces no new infrastructure. It produces a carefully crafted string constant and a thin composition function, then validates that the prompt content meets requirements.

## Common Pitfalls

### Pitfall 1: Prompt Bloat

**What goes wrong:** System prompt grows to 5000+ tokens trying to cover every edge case, crowding out conversation history in the context window.
**Why it happens:** Temptation to add "just one more rule" for each potential scenario.
**How to avoid:** Set a hard budget (2500 tokens max for the static prompt). The BuildContext adds ~1000 tokens. Total system prompt should be under 3500 tokens, leaving 196K+ tokens for conversation.
**Warning signs:** Prompt exceeds 10,000 characters (~2500 tokens).

### Pitfall 2: Hallucination-Prone Gap Handling

**What goes wrong:** The LLM invents plausible-sounding capacity numbers when data is missing rather than acknowledging the gap.
**Why it happens:** The system prompt says "acknowledge gaps" but doesn't specify HOW to distinguish material vs immaterial gaps, or what to do instead of making something up.
**How to avoid:** The prompt must explicitly list the data source markers (NOT CONFIGURED, UNAVAILABLE) and instruct the LLM to: (a) check if the missing data is relevant to the current question, (b) if relevant, state what's unknown and what analysis cannot be performed, (c) if irrelevant, proceed without caveat. The prompt should also explicitly forbid inventing numbers.
**Warning signs:** Testing with partial data produces responses with specific numbers that don't appear in the context.

### Pitfall 3: Generic Capacity Advice

**What goes wrong:** The advisor gives textbook advice ("consider adding more capacity") instead of specific recommendations tied to the operator's actual data.
**Why it happens:** Domain knowledge in the prompt is too abstract -- rules without connection to the specific metrics available in BuildContext.
**How to avoid:** The prompt must map domain heuristics to specific BuildContext sections. For example: "When the Diego Cells section shows utilization above 80% [HIGH], recommend adding cells. Cite the specific utilization percentage and cell count from the context."
**Warning signs:** Advisor responses that could apply to any TAS deployment without referencing specific numbers from the operator's environment.

### Pitfall 4: Tone Mismatch

**What goes wrong:** Advisor sounds like a chatbot ("Great question! Let me help you with that!") instead of a senior engineer.
**Why it happens:** Default LLM behavior without explicit tone guidance.
**How to avoid:** Explicit negative instruction: "Do not use conversational filler. Do not express enthusiasm about questions. Write like a senior engineer's capacity review notes."
**Warning signs:** Responses starting with "Great question" or "I'd be happy to help."

### Pitfall 5: Ignoring Threshold Flags

**What goes wrong:** The advisor restates data without interpreting the [HIGH] and [CRITICAL] flags that BuildContext already inserts.
**Why it happens:** The prompt doesn't explain what these inline annotations mean.
**How to avoid:** The prompt must define what [HIGH] and [CRITICAL] flags mean: "[HIGH] indicates a metric approaching a dangerous threshold. [CRITICAL] indicates a metric that requires immediate action. When you see these flags in the infrastructure context, prioritize discussing them and recommend corrective action."
**Warning signs:** Advisor mentions utilization numbers but doesn't flag high-utilization scenarios as urgent.

## Code Examples

### System Prompt Structure (Recommended)

```go
// prompt.go
package ai

const systemPrompt = `You are a TAS/Diego capacity planning advisor. You analyze live infrastructure data and provide actionable procurement guidance for platform engineering teams.

<domain_knowledge>
## Capacity Planning Heuristics

### N-1 Redundancy
Every cluster must survive the loss of its largest host without app impact. After removing one host, the remaining hosts must have enough memory and CPU to run all VMs. If a cluster cannot survive N-1, it is at-risk.

### HA Admission Control
vSphere HA reserves a percentage of cluster resources (typically 25-33%) for failover. This reservation reduces the usable capacity below the raw total. HA-usable memory is what remains after the HA reservation.

### vCPU:pCPU Ratios
- 4:1 or below: safe for production workloads
- 4:1 to 8:1: elevated risk; CPU contention likely under load
- Above 8:1: aggressive overcommit; performance degradation probable

### Cell Sizing
- Typical production cells: 32-64 GB memory, 4-8 vCPUs
- Larger cells (64 GB) reduce management overhead but increase blast radius per cell failure
- Smaller cells (32 GB) improve fault isolation but increase cell count and management burden

### Utilization Targets
- Below 70%: healthy headroom for growth and failure absorption
- 70-80%: acceptable but plan procurement now (lead times are 8-12 weeks)
- Above 80% [HIGH]: capacity constrained; procurement is urgent
- Above 90% [CRITICAL]: immediate risk of app placement failures

### Isolation Segments
Each isolation segment has independent cell pools. A segment with only 2-3 cells has poor fault tolerance because losing one cell shifts a large percentage of workload. Minimum recommended: 4 cells per segment for meaningful N-1 tolerance.

### Diego Auction Mechanics
Diego places app instances on cells with available capacity. The auction considers remaining memory, disk, and container count. When no single cell has a large enough contiguous block of memory for a new app instance, placement fails even if aggregate free memory exists across cells.

### Small Footprint TAS
Small Footprint deployments colocate Diego on compute VMs alongside other platform components. Cell capacity is reduced because compute VMs share resources with routers, brains, and other processes. Capacity analysis must account for this shared overhead.
</domain_knowledge>

<procurement_framing>
## Procurement Context

Frame capacity findings in procurement terms:
- Hardware lead times are typically 8-12 weeks from order to rack-ready
- Budget requests often align with quarterly or annual cycles
- Growth projections should cover 6-12 months to account for procurement lag
- When utilization exceeds 80%, procurement should already be in progress
- Express capacity needs in concrete terms: "N additional hosts at X GB each" or "N additional Diego cells at X GB"
- Include cost context when possible: fewer larger hosts vs more smaller hosts
</procurement_framing>

<response_rules>
## Response Structure

1. State the finding clearly
2. Cite specific numbers from the infrastructure context (cell counts, utilization percentages, memory values)
3. Recommend a specific action

Keep responses concise: 2-4 paragraphs typical. Use tables for comparisons. Do not use conversational filler. Do not express enthusiasm about questions. Write like a senior engineer's capacity review notes -- direct, data-driven, and actionable.

When the infrastructure context contains [HIGH] or [CRITICAL] flags, prioritize discussing these and recommend corrective action. [HIGH] means approaching a dangerous threshold. [CRITICAL] means immediate action required.

When recommending procurement, state concrete quantities: "Add 2 hosts at 128 GB each" not "consider adding more capacity."
</response_rules>

<data_gap_handling>
## Handling Missing Data

The infrastructure context marks missing data sources:
- "NOT CONFIGURED" -- the data source is not set up in this environment
- "UNAVAILABLE" -- the data source is configured but currently unreachable
- "No scenario comparison has been run" -- no what-if analysis has been performed

Rules for acknowledging gaps:
- Only mention a gap if it is MATERIAL to the question being asked
- If someone asks about cell sizing and vSphere data shows NOT CONFIGURED, acknowledge that physical host constraints cannot be evaluated
- If someone asks about app memory usage and vSphere is missing, do not mention vSphere -- it is not relevant
- Never invent or estimate data values that are not present in the context
- When a gap is material, state: what data is missing, what analysis cannot be performed, and what conclusions can still be drawn from available data
</data_gap_handling>

The following section contains live infrastructure data for the operator's environment. Reference these specific values when making claims.
`

// BuildSystemPrompt combines static domain expertise with live infrastructure context.
func BuildSystemPrompt(context string) string {
	return systemPrompt + "\n<infrastructure_context>\n" + context + "</infrastructure_context>"
}
```

### Prompt Composition Test

```go
// prompt_test.go
package ai

import (
	"strings"
	"testing"
)

func TestSystemPromptContainsDomainKnowledge(t *testing.T) {
	requiredSections := []string{
		"<domain_knowledge>",
		"<procurement_framing>",
		"<response_rules>",
		"<data_gap_handling>",
	}
	for _, section := range requiredSections {
		if !strings.Contains(systemPrompt, section) {
			t.Errorf("system prompt missing required section: %s", section)
		}
	}
}

func TestSystemPromptContainsHeuristics(t *testing.T) {
	heuristics := []string{
		"N-1",
		"HA Admission Control",
		"vCPU:pCPU",
		"isolation segment",
	}
	for _, h := range heuristics {
		if !strings.Contains(systemPrompt, h) {
			t.Errorf("system prompt missing required heuristic: %s", h)
		}
	}
}

func TestBuildSystemPromptIncludesContext(t *testing.T) {
	ctx := "## Diego Cells\n**shared**: 6 cells, 196608 MB total"
	result := BuildSystemPrompt(ctx)

	if !strings.Contains(result, "<infrastructure_context>") {
		t.Error("composed prompt missing infrastructure_context tag")
	}
	if !strings.Contains(result, ctx) {
		t.Error("composed prompt missing context data")
	}
	if !strings.Contains(result, "N-1") {
		t.Error("composed prompt missing domain knowledge")
	}
}

func TestSystemPromptTokenBudget(t *testing.T) {
	// Static prompt should be under ~10000 chars (~2500 tokens)
	const maxChars = 10000
	if len(systemPrompt) > maxChars {
		t.Errorf("system prompt is %d chars (~%d tokens), exceeds budget of %d chars (~%d tokens)",
			len(systemPrompt), len(systemPrompt)/4, maxChars, maxChars/4)
	}
}
```

## State of the Art

| Old Approach                                     | Current Approach                                        | When Changed                 | Impact                                                                 |
| ------------------------------------------------ | ------------------------------------------------------- | ---------------------------- | ---------------------------------------------------------------------- |
| Generic "you are a helpful assistant" roles      | Domain-specific system prompts with structured XML tags | Anthropic guidance 2024-2025 | Much better domain adherence and instruction following                 |
| Few-shot examples for output format              | Explicit format instructions with XML tags              | Claude 3.5+ (2024)           | Saves tokens, equally reliable for structured responses                |
| Prefilled assistant responses for format control | Direct instructions + structured output tools           | Claude 4.6 (2026)            | Prefills deprecated in Claude 4.6; instruction-following is sufficient |

**Deprecated/outdated:**

- Prefilled assistant responses: No longer supported in Claude 4.6 models. Use explicit instructions instead.
- Heavy anti-laziness prompting: Claude 4.6 is more proactive by default; overly aggressive "you MUST" language may cause over-triggering.

## Open Questions

1. **Exact prompt wording for materiality-based gap handling**
   - What we know: The prompt must distinguish material vs immaterial gaps. The markers are well-defined (NOT CONFIGURED, UNAVAILABLE).
   - What's unclear: Whether explicit if/then rules (verbose but unambiguous) or general guidance (shorter but relies on LLM judgment) works better in practice.
   - Recommendation: Start with explicit rules. The planner should include a manual testing step where the implementer verifies gap handling with 2-3 test prompts against partial data contexts. If the explicit rules produce unnatural responses, simplify.

2. **Token budget split between domain knowledge and conversation**
   - What we know: Claude Sonnet has a 200K token context window. The system prompt (static + context) should be under 4000 tokens total.
   - What's unclear: Whether the domain knowledge section (heuristics) should be exhaustive or minimal. More knowledge = better base responses but less room for long conversations.
   - Recommendation: Target 2000 tokens for static prompt, 1000 tokens for BuildContext. This leaves 197K tokens for conversation history -- more than enough. The code example above is approximately this size.

## Sources

### Primary (HIGH confidence)

- Anthropic prompt engineering best practices (https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering/claude-4-best-practices) -- XML tag structuring, role assignment, output formatting, Claude 4.6 considerations
- Existing codebase: `backend/services/ai/` -- provider.go, options.go, context.go, anthropic.go establish the integration points

### Secondary (MEDIUM confidence)

- VMware Tanzu capacity management blog (https://blogs.vmware.com/tanzu/keep-your-app-platform-in-a-happy-state-an-operators-guide-to-capacity-management-on-pivotal-cloud-foundry/) -- free chunks, 35% headroom, monitoring metrics
- VMware Tanzu key scaling indicators (https://docs.vmware.com/en/VMware-Tanzu-Application-Service/6.0/tas-for-vms/monitoring-key-cap-scaling.html) -- Diego cell capacity metrics and scaling triggers
- vSphere HA Admission Control (https://techdocs.broadcom.com/us/en/vmware-cis/vsphere/vsphere/8-0/vsphere-availability/creating-and-using-vsphere-ha-clusters/vsphere-ha-admission-control.html) -- percentage-based cluster resource reservation for failover

### Tertiary (LOW confidence)

- VMware redundancy blog (https://blogs.vmware.com/tanzu/redundancy-free-capacity-and-example-calculations/) -- 50% headroom recommendation (specific to GemFire, not Diego, but principle applies)

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH -- this phase uses only existing infrastructure (Go const, WithSystem, BuildContext); no new dependencies
- Architecture: HIGH -- the pattern (static prompt const + composition function + tests) is straightforward and well-supported by the existing codebase
- Domain knowledge content: MEDIUM -- TAS/Diego heuristics are well-established industry knowledge, but some thresholds (e.g., 80% utilization target) are customary rather than documented in a single authoritative source. The codebase already encodes these same thresholds in its flag functions.
- Pitfalls: HIGH -- based on direct observation of common LLM system prompt issues and Anthropic's published guidance

**Research date:** 2026-02-24
**Valid until:** 2026-03-24 (stable domain -- capacity planning heuristics and prompt engineering patterns change slowly)
