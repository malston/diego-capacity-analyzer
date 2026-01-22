# Claude Code Auto-Update Scripts

Automated scripts for keeping Claude Code and its plugins up to date.

## Scripts

### auto-upgrade-claude.sh

Automatically upgrades Claude Code and claudeup, then displays the changelog.

**Usage:**

```bash
# Normal upgrade (once per day)
./scripts/auto-upgrade-claude.sh

# Force upgrade even if already checked today
./scripts/auto-upgrade-claude.sh --force
```

**What it does:**

1. Checks if already run today (skips unless `--force`)
2. Upgrades Claude Code via `brew upgrade --cask claude-code`
3. Detects version changes and displays changelog from GitHub
4. Upgrades claudeup to the latest release
5. Records check date to avoid duplicate runs

**When to use:**

- Automatically called by `.envrc` when entering the directory (if configured)
- Manually run with `--force` to check for updates immediately

**Example output:**

```bash
Checking for Claude Code updates...

✨ Claude Code upgraded: 2.0.59 → 2.0.60

Fetching changelog...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
## 2.0.60

### Features
- Add support for custom output styles
- Improve hook system performance

### Bug Fixes
- Fix issue with MCP server initialization
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Checking for claudeup updates...
Upgrading claudeup: 0.5.0 → 0.6.0
✓ claudeup upgraded
```

---

### auto-update-plugins.sh

Checks for plugin and marketplace updates using claudeup.

**Usage:**

```bash
# Normal update check (once per day)
./scripts/auto-update-plugins.sh

# Force check even if already ran today
./scripts/auto-update-plugins.sh --force
```

**What it does:**

1. Checks if already run today (skips unless `--force`)
2. Runs `claudeup update` to check and prompt for updates
3. Records check date to avoid duplicate runs

**When to use:**

- Automatically called by `.envrc` when entering the directory (if configured)
- Manually run with `--force` to check for plugin updates immediately

**What claudeup update does:**

- Checks all installed marketplaces for updates
- Checks all installed plugins for updates
- Prompts to update outdated items
- Handles the update process automatically

---

## Setup for Auto-Updates

To enable automatic updates when entering the directory, create `.envrc` (gitignored):

```bash
#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run updates in background to avoid blocking shell
"$SCRIPT_DIR/scripts/auto-upgrade-claude.sh" &
"$SCRIPT_DIR/scripts/auto-update-plugins.sh" &
```

Then allow direnv:

```bash
direnv allow
```

**Requirements:**

- `direnv` installed: `brew install direnv` (macOS) or `sudo apt-get install direnv` (Ubuntu/Debian)
- Shell integration: Add `eval "$(direnv hook zsh)"` to `~/.zshrc` (or bash equivalent)

---

### find-mcp-servers.sh

Discovers and displays all MCP server configuration sources across different scopes.

**Usage:**

```bash
# Run the discovery script
~/.claude/scripts/find-mcp-servers.sh
```

**What it does:**

1. Checks user-scoped configuration (`~/.claude.json`)
2. Checks project-scoped configuration (`.mcp.json` in current directory)
3. Checks local-scoped configuration (project-specific in `~/.claude.json`)
4. Checks enterprise managed configuration (system-wide)
5. Scans installed plugins for bundled MCP servers

**When to use:**

- Troubleshooting: "Why am I seeing these MCP servers?"
- Documentation: Understanding your MCP server setup
- Auditing: Verifying which plugins provide which MCP servers

**What it shows:**

For each scope, the script displays:

- Whether configuration exists
- Server names and types (stdio, http, sse)
- Commands or URLs for each server
- Plugin-bundled servers (inline or in `.mcp.json`)

**Example output:**

```bash
=== MCP Server Configuration Discovery ===

[1] User Scope (Global)
    File: ~/.claude.json
  File exists but no mcpServers configured

[2] Project Scope (Current Project)
    File: /path/to/project/.mcp.json
  Not found

[3] Local Scope (Project-Specific in User Config)
    Path: Project-specific entries in ~/.claude.json
  No local-scoped servers for this project

[4] Enterprise Managed (System-Wide)
    File: /Library/Application Support/ClaudeCode/managed-mcp.json
  Not configured

[5] Installed Plugins
    Checking: /Users/you/.claude/plugins

  Plugin: my-plugin
    Inline MCP servers in plugin.json:
      - server-name-1
      - server-name-2
```

---

## Manual Plugin Management

For manual plugin management, use `claudeup` directly:

```bash
# Check for updates
claudeup update

# List installed plugins
claudeup list

# Update a specific marketplace
claude plugin marketplace update <marketplace-name>

# Reinstall a plugin
claude plugin uninstall <plugin-name>@<marketplace>
claude plugin install <plugin-name>@<marketplace>
```

For more information, run `claudeup --help` or visit:
<https://github.com/malston/claudeup>

---

## Files Created

- `~/.claude/.last_brew_check` - Tracks last Claude Code upgrade check
- `~/.claude/.last_plugin_check` - Tracks last plugin update check

These files prevent the scripts from running multiple times per day.

## TMUX

These commands automate running Claude Code in a tmux session and capturing its output. Here's what each does:

**Command breakdown:**

1. `tmux kill-session -t test-session 2>/dev/null` - Kills any existing session (suppresses errors if none exists)
2. `tmux new-session -d -s test-session` - Creates a new detached session named "test-session"
3. `tmux send-keys -t test-session 'claude' Enter` - Starts Claude Code CLI
4. `sleep 2` - Waits for Claude to initialize
5. `tmux send-keys -t test-session '/context' Enter` - Sends the `/context` command
6. `sleep 1` - Waits for command to execute
7. `tmux capture-pane -t test-session -p` - Captures and prints the pane content

