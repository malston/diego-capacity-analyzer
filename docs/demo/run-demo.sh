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

# Check if a port is in use
port_in_use() {
    local port="$1"
    if command -v lsof &>/dev/null; then
        lsof -i :"$port" &>/dev/null
    elif command -v nc &>/dev/null; then
        nc -z localhost "$port" &>/dev/null
    else
        # Fallback: try to connect with bash
        (echo >/dev/tcp/localhost/"$port") 2>/dev/null
    fi
}

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
    local port="${BACKEND_PORT:-8080}"
    if port_in_use "$port"; then
        log_warn "Backend port $port already in use -- skipping (using existing service)"
        return 0
    fi

    log_info "Starting backend on port $port (auth disabled for demo)..."
    cd "$PROJECT_ROOT/backend"
    go build -o capacity-backend . || { log_error "Backend build failed"; exit 1; }
    AUTH_MODE=disabled ./capacity-backend &
    BACKEND_PID=$!
    cd "$PROJECT_ROOT"
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
    local port="${FRONTEND_PORT:-3000}"
    if port_in_use "$port"; then
        log_warn "Frontend port $port already in use -- skipping (using existing service)"
        return 0
    fi

    log_info "Starting frontend on port $port..."
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
    local port="${SLIDES_PORT:-8888}"
    if port_in_use "$port"; then
        log_warn "Slides port $port already in use -- skipping (using existing service)"
        return 0
    fi

    log_info "Starting slides server on port $port..."
    cd "$PROJECT_ROOT/docs"
    python3 -m http.server "$port" &
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
    if [[ "${SLIDES:-true}" == "true" ]]; then
        start_slides
    fi

    echo ""
    log_success "Demo is ready!"
    echo ""
    echo "  📊 Dashboard:    http://localhost:${FRONTEND_PORT:-3000}"
    echo "  🔌 Backend API:  http://localhost:${BACKEND_PORT:-8080}/api/v1/health"
    if [[ "${SLIDES:-true}" == "true" ]]; then
        echo "  📽️  Slides:       http://localhost:${SLIDES_PORT:-8888}/demo/"
    fi
    echo ""

    # Check if we started any processes
    local started_any=false
    if [[ -n "${BACKEND_PID:-}" ]] || [[ -n "${FRONTEND_PID:-}" ]] || [[ -n "${SLIDES_PID:-}" ]]; then
        started_any=true
    fi

    if [[ "$started_any" == "true" ]]; then
        echo "  Press Ctrl+C to stop the demo"
        echo ""

        # Open browser to dashboard
        if [[ "${NO_BROWSER:-false}" != "true" ]]; then
            sleep 1
            open_browser "http://localhost:${FRONTEND_PORT:-3000}"
        fi

        # Wait for user interrupt
        wait
    else
        echo "  All services already running -- nothing to manage"
        echo ""

        # Open browser to dashboard
        if [[ "${NO_BROWSER:-false}" != "true" ]]; then
            open_browser "http://localhost:${FRONTEND_PORT:-3000}"
        fi
    fi
}

# Help
if [[ "${1:-}" == "-h" ]] || [[ "${1:-}" == "--help" ]]; then
    cat <<EOF
Usage: ./run-demo.sh [options]

Starts the Diego Capacity Analyzer in demo mode.
Authentication is disabled (AUTH_MODE=disabled) so all features
work without credentials.

Environment variables:
  BACKEND_PORT   Backend server port (default: 8080)
  FRONTEND_PORT  Frontend dev server port (default: 3000)
  SLIDES_PORT    Slides HTTP server port (default: 8888)
  SLIDES=false   Skip starting the presentation slides server
  NO_BROWSER=true  Don't auto-open browser

Examples:
  ./run-demo.sh                    # Start demo with slides, open browser
  SLIDES=false ./run-demo.sh       # Skip presentation slides server
  NO_BROWSER=true ./run-demo.sh    # Don't open browser

Presentation slides available at:
  docs/demo/demo-slides.html         # R&D presentation
  docs/demo/feature-walkthrough.html # Feature walkthrough

EOF
    exit 0
fi

main
