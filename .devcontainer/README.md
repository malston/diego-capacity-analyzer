# DevContainer Configuration

VS Code development container for Claude Code development.

## Prerequisites

1. **VS Code** with Dev Containers extension
2. **Docker Desktop** (4.25+) running

## Quick Start

1. **Open project in VS Code**
   ```bash
   code .
   ```

2. **Reopen in Container**
   - Click "Reopen in Container" when prompted
   - OR: `Cmd/Ctrl+Shift+P` → "Dev Containers: Reopen in Container"

3. **Wait for first build** (~2-3 minutes)

4. **Verify installation**
   ```bash
   claude --version
   ```

## What Gets Installed

### AI Assistant
- **Claude Code** - Anthropic's AI coding assistant with MCP servers

### MCP Servers (Pre-configured)
- **Context7** - Library documentation (1000+ libraries)
- **Cloudflare Docs** - Cloudflare products documentation

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

## Configuration Files

| File | Purpose |
|------|---------|
| `devcontainer.json` | Main devcontainer configuration |
| `Dockerfile` | Container image definition |
| `init-claude-config.sh` | Initialize Claude Code MCP servers |
| `init-claude-hooks.sh` | Deploy Claude Code hooks to workspace |
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
ghub-config-${devcontainerId}             → /home/node/.config/gh
npm-global-${devcontainerId}              → /home/node/.npm-global
local-bin-${devcontainerId}               → /home/node/.local
cargo-${devcontainerId}                   → /home/node/.cargo
bun-${devcontainerId}                     → /home/node/.bun
aws-config-${devcontainerId}              → /home/node/.aws
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

### MCP Servers Not Loading

```bash
# Check MCP configuration
cat ~/.claude/mcp.json

# Check Claude logs
claude --verbose
```

### API Keys Not Persisting

Check volume exists:
```bash
docker volume ls | grep claude-config
```

## Customization

### Add VS Code Extensions

Edit `devcontainer.json` → `customizations.vscode.extensions` array.

### Add System Packages

Edit `Dockerfile` and add to the `apt-get install` block.

### Add MCP Servers

Edit `mcp.json.template` and rebuild container.

## Further Reading

- [DEVCONTAINER-FILES.md](DEVCONTAINER-FILES.md) - Detailed file documentation
- [VS Code Dev Containers](https://code.visualstudio.com/docs/devcontainers/containers)
- [Claude Code Docs](https://docs.anthropic.com/en/docs/claude-code)
