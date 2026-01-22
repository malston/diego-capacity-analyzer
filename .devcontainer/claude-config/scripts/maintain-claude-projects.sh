#!/usr/bin/env bash
# Maintenance script for ~/.claude
# Cleans up caches, build artifacts, and stale session data

set -e

CLAUDE_DIR="${HOME}/.claude"
DRY_RUN=false
VERBOSE=false
FORCE=false
SAFE_ONLY=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Maintenance script for ~/.claude cleanup

OPTIONS:
  -s, --safe          Safe mode: only delete debug and shell-snapshots
  -d, --dry-run       Show what would be deleted without actually deleting
  -v, --verbose       Show detailed information
  -f, --force         Skip confirmation prompt
  -h, --help          Show this help message

MODES:
  Default (no --safe):
    Deletes all caches: debug, shell-snapshots, file-history, cache,
    telemetry, statsig, paste-cache, .mypy_cache

  Safe mode (--safe):
    Only deletes: debug, shell-snapshots
    These are always regenerated and safe to remove.

EXAMPLES:
  $(basename "$0") --safe --dry-run   # Preview safe cleanup
  $(basename "$0") --safe             # Safe cleanup only
  $(basename "$0") --dry-run          # Preview full cleanup
  $(basename "$0") -f                 # Full cleanup without confirmation
EOF
  exit 0
}

log() {
  echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $*"
}

log_success() {
  echo -e "${GREEN}✓${NC} $*"
}

log_warning() {
  echo -e "${YELLOW}⚠${NC} $*"
}

log_error() {
  echo -e "${RED}✗${NC} $*" >&2
}

# Get size in bytes for accurate calculations
get_size_bytes() {
  du -sk "$1" 2>/dev/null | cut -f1 || echo 0
}

# Human readable size
human_size() {
  local bytes=$1
  if [[ $bytes -ge 1048576 ]]; then
    echo "$(( bytes / 1024 ))M"
  elif [[ $bytes -ge 1024 ]]; then
    echo "${bytes}K"
  else
    echo "${bytes}B"
  fi
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -s|--safe)    SAFE_ONLY=true; shift ;;
    -d|--dry-run) DRY_RUN=true; shift ;;
    -v|--verbose) VERBOSE=true; shift ;;
    -f|--force)   FORCE=true; shift ;;
    -h|--help)    usage ;;
    *)            log_error "Unknown option: $1"; usage ;;
  esac
done

# Check if directory exists
if [[ ! -d "$CLAUDE_DIR" ]]; then
  log_error "Directory not found: $CLAUDE_DIR"
  exit 1
fi

# Define cache directories by safety level
SAFE_CACHE_DIRS=(
  "debug"
  "shell-snapshots"
)

EXTRA_CACHE_DIRS=(
  "file-history"
  "cache"
  "telemetry"
  "statsig"
  "paste-cache"
  ".mypy_cache"
)

# Select which dirs to clean
if [[ "$SAFE_ONLY" == true ]]; then
  CACHE_DIRS=("${SAFE_CACHE_DIRS[@]}")
  MODE="safe"
else
  CACHE_DIRS=("${SAFE_CACHE_DIRS[@]}" "${EXTRA_CACHE_DIRS[@]}")
  MODE="full"
fi

log "Starting $MODE maintenance of $CLAUDE_DIR"
[[ "$DRY_RUN" == true ]] && log_warning "DRY RUN MODE - no files will be deleted"

INITIAL_BYTES=$(get_size_bytes "$CLAUDE_DIR")
log "Initial size: $(du -sh "$CLAUDE_DIR" | cut -f1)"
echo ""

# Confirmation prompt (unless dry-run or force)
if [[ "$DRY_RUN" == false && "$FORCE" == false ]]; then
  if [[ "$SAFE_ONLY" == true ]]; then
    echo "Will delete: ${SAFE_CACHE_DIRS[*]}"
  else
    echo "Will delete: ${CACHE_DIRS[*]}"
    echo ""
    log_warning "Full mode includes file-history, telemetry, etc."
    echo "Use --safe for conservative cleanup."
  fi
  echo ""
  read -p "Proceed with cleanup? [y/N] " -n 1 -r
  echo ""
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_warning "Aborted by user"
    exit 0
  fi
