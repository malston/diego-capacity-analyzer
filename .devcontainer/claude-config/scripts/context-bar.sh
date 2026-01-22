#!/bin/bash
# ABOUTME: Claude Code statusline script displaying model, directory, git status, context, and cost.
# ABOUTME: Uses JSON input from Claude Code's statusline API with git caching for performance.

# Color theme: gray, orange, blue, teal, green, lavender, rose, gold, slate, cyan
# Preview colors with: bash scripts/color-preview.sh
COLOR="blue"

# Git cache settings (seconds)
GIT_CACHE_TTL=5
GIT_CACHE_FILE="/tmp/claude-statusline-git-cache-$$"

# Color codes
C_RESET='\033[0m'
C_GRAY='\033[38;5;245m'
C_DIM='\033[38;5;240m'
C_WHITE='\033[38;5;252m'
C_BAR_EMPTY='\033[38;5;238m'

# Semantic colors
C_CLEAN='\033[38;5;71m'    # green - clean/synced state
C_DIRTY='\033[38;5;173m'   # orange - uncommitted changes
C_WARN='\033[38;5;136m'    # gold - behind/diverged
C_COST='\033[38;5;139m'    # lavender - cost display

# Theme accent color
case "$COLOR" in
    orange)   C_ACCENT='\033[38;5;173m' ;;
    blue)     C_ACCENT='\033[38;5;74m' ;;
    teal)     C_ACCENT='\033[38;5;66m' ;;
    green)    C_ACCENT='\033[38;5;71m' ;;
    lavender) C_ACCENT='\033[38;5;139m' ;;
    rose)     C_ACCENT='\033[38;5;132m' ;;
    gold)     C_ACCENT='\033[38;5;136m' ;;
    slate)    C_ACCENT='\033[38;5;60m' ;;
    cyan)     C_ACCENT='\033[38;5;37m' ;;
    *)        C_ACCENT="$C_GRAY" ;;
esac

# Read JSON input once
input=$(cat)

# Extract core fields from JSON input
model=$(echo "$input" | jq -r '.model.display_name // .model.id // "?"')
cwd=$(echo "$input" | jq -r '.workspace.current_dir // .cwd // empty')
dir=$(basename "$cwd" 2>/dev/null || echo "?")

# Extract context window info directly from JSON (no transcript parsing needed)
max_context=$(echo "$input" | jq -r '.context_window.context_window_size // 200000')
max_k=$((max_context / 1000))

# Get current token usage from the API response
current_usage=$(echo "$input" | jq -r '.context_window.current_usage // empty')
if [[ -n "$current_usage" && "$current_usage" != "null" ]]; then
    input_tokens=$(echo "$current_usage" | jq -r '.input_tokens // 0')
    cache_creation=$(echo "$current_usage" | jq -r '.cache_creation_input_tokens // 0')
    cache_read=$(echo "$current_usage" | jq -r '.cache_read_input_tokens // 0')
    context_length=$((input_tokens + cache_creation + cache_read))
else
    # Baseline estimate when no usage data yet (~20k for system prompt, tools, memory)
    context_length=20000
fi

# Extract session cost
cost_usd=$(echo "$input" | jq -r '.cost.total_cost_usd // 0')

# Get transcript path for last message feature
transcript_path=$(echo "$input" | jq -r '.transcript_path // empty')

