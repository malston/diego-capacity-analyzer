# Phase 2: Context Builder - Research

**Researched:** 2026-02-24
**Domain:** Go text serialization / LLM context assembly from in-memory model types
**Confidence:** HIGH

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions

- Section order: data source summary, infrastructure (vSphere), Diego cells (per-segment aggregates), apps (top-N consumers), scenario comparison (if run)
- Infrastructure comes first -- physical capacity is the foundation for understanding cell utilization
- Cell metrics are aggregated per isolation segment (totals, averages, utilization %) -- no per-cell detail
- App data includes top 5-10 consumers by memory allocation, not the full app list
- Scenario comparisons appear as compact before-and-after summaries (current vs proposed deltas)
- Markdown format with ## headings per section
- Brief one-line section intros explaining what the LLM is reading (e.g., "Physical hosts and clusters backing Diego cells")
- Target ~500-1000 tokens total -- compact with key metrics, no narrative padding
- Flag notable conditions at thresholds (e.g., utilization >80%, N-1 violations) without pre-computing full assessments
- Data source summary block at top: "Data Sources: CF API check, BOSH check, vSphere x (not configured), Log Cache check"
- Explicit per-section markers distinguishing "NOT CONFIGURED" (admin choice) vs "UNAVAILABLE" (connection error)
- Partial data gaps (e.g., Log Cache missing for some apps) noted at section level: "Note: Memory usage unavailable for 3 of 47 apps"
- Sections for unconfigured sources still appear with the marker -- never silently omitted
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
- Internal function signatures and decomposition

### Deferred Ideas (OUT OF SCOPE)

None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>

## Phase Requirements

| ID     | Description                                                                                                                                          | Research Support                                                                                            |
| ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| CTX-01 | Context builder serializes current dashboard state (cell counts, memory utilization, isolation segments, app counts) into annotated text for the LLM | Pure function taking `models.DashboardResponse`; aggregates cells per segment, selects top-N apps by memory |
| CTX-02 | Context builder serializes infrastructure state (clusters, hosts, VMs) when vSphere data is available                                                | Accepts `*models.InfrastructureState` (nil when absent); renders cluster table or "NOT CONFIGURED" marker   |
| CTX-03 | Context builder serializes scenario comparison results when a scenario has been run                                                                  | Accepts `*models.ScenarioComparison` (nil when absent); renders compact before/after delta summary          |
| CTX-04 | Context builder flags missing data sources with explicit markers the LLM can reference                                                               | Data source summary block with per-source status; section-level markers for partial gaps                    |
| CTX-05 | Context builder reads from existing Handler state without making additional API calls                                                                | Function signature accepts only model types; never receives config, HTTP clients, or service pointers       |

</phase_requirements>

## Summary

Phase 2 is a serialization problem, not an integration problem. The context builder is a pure function (or small set of pure functions) that takes in-memory model types already present in the Handler struct and produces a single annotated markdown string for the LLM. No new dependencies are needed. No API calls, no credentials, no HTTP. The entire implementation lives in Go's standard library (`fmt`, `strings`, `sort`).

The codebase already has all the model types needed: `models.DashboardResponse` (cells, apps, segments, metadata), `models.InfrastructureState` (clusters, hosts, utilization, HA status), and `models.ScenarioComparison` (current vs proposed results with deltas and warnings). The Handler struct holds these as `*models.InfrastructureState` (mutex-protected) and the dashboard response is available via cache. The context builder function just needs to accept these as parameters and produce text.

**Primary recommendation:** Create a single `services/ai/context.go` file with a `BuildContext` function that accepts model types and data-source availability booleans, and returns a markdown string. Keep it in the `ai` package since it produces LLM-specific output. Test it thoroughly with table-driven tests covering full data, partial data, all-missing data, and credential non-leakage scenarios.

## Standard Stack

### Core

