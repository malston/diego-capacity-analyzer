#!/usr/bin/env bash
# ABOUTME: Installs claudeup and applies the docker profile for plugin management.
# ABOUTME: Uses marker file to prevent re-running setup on container restarts.

set -euo pipefail

CLAUDEUP_HOME="/home/node/.claudeup"
PROFILE_DIR="$CLAUDEUP_HOME/profiles"
PROFILE_FILE="$PROFILE_DIR/docker.json"
PROFILE_TEMPLATE="/usr/local/share/claude-defaults/docker-profile.json"
MARKER_FILE="$CLAUDEUP_HOME/.setup-complete"

echo "Initializing claudeup..."

# Skip if setup already completed
if [ -f "$MARKER_FILE" ]; then
    echo "[SKIP] Claudeup setup already complete"
    exit 0
fi

# Ensure directories exist
mkdir -p "$PROFILE_DIR"

# Install claudeup if not present
if ! command -v claudeup &> /dev/null; then
    echo "Installing claudeup..."
    curl -fsSL https://raw.githubusercontent.com/claudeup/claudeup/main/install.sh | bash
    # Add to PATH for this session
    export PATH="$HOME/.local/bin:$PATH"
    echo "[OK] claudeup installed"
else
    echo "[SKIP] claudeup already installed"
fi

# Copy docker profile if it doesn't exist
if [ ! -f "$PROFILE_FILE" ] && [ -f "$PROFILE_TEMPLATE" ]; then
    cp "$PROFILE_TEMPLATE" "$PROFILE_FILE"
    echo "[OK] Docker profile copied"
else
    echo "[SKIP] Docker profile exists, preserving"
fi

# Apply the docker profile
echo "Applying docker profile (this may take a few minutes)..."
if claudeup setup --profile docker -y; then
    echo "[OK] Docker profile applied"
    # Create marker file after successful setup
    touch "$MARKER_FILE"
    echo "[OK] Setup complete marker created"
else
    echo "[WARN] claudeup setup failed, will retry on next container start"
    exit 1
fi

echo "Claudeup initialization complete"