# --- Git Status with Caching ---
# Returns: branch|file_count|file_info|sync_state|sync_text
# sync_state: synced, ahead, behind, diverged, none
get_git_status() {
    local cwd="$1"
    local cache_file="${GIT_CACHE_FILE}-${cwd//\//_}"
    local now=$(date +%s)

    # Check cache validity
    if [[ -f "$cache_file" ]]; then
        local cache_time=$(head -1 "$cache_file")
        if [[ $((now - cache_time)) -lt $GIT_CACHE_TTL ]]; then
            tail -n +2 "$cache_file"
            return
        fi
    fi

    # Generate fresh git status
    local branch=""
    local file_count=0
    local file_info=""
    local sync_state="none"
    local sync_text=""

    branch=$(git -C "$cwd" branch --show-current 2>/dev/null)
    if [[ -n "$branch" ]]; then
        # Count uncommitted files
        file_count=$(git -C "$cwd" --no-optional-locks status --porcelain -uall 2>/dev/null | wc -l | tr -d ' ')

        # Get single filename if only one file
        if [[ "$file_count" -eq 1 ]]; then
            file_info=$(git -C "$cwd" --no-optional-locks status --porcelain -uall 2>/dev/null | head -1 | sed 's/^...//')
        fi

        # Check sync status with upstream
        local upstream=$(git -C "$cwd" rev-parse --abbrev-ref @{upstream} 2>/dev/null)
        if [[ -n "$upstream" ]]; then
            # Get last fetch time
            local fetch_head="$cwd/.git/FETCH_HEAD"
            local fetch_ago=""
            if [[ -f "$fetch_head" ]]; then
                local fetch_time=$(stat -f %m "$fetch_head" 2>/dev/null || stat -c %Y "$fetch_head" 2>/dev/null)
                if [[ -n "$fetch_time" ]]; then
                    local diff=$((now - fetch_time))
                    if [[ $diff -lt 60 ]]; then
                        fetch_ago="<1m"
                    elif [[ $diff -lt 3600 ]]; then
                        fetch_ago="$((diff / 60))m"
                    elif [[ $diff -lt 86400 ]]; then
                        fetch_ago="$((diff / 3600))h"
                    else
                        fetch_ago="$((diff / 86400))d"
                    fi
                fi
            fi

            local counts=$(git -C "$cwd" rev-list --left-right --count HEAD...@{upstream} 2>/dev/null)
            local ahead=$(echo "$counts" | cut -f1)
            local behind=$(echo "$counts" | cut -f2)
            if [[ "$ahead" -eq 0 && "$behind" -eq 0 ]]; then
                sync_state="synced"
                sync_text="✓${fetch_ago:+ $fetch_ago}"
            elif [[ "$ahead" -gt 0 && "$behind" -eq 0 ]]; then
                sync_state="ahead"
                sync_text="↑${ahead}"
            elif [[ "$ahead" -eq 0 && "$behind" -gt 0 ]]; then
                sync_state="behind"
                sync_text="↓${behind}"
            else
                sync_state="diverged"
                sync_text="↑${ahead}↓${behind}"
            fi
        fi
    fi

    # Write to cache (pipe-delimited for easy parsing)
    local result="${branch}|${file_count}|${file_info}|${sync_state}|${sync_text}"
    {
        echo "$now"
        echo "$result"
    } > "$cache_file"

    echo "$result"
}

# Get cached git status
branch=""
file_count=0
file_info=""
sync_state=""
sync_text=""
if [[ -n "$cwd" && -d "$cwd" ]]; then
    git_data=$(get_git_status "$cwd")
    IFS='|' read -r branch file_count file_info sync_state sync_text <<< "$git_data"
fi

# --- Build Context Bar ---
bar_width=10
if [[ "$context_length" -gt 0 ]]; then
    pct=$((context_length * 100 / max_context))
else
    pct=10  # Default ~10% for baseline
fi
[[ $pct -gt 100 ]] && pct=100

bar=""
for ((i=0; i<bar_width; i++)); do
    bar_start=$((i * 10))
    progress=$((pct - bar_start))
    if [[ $progress -ge 8 ]]; then
        bar+="${C_ACCENT}█${C_RESET}"
    elif [[ $progress -ge 3 ]]; then
        bar+="${C_ACCENT}▄${C_RESET}"
    else
        bar+="${C_BAR_EMPTY}░${C_RESET}"
    fi
done

ctx="${bar} ${C_GRAY}${pct}% of ${max_k}k"

# --- Format Cost ---
cost_display=""
if [[ "$cost_usd" != "0" && "$cost_usd" != "null" ]]; then
    # Format cost nicely (show cents for small amounts)
    if (( $(echo "$cost_usd < 0.01" | bc -l) )); then
        cost_display=" ${C_DIM}|${C_RESET} ${C_COST}<\$0.01"
    elif (( $(echo "$cost_usd < 1" | bc -l) )); then
        cost_cents=$(printf "%.0f" $(echo "$cost_usd * 100" | bc -l))
        cost_display=" ${C_DIM}|${C_RESET} ${C_COST}${cost_cents}¢"
    else
        cost_formatted=$(printf "%.2f" "$cost_usd")
        cost_display=" ${C_DIM}|${C_RESET} ${C_COST}\$${cost_formatted}"
    fi
