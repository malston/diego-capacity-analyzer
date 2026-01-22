#!/usr/bin/env bash
# .claude/scripts/parse-claude-output.sh
# Fixed JSON escaping

set -euo pipefail

COMMAND="${1:-/context}"
FORMAT="${2:-json}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Get raw output
RAW_OUTPUT=$("$SCRIPT_DIR/run-claude-command.sh" "$COMMAND" 2>&1)

# Filter out script messages
FILTERED_OUTPUT=$(echo "$RAW_OUTPUT" | grep -v "^Starting\|^Waiting\|^Sending\|^Completed\|^Session:\|^Claude ready!")

parse_context() {
    local output="$1"
    local format="$2"

    MODEL=$(echo "$output" | grep -oE '(Opus|Sonnet|Haiku) [0-9.]+' | head -1 || echo "")
    PLAN=$(echo "$output" | grep -oE 'Claude (Max|Pro|Free)' | sed 's/Claude //' | head -1 || echo "")
    [[ -z "$PLAN" ]] && PLAN=$(echo "$output" | grep -oE '\| (Max|Pro|Free)\]' | grep -oE 'Max|Pro|Free' | head -1 || echo "")
    USERNAME=$(echo "$output" | grep -E '^\s*[a-zA-Z0-9_-]+\s*$' | tr -d '[:space:]' | head -1 || echo "")

    STATS_LINE=$(echo "$output" | grep -E 'CLAUDE\.md.*MCP.*hook' | sed 's/^[[:space:]]*//' || echo "")
    CLAUDE_MD=$(echo "$STATS_LINE" | grep -oE '[0-9]+[[:space:]]+CLAUDE\.md' | grep -oE '[0-9]+' || echo "0")
    MCPS=$(echo "$STATS_LINE" | grep -oE '[0-9]+[[:space:]]+MCPs?' | grep -oE '[0-9]+' || echo "0")
    HOOKS=$(echo "$STATS_LINE" | grep -oE '[0-9]+[[:space:]]+hooks?' | grep -oE '[0-9]+' || echo "0")

    USAGE_LINE=$(echo "$output" | grep -E '[0-9]+h:.*[0-9]+%' || echo "")
    TIME_LIMIT=$(echo "$USAGE_LINE" | grep -oE '[0-9]+h' | head -1 | grep -oE '[0-9]+' || echo "0")
    USAGE_PERCENT=$(echo "$output" | grep -oE '[0-9]+%' | head -1 | grep -oE '[0-9]+' || echo "0")
    TIME_REMAINING=$(echo "$output" | grep -oE '\([0-9]+h[[:space:]]+[0-9]+m\)' | tr -d '()' || echo "unknown")

    case "$format" in
        json)
            cat << EOF
{
  "command": "/context",
  "model": "${MODEL:-unknown}",
  "plan": "${PLAN:-unknown}",
  "username": "${USERNAME:-unknown}",
  "stats": {
    "claude_md_files": ${CLAUDE_MD},
    "mcps": ${MCPS},
    "hooks": ${HOOKS}
  },
  "usage": {
    "time_limit_hours": ${TIME_LIMIT},
    "percent_used": ${USAGE_PERCENT},
    "time_remaining": "${TIME_REMAINING}"
  }
}
EOF
            ;;
        yaml)
            cat << EOF
command: /context
model: ${MODEL:-unknown}
plan: ${PLAN:-unknown}
username: ${USERNAME:-unknown}
stats:
  claude_md_files: ${CLAUDE_MD}
  mcps: ${MCPS}
  hooks: ${HOOKS}
usage:
  time_limit_hours: ${TIME_LIMIT}
  percent_used: ${USAGE_PERCENT}
  time_remaining: ${TIME_REMAINING}
EOF
            ;;
        env)
            cat << EOF
CLAUDE_MODEL="${MODEL:-unknown}"
CLAUDE_PLAN="${PLAN:-unknown}"
CLAUDE_USERNAME="${USERNAME:-unknown}"
CLAUDE_MD_FILES=${CLAUDE_MD}
CLAUDE_MCPS=${MCPS}
CLAUDE_HOOKS=${HOOKS}
CLAUDE_TIME_LIMIT=${TIME_LIMIT}
CLAUDE_USAGE_PERCENT=${USAGE_PERCENT}
CLAUDE_TIME_REMAINING="${TIME_REMAINING}"
EOF
            ;;
        raw)
            echo "$FILTERED_OUTPUT"
            ;;
    esac
}

case "$COMMAND" in
    /context|/mcp)
        # Both commands produce the same output
        parse_context "$FILTERED_OUTPUT" "$FORMAT"
        ;;
    *)
        # For unknown commands, return raw or JSON-wrapped output
        case "$FORMAT" in
            json)
                # Use jq to properly escape the output
                RAW_JSON=$(echo "$FILTERED_OUTPUT" | jq -Rs .)
                echo "{\"command\": \"$COMMAND\", \"raw_output\": $RAW_JSON}"
                ;;
            yaml)
                echo "command: $COMMAND"
                echo "raw_output: |"
                echo "$FILTERED_OUTPUT" | sed 's/^/  /'
                ;;
            env)
                echo "CLAUDE_COMMAND=\"$COMMAND\""
                ;;
            raw|*)
                echo "$FILTERED_OUTPUT"
                ;;
        esac
        ;;
esac