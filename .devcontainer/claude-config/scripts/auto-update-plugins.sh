#!/usr/bin/env bash
# ABOUTME: Automatically check and update Claude Code plugins and marketplaces
# ABOUTME: Called by .envrc when entering the directory
# ABOUTME: Uses mkdir for atomic locking (portable)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LAST_CHECK_FILE="$SCRIPT_DIR/../.last_plugin_check"
LOCK_DIR="/tmp/claude-plugin-update.lock"

BLUE='\033[38;2;6;176;204m'
NC='\033[0m'

# Parse arguments
FORCE=false
QUIET=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --force) FORCE=true; shift ;;
        --quiet) QUIET=true; shift ;;
        *) shift ;;
    esac
done

log() {
    [[ "$QUIET" == false ]] && printf "${BLUE}ℹ %s${NC}\n" "$1"
}

# Acquire lock
if ! mkdir "$LOCK_DIR" 2>/dev/null; then
    if [[ -d "$LOCK_DIR" ]]; then
        lock_age=$(( $(date +%s) - $(stat -f %m "$LOCK_DIR") ))
        if (( lock_age > 3600 )); then
            rmdir "$LOCK_DIR" 2>/dev/null
            mkdir "$LOCK_DIR" 2>/dev/null || exit 0
        else
            exit 0
        fi
    fi
fi
trap 'rmdir "$LOCK_DIR" 2>/dev/null' EXIT

# Check if we've already run today (unless --force)
if [[ "$FORCE" == false && -f "$LAST_CHECK_FILE" ]]; then
    LAST_CHECK=$(cat "$LAST_CHECK_FILE")
    TODAY=$(date +%Y-%m-%d)
    if [[ "$LAST_CHECK" == "$TODAY" ]]; then
        exit 0
    fi
fi

# Display current settings
log "Displaying Claude Code user settings..."
if [[ "$QUIET" == false ]]; then
    "$SCRIPT_DIR"/claude-config list -e
fi

# Run claudeup with closed stdin and explicit wait
if [[ "$QUIET" == true ]]; then
    claudeup update > /dev/null 2>&1
    claudeup upgrade > /dev/null 2>&1
else
    claudeup update
    claudeup upgrade
fi

# Mark as checked today
date +%Y-%m-%d > "$LAST_CHECK_FILE"
