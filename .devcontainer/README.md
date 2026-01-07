# DevContainer Configuration

VS Code and CLI development container for Claude Code development.

## Prerequisites

1. **Docker Desktop** (4.25+) running
2. **devcontainer CLI** - `npm install -g @devcontainers/cli`
3. Optional: **VS Code** with Dev Containers extension

## Quick Start (CLI)

```bash
cd .devcontainer

# Set environment variables (add to ~/.bashrc or ~/.zshrc)
export GIT_USER_NAME="Your Name"
export GIT_USER_EMAIL="your@email.com"
export GITHUB_TOKEN="ghp_..."
export DOTFILES_REPO="https://github.com/you/dotfiles.git"  # optional
export DOTFILES_BRANCH="main"  # optional

# Build and start
make rebuild
make up
make shell
```

## Quick Start (VS Code)

1. Open project in VS Code: `code .`
2. Click "Reopen in Container" when prompted
3. Wait for build (~2-3 minutes)
4. Verify: `claude --version`

## Make Targets

Run from `.devcontainer/` directory:

```bash
make help      # Show all targets
make build     # Build container image
make rebuild   # Rebuild (no cache)
make up        # Start container
make shell     # Open interactive shell
make stop      # Stop container
make down      # Stop and remove container
make status    # Show container status
make run CMD="make test"  # Run command in container
```

Or from project root: `make -C .devcontainer <target>`

## What Gets Installed

### AI Assistant
- **Claude Code** - Anthropic's AI coding assistant
- **claudeup** - Plugin manager with docker profile (17 plugins)

### Plugins (via docker-profile.json)
- superpowers, episodic-memory, claude-mem
- code-review, code-documentation, commit-commands
- feature-dev, frontend-design, pr-review-toolkit
- security-guidance, hookify, plugin-dev

### Development Tools
- **Languages**: Node.js 22, Go 1.23
- **Package Managers**: npm, bun
- **Version Control**: git, git-lfs, gh (GitHub CLI)
- **Search**: ripgrep (rg), fd-find
- **Text Processing**: jq, yq
- **Editors**: nano, vim

### Configuration
- **Skip Permission Prompts** - Claude Code runs with `dangerously_skip_permissions: true`
- **Extended Timeouts** - Bash commands default to 5 minutes, max 10 minutes
- **High Token Limits** - MCP output up to 60K tokens

## Environment Variables

Set these before starting the container:

| Variable | Required | Description |
|----------|----------|-------------|
| `GIT_USER_NAME` | Yes | Git commit author name |
| `GIT_USER_EMAIL` | Yes | Git commit author email |
| `GITHUB_TOKEN` | Yes | GitHub PAT for git operations |
| `DOTFILES_REPO` | No | Dotfiles repo URL to clone |
| `DOTFILES_BRANCH` | No | Dotfiles branch (default: main) |
| `CONTEXT7_API_KEY` | No | Context7 MCP server API key |

## Configuration Files

| File | Purpose |
|------|---------|
| `devcontainer.json` | Main devcontainer configuration |
| `Dockerfile` | Container image definition |
| `Makefile` | Make targets for container management |
| `devcontainer.sh` | CLI wrapper script |
| `init-claude-config.sh` | Git identity, dotfiles, MCP setup |
| `init-claude-hooks.sh` | Deploy Claude Code hooks |
| `init-claudeup.sh` | Install claudeup and apply plugin profile |
| `docker-profile.json` | Claudeup plugin/marketplace definitions |
| `setup-git-hooks.sh` | Install git pre-commit hook |
| `mcp.json.template` | MCP server configuration |
| `settings.json.template` | Claude Code settings |
| `session-start.sh.template` | Session startup validation |

For detailed documentation on each file, see [DEVCONTAINER-FILES.md](DEVCONTAINER-FILES.md).

## Persistent Volumes

Configuration persists across container rebuilds via named volumes:

```text
claude-code-bashhistory-${devcontainerId} → /commandhistory
claude-config-${devcontainerId}           → /home/node/.claude
claudeup-config-${devcontainerId}         → /home/node/.claudeup
ghub-config-${devcontainerId}             → /home/node/.config/gh
npm-global-${devcontainerId}              → /home/node/.npm-global
local-bin-${devcontainerId}               → /home/node/.local
cargo-${devcontainerId}                   → /home/node/.cargo
bun-${devcontainerId}                     → /home/node/.bun
aws-config-${devcontainerId}              → /home/node/.aws
dotfiles-${devcontainerId}                → /home/node/dotfiles
```

### Host Mounts

These directories are mounted from your host machine:

```text
~/.claude-mem  → /home/node/.claude-mem  (shared memory across containers)
~/.ssh         → /home/node/.ssh         (SSH keys, read-only)
```

## Troubleshooting

### Container Won't Start

**Check Docker is running:**
```bash
docker info
```

**Check memory requirements** (needs 4GB):
```bash
docker system info | grep Memory
```

### Environment Variables Not Set

The `${localEnv:...}` syntax only works with VS Code. For CLI usage, the `devcontainer.sh` script passes environment variables via `--remote-env` flags. Ensure variables are exported in your shell.

### MCP Servers Not Loading

```bash
# Check MCP configuration
cat ~/.claude/mcp.json

# Check Claude logs
claude --verbose
```

### Plugin Errors (bun not found)

Ensure Bun is in PATH. The container installs Bun to `/home/node/.bun/bin`. Check:
```bash
which bun
echo $PATH
```

### Git Worktree Issues

Git commands may fail with "not a git repository" errors due to worktree `.git` files. The init scripts run from `/tmp` to avoid this.

## Customization

### Add VS Code Extensions

Edit `devcontainer.json` → `customizations.vscode.extensions` array.

### Add System Packages

Edit `Dockerfile` and add to the `apt-get install` block.

### Add MCP Servers

Edit `mcp.json.template` and rebuild container.

### Add/Remove Plugins

Edit `docker-profile.json` and rebuild container.

## Further Reading

- [DEVCONTAINER-FILES.md](DEVCONTAINER-FILES.md) - Detailed file documentation
- [VS Code Dev Containers](https://code.visualstudio.com/docs/devcontainers/containers)
- [Claude Code Docs](https://docs.anthropic.com/en/docs/claude-code)
