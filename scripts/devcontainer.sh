#!/usr/bin/env bash
# ABOUTME: Helper script for devcontainer CLI operations
# ABOUTME: Wraps common commands for inner-loop development workflow

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[OK]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

usage() {
    cat <<EOF
Usage: $(basename "$0") <command> [options]

Commands:
  build       Build the dev container image
  rebuild     Rebuild the dev container image (no cache)
  up          Start the dev container
  run         Run a command in the running container
  shell       Open an interactive shell in the container
  stop        Stop the running container
  down        Stop and remove the container
  logs        Show container logs
  status      Show container status

Options:
  -h, --help  Show this help message

Examples:
  $(basename "$0") build              # Build the container
  $(basename "$0") rebuild            # Rebuild without cache
  $(basename "$0") up                 # Start the container
  $(basename "$0") run make test      # Run 'make test' in container
  $(basename "$0") shell              # Open interactive shell
  $(basename "$0") down               # Stop and remove container
EOF
}

check_devcontainer_cli() {
    if ! command -v devcontainer &>/dev/null; then
        log_error "devcontainer CLI not found. Install with: npm install -g @devcontainers/cli"
        exit 1
    fi
}

get_container_id() {
    docker ps -q --filter "label=devcontainer.local_folder=$WORKSPACE_ROOT" 2>/dev/null || true
}

cmd_build() {
    log_info "Building dev container..."
    devcontainer build --workspace-folder "$WORKSPACE_ROOT" "$@"
    log_success "Build complete"
}

cmd_rebuild() {
    log_info "Rebuilding dev container (no cache)..."
    devcontainer build --workspace-folder "$WORKSPACE_ROOT" --no-cache "$@"
    log_success "Rebuild complete"
}

cmd_up() {
    log_info "Starting dev container..."
    devcontainer up --workspace-folder "$WORKSPACE_ROOT" "$@"
    log_success "Container started"
}

cmd_run() {
    local container_id
    container_id=$(get_container_id)

    if [[ -z "$container_id" ]]; then
        log_error "No running container found. Run '$(basename "$0") up' first."
        exit 1
    fi

    if [[ $# -eq 0 ]]; then
        log_error "No command specified. Usage: $(basename "$0") run <command>"
        exit 1
    fi

    log_info "Running: $*"
    devcontainer exec --workspace-folder "$WORKSPACE_ROOT" "$@"
}

cmd_shell() {
    local container_id
    container_id=$(get_container_id)

    if [[ -z "$container_id" ]]; then
        log_error "No running container found. Run '$(basename "$0") up' first."
        exit 1
    fi

    log_info "Opening interactive shell..."
    devcontainer exec --workspace-folder "$WORKSPACE_ROOT" /bin/zsh
}

cmd_stop() {
    local container_id
    container_id=$(get_container_id)

    if [[ -z "$container_id" ]]; then
        log_warn "No running container found"
        return 0
    fi

    log_info "Stopping container $container_id..."
    docker stop "$container_id"
    log_success "Container stopped"
}

cmd_down() {
    local container_id
    container_id=$(get_container_id)

    if [[ -z "$container_id" ]]; then
        container_id=$(docker ps -qa --filter "label=devcontainer.local_folder=$WORKSPACE_ROOT" 2>/dev/null || true)
    fi

    if [[ -z "$container_id" ]]; then
        log_warn "No container found"
        return 0
    fi

    log_info "Stopping and removing container $container_id..."
    docker rm -f "$container_id"
    log_success "Container removed"
}

cmd_logs() {
    local container_id
    container_id=$(get_container_id)

    if [[ -z "$container_id" ]]; then
        log_error "No running container found"
        exit 1
    fi

    docker logs "$@" "$container_id"
}

cmd_status() {
    local container_id
    container_id=$(get_container_id)

    if [[ -z "$container_id" ]]; then
        echo "Status: Not running"
        local stopped_id
        stopped_id=$(docker ps -qa --filter "label=devcontainer.local_folder=$WORKSPACE_ROOT" 2>/dev/null || true)
        if [[ -n "$stopped_id" ]]; then
            echo "Stopped container: $stopped_id"
        fi
    else
        echo "Status: Running"
        echo "Container ID: $container_id"
        docker ps --format "table {{.ID}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" --filter "id=$container_id"
    fi
}

main() {
    if [[ $# -eq 0 ]]; then
        usage
        exit 1
    fi

    local command="$1"
    shift

    case "$command" in
        -h|--help) usage; exit 0 ;;
        build) check_devcontainer_cli; cmd_build "$@" ;;
        rebuild) check_devcontainer_cli; cmd_rebuild "$@" ;;
        up) check_devcontainer_cli; cmd_up "$@" ;;
        run) check_devcontainer_cli; cmd_run "$@" ;;
        shell) check_devcontainer_cli; cmd_shell ;;
        stop) cmd_stop ;;
        down) cmd_down ;;
        logs) cmd_logs "$@" ;;
        status) cmd_status ;;
        *) log_error "Unknown command: $command"; usage; exit 1 ;;
    esac
}

main "$@"
