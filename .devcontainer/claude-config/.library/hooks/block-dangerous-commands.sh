#!/usr/bin/env bash
# ABOUTME: PreToolUse hook that blocks dangerous Bash commands.
# ABOUTME: Exits with code 2 to block, 0 to allow.
#
# USAGE:
#     As a Claude Code hook (automatic):
#         Triggered automatically before Bash tool execution.
#
#     Manual invocation (test blocking):
#         echo '{"tool_input": {"command": "rm -rf /"}}' | ./block-dangerous-commands.sh
#         # Returns exit code 2 (blocked)
#
#     Manual invocation (test allowing):
#         echo '{"tool_input": {"command": "git status"}}' | ./block-dangerous-commands.sh
#         # Returns exit code 0 (allowed)
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
#                 "command": "~/.claude/hooks/block-dangerous-commands.sh"
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
#         "command": "rm -rf /"
#       }
#     }
#
# BLOCKED COMMANDS:
#     - rm -rf / or ~            - Recursive delete of root/home
#     - rm -rf *                 - Recursive delete with wildcard
#     - rm lock files            - package-lock.json, yarn.lock, go.sum
#     - git push --force/-f      - Force push (suggests --force-with-lease)
#     - git reset --hard origin  - Destructive reset
#     - cf delete-org/space      - Cloud Foundry resource deletion
#     - kubectl delete --all     - Kubernetes mass deletion
#     - docker system prune -a   - Full Docker prune
#     - chmod -R 777             - Insecure permissions
#     - curl/wget | bash/sh      - Piping to shell
#     - > /etc/                  - Writing to /etc
#     - dd if=* of=/dev/         - Direct disk write
#     - mkfs.*                   - Filesystem format
#     - :(){:|:&};:              - Fork bomb
#
# DEPENDENCIES:
#     - jq (required for JSON parsing)
#     - grep with -E (extended regex)
#
# EXIT CODES:
#     0 - Command allowed
#     2 - Command blocked (Claude Code convention for blocking)

set -uo pipefail

# Read JSON input from stdin
input=$(cat)

# Extract command from tool_input
command=$(echo "$input" | jq -r '.tool_input.command // empty' 2>/dev/null)

# Exit silently if no command
if [[ -z "$command" ]]; then
    exit 0
fi

# Dangerous patterns to block
# Each pattern: regex|human-readable message
dangerous_patterns=(
    'rm\s+-rf\s+/$|Recursive delete from root'
    'rm\s+-rf\s+/[^a-zA-Z]|Recursive delete from root'
    'rm\s+-rf\s+~|Recursive delete of home directory'
    'rm\s+-rf\s+\*|Recursive delete with wildcard'
    'rm\s+.*package-lock\.json|Deletion of package-lock.json'
    'rm\s+.*yarn\.lock|Deletion of yarn.lock'
    'rm\s+.*go\.sum|Deletion of go.sum'
    'git\s+push.*--force|Force push (use --force-with-lease instead)'
    'git\s+push.*-f\s|Force push (use --force-with-lease instead)'
    'git\s+reset\s+--hard\s+origin|Hard reset to origin (destructive)'
    'cf\s+delete-org|Cloud Foundry org deletion'
    'cf\s+delete-space|Cloud Foundry space deletion'
    'cf\s+delete\s+-f|Cloud Foundry force delete'
    'kubectl\s+delete.*--all|Kubernetes mass deletion'
    'docker\s+system\s+prune\s+-a|Docker full system prune'
    'chmod\s+-R\s+777|Insecure permissions (777)'
    'curl.*\|\s*bash|Piping curl to bash (security risk)'
    'curl.*\|\s*sh|Piping curl to sh (security risk)'
    'wget.*\|\s*bash|Piping wget to bash (security risk)'
    '>\s*/etc/|Writing to /etc'
    'dd\s+if=.*of=/dev/|Direct disk write'
    'mkfs\.|Filesystem format command'
    ':(){:|Fork bomb pattern'
)

# Check each pattern
for pattern_msg in "${dangerous_patterns[@]}"; do
    pattern="${pattern_msg%|*}"
    message="${pattern_msg#*|}"

    if echo "$command" | grep -qEi "$pattern"; then
        echo "BLOCKED: $message" >&2
        echo "Command: $command" >&2
        exit 2
    fi
done

# Allow the command
exit 0
