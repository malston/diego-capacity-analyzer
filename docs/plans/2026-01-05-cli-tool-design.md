# CLI Tool Design

A command-line interface for the Diego Capacity Analyzer API, enabling CI/CD pipelines to monitor capacity and alert when thresholds are exceeded.

## Goals

- Enable automated capacity checks in CI pipelines
- Provide human-readable output for debugging, JSON for parsing
- Exit with non-zero status when thresholds exceeded
- Single binary, no runtime dependencies

## Commands

### diego-capacity health

Check backend connectivity.

```text
diego-capacity health
```

Output:
```text
Backend: http://localhost:8080
CF API:  ok
BOSH:    ok
```

Exit codes: 0 (healthy), 2 (error)

### diego-capacity status

Show current infrastructure status.

```text
diego-capacity status
```

Output:
```text
Infrastructure: vcenter.example.com (vsphere)
Clusters:       2
Hosts:          8
Diego Cells:    20

N-1 Capacity:     72% [ok]
Memory:           78% [warning]
Constraining:     memory
```

Exit codes: 0 (success), 2 (error/no data)

### diego-capacity check

Check thresholds and exit non-zero if any exceeded.

```text
diego-capacity check [flags]
```

Flags:
- `--n1-threshold <percent>` - N-1 capacity threshold (default: 85)
- `--memory-threshold <percent>` - Memory utilization threshold (default: 90)
- `--staging-threshold <count>` - Minimum free staging chunks (default: 200)

Output:
```text
✓ N-1 capacity: 72% (threshold: 85%)
✗ Memory utilization: 92% (threshold: 90%)
✓ Staging chunks: 450 (threshold: 200)

FAILED: 1 check exceeded threshold
```

Exit codes:
- 0: All checks passed
- 1: One or more thresholds exceeded
- 2: Error (connectivity, no data, invalid input)

## Global Flags

- `--json` - Output JSON instead of human-readable text
- `--api-url <url>` - Override DIEGO_CAPACITY_API_URL environment variable

## Configuration

Environment variables:
- `DIEGO_CAPACITY_API_URL` - Backend URL (default: http://localhost:8080)

## Project Layout

```text
cmd/
└── cli/
    ├── main.go           # Entry point, cobra setup
    └── cmd/
        ├── root.go       # Root command, global flags
        ├── health.go     # health subcommand
        ├── status.go     # status subcommand
        └── check.go      # check subcommand

internal/
└── client/
    └── client.go         # HTTP client wrapping API calls
```

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- Reuses `backend/models/` for response types

## Build

Makefile targets:
- `make cli` - Build CLI binary
- `make cli-install` - Install to $GOPATH/bin

Binary name: `diego-capacity`

Release workflow updated to include CLI binary for all platforms (linux/darwin × amd64/arm64).

## CI Pipeline Example

Concourse:
```yaml
- task: capacity-check
  config:
    platform: linux
    image_resource:
      type: registry-image
      source: { repository: alpine }
    run:
      path: diego-capacity
      args: [check, --n1-threshold, "80"]
    params:
      DIEGO_CAPACITY_API_URL: ((capacity-api-url))
```

GitHub Actions:
```yaml
- name: Check capacity
  env:
    DIEGO_CAPACITY_API_URL: ${{ secrets.CAPACITY_API_URL }}
  run: diego-capacity check --n1-threshold 80
```

## Error Handling

| Scenario | Exit Code | Message |
|----------|-----------|---------|
| Backend unreachable | 2 | `Error: cannot connect to backend at http://...` |
| No infrastructure data | 2 | `Error: no infrastructure data. Load data via UI or API first.` |
| Invalid threshold | 2 | `Error: --n1-threshold must be between 0 and 100` |
| Threshold exceeded | 1 | Shows which checks failed |

## Testing

- Unit tests for threshold logic and output formatting
- Integration tests using httptest with mock API responses
- No external dependencies required

## Out of Scope (v1)

- Authentication (backend doesn't require it)
- Config file support (~/.diego-capacity.yaml)
- Watch mode (--watch for continuous monitoring)
- Direct alerting (Slack, email) - delegated to CI pipeline

## Future Considerations

- `diego-capacity dashboard` - Fetch full dashboard data
- `diego-capacity scenario` - Run what-if comparisons
- `diego-capacity init` - Generate config file interactively