| Library   | Version | Purpose                                              | Why Standard                                                          |
| --------- | ------- | ---------------------------------------------------- | --------------------------------------------------------------------- |
| `fmt`     | stdlib  | Format numeric values, Sprintf for section templates | Go standard; no external dependency needed                            |
| `strings` | stdlib  | `strings.Builder` for efficient string concatenation | Idiomatic Go for building multi-section text                          |
| `sort`    | stdlib  | Sort apps by memory for top-N selection              | Already used in `models/bottleneck.go` and `models/recommendation.go` |

### Supporting

| Library | Version | Purpose                          | When to Use                                      |
| ------- | ------- | -------------------------------- | ------------------------------------------------ |
| `math`  | stdlib  | Rounding utilization percentages | Only if formatting needs rounding beyond Sprintf |

### Alternatives Considered

| Instead of        | Could Use       | Tradeoff                                                                                                                                                                             |
| ----------------- | --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `strings.Builder` | `text/template` | Templates add complexity and indirection for a function that's mostly `Sprintf` calls; Builder is simpler, easier to test, and the output structure is fixed (not user-configurable) |
| `strings.Builder` | `bytes.Buffer`  | Functionally equivalent; Builder is slightly more idiomatic for string-only output since it avoids []byte conversion                                                                 |

**Installation:** None -- all standard library.

## Architecture Patterns

### Recommended Project Structure

```
backend/services/ai/
├── provider.go            # ChatProvider interface (existing)
├── options.go             # Functional options (existing)
├── anthropic.go           # Anthropic provider (existing)
├── context.go             # Context builder function (NEW)
└── context_test.go        # Context builder tests (NEW)
```

### Pattern 1: Pure Function with Structured Input

**What:** A single exported function that accepts a struct of inputs (all model types) and returns a string. No methods on a receiver, no state, no side effects.
**When to use:** When the operation has no lifecycle, no configuration that changes between calls, and no dependencies to inject.
**Example:**

```go
// ContextInput bundles all data sources the context builder can serialize.
// Nil pointers indicate absent/unconfigured data sources.
type ContextInput struct {
    Dashboard   *models.DashboardResponse
    Infra       *models.InfrastructureState
    Scenario    *models.ScenarioComparison
    // Data source availability flags (distinct from nil data --
    // a source can be configured but have no data yet)
    BOSHConfigured    bool
    VSphereConfigured bool
    LogCacheAvailable bool
}

// BuildContext serializes capacity data into annotated markdown for the LLM.
func BuildContext(input ContextInput) string {
    var b strings.Builder
    writeDataSourceSummary(&b, input)
    writeInfrastructure(&b, input.Infra)
    writeDiegoCells(&b, input.Dashboard)
    writeApps(&b, input.Dashboard)
    writeScenario(&b, input.Scenario)
    return b.String()
}
```

### Pattern 2: Section Writers with Consistent Signatures

**What:** Each section is written by a private helper function with signature `func writeXxx(b *strings.Builder, data *T)`. The helper handles its own nil check and writes either the data section or the appropriate missing-data marker.
**When to use:** When the output has multiple independent sections with uniform absent-data behavior.
**Example:**

```go
func writeInfrastructure(b *strings.Builder, infra *models.InfrastructureState) {
    b.WriteString("\n## Infrastructure\n")
    if infra == nil {
        b.WriteString("vSphere data: NOT CONFIGURED\n")
        return
    }
    b.WriteString("Physical hosts and clusters backing Diego cells.\n\n")
    // ... render cluster data
}
```

### Pattern 3: Top-N Selection for Apps

**What:** Sort apps by memory allocation descending, take the top N, and note how many were omitted.
**When to use:** When the full list is too large for the LLM token budget.
**Example:**

```go
func topAppsByMemory(apps []models.App, n int) (top []models.App, total int) {
    total = len(apps)
    sorted := make([]models.App, total)
    copy(sorted, apps)
    sort.SliceStable(sorted, func(i, j int) bool {
        return sorted[i].RequestedMB > sorted[j].RequestedMB
    })
    if n > total {
        n = total
    }
    return sorted[:n], total
}
```

