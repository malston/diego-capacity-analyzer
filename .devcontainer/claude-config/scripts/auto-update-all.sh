#!/usr/bin/env bash
# ABOUTME: Auto-update Claude Code, plugins, and marketplaces
# ABOUTME: Uses mkdir for atomic locking (portable)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOCK_DIR="/tmp/claude-auto-update.lock"

FORCE=false
QUIET=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        --force) FORCE=true; shift ;;
        --quiet) QUIET=true; shift ;;
        *) shift ;;
    esac
done

# Acquire lock (mkdir is atomic)
if ! mkdir "$LOCK_DIR" 2>/dev/null; then
    # Check if lock is stale (older than 1 hour)
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

# Clean up lock on exit
trap 'rmdir "$LOCK_DIR" 2>/dev/null' EXIT

# Build args array
args=()
[[ "$FORCE" == true ]] && args+=("--force")
[[ "$QUIET" == true ]] && args+=("--quiet")

# Run both updates
"$SCRIPT_DIR/auto-upgrade-claude.sh" "${args[@]}"
"$SCRIPT_DIR/auto-update-plugins.sh" "${args[@]}"