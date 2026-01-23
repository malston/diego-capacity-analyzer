#!/usr/bin/env bash
# ABOUTME: Simple script to build and run the Diego Capacity Analyzer demo
# ABOUTME: Starts backend, frontend, and optionally serves presentation slides

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[OK]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# Cleanup function
cleanup() {
    log_info "Shutting down demo..."
    if [[ -n "${BACKEND_PID:-}" ]]; then
        kill "$BACKEND_PID" 2>/dev/null || true
    fi
    if [[ -n "${FRONTEND_PID:-}" ]]; then
        kill "$FRONTEND_PID" 2>/dev/null || true
    fi
    if [[ -n "${SLIDES_PID:-}" ]]; then
        kill "$SLIDES_PID" 2>/dev/null || true
    fi
    log_success "Demo stopped"
}

trap cleanup EXIT

# Check dependencies
check_deps() {
    local missing=()

    if ! command -v go &>/dev/null; then
        missing+=("go")
    fi

    if ! command -v node &>/dev/null; then
        missing+=("node")
    fi

    # Prefer bun over npm if available
    if command -v bun &>/dev/null; then
        PKG_MANAGER="bun"
    elif command -v npm &>/dev/null; then
        PKG_MANAGER="npm"
    else
        missing+=("bun or npm")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing dependencies: ${missing[*]}"
        exit 1
    fi

    log_success "Dependencies OK (using $PKG_MANAGER)"
}

# Install frontend dependencies if needed
install_deps() {
    cd "$PROJECT_ROOT/frontend"
    if [[ ! -d "node_modules" ]]; then
        log_info "Installing frontend dependencies..."
        $PKG_MANAGER install
    fi
}

# Start backend
start_backend() {
    log_info "Starting backend on port ${BACKEND_PORT:-8080}..."
    cd "$PROJECT_ROOT"
    go build -o ./backend/diego-analyzer ./backend/... 2>/dev/null
    ./backend/diego-analyzer &
    BACKEND_PID=$!
    sleep 2

    if kill -0 "$BACKEND_PID" 2>/dev/null; then
        log_success "Backend running (PID: $BACKEND_PID)"
    else
        log_error "Backend failed to start"
        exit 1
    fi
}

# Start frontend
start_frontend() {
    log_info "Starting frontend on port ${FRONTEND_PORT:-5173}..."
    cd "$PROJECT_ROOT/frontend"
    $PKG_MANAGER run dev &
    FRONTEND_PID=$!
    sleep 3

    if kill -0 "$FRONTEND_PID" 2>/dev/null; then
        log_success "Frontend running (PID: $FRONTEND_PID)"
    else
        log_error "Frontend failed to start"
        exit 1
    fi
}

# Serve presentation slides
start_slides() {
    log_info "Starting slides server on port ${SLIDES_PORT:-8888}..."
    cd "$PROJECT_ROOT/docs"
    python3 -m http.server "${SLIDES_PORT:-8888}" &
    SLIDES_PID=$!
    sleep 1

    if kill -0 "$SLIDES_PID" 2>/dev/null; then
        log_success "Slides server running (PID: $SLIDES_PID)"
    else
        log_warn "Slides server failed to start (python3 not available?)"
    fi
}

# Open browser
open_browser() {
    local url="$1"
    if command -v open &>/dev/null; then
        open "$url"
    elif command -v xdg-open &>/dev/null; then
        xdg-open "$url"
    else
        log_info "Open in browser: $url"
    fi
}

# Main
main() {
    echo ""
    echo "╔═══════════════════════════════════════════════╗"
    echo "║     Diego Capacity Analyzer - Demo Mode       ║"
    echo "╚═══════════════════════════════════════════════╝"
    echo ""

    check_deps
    install_deps
    start_backend
    start_frontend

    # Optionally start slides server
    if [[ "${SLIDES:-false}" == "true" ]]; then
        start_slides
    fi

    echo ""
    log_success "Demo is ready!"
    echo ""
    echo "  📊 Dashboard:    http://localhost:${FRONTEND_PORT:-5173}"
    echo "  🔌 Backend API:  http://localhost:${BACKEND_PORT:-8080}/api/v1/health"
    if [[ "${SLIDES:-false}" == "true" ]]; then
        echo "  📽️  Slides:       http://localhost:${SLIDES_PORT:-8888}/demo/"
    fi
    echo ""
    echo "  Press Ctrl+C to stop the demo"
    echo ""

    # Open browser to dashboard
    if [[ "${NO_BROWSER:-false}" != "true" ]]; then
        sleep 1
        open_browser "http://localhost:${FRONTEND_PORT:-5173}"
    fi

    # Wait for user interrupt
    wait
}

# Help
if [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]]; then
    cat <<EOF
Usage: ./run-demo.sh [options]

Starts the Diego Capacity Analyzer in demo mode.

Environment variables:
  BACKEND_PORT   Backend server port (default: 8080)
  FRONTEND_PORT  Frontend dev server port (default: 5173)
  SLIDES_PORT    Slides HTTP server port (default: 8888)
  SLIDES=true    Also start the presentation slides server
  NO_BROWSER=true  Don't auto-open browser

Examples:
  ./run-demo.sh                    # Start demo, open browser
  SLIDES=true ./run-demo.sh        # Also serve presentation slides
  NO_BROWSER=true ./run-demo.sh    # Don't open browser

Presentation slides available at:
  docs/demo/demo-slides.html         # R&D presentation
  docs/demo/feature-walkthrough.html # Feature walkthrough

EOF
    exit 0
fi

main
