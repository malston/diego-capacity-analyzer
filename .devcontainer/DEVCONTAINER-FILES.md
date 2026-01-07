# Dev Container Configuration Guide

This document describes each file in the `.devcontainer/` directory and how to customize them.

## File Overview

| File | Purpose | When to Modify |
|------|---------|----------------|
| `devcontainer.json` | Main configuration | Adding extensions, changing resources, modifying volumes |
| `Dockerfile` | Container image definition | Adding system packages, tools |
| `Makefile` | CLI make targets | Adding/modifying container management commands |
| `devcontainer.sh` | CLI wrapper script | Changing devcontainer CLI behavior |
| `init-claude-config.sh` | Git identity, dotfiles, MCP setup | Changing git config, dotfiles, or MCP defaults |
| `init-claude-hooks.sh` | Claude hooks deployment | Changing hook installation behavior |
| `init-claudeup.sh` | Plugin manager setup | Changing claudeup installation or profile |
| `docker-profile.json` | Claudeup plugin definitions | Adding/removing plugins or marketplaces |
| `setup-git-hooks.sh` | Git pre-commit hook | Modifying branch protection rules |
| `mcp.json.template` | MCP server definitions | Adding/removing MCP servers |
| `settings.json.template` | Claude Code settings | Changing timeouts, environment variables |
| `session-start.sh.template` | Startup validation hook | Customizing session startup checks |

---

## devcontainer.json

**Purpose**: Main VS Code Dev Container configuration file.

### Key Sections

```json
{
  "name": "...",           // Container name shown in VS Code
  "build": {},             // Dockerfile and build args
  "features": {},          // Dev container features (git, go, etc.)
  "hostRequirements": {},  // CPU, memory, storage minimums
  "customizations": {},    // VS Code extensions and settings
  "mounts": [],            // Volume mounts for persistence
  "postCreateCommand": "", // Runs once after container creation
  "postStartCommand": ""   // Runs every time container starts
}
```

### Common Modifications

**Add a VS Code extension:**
```json
"customizations": {
  "vscode": {
    "extensions": [
      "anthropic.claude-code",
      "your.new-extension"  // Add here
    ]
  }
}
```

**Add a volume mount** (persists data across rebuilds):
```json
"mounts": [
  "source=mydata-${devcontainerId},target=/home/node/.mydata,type=volume"
]
```

**Change resource requirements:**
```json
"hostRequirements": {
  "cpus": 4,         // Minimum CPU cores
  "memory": "8gb",   // Minimum RAM
  "storage": "32gb"  // Minimum disk
}
```

**Add environment variables:**
```json
"containerEnv": {
  "MY_VAR": "my_value"
}
```

---

## Dockerfile

**Purpose**: Defines the container image with system packages and configuration.

### Structure

```dockerfile
FROM node:22                    # Base image

# Install system packages
RUN apt-get update && apt-get install -y \
    package1 \
    package2

# Create directories
RUN mkdir -p /path/to/dir

# Copy scripts and templates
COPY script.sh /usr/local/bin/
COPY template.json /usr/local/share/claude-defaults/

# Set permissions
RUN chmod +x /usr/local/bin/script.sh

USER node  # Run as non-root user
```

### Common Modifications

**Add a system package:**
```dockerfile
RUN apt-get update && apt-get install -y \
    # existing packages...
    your-new-package \  # Add here
    && apt-get clean && rm -rf /var/lib/apt/lists/*
```

**Add a custom script:**
1. Create the script in `.devcontainer/`
2. Add to Dockerfile:
```dockerfile
COPY your-script.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/your-script.sh
```

**Install a tool from URL:**
```dockerfile
RUN curl -LsSf https://example.com/install.sh | sh
```

---

## init-claude-config.sh

**Purpose**: Configures git identity, clones dotfiles, and copies Claude Code configuration templates.

### What It Does

1. **Git identity** (from environment variables):
   - Sets `git config --global user.name` from `$GIT_USER_NAME`
   - Sets `git config --global user.email` from `$GIT_USER_EMAIL`
   - Configures GitHub token URL rewriting from `$GITHUB_TOKEN`

2. **Dotfiles** (if `$DOTFILES_REPO` is set):
   - Clones dotfiles repo to `/home/node/dotfiles`
   - Uses `$DOTFILES_BRANCH` (default: main)
   - Runs `install.sh` if present

