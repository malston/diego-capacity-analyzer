#!/usr/bin/env bash
# ABOUTME: Installs git pre-commit hook to block direct commits to main/master.
# ABOUTME: Handles both regular repos and git worktrees.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$REPO_ROOT"

# Validate git repository (works with worktrees)
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "[SKIP] Not a git repository"
    exit 0
fi

GIT_DIR="$(git rev-parse --git-dir)"
HOOKS_DIR="$GIT_DIR/hooks"

mkdir -p "$HOOKS_DIR"

# Install pre-commit hook
cat > "$HOOKS_DIR/pre-commit" << 'EOF'
#!/bin/bash
BRANCH=$(git branch --show-current)
if [ "$BRANCH" = "main" ] || [ "$BRANCH" = "master" ]; then
    echo "[BLOCKED] Direct commits to '$BRANCH' not allowed"
    echo "Create a feature branch: git checkout -b feature/your-feature"
    exit 1
fi
exit 0
EOF

chmod +x "$HOOKS_DIR/pre-commit"
echo "[OK] Git pre-commit hook installed"
