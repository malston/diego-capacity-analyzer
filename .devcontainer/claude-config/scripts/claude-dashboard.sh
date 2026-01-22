#!/usr/bin/env bash
# ~/.claude/scripts/claude-dashboard.sh
# Display comprehensive Claude status

DATA=$(.claude/scripts/parse-claude-output.sh /context json)

cat << EOF
╭─── Claude Code Status ────────────────────────────╮
│ Model:      $(echo "$DATA" | jq -r '.model')
│ Plan:       $(echo "$DATA" | jq -r '.plan')
│ User:       $(echo "$DATA" | jq -r '.username')
├───────────────────────────────────────────────────┤
│ Files:      $(echo "$DATA" | jq -r '.stats.claude_md_files') CLAUDE.md
│ MCPs:       $(echo "$DATA" | jq -r '.stats.mcps')
│ Hooks:      $(echo "$DATA" | jq -r '.stats.hooks')
├───────────────────────────────────────────────────┤
│ Time Limit: $(echo "$DATA" | jq -r '.usage.time_limit_hours') hours
│ Used:       $(echo "$DATA" | jq -r '.usage.percent_used')%
│ Remaining:  $(echo "$DATA" | jq -r '.usage.time_remaining')
╰───────────────────────────────────────────────────╯
EOF