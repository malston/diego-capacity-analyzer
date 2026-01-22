#!/usr/bin/env bash
# ABOUTME: Automatically upgrade Claude Code and display changelog
# ABOUTME: Called by .envrc when entering the directory
# ABOUTME: Uses flock to prevent concurrent runs

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LAST_CHECK_FILE="$SCRIPT_DIR/../.last_claude_update_check"
LOCK_FILE="/tmp/claude-auto-upgrade.lock"

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
    [[ "$QUIET" == false ]] && echo "$@"
}

# Acquire lock (non-blocking)
exec 9>"$LOCK_FILE"
if ! flock -n 9; then
    [[ "$QUIET" == false ]] && echo "Claude upgrade already in progress, skipping..."
    exit 0
fi

# Check if we've already run today (unless --force)
if [[ "$FORCE" == false && -f "$LAST_CHECK_FILE" ]]; then
    LAST_CHECK=$(cat "$LAST_CHECK_FILE")
    TODAY=$(date +%Y-%m-%d)

    if [[ "$LAST_CHECK" == "$TODAY" ]]; then
        exit 0
    fi
fi

log "Checking for Claude Code updates..."

# Get current version before upgrading
OLD_VERSION=$(claude --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

# Run the upgrade
UPGRADE_OUTPUT=$(claude update 2>&1)

# Get new version after upgrading
NEW_VERSION=$(claude --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)

# Check if version changed
if [[ -n "$OLD_VERSION" && -n "$NEW_VERSION" && "$OLD_VERSION" != "$NEW_VERSION" ]]; then
    log ""
    log "✨ Claude Code upgraded: $OLD_VERSION → $NEW_VERSION"
    log ""
    log "Fetching changelog..."

    CHANGELOG_URL="https://raw.githubusercontent.com/anthropics/claude-code/refs/heads/main/CHANGELOG.md"
    VERSION_CHANGES=$(curl -sL "$CHANGELOG_URL" | python3 -c "
import sys, re
version = '${NEW_VERSION}'
changelog = sys.stdin.read()
sections = re.split(r'^## ', changelog, flags=re.MULTILINE)
for section in sections:
    if section.startswith(version):
        print(f'## {section.split(chr(10) + chr(10) + \"## \")[0].strip()}')
        break
")

    if [[ -n "$VERSION_CHANGES" ]]; then
        log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        log "$VERSION_CHANGES"
        log "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    else
        log "Full changelog: https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md"
    fi
elif echo "$UPGRADE_OUTPUT" | grep -q "is already installed"; then
    log "Claude Code is up to date ($OLD_VERSION)"
else
    log "$UPGRADE_OUTPUT"
fi

# Mark as checked today
date +%Y-%m-%d > "$LAST_CHECK_FILE"