3. **Claude Code config** (if not exists):
   - Copies `mcp.json.template` → `/home/node/.claude/mcp.json`
   - Copies `settings.json.template` → `/home/node/.claude/settings.json`

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `GIT_USER_NAME` | Git commit author name |
| `GIT_USER_EMAIL` | Git commit author email |
| `GITHUB_TOKEN` | Enables `https://github.com/` → `https://TOKEN@github.com/` rewriting |
| `DOTFILES_REPO` | URL of dotfiles repository to clone |
| `DOTFILES_BRANCH` | Branch to checkout (default: main) |

### When to Modify

- To add more git configuration
- To change dotfiles installation behavior
- To modify template copy logic

---

## init-claude-hooks.sh

**Purpose**: Deploys Claude Code hooks to the workspace.

### What It Does

1. Creates `.claude/hooks/` in the workspace
2. Copies `session-start.sh.template` to the workspace hooks directory

### When to Modify

- To change where hooks are installed
- To add additional hooks
- To change the idempotency behavior

### Add a New Hook

1. Create `your-hook.sh.template` in `.devcontainer/`
2. Add COPY in Dockerfile:
```dockerfile
COPY --chmod=755 your-hook.sh.template /usr/local/share/claude-defaults/hooks/your-hook.sh
```
3. Add deployment in `init-claude-hooks.sh`:
```bash
if [ ! -f "$WORKSPACE_HOOKS/your-hook.sh" ]; then
    cp "$TEMPLATE_DIR/your-hook.sh" "$WORKSPACE_HOOKS/"
    chmod +x "$WORKSPACE_HOOKS/your-hook.sh"
fi
```

---

## setup-git-hooks.sh

**Purpose**: Installs git pre-commit hook to prevent direct commits to main/master.

### What It Does

1. Detects if running in a git repository
2. Creates `.git/hooks/pre-commit` script
3. The hook blocks commits when on `main` or `master` branch

### When to Modify

**Add more protected branches:**
```bash
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ] || [ "$BRANCH" = "production" ]; then
```

**Add pre-commit checks** (linting, tests):
```bash
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/bash
BRANCH=$(git branch --show-current)

# Branch protection
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
    echo "[BLOCKED] Direct commits to '$BRANCH' not allowed"
    exit 1
fi

# Run linter
npm run lint || exit 1

# Run tests
npm test || exit 1

exit 0
EOF
```

---

## mcp.json.template

**Purpose**: Defines MCP (Model Context Protocol) servers available to Claude Code.

### Structure

```json
{
  "mcpServers": {
    "server-name": {
      "command": "npx",
      "args": ["-y", "@package/name"],
      "env": {
        "API_KEY": "${ENV_VAR}"
      }
    }
  }
}
```

### Common Modifications

**Add an MCP server:**
```json
{
  "mcpServers": {
    "existing-server": { ... },
    "new-server": {
      "command": "npx",
      "args": ["-y", "@your/mcp-package"],
      "env": {}
    }
  }
}
```

**Remove an MCP server:** Delete the entire server block.

**Available MCP Servers** (examples):
- `@upstash/context7-mcp` - Documentation context
- `mcp-remote https://docs.mcp.cloudflare.com/mcp` - Cloudflare docs
- `@executeautomation/chromemcp` - Chrome DevTools automation

---

## settings.json.template

**Purpose**: Claude Code application settings and environment.

### Structure

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "dangerously_skip_permissions": true,
  "verbose": true,
  "hooks": { ... },
  "env": { ... },
  "includeCoAuthoredBy": false
}
```

### Key Settings

| Setting | Purpose | Values |
|---------|---------|--------|
| `dangerously_skip_permissions` | Skip permission prompts | `true`/`false` |
| `verbose` | Enable verbose logging | `true`/`false` |
| `includeCoAuthoredBy` | Add co-author to commits | `true`/`false` |

### Environment Variables

```json
"env": {
  "MAX_MCP_OUTPUT_TOKENS": "60000",    // Max tokens from MCP servers
  "BASH_DEFAULT_TIMEOUT_MS": "300000", // Default bash timeout (5 min)
  "BASH_MAX_TIMEOUT_MS": "600000"      // Max bash timeout (10 min)
}
```

### Common Modifications

**Increase timeouts:**
```json
"env": {
  "BASH_DEFAULT_TIMEOUT_MS": "600000",  // 10 minutes
  "BASH_MAX_TIMEOUT_MS": "900000"       // 15 minutes
}
```

**Add custom environment variable:**
```json
"env": {
  "MY_CUSTOM_VAR": "value"
}
```

---

## session-start.sh.template

**Purpose**: Runs at the start of each Claude Code session to validate the environment.

### What It Does

1. Detects if running in a devcontainer
2. Checks volume mounts
3. Lists available MCP servers
4. Verifies tool availability
5. Shows workspace and git info

### When to Modify

**Add a tool check:**
```bash
echo "Tools:"
for tool in claude gh node go your-tool; do
    if command -v "$tool" &> /dev/null; then
        echo "  [OK] $tool"
    else
        echo "  [MISSING] $tool"
    fi