fi

TOTAL_RECLAIMED=0

# ============================================
# 1. Clean ~/.claude cache directories
# ============================================
log "Cleaning Claude Code cache directories..."

for dir in "${CACHE_DIRS[@]}"; do
  target="$CLAUDE_DIR/$dir"
  if [[ -d "$target" ]]; then
    size_kb=$(get_size_bytes "$target")
    size_human=$(du -sh "$target" 2>/dev/null | cut -f1)

    if [[ "$DRY_RUN" == true ]]; then
      log_warning "Would delete $dir ($size_human)"
    else
      rm -rf "$target"
      log_success "Deleted $dir ($size_human)"
      ((TOTAL_RECLAIMED += size_kb))
    fi
  elif [[ "$VERBOSE" == true ]]; then
    echo "  Skipping $dir (not found)"
  fi
done

echo ""

# ============================================
# 2. Clean build artifacts in projects (full mode only)
# ============================================
CLAUDE_PROJECTS="$CLAUDE_DIR/projects"

if [[ "$SAFE_ONLY" == false && -d "$CLAUDE_PROJECTS" ]]; then
  log "Cleaning build artifacts in projects..."

  BUILD_ARTIFACTS=(
    "node_modules"
    ".next"
    ".nuxt"
    "dist"
    "build"
    "__pycache__"
    ".pytest_cache"
    ".venv"
    "venv"
    ".gradle"
    "target"
    ".m2"
    ".cache"
    ".tmp"
    "coverage"
    ".nyc_output"
    ".eslintcache"
  )

  for pattern in "${BUILD_ARTIFACTS[@]}"; do
    matches=$(find "$CLAUDE_PROJECTS" -depth -type d -name "$pattern" 2>/dev/null)

    if [[ -n "$matches" ]]; then
      count=$(echo "$matches" | wc -l | tr -d ' ')
      size_kb=$(echo "$matches" | xargs du -sk 2>/dev/null | awk '{sum+=$1} END {print sum+0}')

      if [[ "$DRY_RUN" == true ]]; then
        log_warning "Would delete $count instance(s) of '$pattern' ($(human_size "$size_kb"))"
        if [[ "$VERBOSE" == true ]]; then
          echo "$matches" | while read -r d; do
            echo "  → $d"
          done
        fi
      else
        echo "$matches" | xargs rm -rf 2>/dev/null || true
        log_success "Deleted $count instance(s) of '$pattern' ($(human_size "$size_kb"))"
        ((TOTAL_RECLAIMED += size_kb))
      fi
    elif [[ "$VERBOSE" == true ]]; then
      echo "  No matches for: $pattern"
    fi
  done

  echo ""

  log "Cleaning empty directories in projects..."
  if [[ "$DRY_RUN" == true ]]; then
    empty_count=$(find "$CLAUDE_PROJECTS" -type d -empty 2>/dev/null | wc -l | tr -d ' ')
    [[ $empty_count -gt 0 ]] && log_warning "Would delete $empty_count empty directories"
  else
    find "$CLAUDE_PROJECTS" -depth -type d -empty -delete 2>/dev/null || true
    log_success "Cleaned empty directories"
  fi

  echo ""
fi

# ============================================
# Summary
# ============================================
FINAL_BYTES=$(get_size_bytes "$CLAUDE_DIR")

if [[ "$DRY_RUN" == true ]]; then
  log_warning "This was a dry run. Run without --dry-run to actually delete files."
else
  log_success "Maintenance complete!"
  echo ""
  echo "Summary:"
  echo "  Mode:            $MODE"
  echo "  Initial size:    $(human_size "$INITIAL_BYTES")"
  echo "  Final size:      $(human_size "$FINAL_BYTES")"
  echo "  Space reclaimed: $(human_size $((INITIAL_BYTES - FINAL_BYTES)))"
fi