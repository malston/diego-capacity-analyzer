# Phase 2: Context Builder - Context

**Gathered:** 2026-02-24
**Status:** Ready for planning

<domain>
## Phase Boundary

A pure function that serializes the handler's in-memory state (dashboard data, infrastructure, scenario results) into annotated markdown text for the LLM. Reads only model types. Never touches credentials or makes API calls. Output is a single string consumed by the Chat endpoint (Phase 4).

</domain>

<decisions>
## Implementation Decisions

### Data sections & ordering

- Section order: data source summary, infrastructure (vSphere), Diego cells (per-segment aggregates), apps (top-N consumers), scenario comparison (if run)
- Infrastructure comes first -- physical capacity is the foundation for understanding cell utilization
- Cell metrics are aggregated per isolation segment (totals, averages, utilization %) -- no per-cell detail
- App data includes top 5-10 consumers by memory allocation, not the full app list
- Scenario comparisons appear as compact before-and-after summaries (current vs proposed deltas)

### Annotation style

- Markdown format with ## headings per section
- Brief one-line section intros explaining what the LLM is reading (e.g., "Physical hosts and clusters backing Diego cells")
- Target ~500-1000 tokens total -- compact with key metrics, no narrative padding
- Flag notable conditions at thresholds (e.g., utilization >80%, N-1 violations) without pre-computing full assessments

### Missing data markers

- Data source summary block at top: "Data Sources: CF API check, BOSH check, vSphere x (not configured), Log Cache check"
- Explicit per-section markers distinguishing "NOT CONFIGURED" (admin choice) vs "UNAVAILABLE" (connection error)
- Partial data gaps (e.g., Log Cache missing for some apps) noted at section level: "Note: Memory usage unavailable for 3 of 47 apps"
- Sections for unconfigured sources still appear with the marker -- never silently omitted

### Credential scrubbing

- Context builder accepts only model types (dashboard data, infrastructure state, scenario results) -- never receives config.Config
- Credentials cannot leak because they are never passed to the function
- Hostnames and deployment names are allowed in output -- they are topology, not credentials
- Unit test uses distinctive sentinel values (e.g., "CREDENTIAL_PASSWORD_VALUE") to verify no config credential fields appear in output
- Scrub scope limited to config.Config credential fields: passwords, client secrets, API keys, CA certs

### Claude's Discretion

- Exact threshold values for flagged conditions
- Precise section intro wording
- How to represent numeric data (inline vs mini-tables)
- Token budget allocation across sections when data is unusually large

</decisions>

<specifics>
## Specific Ideas

No specific requirements -- open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

_Phase: 02-context-builder_
_Context gathered: 2026-02-24_