done
```

**Add a volume check:**
```bash
for vol in ".claude" ".config/gh" ".your-data"; do
    # ...
done
```

**Add custom startup logic:**
```bash
# After the Ready! message
echo ""
echo "Custom startup tasks..."
your-startup-command
```

---

## Makefile

**Purpose**: Provides make targets for container management via CLI.

### Targets

| Target | Description |
|--------|-------------|
| `make help` | Show all targets |
| `make build` | Build container image |
| `make rebuild` | Rebuild with no cache |
| `make up` | Start container |
| `make shell` | Open interactive shell |
| `make stop` | Stop container |
| `make down` | Stop and remove container |
| `make status` | Show container status |
| `make run CMD="..."` | Run command in container |

### Usage

```bash
cd .devcontainer
make rebuild
make up
make shell
```

Or from project root: `make -C .devcontainer <target>`

---

## devcontainer.sh

**Purpose**: Wrapper script for devcontainer CLI with environment variable passthrough.

### What It Does

1. Wraps `devcontainer` CLI commands (build, up, exec, etc.)
2. Passes environment variables via `--remote-env` flags
3. Provides colored output and status messages

### Environment Variables Passed

- `GIT_USER_NAME`
- `GIT_USER_EMAIL`
- `GITHUB_TOKEN`
- `DOTFILES_REPO`
- `DOTFILES_BRANCH`
- `CONTEXT7_API_KEY`

### When to Modify

- To pass additional environment variables
- To change default CLI behavior
- To add new commands

---

## init-claudeup.sh

**Purpose**: Installs claudeup plugin manager and applies the docker profile.

### What It Does

1. Checks for setup completion marker (`~/.claudeup/.setup-complete`)
2. Installs claudeup if not present
3. Copies `docker-profile.json` to `~/.claudeup/profiles/docker.json`
4. Runs `claudeup setup --profile docker -y`
5. Creates marker file to prevent re-running

### When to Modify

- To change the plugin profile applied
- To add post-installation steps
- To change marker file location

---

## docker-profile.json

**Purpose**: Defines marketplaces and plugins for claudeup to install.

### Structure

```json
{
  "name": "docker",
  "description": "Default plugins for Docker environment",
  "marketplaces": [
    { "source": "github", "repo": "org/repo" },
    { "source": "git", "url": "https://..." }
  ],
  "plugins": [
    "plugin-name@marketplace-name"
  ]
}
```

### Current Marketplaces (8)

- anthropics/claude-plugins-official
- anthropics/claude-code
- obra/superpowers-marketplace
- thedotmack/claude-mem
- ccplugins/awesome-claude-code-plugins
- EveryInc/compound-engineering-plugin
- anthropics/skills
- wshobson/agents

### Current Plugins (16)

- superpowers, episodic-memory, elements-of-style (superpowers-marketplace)
- claude-mem (thedotmack)
- code-review, code-documentation, commit-commands, feature-dev (official)
- frontend-design, pr-review-toolkit, security-guidance (official)
- code-review, commit-commands, feature-dev, hookify, plugin-dev (claude-code-plugins)

### When to Modify

- To add/remove marketplaces
- To add/remove plugins
- To create a different plugin profile

---

## Rebuilding the Container

After modifying any file:

```bash
cd .devcontainer
make rebuild
make up
```

Or in VS Code: `Cmd+Shift+P` → "Dev Containers: Rebuild Container"

---

## Troubleshooting

### Container won't start

1. Check Docker is running
2. Check memory requirements in `devcontainer.json`
3. View logs: `devcontainer up --workspace-folder . 2>&1`

### Changes not taking effect

1. Rebuild container (changes to Dockerfile/devcontainer.json)
2. Delete volumes to reset config: `docker volume rm <volume-name>`

### MCP servers not working

1. Check `/home/node/.claude/mcp.json` exists
2. Verify required environment variables are set
3. Run `claude mcp list` in container

### Git hooks not working

1. Verify `.git/hooks/pre-commit` exists and is executable
2. Check if in a git worktree (hooks are in different location)
