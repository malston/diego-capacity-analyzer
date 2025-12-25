# Product Guidelines: TAS Capacity Analyzer

## Voice and Tone

### Technical and Precise
Use industry-standard terminology without simplification. Platform engineers managing TAS infrastructure are experts who expect accurate technical language.

**Do:**
- "Diego cell memory utilization: 78.3%"
- "BOSH Director connection failed: certificate verify failed"
- "Container memory (actual) vs. allocated memory"

**Don't:**
- "Your system is running a bit hot"
- "Something went wrong connecting to your server"
- "Memory being used vs. memory requested"

### Terminology Standards
- Use "Diego cell" not "cell" or "VM" when referring to TAS compute units
- Use "isolation segment" not "segment" or "org boundary"
- Use "container" when referring to application instances
- Use "allocated" vs "actual" when distinguishing requested from used resources

## Visual Design Principles

### Data-Dense and Information-Rich
Maximize information density per screen. Operators want comprehensive visibility without navigating between views.

- Display multiple metrics simultaneously (memory, CPU, container count)
- Use compact table layouts with sortable columns
- Show aggregates and drill-down details on the same screen
- Avoid excessive whitespace that forces scrolling

### Status-Based Color System
Use consistent color coding based on utilization thresholds:

| Status | Color | Threshold |
|--------|-------|-----------|
| Healthy | Green | < 70% utilization |
| Warning | Yellow/Amber | 70% - 85% utilization |
| Critical | Red | > 85% utilization |

Apply these colors consistently to:
- Cell utilization indicators
- Progress bars and gauges
- Table row highlights
- Chart segments

### Typography
- Use monospace fonts for numeric values, IDs, and technical identifiers
- Use proportional fonts for labels and descriptive text
- Right-align numeric columns for easy comparison

## Error Handling

### Inline Technical Details
Display errors with sufficient technical context for troubleshooting. Platform engineers need details to diagnose issues.

**Error messages should include:**
- The specific operation that failed
- API endpoint or service involved
- HTTP status code or error type
- Actionable troubleshooting hints when possible

**Example:**
```text
BOSH Director Error
Failed to fetch VM vitals from https://10.0.0.6:25555/deployments/cf-abc123/vms
Status: 401 Unauthorized
Hint: Verify BOSH_CLIENT and BOSH_CLIENT_SECRET environment variables
```

### Partial Data Display
When some data sources fail:
- Show available data rather than blocking the entire view
- Mark unavailable sections with clear "Data unavailable" indicators
- Provide retry actions where appropriate

## Interaction Patterns

### Mouse-Optimized with Hover States
Design for mouse-driven interaction with rich contextual information.

**Hover behaviors:**
- Tooltips on truncated text and chart elements
- Preview cards for cells and applications
- Highlight related elements (e.g., hovering a cell highlights its apps)

**Click-to-drill-down:**
- Click cells to view detailed metrics and running applications
- Click applications to see container-level stats
- Click charts to filter the view to that segment

### Data Tables
- Sortable columns with clear sort indicators
- Filterable by text search and dropdown selectors
- Row hover highlighting
- Click-to-expand for additional details

## Accessibility

- Maintain sufficient color contrast (WCAG AA minimum)
- Provide text alternatives for color-coded status (icons, labels)
- Support standard browser zoom without layout breaking
- Ensure interactive elements have visible focus states