fi

# --- Build Git Status Display ---
git_display=""
if [[ -n "$branch" ]]; then
    # Branch name in accent color
    git_display="${C_ACCENT}${branch}${C_RESET}"

    # Combine clean/dirty state with sync status
    # Clean + synced: ✓ 27m (single checkmark with time)
    # Clean + ahead/behind: ✓ ↑1 or ✓ ↓2
    # Dirty + synced: *3 27m
    # Dirty + ahead/behind: *3 ↑1 or *3 ↓2
    if [[ "$file_count" -eq 0 ]]; then
        # Clean state
        if [[ "$sync_state" == "synced" ]]; then
            # Extract just the time from sync_text (remove the ✓)
            sync_time="${sync_text#✓}"
            git_display+=" ${C_CLEAN}✓${sync_time}${C_RESET}"
        elif [[ -n "$sync_text" ]]; then
            git_display+=" ${C_CLEAN}✓${C_RESET}"
            case "$sync_state" in
                ahead)    git_display+=" ${C_ACCENT}${sync_text}${C_RESET}" ;;
                behind)   git_display+=" ${C_WARN}${sync_text}${C_RESET}" ;;
                diverged) git_display+=" ${C_WARN}${sync_text}${C_RESET}" ;;
            esac
        else
            git_display+=" ${C_CLEAN}✓${C_RESET}"
        fi
    else
        # Dirty state
        if [[ "$file_count" -eq 1 ]]; then
            git_display+=" ${C_DIRTY}*${file_info}${C_RESET}"
        else
            git_display+=" ${C_DIRTY}*${file_count}${C_RESET}"
        fi
        # Add sync status
        if [[ -n "$sync_text" ]]; then
            case "$sync_state" in
                synced)   sync_time="${sync_text#✓}"; git_display+=" ${C_DIM}${sync_time}${C_RESET}" ;;
                ahead)    git_display+=" ${C_ACCENT}${sync_text}${C_RESET}" ;;
                behind)   git_display+=" ${C_WARN}${sync_text}${C_RESET}" ;;
                diverged) git_display+=" ${C_WARN}${sync_text}${C_RESET}" ;;
            esac
        fi
    fi
fi

# --- Build Output ---
output="${C_ACCENT}${model}${C_RESET} ${C_DIM}|${C_RESET} ${C_WHITE}${dir}${C_RESET}"
[[ -n "$git_display" ]] && output+=" ${C_DIM}|${C_RESET} ${git_display}"
output+=" ${C_DIM}|${C_RESET} ${ctx}${cost_display}${C_RESET}"

printf '%b\n' "$output"

# --- Optional: Last User Message (second line) ---
if [[ -n "$transcript_path" && -f "$transcript_path" ]]; then
    # Calculate max length for truncation (rough estimate)
    plain_output="${model} | ${dir} | ${branch} *${file_count} ${sync_text} | xxxxxxxxxx ${pct}% of ${max_k}k"
    max_len=${#plain_output}

    last_user_msg=$(jq -rs '
        def is_unhelpful:
            startswith("[Request interrupted") or
            startswith("[Request cancelled") or
            . == "";

        [.[] | select(.type == "user") |
         select(.message.content | type == "string" or
                (type == "array" and any(.[]; .type == "text")))] |
        reverse |
        map(.message.content |
            if type == "string" then .
            else [.[] | select(.type == "text") | .text] | join(" ") end |
            gsub("\n"; " ") | gsub("  +"; " ")) |
        map(select(is_unhelpful | not)) |
        first // ""
    ' < "$transcript_path" 2>/dev/null)

    if [[ -n "$last_user_msg" ]]; then
        if [[ ${#last_user_msg} -gt $max_len ]]; then
            echo "> ${last_user_msg:0:$((max_len - 3))}..."
        else
            echo "> ${last_user_msg}"
        fi
    fi
fi