### Pattern 4: Threshold Flagging

**What:** After writing a metric value, append a brief flag if it crosses a threshold (e.g., "82% [HIGH]"). Flags are inline annotations, not separate assessment sections.
**When to use:** For utilization percentages, HA status, vCPU ratios.
**Example:**

```go
func utilizationFlag(pct float64) string {
    if pct > 90 {
        return " [CRITICAL]"
    }
    if pct > 80 {
        return " [HIGH]"
    }
    return ""
}
```

### Anti-Patterns to Avoid

- **Passing config.Config to the builder:** The function must never receive the Config struct. Credential isolation is enforced at the function signature level, not by runtime scrubbing.
- **Making API calls inside the builder:** The builder is a pure serializer. All data must be fetched beforehand and passed in.
- **One giant function:** Each section should be a separate helper for testability. The top-level function orchestrates section order.
- **Using text/template:** Adds unnecessary complexity. The output format is developer-controlled, not user-configurable, so direct string building is simpler and more debuggable.
- **Including per-cell detail:** Decision locked: aggregate per isolation segment, no individual cell rows.

## Don't Hand-Roll

| Problem               | Don't Build       | Use Instead                  | Why                                                                     |
| --------------------- | ----------------- | ---------------------------- | ----------------------------------------------------------------------- |
| String concatenation  | Manual `+` chains | `strings.Builder`            | Efficient, idiomatic, avoids O(n^2) allocation                          |
| Top-N sorting         | Custom sort algo  | `sort.SliceStable`           | Already used throughout the codebase (bottleneck.go, recommendation.go) |
| Percentage formatting | Manual math       | `fmt.Sprintf("%.1f%%", val)` | Consistent formatting, handles edge cases                               |

**Key insight:** This phase needs zero external dependencies. The complexity is in the data selection and formatting decisions (what to include, how to annotate), not in the tooling. Keep it simple.

## Common Pitfalls

### Pitfall 1: Token Budget Overrun

**What goes wrong:** With many clusters, apps, or isolation segments, the output exceeds the ~500-1000 token target, wasting context window.
**Why it happens:** No upper bound on list lengths; each cluster/segment adds several lines.
**How to avoid:** Cap app list at top-N (5-10 per decision). For clusters, show all (typical environments have 1-3 clusters). If an environment has many isolation segments, aggregate the smallest ones into an "other" bucket.
**Warning signs:** Test with realistic data (50+ apps, 3+ segments, 2+ clusters) and count approximate tokens.

### Pitfall 2: Nil Pointer Panic on Optional Data

**What goes wrong:** Accessing fields on nil `InfrastructureState` or nil `ScenarioComparison` causes a panic.
**Why it happens:** These are optional data sources -- vSphere may not be configured, scenario may not have been run.
**How to avoid:** Each section writer checks for nil at the top and writes the missing-data marker instead. The `ContextInput` struct uses pointer types for optional sources.
**Warning signs:** Tests that only cover the "everything available" case.

### Pitfall 3: Credential Leakage Through Transitive Fields

