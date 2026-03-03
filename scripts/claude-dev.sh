#!/bin/bash
# Start a tmux session with Claude Code and monitoring panes
# for diego-capacity-analyzer development.
#
# Layout:
#   +----------------------------+----------------+
#   |                            | backend-dev    |
#   |   Claude Code              | (watchexec/air)|
#   |                            |                |
#   +----------------------------+----------------+
#   | Go test watcher            | frontend-dev   |
#   | (entr -c go test ./...)    | (vite :5173)   |
#   +----------------------------+----------------+

set -euo pipefail

SESSION="diego-cap"
DIR="/Users/markalston/code/diego-capacity-analyzer"

if tmux has-session -t "$SESSION" 2>/dev/null; then
    echo "Session '$SESSION' already exists. Attaching..."
    exec tmux attach -t "$SESSION"
fi

# Create session -- capture pane IDs from each split-window via -P -F
CLAUDE_PANE=$(tmux new-session -d -s "$SESSION" -c "$DIR" -P -F '#{pane_id}')
tmux send-keys -t "$CLAUDE_PANE" "claude" Enter

# Backend dev server (top right, 35% width)
BACKEND_PANE=$(tmux split-window -h -t "$CLAUDE_PANE" -c "$DIR" -p 35 -P -F '#{pane_id}')
tmux send-keys -t "$BACKEND_PANE" "make backend-dev" Enter

# Go test watcher (bottom left, 35% height)
TEST_PANE=$(tmux split-window -v -t "$CLAUDE_PANE" -c "$DIR" -p 35 -P -F '#{pane_id}')
tmux send-keys -t "$TEST_PANE" \
    "find backend cli -name '*.go' | entr -c sh -c 'cd backend && go test ./... && cd ../cli && go test ./...'" Enter

# Frontend dev server (bottom right, 50% of backend pane height)
FRONTEND_PANE=$(tmux split-window -v -t "$BACKEND_PANE" -c "$DIR" -p 50 -P -F '#{pane_id}')
tmux send-keys -t "$FRONTEND_PANE" "make frontend-dev" Enter

# Focus Claude pane
tmux select-pane -t "$CLAUDE_PANE"

tmux attach -t "$SESSION"
