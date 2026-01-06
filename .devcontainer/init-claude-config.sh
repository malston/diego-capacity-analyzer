#!/usr/bin/env bash
# ABOUTME: Initializes Claude Code configuration from templates.
# ABOUTME: Copies MCP servers and settings if they don't already exist.

set -euo pipefail

CLAUDE_HOME="/home/node/.claude"
MCP_FILE="$CLAUDE_HOME/mcp.json"
MCP_TEMPLATE="/usr/local/share/claude-defaults/mcp.json"
SETTINGS_FILE="$CLAUDE_HOME/settings.json"
SETTINGS_TEMPLATE="/usr/local/share/claude-defaults/settings.json"

echo "Initializing Claude Code configuration..."

# Create .claude directory if it doesn't exist
mkdir -p "$CLAUDE_HOME"

# Copy MCP configuration if it doesn't exist
if [ ! -f "$MCP_FILE" ] && [ -f "$MCP_TEMPLATE" ]; then
    cp "$MCP_TEMPLATE" "$MCP_FILE"
    echo "[OK] MCP servers configured"
else
    echo "[SKIP] MCP config exists, preserving"
fi

# Copy settings.json if it doesn't exist
if [ ! -f "$SETTINGS_FILE" ] && [ -f "$SETTINGS_TEMPLATE" ]; then
    cp "$SETTINGS_TEMPLATE" "$SETTINGS_FILE"
    echo "[OK] Claude settings initialized"
else
    echo "[SKIP] Settings exist, preserving"
fi

# Ensure npm global directory exists
mkdir -p /home/node/.npm-global/lib

echo "Claude configuration complete"
