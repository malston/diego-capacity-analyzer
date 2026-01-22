#!/usr/bin/env bash
# ABOUTME: Initializes Claude Code configuration from project templates.
# ABOUTME: Deploys settings, hooks, skills, agents, and output-styles to ~/.claude.

set -euo pipefail

CLAUDE_HOME="/home/node/.claude"
CONFIG_SOURCE="/usr/local/share/claude-defaults"
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
        echo "Running dotfiles install script${DOTFILES_INSTALL_ARGS:+ with args: $DOTFILES_INSTALL_ARGS}..."
        # shellcheck disable=SC2086 # Word splitting is intentional for args
        cd "$DOTFILES_DIR" && chmod +x install.sh && ./install.sh ${DOTFILES_INSTALL_ARGS:-}
        echo "[OK] Dotfiles install script completed"
    fi
elif [ -n "${DOTFILES_REPO:-}" ]; then
    echo "[SKIP] Dotfiles directory not empty, preserving"
fi

# Create .claude directory structure
mkdir -p "$CLAUDE_HOME"

# Deploy .library (source files for hooks, skills, agents, etc.)
if [ ! -d "$CLAUDE_HOME/.library" ] && [ -d "$CONFIG_SOURCE/.library" ]; then
    cp -r "$CONFIG_SOURCE/.library" "$CLAUDE_HOME/"
    echo "[OK] .library deployed (hooks, skills, agents, output-styles, commands)"
else
    echo "[SKIP] .library exists, preserving"
fi

# Deploy settings.json
if [ ! -f "$CLAUDE_HOME/settings.json" ] && [ -f "$CONFIG_SOURCE/settings.json" ]; then
    cp "$CONFIG_SOURCE/settings.json" "$CLAUDE_HOME/"
    echo "[OK] settings.json deployed"
else
    echo "[SKIP] settings.json exists, preserving"
fi

# Deploy enabled.json
if [ ! -f "$CLAUDE_HOME/enabled.json" ] && [ -f "$CONFIG_SOURCE/enabled.json" ]; then
    cp "$CONFIG_SOURCE/enabled.json" "$CLAUDE_HOME/"
    echo "[OK] enabled.json deployed"
else
    echo "[SKIP] enabled.json exists, preserving"
fi

# Deploy CLAUDE.md from template with USER_NAME substitution
if [ ! -f "$CLAUDE_HOME/CLAUDE.md" ] && [ -f "$CONFIG_SOURCE/CLAUDE.md.template" ]; then
    USER_NAME="${CLAUDE_USER_NAME:-Developer}"
    sed "s/{{USER_NAME}}/$USER_NAME/g" "$CONFIG_SOURCE/CLAUDE.md.template" > "$CLAUDE_HOME/CLAUDE.md"
    echo "[OK] CLAUDE.md deployed (personalized for $USER_NAME)"
else
    echo "[SKIP] CLAUDE.md exists, preserving"
fi

# Deploy MCP configuration
if [ ! -f "$CLAUDE_HOME/mcp.json" ] && [ -f "$CONFIG_SOURCE/mcp.json" ]; then
    cp "$CONFIG_SOURCE/mcp.json" "$CLAUDE_HOME/"
    echo "[OK] MCP servers configured"
else
    echo "[SKIP] MCP config exists, preserving"
fi

# Deploy Claude Code preferences (~/.claude.json at home level, not inside ~/.claude/)
CLAUDE_JSON="/home/node/.claude.json"
if [ ! -f "$CLAUDE_JSON" ] && [ -f "$CONFIG_SOURCE/claude.json.template" ]; then
    cp "$CONFIG_SOURCE/claude.json.template" "$CLAUDE_JSON"
    echo "[OK] Claude Code preferences deployed (theme, notifications)"
else
    echo "[SKIP] ~/.claude.json exists, preserving"
fi

# Create symlinks from top-level directories to .library
# This matches the structure Claude Code expects
create_symlinks() {
    local dir_name="$1"
    local source_dir="$CLAUDE_HOME/.library/$dir_name"
    local target_dir="$CLAUDE_HOME/$dir_name"

    if [ -d "$source_dir" ] && [ ! -d "$target_dir" ]; then
        mkdir -p "$target_dir"
        for item in "$source_dir"/*; do
            if [ -e "$item" ]; then
                local basename=$(basename "$item")
                ln -sf "../.library/$dir_name/$basename" "$target_dir/$basename"
            fi
        done
        echo "[OK] $dir_name/ symlinks created"
    elif [ -d "$target_dir" ]; then
        echo "[SKIP] $dir_name/ exists, preserving"
    fi
}

# Create symlinks for each category
create_symlinks "hooks"
create_symlinks "skills"
create_symlinks "agents"
create_symlinks "commands"
create_symlinks "output-styles"

# Ensure hook scripts are executable
if [ -d "$CLAUDE_HOME/.library/hooks" ]; then
    chmod +x "$CLAUDE_HOME/.library/hooks"/*.sh 2>/dev/null || true
    chmod +x "$CLAUDE_HOME/.library/hooks"/*.py 2>/dev/null || true
fi

# Deploy scripts folder
if [ ! -d "$CLAUDE_HOME/scripts" ] && [ -d "$CONFIG_SOURCE/scripts" ]; then
    cp -r "$CONFIG_SOURCE/scripts" "$CLAUDE_HOME/"
    chmod +x "$CLAUDE_HOME/scripts"/*.sh 2>/dev/null || true
    chmod +x "$CLAUDE_HOME/scripts/claude-config" 2>/dev/null || true
    echo "[OK] scripts/ deployed"
else
    echo "[SKIP] scripts/ exists, preserving"
fi

# Deploy completions folder
if [ ! -d "$CLAUDE_HOME/completions" ] && [ -d "$CONFIG_SOURCE/completions" ]; then
    cp -r "$CONFIG_SOURCE/completions" "$CLAUDE_HOME/"
    echo "[OK] completions/ deployed"
else
    echo "[SKIP] completions/ exists, preserving"
fi

# Add claude-config alias to bashrc if not present
BASHRC="/home/node/.bashrc"
if ! grep -q "alias claude-config=" "$BASHRC" 2>/dev/null; then
    echo "" >> "$BASHRC"
    echo "# Claude Code configuration management" >> "$BASHRC"
    echo "alias claude-config='~/.claude/scripts/claude-config'" >> "$BASHRC"
    echo "[OK] claude-config alias added to .bashrc"
else
    echo "[SKIP] claude-config alias exists"
fi

# Ensure npm global directory exists
mkdir -p /home/node/.npm-global/lib

echo "Claude configuration complete"
