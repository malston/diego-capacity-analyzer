#!/usr/bin/env bash
# ABOUTME: Sets up Claude Code hooks in the workspace.
# ABOUTME: Copies session-start hook template if it doesn't exist.

set -euo pipefail

# Get workspace folder from environment or detect it
if [ -n "${containerWorkspaceFolder:-}" ]; then
    WORKSPACE_FOLDER="$containerWorkspaceFolder"
elif [ -d /workspaces ]; then
    WORKSPACE_FOLDER=$(find /workspaces -maxdepth 1 -mindepth 1 -type d | head -1)
else
    echo "[WARN] Could not determine workspace folder, skipping hooks setup"
    exit 0
fi

WORKSPACE_HOOKS="$WORKSPACE_FOLDER/.claude/hooks"
TEMPLATE_DIR="/usr/local/share/claude-defaults/hooks"

echo "Setting up Claude Code hooks..."

mkdir -p "$WORKSPACE_HOOKS"

if [ ! -f "$WORKSPACE_HOOKS/session-start.sh" ] && [ -f "$TEMPLATE_DIR/session-start.sh" ]; then
    cp "$TEMPLATE_DIR/session-start.sh" "$WORKSPACE_HOOKS/"
    chmod +x "$WORKSPACE_HOOKS/session-start.sh"
    echo "[OK] Session-start hook installed"
else
    echo "[SKIP] Hook exists, preserving"
    chmod +x "$WORKSPACE_HOOKS/session-start.sh" 2>/dev/null || true
fi

chown -R node:node "$WORKSPACE_FOLDER/.claude" 2>/dev/null || true

echo "Hook setup complete"