**What goes wrong:** A model struct happens to contain a field that was populated from a credential (e.g., a URL containing an auth token).
**Why it happens:** The builder is supposed to only see model types, but someone adds config.Config to the input "for convenience."
**How to avoid:** The function signature is the firewall -- it accepts only `models.*` types and booleans. The credential scrub test (CTX-05 success criterion #5) acts as a regression safety net: populate a Config with sentinel values like "CREDENTIAL_PASSWORD_VALUE" and assert none appear in the output.
**Warning signs:** Adding config or service types to `ContextInput`.

### Pitfall 4: Inconsistent Missing-Data Markers

**What goes wrong:** Some sections silently omit themselves when data is missing, while others show a marker. The LLM can't tell whether data was "not loaded yet" vs "not configured."
**Why it happens:** Each section writer handles missing data independently without a shared convention.
**How to avoid:** Define two standard markers up-front (e.g., `"NOT CONFIGURED"` and `"UNAVAILABLE"`) and use them consistently. The data source summary block at the top provides the overview; section-level markers reinforce it. Test that every section emits a marker when its data is nil.
**Warning signs:** A section that returns early with no output when data is nil.

### Pitfall 5: Segment Aggregation Math Errors

**What goes wrong:** Per-segment aggregation double-counts cells or mismatches apps to segments.
**Why it happens:** DiegoCell.IsolationSegment may be empty (shared/default segment), and App.IsolationSegment uses the same convention.
**How to avoid:** Group by IsolationSegment string. Empty string is the "shared" segment. Test with mixed segments including the empty-string default.
**Warning signs:** Cell counts or memory totals in the context don't match DashboardResponse totals.

## Code Examples

### Segment Aggregation

```go
type segmentSummary struct {
    Name          string
    CellCount     int
    TotalMemoryMB int
    AllocatedMB   int
    UsedMB        int
    AppCount      int
}

func aggregateBySegment(cells []models.DiegoCell, apps []models.App) map[string]*segmentSummary {
    segments := make(map[string]*segmentSummary)

    for _, cell := range cells {
        seg := cell.IsolationSegment
        if seg == "" {
            seg = "shared"
        }
        s, ok := segments[seg]
        if !ok {
            s = &segmentSummary{Name: seg}
            segments[seg] = s
        }
        s.CellCount++
        s.TotalMemoryMB += cell.MemoryMB
        s.AllocatedMB += cell.AllocatedMB
        s.UsedMB += cell.UsedMB
    }

    for _, app := range apps {
        seg := app.IsolationSegment
        if seg == "" {
            seg = "shared"
        }
        s, ok := segments[seg]
        if !ok {
            s = &segmentSummary{Name: seg}
            segments[seg] = s
        }
        s.AppCount += app.Instances
    }

    return segments
}
```

### Data Source Summary Block

```go
func writeDataSourceSummary(b *strings.Builder, input ContextInput) {
    b.WriteString("## Data Sources\n")

    // CF API is always available (required config)
    cfStatus := "available"
    if input.Dashboard == nil || len(input.Dashboard.Apps) == 0 {
        cfStatus = "UNAVAILABLE"
    }
    fmt.Fprintf(b, "- CF API: %s\n", cfStatus)

    // BOSH
    if !input.BOSHConfigured {
        b.WriteString("- BOSH: NOT CONFIGURED\n")
    } else if input.Dashboard == nil || !input.Dashboard.Metadata.BOSHAvailable {
        b.WriteString("- BOSH: UNAVAILABLE\n")
    } else {
        b.WriteString("- BOSH: available\n")
    }

    // vSphere
    if !input.VSphereConfigured {
        b.WriteString("- vSphere: NOT CONFIGURED\n")
    } else if input.Infra == nil {
        b.WriteString("- vSphere: UNAVAILABLE\n")
    } else {
        b.WriteString("- vSphere: available\n")
    }
    b.WriteString("\n")
}
```

### Scenario Compact Summary

```go
func writeScenario(b *strings.Builder, scenario *models.ScenarioComparison) {
    b.WriteString("\n## Scenario Comparison\n")
    if scenario == nil {
        b.WriteString("No scenario comparison has been run.\n")
        return
    }
    b.WriteString("Current vs proposed capacity changes.\n\n")
    cur := scenario.Current
    pro := scenario.Proposed
    delta := scenario.Delta
    fmt.Fprintf(b, "| Metric | Current | Proposed | Delta |\n")
    fmt.Fprintf(b, "|--------|---------|----------|-------|\n")
    fmt.Fprintf(b, "| Cells | %d | %d | %+d |\n",
        cur.CellCount, pro.CellCount, pro.CellCount-cur.CellCount)
    fmt.Fprintf(b, "| Cell Size | %s | %s | - |\n",
        cur.CellSize(), pro.CellSize())
    fmt.Fprintf(b, "| App Capacity | %d GB | %d GB | %+d GB |\n",
        cur.AppCapacityGB, pro.AppCapacityGB, delta.CapacityChangeGB)
    fmt.Fprintf(b, "| Utilization | %.1f%% | %.1f%% | %+.1f%% |\n",
        cur.UtilizationPct, pro.UtilizationPct, delta.UtilizationChangePct)
}
```

## State of the Art

| Old Approach                      | Current Approach                              | When Changed                   | Impact                                                                    |
| --------------------------------- | --------------------------------------------- | ------------------------------ | ------------------------------------------------------------------------- |
| Template-based context formatting | Direct string building with `strings.Builder` | Stable pattern in Go ecosystem | Templates add indirection without benefit for developer-controlled output |
| Pass full config/handler          | Pure function with model-only inputs          | Established security pattern   | Prevents credential leakage by construction                               |

**Deprecated/outdated:**

- Nothing relevant -- this is standard Go string manipulation with no external dependencies.

## Open Questions

1. **Exact top-N count for apps**
   - What we know: CONTEXT.md says "5-10 consumers by memory allocation"
   - What's unclear: Whether to use 5 or 10, or make it configurable
   - Recommendation: Default to 10; the planner can set a const. At ~50 tokens for 10 apps (one line each), this fits within budget. Drop to 5 only if token measurements prove tight.

2. **How the chat endpoint will call BuildContext**
   - What we know: Phase 4 (chat endpoint) will assemble the system prompt with context. The Handler has access to dashboard cache and infrastructure state under mutex.
   - What's unclear: Whether BuildContext should be called from the handler or from a middleware.
   - Recommendation: BuildContext is a pure function in the `ai` package. The Phase 4 handler will call it, passing data it already holds. No further coupling needed now.

3. **Log Cache availability signal**
   - What we know: `ContextInput` needs a `LogCacheAvailable` boolean. Log Cache data surfaces as `UsedMB` on DiegoCell (when BOSH+Log Cache are active) and `ActualMB` on App.
   - What's unclear: There's no explicit `LogCacheConfigured` field on config or handler today.
   - Recommendation: Infer Log Cache availability from the data itself: if any cell has `UsedMB > 0` or any app has `ActualMB > 0`, Log Cache contributed data. Alternatively, count apps where `ActualMB == 0` as "memory usage unavailable" (matches the CONTEXT.md pattern). The planner should determine the exact heuristic.

## Sources

### Primary (HIGH confidence)

- Codebase inspection: `backend/models/models.go`, `backend/models/infrastructure.go`, `backend/models/scenario.go` -- all model types the builder will consume
- Codebase inspection: `backend/handlers/handlers.go` -- Handler struct showing available state (infrastructureState, cache, chatProvider)
- Codebase inspection: `backend/handlers/health.go`, `backend/handlers/infrastructure.go` -- how dashboard and infra data are fetched and cached
- Codebase inspection: `backend/config/config.go` -- all credential fields that must never appear in output
- Codebase inspection: `backend/services/ai/` -- existing package structure for AI-related code

### Secondary (MEDIUM confidence)

- Go standard library documentation for `strings.Builder`, `fmt.Fprintf`, `sort.SliceStable` -- stable, well-documented APIs

### Tertiary (LOW confidence)

- None -- this phase requires no external libraries or ecosystem research.

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - Pure Go stdlib, no dependencies needed
- Architecture: HIGH - Function signature, data flow, and file placement are clear from codebase inspection
- Pitfalls: HIGH - Derived directly from model types and CONTEXT.md decisions; all verifiable in code

**Research date:** 2026-02-24
**Valid until:** Indefinite (Go stdlib is stable; model types are the only moving target)
