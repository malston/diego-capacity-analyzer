#!/usr/bin/env bash
# ABOUTME: PreToolUse hook that logs all Bash commands to an audit file.
# ABOUTME: Useful for debugging and reviewing what Claude executed.
#
# USAGE:
#     As a Claude Code hook (automatic):
#         Triggered automatically before Bash tool execution.
#
#     Manual invocation:
#         echo '{"tool_input": {"command": "ls -la", "description": "List files"}}' | ./log-bash-commands.sh
#
#     View log:
#         tail -f ~/.claude/bash-command-log.txt
#
# CONFIGURATION:
#     Add to ~/.claude/settings.json:
#
#     {
#       "hooks": {
#         "PreToolUse": [
#           {
#             "matcher": "Bash",
#             "hooks": [
#               {
#                 "type": "command",
#                 "command": "~/.claude/hooks/log-bash-commands.sh"
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
#         "command": "git status",
#         "description": "Check git status"
#       }
#     }
#
# OUTPUT FORMAT (log file):
#     [2026-01-21 09:48:25] git status - Check git status
#
# LOG FILE:
#     ~/.claude/bash-command-log.txt
#
# DEPENDENCIES:
#     - jq (required for JSON parsing)
#
# EXIT CODES:
#     0 - Always (logging should never block commands)

set -uo pipefail

LOG_FILE="$HOME/.claude/bash-command-log.txt"

# Read JSON input from stdin
input=$(cat)

# Extract command and description from tool_input
command=$(echo "$input" | jq -r '.tool_input.command // "unknown"' 2>/dev/null)
description=$(echo "$input" | jq -r '.tool_input.description // "No description"' 2>/dev/null)

# Get timestamp
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

# Log to file
echo "[$timestamp] $command - $description" >> "$LOG_FILE"

# Always allow (exit 0)
exit 0
