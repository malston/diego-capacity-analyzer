#!/usr/bin/env bash
# Install Claude projects maintenance to crontab

CRON_JOB="0 2 1 * * ~/.claude/scripts/maintain-claude-projects.sh >> ~/Library/Logs/claude-maintenance.log 2>&1"
COMMENT="# Claude projects maintenance - runs monthly on the 1st at 2am"

# Get existing crontab, filter out any existing claude maintenance entries, add the new one
(crontab -l 2>/dev/null | grep -v "maintain-claude-projects" ; echo "$COMMENT" ; echo "$CRON_JOB") | crontab -

echo "✓ Cron job installed:"
echo "  $CRON_JOB"
echo ""
crontab -l | grep maintain