## Bash Scripts

### Option 1: Simple wrapper script

```bash
#!/usr/bin/env bash
# run-claude-context.sh
# Executes a Claude command in tmux and captures output

set -euo pipefail

SESSION_NAME="${1:-claude-session}"
COMMAND="${2:-/context}"
STARTUP_WAIT="${3:-2}"
COMMAND_WAIT="${4:-1}"

# Clean up any existing session
tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true

# Create new detached session
tmux new-session -d -s "$SESSION_NAME"

# Start Claude
tmux send-keys -t "$SESSION_NAME" 'claude' Enter
sleep "$STARTUP_WAIT"

# Send command
tmux send-keys -t "$SESSION_NAME" "$COMMAND" Enter
sleep "$COMMAND_WAIT"

# Capture and print output
tmux capture-pane -t "$SESSION_NAME" -p

# Optional: keep session alive or kill it
# tmux kill-session -t "$SESSION_NAME"
```

**Usage:**

```bash
chmod +x run-claude-context.sh
./run-claude-context.sh                    # Uses defaults
./run-claude-context.sh my-session         # Custom session name
./run-claude-context.sh my-session /help 3 2  # All custom params
```

### Option 2: Enhanced script with options

```bash
#!/usr/bin/env bash
# claude-tmux-runner.sh
# Advanced wrapper for running Claude commands in tmux

set -euo pipefail

# Defaults
SESSION_NAME="claude-session"
COMMAND="/context"
STARTUP_WAIT=2
COMMAND_WAIT=1
KEEP_SESSION=false
VERBOSE=false

# Usage function
usage() {
    cat << EOF
Usage: $(basename "$0") [OPTIONS]

Run Claude Code commands in a tmux session and capture output.

OPTIONS:
    -s, --session NAME      Session name (default: claude-session)
    -c, --command CMD       Command to send (default: /context)
    -w, --startup-wait SEC  Wait time for Claude startup (default: 2)
    -d, --command-wait SEC  Wait time for command execution (default: 1)
    -k, --keep-session      Keep tmux session alive after capture
    -v, --verbose           Show debug output
    -h, --help              Show this help message

EXAMPLES:
    $(basename "$0") --command /help
    $(basename "$0") -s my-session -c "/context" -k
    $(basename "$0") -c "/tools" -w 3 -d 2 --verbose
EOF
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--session)
            SESSION_NAME="$2"
            shift 2
            ;;
        -c|--command)
            COMMAND="$2"
            shift 2
            ;;
        -w|--startup-wait)
            STARTUP_WAIT="$2"
            shift 2
            ;;
        -d|--command-wait)
            COMMAND_WAIT="$2"
            shift 2
            ;;
        -k|--keep-session)
            KEEP_SESSION=true
            shift
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1"
            usage
            ;;
    esac
done

# Logging function
log() {
    if [[ "$VERBOSE" == true ]]; then
        echo "[DEBUG] $*" >&2
    fi
}

# Main execution
main() {
    log "Session: $SESSION_NAME, Command: $COMMAND"

    # Kill existing session
    log "Cleaning up existing session..."
    tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true

    # Create new session
    log "Creating new tmux session..."
    tmux new-session -d -s "$SESSION_NAME"

    # Start Claude
    log "Starting Claude (waiting ${STARTUP_WAIT}s)..."
    tmux send-keys -t "$SESSION_NAME" 'claude' Enter
    sleep "$STARTUP_WAIT"

    # Send command
    log "Sending command: $COMMAND (waiting ${COMMAND_WAIT}s)..."
    tmux send-keys -t "$SESSION_NAME" "$COMMAND" Enter
    sleep "$COMMAND_WAIT"

    # Capture output
    log "Capturing pane output..."
    tmux capture-pane -t "$SESSION_NAME" -p

    # Cleanup
    if [[ "$KEEP_SESSION" == false ]]; then
        log "Killing session..."
        tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true
    else
        log "Session kept alive: tmux attach -t $SESSION_NAME"
    fi
}

main
```

**Usage:**

```bash
chmod +x claude-tmux-runner.sh
./claude-tmux-runner.sh --help
./claude-tmux-runner.sh -c /tools -v
./claude-tmux-runner.sh -s well-fargo-session -c "/context" -k
```

### Option 3: Function for your `.bashrc`

```bash
# Add to ~/.bashrc or ~/.bash_profile
claude-tmux() {
    local session="${1:-claude-tmp-$$}"
    local command="${2:-/context}"
    local startup_wait="${3:-2}"
    local command_wait="${4:-1}"

    tmux kill-session -t "$session" 2>/dev/null || true
    tmux new-session -d -s "$session"
    tmux send-keys -t "$session" 'claude' Enter
    sleep "$startup_wait"
    tmux send-keys -t "$session" "$command" Enter
    sleep "$command_wait"
    tmux capture-pane -t "$session" -p
    tmux kill-session -t "$session" 2>/dev/null || true
}

# Usage: claude-tmux [session] [command] [startup_wait] [command_wait]
```

The enhanced script (Option 2) is probably most useful for your consulting work - it gives you flexibility and clean output capture for automation workflows. Would you like me to add features like output file saving or multiple command execution?
