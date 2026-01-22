#!/usr/bin/env bash
# ABOUTME: PostToolUse hook that auto-formats files after Claude edits them.
# ABOUTME: Runs gofmt for Go files and prettier for JS/TS/JSX/TSX files.
#
# USAGE:
#     As a Claude Code hook (automatic):
#         Triggered automatically after Edit|Write operations.
#
#     Manual invocation:
#         echo '{"tool_input": {"file_path": "/path/to/file.go"}}' | ./format-on-save.sh
#
# CONFIGURATION:
#     Add to ~/.claude/settings.json:
#
#     {
#       "hooks": {
#         "PostToolUse": [
#           {
#             "matcher": "Edit|Write",
#             "hooks": [
#               {
#                 "type": "command",
#                 "command": "~/.claude/hooks/format-on-save.sh"
#               }
#             ]
#           }
#         ]
#       }
#     }
#
# INPUT FORMAT (stdin):
#     {
#       "tool_input": {
#         "file_path": "/path/to/file.go"
#       }
#     }
#
# SUPPORTED FILE TYPES:
#     .go                 - Formatted with gofmt
#     .js, .jsx, .ts, .tsx - Formatted with prettier
#     .json, .css, .scss   - Formatted with prettier
#     .md, .yaml, .yml     - Formatted with prettier
#
# DEPENDENCIES:
#     - jq (required for JSON parsing)
#     - gofmt (for Go files)
#     - prettier or npx (for JS/TS/CSS/etc files)
#
# EXIT CODES:
#     0 - Success (file formatted or skipped if unsupported type)

set -euo pipefail

# Read JSON input from stdin
input=$(cat)

# Extract file_path from tool_input
file_path=$(echo "$input" | jq -r '.tool_input.file_path // empty' 2>/dev/null)

# Exit silently if no file path (shouldn't happen, but be defensive)
if [[ -z "$file_path" ]]; then
    exit 0
fi

# Exit if file doesn't exist (might have been deleted)
if [[ ! -f "$file_path" ]]; then
    exit 0
fi

# Format Go files with gofmt
if [[ "$file_path" == *.go ]]; then
    if command -v gofmt &>/dev/null; then
        gofmt -w "$file_path" 2>/dev/null || true
    fi
    exit 0
fi

# Format JavaScript/TypeScript files with prettier
if [[ "$file_path" =~ \.(js|jsx|ts|tsx|json|css|scss|md|yaml|yml)$ ]]; then
    if command -v prettier &>/dev/null; then
        prettier --write "$file_path" 2>/dev/null || true
    elif command -v npx &>/dev/null; then
        # Fallback to npx if prettier not globally installed
        npx prettier --write "$file_path" 2>/dev/null || true
    fi
    exit 0
fi

exit 0
