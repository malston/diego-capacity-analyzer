# ABOUTME: Dev container image for Claude Code development environment.
# ABOUTME: Provides Node.js, essential tools, and Claude Code configuration scaffolding.

FROM node:22

ARG TZ=America/Denver
ENV TZ="$TZ"

# Essential development tools only
RUN apt-get update && apt-get install -y --no-install-recommends \
    # Core utilities
    curl \
    wget \
    ca-certificates \
    gnupg2 \
    # Version control
    git \
    git-lfs \
    gh \
    # Process tools
    procps \
    htop \
    # Text processing
    jq \
    yq \
    # Fast search (modern grep/find)
    ripgrep \
    fd-find \
    # Build essentials (for native npm modules)
    build-essential \
    make \
    # Python (for node-gyp)
    python3 \
    # Editors
    nano \
    vim \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Create python symlink
RUN update-alternatives --install /usr/bin/python python /usr/bin/python3 1

ARG USERNAME=node

# Persist bash history and configure PATH
RUN mkdir /commandhistory && \
    touch /commandhistory/.bash_history && \
    chown -R $USERNAME /commandhistory && \
    echo 'export PATH=$PATH:/home/node/.npm-global/bin:/home/node/.local/bin:/home/node/.bun/bin' >> /home/node/.bashrc && \
    echo 'export HISTFILE=/commandhistory/.bash_history' >> /home/node/.bashrc && \
    chown node:node /home/node/.bashrc

ENV DEVCONTAINER=true

# Create config directories
RUN mkdir -p /home/node/.config/gh /home/node/.claude /home/node/.claudeup/profiles /home/node/.claude-mem /home/node/dotfiles /home/node/.npm-global/lib /home/node/.aws /home/node/.local/bin && \
    chown -R node:node /home/node/.config /home/node/.claude /home/node/.claudeup /home/node/.claude-mem /home/node/dotfiles /home/node/.npm-global /home/node/.aws /home/node/.local

WORKDIR /workspaces

USER node

# npm global path
ENV NPM_CONFIG_PREFIX=/home/node/.npm-global
ENV PATH=$PATH:/home/node/.npm-global/bin:/home/node/.local/bin:/home/node/.bun/bin

# Install Bun (required by some Claude Code plugins)
RUN curl -fsSL https://bun.sh/install | bash

# Configure git to ignore Claude's local settings
RUN git config --global core.excludesfile ~/.gitignore_global && \
    echo ".claude/settings.local.json" > /home/node/.gitignore_global

USER root

# Create directory for Claude defaults
RUN mkdir -p /usr/local/share/claude-defaults/hooks && \
    chown -R node:node /usr/local/share/claude-defaults

# Copy scripts and templates
COPY --chmod=755 init-claude-config.sh /usr/local/bin/
COPY --chmod=755 init-claude-hooks.sh /usr/local/bin/
COPY --chmod=755 init-claudeup.sh /usr/local/bin/
COPY --chown=node:node mcp.json.template /usr/local/share/claude-defaults/mcp.json
COPY --chmod=755 session-start.sh.template /usr/local/share/claude-defaults/hooks/session-start.sh
COPY --chown=node:node docker-profile.json /usr/local/share/claude-defaults/docker-profile.json

# Copy Claude Code configuration snapshot
COPY --chown=node:node claude-config/settings.json /usr/local/share/claude-defaults/settings.json
COPY --chown=node:node claude-config/enabled.json /usr/local/share/claude-defaults/enabled.json
COPY --chown=node:node claude-config/CLAUDE.md.template /usr/local/share/claude-defaults/CLAUDE.md.template
COPY --chown=node:node claude-config/claude.json.template /usr/local/share/claude-defaults/claude.json.template
COPY --chown=node:node claude-config/.library /usr/local/share/claude-defaults/.library
COPY --chown=node:node --chmod=755 claude-config/scripts /usr/local/share/claude-defaults/scripts
COPY --chown=node:node claude-config/completions /usr/local/share/claude-defaults/completions

USER node
