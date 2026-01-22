#!/usr/bin/env bash
# .claude/scripts/run-claude-command.sh
# Clean version with permission bypass

set -euo pipefail

COMMAND="${1:-/help}"
SESSION_NAME="${2:-claude-cmd-$$}"
KEEP_ALIVE="${3:-false}"
TIMEOUT=60

# Create session with our full PATH already set
tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true
tmux new-session -d -s "$SESSION_NAME" "export PATH='$PATH' && exec \$SHELL"

# Wait for shell to be ready
sleep 1

# Start Claude with permission bypass
echo "Starting Claude..." >&2
tmux send-keys -t "$SESSION_NAME" 'claude --dangerously-skip-permissions' Enter

# Wait for Claude prompt
echo "Waiting for Claude prompt..." >&2
for i in {1..30}; do
    if tmux capture-pane -t "$SESSION_NAME" -p | grep -q "cluster01"; then
        echo "Claude ready!" >&2
        break
    fi
    sleep 1
done

sleep 2

# Clear screen and history
tmux send-keys -t "$SESSION_NAME" C-l
sleep 0.5
tmux clear-history -t "$SESSION_NAME"
sleep 0.5

# Send command
echo "Sending: $COMMAND" >&2
tmux send-keys -t "$SESSION_NAME" "$COMMAND" Enter

# Wait for completion
echo "Waiting for completion..." >&2
for i in $(seq 1 "$TIMEOUT"); do
    pane=$(tmux capture-pane -t "$SESSION_NAME" -p)

    # Done when no spinner and prompt is back
    if ! echo "$pane" | grep -q "∙" && echo "$pane" | grep -q "❯"; then
        echo "Completed after ${i}s!" >&2
        sleep 2
        break
    fi

    if (( i % 10 == 0 )); then
        echo "Still waiting... (${i}/${TIMEOUT}s)" >&2
    fi
    sleep 1
done

# Capture output
tmux capture-pane -t "$SESSION_NAME" -p -S -100

# Cleanup or keep alive
if [[ "$KEEP_ALIVE" == "true" ]]; then
    echo "" >&2
    echo "Session: tmux attach -t $SESSION_NAME" >&2
else
    tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true
fi