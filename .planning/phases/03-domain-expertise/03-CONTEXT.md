# Phase 3: Domain Expertise - Context

**Gathered:** 2026-02-24
**Status:** Ready for planning

<domain>
## Phase Boundary

System prompt that transforms the LLM from a generic chatbot into a TAS/Diego capacity planning domain expert. The prompt encodes operational knowledge and instructs the LLM to reason about procurement decisions using the operator's live infrastructure data (provided via BuildContext from Phase 2). The chat endpoint (Phase 4) and UI (Phase 5-6) are separate phases.

</domain>

<decisions>
## Implementation Decisions

### Domain knowledge depth

- Platform architect level expertise: N-1 redundancy, HA Admission Control, vCPU:pCPU ratios, cell sizing heuristics, isolation segment tradeoffs, AZ placement, Diego auction mechanics, Garden container lifecycle
- Encode all standard VMware/TAS heuristics: N-1 host failure tolerance, HA Admission Control at 25-33%, vCPU:pCPU ratio ceilings (4:1 safe, 8:1 risky), cell memory sizing (32-64GB typical), 80% utilization target, isolation segment overhead
- Cover both standard TAS (dedicated diego_cell VMs) and Small Footprint TAS (colocated Diego on compute VMs) -- the dashboard already detects both
- Stay at the logical capacity layer for vSphere: host memory, CPU, and HA. Do not encode DRS rules, resource pools, or storage I/O knowledge since the dashboard doesn't provide that data

### Response tone and framing

- Primary audience: platform engineers making procurement cases to management. They understand TAS but need data-backed arguments for capacity review documents
- Proactively recommend actions when data shows clear issues (high utilization, N-1 violations, ratio overcommit) -- don't wait to be asked
- Frame procurement recommendations around real-world realities: lead times (8-12 week typical), budget cycles, growth planning horizons
- No persona or name -- just "the advisor." Professional and tool-like, reads like a senior engineer's capacity review notes

### Data gap handling

- Acknowledge material gaps only -- caveat when missing data is relevant to the question being asked
- If someone asks about cell sizing and vSphere is NOT CONFIGURED, that's material -- acknowledge it
- If someone asks about app memory and vSphere is missing, that's not material -- skip the caveat
- Never hallucinate data to fill gaps; state what's unknown and what can still be concluded

### Response structure

- Consistent format: finding + evidence + recommendation
- State the finding, cite specific numbers from the provided context, then recommend action
- Concise: 2-4 paragraphs typical, use tables for comparisons
- Always reference actual data values from context when making claims

### Claude's Discretion

- Exact system prompt length and token budget allocation
- How to organize domain knowledge sections within the prompt (topical vs priority-ordered)
- Whether to use few-shot examples in the system prompt or rely on instruction-following
- Specific wording of data gap acknowledgment patterns

</decisions>

<specifics>
## Specific Ideas

No specific requirements -- open to standard approaches for system prompt engineering. The key constraint is that the prompt must work with the BuildContext output format established in Phase 2 (markdown with section headers, threshold flags, and NOT CONFIGURED/UNAVAILABLE markers).

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

_Phase: 03-domain-expertise_
_Context gathered: 2026-02-24_
