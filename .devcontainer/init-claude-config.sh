#!/usr/bin/env bash
# ABOUTME: Initializes Claude Code configuration from templates.
# ABOUTME: Configures git identity, dotfiles, MCP servers, and settings.

set -euo pipefail

CLAUDE_HOME="/home/node/.claude"
MCP_FILE="$CLAUDE_HOME/mcp.json"
MCP_TEMPLATE="/usr/local/share/claude-defaults/mcp.json"
SETTINGS_FILE="$CLAUDE_HOME/settings.json"
SETTINGS_TEMPLATE="/usr/local/share/claude-defaults/settings.json"
DOTFILES_DIR="/home/node/dotfiles"

echo "Initializing Claude Code configuration..."

# Configure git identity if env vars are set
# Run from /tmp to avoid worktree .git file issues
pushd /tmp > /dev/null
if [ -n "${GIT_USER_NAME:-}" ]; then
    git config --global user.name "$GIT_USER_NAME"
    echo "[OK] Git user.name: $GIT_USER_NAME"
fi

if [ -n "${GIT_USER_EMAIL:-}" ]; then
    git config --global user.email "$GIT_USER_EMAIL"
    echo "[OK] Git user.email: $GIT_USER_EMAIL"
fi

# Configure GitHub token for git operations
if [ -n "${GITHUB_TOKEN:-}" ]; then
    git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"
    echo "[OK] GitHub token configured for git"
fi
popd > /dev/null

# Clone dotfiles if repo is set and directory is empty
if [ -n "${DOTFILES_REPO:-}" ] && [ -z "$(ls -A "$DOTFILES_DIR" 2>/dev/null)" ]; then
    echo "Cloning dotfiles from $DOTFILES_REPO (branch: ${DOTFILES_BRANCH:-main})..."
    git clone --branch "${DOTFILES_BRANCH:-main}" "$DOTFILES_REPO" "$DOTFILES_DIR"
    echo "[OK] Dotfiles cloned to $DOTFILES_DIR"

    # Run install script if it exists
    if [ -f "$DOTFILES_DIR/install.sh" ]; then
        echo "Running dotfiles install script..."
        cd "$DOTFILES_DIR" && chmod +x install.sh && ./install.sh
        echo "[OK] Dotfiles install script completed"
    fi
elif [ -n "${DOTFILES_REPO:-}" ]; then
    echo "[SKIP] Dotfiles directory not empty, preserving"
fi

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
