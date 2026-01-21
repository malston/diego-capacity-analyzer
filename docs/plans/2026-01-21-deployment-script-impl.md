# Deployment Script Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create an automated deployment script that follows docs/DEPLOYMENT.md with checkpoint/resume support.

**Architecture:** Single bash script with modular phase functions, state persistence via `.state/deploy-state`, config loading from `config/deploy.conf` with environment variable fallback.

**Tech Stack:** Bash (strict mode), integrates with cf CLI, om CLI, existing generate-env.sh

---

## Task 1: Update .gitignore

**Files:**

- Modify: `.gitignore`

**Step 1: Add ignore patterns for state and config**

Add to `.gitignore`:

```
# Deployment script state and config
.state/
config/deploy.conf
```

**Step 2: Verify patterns added**

Run: `grep -E "^\.state|^config/deploy\.conf" .gitignore`
Expected: Both patterns shown

**Step 3: Commit**

```bash
git add .gitignore
git commit -m "chore: add .state/ and config/deploy.conf to gitignore"
```

---

## Task 2: Create config directory and template

**Files:**

- Create: `config/deploy.conf.example`

**Step 1: Create config directory**

Run: `mkdir -p config`

**Step 2: Write config template**

Create `config/deploy.conf.example`:

```bash
# ABOUTME: Template configuration for deployment script.
# ABOUTME: Copy to deploy.conf and fill in values for your environment.

# Ops Manager credentials (required)
# Used by generate-env.sh to derive all other credentials
OM_TARGET=opsman.example.com
OM_USERNAME=admin
OM_PASSWORD=

# Optional: Use client credentials instead of username/password
# OM_CLIENT_ID=
# OM_CLIENT_SECRET=

# Optional: For non-routable BOSH networks (SSH tunnel through Ops Manager)
# OM_PRIVATE_KEY=~/.ssh/opsman_key

# Optional: Skip SSL validation (not recommended for production)
OM_SKIP_SSL_VALIDATION=false

# CF target org/space for deployment (default: system/system)
CF_ORG=system
CF_SPACE=system

# App names (default: capacity-backend, capacity-ui)
BACKEND_APP_NAME=capacity-backend
FRONTEND_APP_NAME=capacity-ui
```

**Step 3: Verify file created**

Run: `head -5 config/deploy.conf.example`
Expected: ABOUTME comments and OM_TARGET line

**Step 4: Commit**

```bash
git add config/deploy.conf.example
git commit -m "feat: add deployment config template"
```

---

## Task 3: Create deploy.sh skeleton with argument parsing

**Files:**

- Create: `scripts/deploy.sh`

**Step 1: Create scripts directory**

Run: `mkdir -p scripts`

**Step 2: Write script skeleton with strict mode and argument parsing**

Create `scripts/deploy.sh`:

```bash
#!/usr/bin/env bash
# ABOUTME: Automated deployment script for TAS Capacity Analyzer.
# ABOUTME: Deploys backend and frontend to Cloud Foundry with checkpoint/resume support.

set -Eeuo pipefail

#######################################
# Constants
#######################################
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly CONFIG_FILE="$PROJECT_ROOT/config/deploy.conf"
readonly STATE_DIR="$PROJECT_ROOT/.state"
readonly STATE_FILE="$STATE_DIR/deploy-state"

#######################################
# Default configuration values
#######################################
: "${CF_ORG:=system}"
: "${CF_SPACE:=system}"
: "${BACKEND_APP_NAME:=capacity-backend}"
: "${FRONTEND_APP_NAME:=capacity-ui}"
: "${OM_SKIP_SSL_VALIDATION:=false}"

#######################################
# Script options
#######################################
VERBOSE=false
DRY_RUN=false
FRESH=false
SKIP_PREREQS=false
PHASE=""

#######################################
# Current execution state
#######################################
CURRENT_PHASE=""

#######################################
# Logging functions
#######################################
log_info() {
    echo -e "\033[0;34m[INFO]\033[0m $*"
}

log_success() {
    echo -e "\033[0;32m[OK]\033[0m   $*"
}

log_warn() {
    echo -e "\033[0;33m[WARN]\033[0m $*"
}

log_error() {
    echo -e "\033[0;31m[ERROR]\033[0m $*" >&2
}

log_debug() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "\033[0;90m[DEBUG]\033[0m $*"
    fi
}

#######################################
# Usage
#######################################
usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Deploy TAS Capacity Analyzer to Cloud Foundry.

Options:
    -h, --help          Show this help message
    -v, --verbose       Enable verbose output
    -n, --dry-run       Show what would be done without executing
    -f, --fresh         Clear state and run from scratch
    --skip-prereqs      Skip prerequisite checks
    --phase=PHASE       Run only specified phase:
                        prereqs, env, backend, frontend, verify

Phases:
    prereqs     Check required tools and connectivity
    env         Generate .env credentials via generate-env.sh
    backend     Deploy backend service to CF
    frontend    Build and deploy frontend to CF
    verify      Verify deployment health

Examples:
    $(basename "$0")                    # Run all phases
    $(basename "$0") --phase=backend    # Run only backend deployment
    $(basename "$0") --fresh            # Start fresh, ignore previous state
    $(basename "$0") --dry-run          # Preview without executing

Configuration:
    Copy config/deploy.conf.example to config/deploy.conf and fill in values.
    Alternatively, set environment variables directly.

EOF
    exit 0
}

#######################################
# Argument parsing
#######################################
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -n|--dry-run)
                DRY_RUN=true
                shift
                ;;
            -f|--fresh)
                FRESH=true
                shift
                ;;
            --skip-prereqs)
                SKIP_PREREQS=true
                shift
                ;;
            --phase=*)
                PHASE="${1#*=}"
                shift
                ;;
            --phase)
                PHASE="$2"
                shift 2
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                ;;
            *)
                log_error "Unexpected argument: $1"
                usage
                ;;
        esac
    done

    # Validate phase if specified
    if [[ -n "$PHASE" ]]; then
        case "$PHASE" in
            prereqs|env|backend|frontend|verify)
                ;;
            *)
                log_error "Invalid phase: $PHASE"
                log_error "Valid phases: prereqs, env, backend, frontend, verify"
                exit 1
                ;;
        esac
    fi
}

#######################################
# Main entry point (placeholder)
#######################################
main() {
    parse_args "$@"

    log_info "TAS Capacity Analyzer Deployment Script"
    log_info "Project root: $PROJECT_ROOT"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_warn "Dry-run mode: no changes will be made"
    fi

    if [[ "$VERBOSE" == "true" ]]; then
        log_debug "Verbose mode enabled"
    fi

    if [[ -n "$PHASE" ]]; then
        log_info "Running single phase: $PHASE"
    fi

    log_info "Argument parsing complete (implementation pending)"
}

main "$@"
```

**Step 3: Make script executable**

Run: `chmod +x scripts/deploy.sh`

**Step 4: Test argument parsing**

Run: `./scripts/deploy.sh --help`
Expected: Usage message displayed

Run: `./scripts/deploy.sh --verbose --dry-run`
Expected: Shows info messages with verbose and dry-run warnings

Run: `./scripts/deploy.sh --phase=invalid`
Expected: Error about invalid phase

**Step 5: Commit**

```bash
git add scripts/deploy.sh
git commit -m "feat: add deploy.sh skeleton with argument parsing"
```

---

## Task 4: Add configuration loading

**Files:**

- Modify: `scripts/deploy.sh`

**Step 1: Add config loading function after logging functions**

Insert after the logging functions section:

```bash
#######################################
# Configuration loading
#######################################
load_config() {
    # Load from config file if it exists
    if [[ -f "$CONFIG_FILE" ]]; then
        log_info "Loading config from $CONFIG_FILE"
        # shellcheck source=/dev/null
        source "$CONFIG_FILE"
    else
        log_debug "No config file found at $CONFIG_FILE, using environment variables"
    fi

    # Apply defaults for any unset variables
    : "${CF_ORG:=system}"
    : "${CF_SPACE:=system}"
    : "${BACKEND_APP_NAME:=capacity-backend}"
    : "${FRONTEND_APP_NAME:=capacity-ui}"
    : "${OM_SKIP_SSL_VALIDATION:=false}"

    # Export for child processes (generate-env.sh)
    export OM_TARGET OM_USERNAME OM_PASSWORD OM_SKIP_SSL_VALIDATION
    export OM_CLIENT_ID OM_CLIENT_SECRET OM_PRIVATE_KEY
    export CF_ORG CF_SPACE BACKEND_APP_NAME FRONTEND_APP_NAME
}
```

**Step 2: Update main() to call load_config**

Update main() to call load_config after parse_args:

```bash
main() {
    parse_args "$@"

    log_info "TAS Capacity Analyzer Deployment Script"
    log_info "Project root: $PROJECT_ROOT"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_warn "Dry-run mode: no changes will be made"
    fi

    load_config

    if [[ "$VERBOSE" == "true" ]]; then
        log_debug "Verbose mode enabled"
        log_debug "CF_ORG=$CF_ORG"
        log_debug "CF_SPACE=$CF_SPACE"
        log_debug "BACKEND_APP_NAME=$BACKEND_APP_NAME"
        log_debug "FRONTEND_APP_NAME=$FRONTEND_APP_NAME"
    fi

    if [[ -n "$PHASE" ]]; then
        log_info "Running single phase: $PHASE"
    fi

    log_info "Configuration loaded (implementation pending)"
}
```

**Step 3: Test config loading**

Run: `./scripts/deploy.sh --verbose`
Expected: Shows default values for CF_ORG, CF_SPACE, etc.

Run: `CF_ORG=myorg ./scripts/deploy.sh --verbose`
Expected: Shows CF_ORG=myorg

**Step 4: Commit**

```bash
git add scripts/deploy.sh
git commit -m "feat: add configuration loading to deploy.sh"
```

---

## Task 5: Add state management functions

**Files:**

- Modify: `scripts/deploy.sh`

**Step 1: Add state management functions after config loading**

Insert after load_config function:

```bash
#######################################
# State management
#######################################
state_init() {
    mkdir -p "$STATE_DIR"
    if [[ -f "$STATE_FILE" ]]; then
        log_debug "Loading state from $STATE_FILE"
        # shellcheck source=/dev/null
        source "$STATE_FILE"
    fi
}

state_get() {
    local key="$1"
    local var_name="STATE_$key"
    echo "${!var_name:-}"
}

state_set() {
    local key="$1"
    local value="$2"
    local var_name="STATE_$key"

    # Set in memory
    declare -g "$var_name=$value"

    # Persist to file
    if [[ "$DRY_RUN" != "true" ]]; then
        # Rewrite entire state file
        {
            echo "# Auto-generated by scripts/deploy.sh - do not edit manually"
            echo "# Last updated: $(date -Iseconds)"
            for var in $(compgen -v | grep "^STATE_"); do
                echo "$var=\"${!var}\""
            done
        } > "$STATE_FILE"
    fi
}

state_is_complete() {
    local phase="$1"
    [[ "$(state_get "$phase")" == "complete" ]]
}

state_clear() {
    log_info "Clearing deployment state"
    if [[ "$DRY_RUN" != "true" ]]; then
        rm -f "$STATE_FILE"
    fi
    # Clear in-memory state
    for var in $(compgen -v | grep "^STATE_"); do
        unset "$var"
    done
}
```

**Step 2: Update main() to initialize state**

Update main() to handle state:

```bash
main() {
    parse_args "$@"

    log_info "TAS Capacity Analyzer Deployment Script"
    log_info "Project root: $PROJECT_ROOT"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_warn "Dry-run mode: no changes will be made"
    fi

    load_config

    # Handle --fresh flag
    if [[ "$FRESH" == "true" ]]; then
        state_clear
    fi

    state_init

    if [[ "$VERBOSE" == "true" ]]; then
        log_debug "Verbose mode enabled"
        log_debug "CF_ORG=$CF_ORG"
        log_debug "CF_SPACE=$CF_SPACE"
    fi

    if [[ -n "$PHASE" ]]; then
        log_info "Running single phase: $PHASE"
    fi

    log_info "State management ready (phase implementation pending)"
}
```

**Step 3: Test state management**

Run: `./scripts/deploy.sh --verbose`
Expected: Completes without error

Run: `ls -la .state/`
Expected: Directory created (may be empty if dry-run)

**Step 4: Commit**

```bash
git add scripts/deploy.sh
git commit -m "feat: add state management to deploy.sh"
```

---

## Task 6: Add error handling

**Files:**

- Modify: `scripts/deploy.sh`

**Step 1: Add error handler after set -Eeuo pipefail**

Insert after the set command:

```bash
#######################################
# Error handling
#######################################
cleanup() {
    local exit_code=$?
    if [[ $exit_code -ne 0 && -n "${CURRENT_PHASE:-}" ]]; then
        log_error "Deployment failed during phase: $CURRENT_PHASE"
        log_info "Re-run the script to resume from this phase"
    fi
}
trap cleanup EXIT

error_handler() {
    local line=$1
    local command=$2
    log_error "Command failed at line $line"
    log_error "Command: $command"
}
trap 'error_handler $LINENO "$BASH_COMMAND"' ERR
```

**Step 2: Add require_cmd helper function after error handling**

```bash
#######################################
# Utility functions
#######################################
require_cmd() {
    local cmd="$1"
    local msg="${2:-Required command not found: $cmd}"
    if ! command -v "$cmd" >/dev/null 2>&1; then
        log_error "$msg"
        return 1
    fi
}
```

**Step 3: Test error handling**

Run: `./scripts/deploy.sh`
Expected: Completes normally

**Step 4: Commit**

```bash
git add scripts/deploy.sh
git commit -m "feat: add error handling to deploy.sh"
```

---

## Task 7: Implement phase_prereqs

**Files:**

- Modify: `scripts/deploy.sh`

**Step 1: Add phase_prereqs function before main()**

```bash
#######################################
# Phase 1: Prerequisites
#######################################
phase_prereqs() {
    CURRENT_PHASE="prereqs"
    log_info "Phase 1: Checking prerequisites"

    if state_is_complete "PREREQS" && [[ "$FRESH" != "true" ]]; then
        log_success "Phase 1 already complete (skipping)"
        return 0
    fi

    local failed=false

    # Check cf CLI
    if require_cmd cf "cf CLI not found. Install from https://github.com/cloudfoundry/cli"; then
        local cf_version
        cf_version=$(cf version | head -1 | awk '{print $3}' | cut -d'+' -f1)
        local cf_major
        cf_major=$(echo "$cf_version" | cut -d. -f1)
        if [[ "$cf_major" -ge 8 ]]; then
            log_success "cf CLI v$cf_version"
        else
            log_error "cf CLI version $cf_version is too old (need v8+)"
            failed=true
        fi
    else
        failed=true
    fi

    # Check om CLI
    if require_cmd om "om CLI not found. Install from https://github.com/pivotal-cf/om"; then
        local om_version
        om_version=$(om version 2>/dev/null || echo "unknown")
        log_success "om CLI $om_version"
    else
        failed=true
    fi

    # Check node
    if require_cmd node "node not found. Install Node.js 18+ from https://nodejs.org"; then
        local node_version
        node_version=$(node --version | sed 's/v//')
        local node_major
        node_major=$(echo "$node_version" | cut -d. -f1)
        if [[ "$node_major" -ge 18 ]]; then
            log_success "node v$node_version"
        else
            log_error "node version $node_version is too old (need v18+)"
            failed=true
        fi
    else
        failed=true
    fi

    # Check npm
    if require_cmd npm "npm not found"; then
        local npm_version
        npm_version=$(npm --version)
        log_success "npm v$npm_version"
    else
        failed=true
    fi

    # Check jq
    if require_cmd jq "jq not found. Install from https://stedolan.github.io/jq/"; then
        local jq_version
        jq_version=$(jq --version)
        log_success "jq $jq_version"
    else
        failed=true
    fi

    # Check CF login
    log_debug "Checking CF login status..."
    if cf target >/dev/null 2>&1; then
        local cf_user
        cf_user=$(cf target | grep -i "user:" | awk '{print $2}')
        log_success "CF logged in as $cf_user"
    else
        log_error "Not logged in to CF. Run: cf login -a https://api.sys.your-domain.com"
        failed=true
    fi

    # Check OM connectivity (only if OM_TARGET is set)
    if [[ -n "${OM_TARGET:-}" ]]; then
        log_debug "Checking Ops Manager connectivity..."
        if om curl -s -p /api/v0/info >/dev/null 2>&1; then
            log_success "Ops Manager reachable at $OM_TARGET"
        else
            log_warn "Cannot reach Ops Manager at $OM_TARGET (may need credentials)"
        fi
    else
        log_debug "OM_TARGET not set, skipping Ops Manager check"
    fi

    if [[ "$failed" == "true" ]]; then
        log_error "Prerequisite checks failed"
        return 1
    fi

    state_set "PREREQS" "complete"
    log_success "Phase 1 complete"
}
```

**Step 2: Update main() to run phases**

Replace the end of main() with phase execution logic:

```bash
main() {
    parse_args "$@"

    log_info "TAS Capacity Analyzer Deployment Script"
    log_info "Project root: $PROJECT_ROOT"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_warn "Dry-run mode: no changes will be made"
    fi

    load_config

    if [[ "$FRESH" == "true" ]]; then
        state_clear
    fi

    state_init

    if [[ "$VERBOSE" == "true" ]]; then
        log_debug "Verbose mode enabled"
        log_debug "CF_ORG=$CF_ORG"
        log_debug "CF_SPACE=$CF_SPACE"
    fi

    # Run phases
    if [[ -n "$PHASE" ]]; then
        # Single phase mode
        case "$PHASE" in
            prereqs)  phase_prereqs ;;
            env)      log_info "Phase env not yet implemented" ;;
            backend)  log_info "Phase backend not yet implemented" ;;
            frontend) log_info "Phase frontend not yet implemented" ;;
            verify)   log_info "Phase verify not yet implemented" ;;
        esac
    else
        # Run all phases
        if [[ "$SKIP_PREREQS" != "true" ]]; then
            phase_prereqs
        fi
        log_info "Remaining phases not yet implemented"
    fi
}
```

**Step 3: Test prerequisites phase**

Run: `./scripts/deploy.sh --phase=prereqs`
Expected: Shows checks for cf, om, node, npm, jq, CF login

**Step 4: Commit**

```bash
git add scripts/deploy.sh
git commit -m "feat: implement prerequisites phase in deploy.sh"
```

---

## Task 8: Implement phase_env

**Files:**

- Modify: `scripts/deploy.sh`

**Step 1: Add phase_env function after phase_prereqs**

```bash
#######################################
# Phase 2: Environment
#######################################
phase_env() {
    CURRENT_PHASE="env"
    log_info "Phase 2: Generating environment"

    if state_is_complete "ENV" && [[ "$FRESH" != "true" ]]; then
        log_success "Phase 2 already complete (skipping)"
        return 0
    fi

    local env_file="$PROJECT_ROOT/.env"
    local generate_script="$PROJECT_ROOT/generate-env.sh"

    # Check if .env already exists with required variables
    if [[ -f "$env_file" ]]; then
        log_debug "Checking existing .env file..."
        local has_required=true
        for var in BOSH_ENVIRONMENT CF_API_URL CF_USERNAME CF_PASSWORD; do
            if ! grep -q "^$var=" "$env_file" 2>/dev/null; then
                log_debug "Missing $var in .env"
                has_required=false
                break
            fi
        done
        if [[ "$has_required" == "true" ]]; then
            log_success ".env already exists with required variables"
            state_set "ENV" "complete"
            log_success "Phase 2 complete"
            return 0
        fi
    fi

    # Need to generate .env
    if [[ ! -f "$generate_script" ]]; then
        log_error "generate-env.sh not found at $generate_script"
        return 1
    fi

    # Check required Ops Manager credentials
    if [[ -z "${OM_TARGET:-}" ]]; then
        log_error "OM_TARGET is required to generate credentials"
        log_error "Set OM_TARGET in config/deploy.conf or environment"
        return 1
    fi

    local has_auth=false
    if [[ -n "${OM_USERNAME:-}" && -n "${OM_PASSWORD:-}" ]]; then
        has_auth=true
    fi
    if [[ -n "${OM_CLIENT_ID:-}" && -n "${OM_CLIENT_SECRET:-}" ]]; then
        has_auth=true
    fi
    if [[ "$has_auth" == "false" ]]; then
        log_error "Missing Ops Manager authentication"
        log_error "Set OM_USERNAME/OM_PASSWORD or OM_CLIENT_ID/OM_CLIENT_SECRET"
        return 1
    fi

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would run: $generate_script"
        state_set "ENV" "complete"
        log_success "Phase 2 complete (dry-run)"
        return 0
    fi

    log_info "Running generate-env.sh..."
    if ! "$generate_script"; then
        log_error "generate-env.sh failed"
        return 1
    fi

    # Verify .env was created
    if [[ ! -f "$env_file" ]]; then
        log_error ".env file was not created"
        return 1
    fi

    log_success ".env generated with credentials"
    state_set "ENV" "complete"
    log_success "Phase 2 complete"
}
```

**Step 2: Update main() to call phase_env**

Update the phase execution in main():

```bash
    # Run phases
    if [[ -n "$PHASE" ]]; then
        # Single phase mode
        case "$PHASE" in
            prereqs)  phase_prereqs ;;
            env)      phase_env ;;
            backend)  log_info "Phase backend not yet implemented" ;;
            frontend) log_info "Phase frontend not yet implemented" ;;
            verify)   log_info "Phase verify not yet implemented" ;;
        esac
    else
        # Run all phases
        if [[ "$SKIP_PREREQS" != "true" ]]; then
            phase_prereqs
        fi
        phase_env
        log_info "Remaining phases not yet implemented"
    fi
```

**Step 3: Test env phase (dry-run)**

Run: `OM_TARGET=test.example.com OM_USERNAME=admin OM_PASSWORD=secret ./scripts/deploy.sh --phase=env --dry-run`
Expected: Shows dry-run message for generate-env.sh

**Step 4: Commit**

```bash
git add scripts/deploy.sh
git commit -m "feat: implement environment phase in deploy.sh"
```

---

## Task 9: Implement phase_backend

**Files:**

- Modify: `scripts/deploy.sh`

**Step 1: Add phase_backend function after phase_env**

```bash
#######################################
# Phase 3: Backend
#######################################
phase_backend() {
    CURRENT_PHASE="backend"
    log_info "Phase 3: Deploying backend"

    if state_is_complete "BACKEND" && [[ "$FRESH" != "true" ]]; then
        log_success "Phase 3 already complete (skipping)"
        return 0
    fi

    local env_file="$PROJECT_ROOT/.env"
    local backend_dir="$PROJECT_ROOT/backend"

    # Source .env for credentials
    if [[ -f "$env_file" ]]; then
        log_debug "Loading credentials from .env"
        set +u  # Temporarily allow unset variables during source
        # shellcheck source=/dev/null
        source "$env_file"
        set -u
    else
        log_error ".env file not found. Run phase 'env' first."
        return 1
    fi

    # Verify backend directory exists
    if [[ ! -d "$backend_dir" ]]; then
        log_error "Backend directory not found: $backend_dir"
        return 1
    fi

    # Target CF org/space
    log_info "Targeting CF org/space: $CF_ORG/$CF_SPACE"
    if [[ "$DRY_RUN" != "true" ]]; then
        cf target -o "$CF_ORG" -s "$CF_SPACE"
    fi

    # Push backend app
    log_info "Pushing $BACKEND_APP_NAME..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would run: cf push $BACKEND_APP_NAME -f $backend_dir/manifest.yml"
    else
        (cd "$backend_dir" && cf push "$BACKEND_APP_NAME")
    fi

    # Set environment variables
    log_info "Setting environment variables..."
    local env_vars=(
        "CF_API_URL=${CF_API_URL:-}"
        "CF_USERNAME=${CF_USERNAME:-}"
        "CF_PASSWORD=${CF_PASSWORD:-}"
        "BOSH_ENVIRONMENT=${BOSH_ENVIRONMENT:-}"
        "BOSH_CLIENT=${BOSH_CLIENT:-}"
        "BOSH_CLIENT_SECRET=${BOSH_CLIENT_SECRET:-}"
        "BOSH_DEPLOYMENT=${BOSH_DEPLOYMENT:-}"
    )

    # Add BOSH_CA_CERT if set (handle multiline)
    if [[ -n "${BOSH_CA_CERT:-}" ]]; then
        env_vars+=("BOSH_CA_CERT=${BOSH_CA_CERT}")
    fi

    # Add BOSH_ALL_PROXY if set
    if [[ -n "${BOSH_ALL_PROXY:-}" ]]; then
        env_vars+=("BOSH_ALL_PROXY=${BOSH_ALL_PROXY}")
    fi

    # Add vSphere vars if set
    if [[ -n "${VSPHERE_HOST:-}" ]]; then
        env_vars+=(
            "VSPHERE_HOST=${VSPHERE_HOST}"
            "VSPHERE_DATACENTER=${VSPHERE_DATACENTER:-}"
            "VSPHERE_USERNAME=${VSPHERE_USERNAME:-}"
            "VSPHERE_PASSWORD=${VSPHERE_PASSWORD:-}"
        )
    fi

    for env_var in "${env_vars[@]}"; do
        local key="${env_var%%=*}"
        local value="${env_var#*=}"
        if [[ -n "$value" ]]; then
            log_debug "Setting $key"
            if [[ "$DRY_RUN" != "true" ]]; then
                cf set-env "$BACKEND_APP_NAME" "$key" "$value" >/dev/null
            fi
        fi
    done

    # Restage to apply env vars
    log_info "Restaging $BACKEND_APP_NAME..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would run: cf restage $BACKEND_APP_NAME"
    else
        cf restage "$BACKEND_APP_NAME"
    fi

    # Get backend URL
    local backend_url
    if [[ "$DRY_RUN" == "true" ]]; then
        backend_url="$BACKEND_APP_NAME.apps.example.com"
    else
        backend_url=$(cf app "$BACKEND_APP_NAME" | grep -E "^routes:" | awk '{print $2}')
    fi
    state_set "BACKEND_URL" "$backend_url"

    log_success "Backend running at $backend_url"
    state_set "BACKEND" "complete"
    log_success "Phase 3 complete"
}
```

**Step 2: Update main() to call phase_backend**

```bash
    # Run phases
    if [[ -n "$PHASE" ]]; then
        # Single phase mode
        case "$PHASE" in
            prereqs)  phase_prereqs ;;
            env)      phase_env ;;
            backend)  phase_backend ;;
            frontend) log_info "Phase frontend not yet implemented" ;;
            verify)   log_info "Phase verify not yet implemented" ;;
        esac
    else
        # Run all phases
        if [[ "$SKIP_PREREQS" != "true" ]]; then
            phase_prereqs
        fi
        phase_env
        phase_backend
        log_info "Remaining phases not yet implemented"
    fi
```

**Step 3: Test backend phase (dry-run)**

Create a minimal .env for testing:

```bash
echo 'CF_API_URL=https://api.example.com
CF_USERNAME=admin
CF_PASSWORD=secret
BOSH_ENVIRONMENT=10.0.0.6' > .env
```

Run: `./scripts/deploy.sh --phase=backend --dry-run`
Expected: Shows dry-run messages for cf push, set-env, restage

Clean up: `rm .env`

**Step 4: Commit**

```bash
git add scripts/deploy.sh
git commit -m "feat: implement backend deployment phase in deploy.sh"
```

---

## Task 10: Implement phase_frontend

**Files:**

- Modify: `scripts/deploy.sh`

**Step 1: Add phase_frontend function after phase_backend**

```bash
#######################################
# Phase 4: Frontend
#######################################
phase_frontend() {
    CURRENT_PHASE="frontend"
    log_info "Phase 4: Deploying frontend"

    if state_is_complete "FRONTEND" && [[ "$FRESH" != "true" ]]; then
        log_success "Phase 4 already complete (skipping)"
        return 0
    fi

    local frontend_dir="$PROJECT_ROOT/frontend"
    local backend_url
    backend_url=$(state_get "BACKEND_URL")

    # Verify frontend directory exists
    if [[ ! -d "$frontend_dir" ]]; then
        log_error "Frontend directory not found: $frontend_dir"
        return 1
    fi

    # Check backend URL is available
    if [[ -z "$backend_url" ]]; then
        log_error "Backend URL not found in state. Run phase 'backend' first."
        return 1
    fi

    # Target CF org/space
    log_info "Targeting CF org/space: $CF_ORG/$CF_SPACE"
    if [[ "$DRY_RUN" != "true" ]]; then
        cf target -o "$CF_ORG" -s "$CF_SPACE"
    fi

    # Create frontend .env with backend URL
    local frontend_env="$frontend_dir/.env"
    log_info "Creating frontend .env with backend URL..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would write VITE_API_URL=https://$backend_url to $frontend_env"
    else
        echo "VITE_API_URL=https://$backend_url" > "$frontend_env"
    fi

    # Install dependencies
    log_info "Installing npm dependencies..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would run: npm install in $frontend_dir"
    else
        (cd "$frontend_dir" && npm install)
    fi

    # Build frontend
    log_info "Building frontend..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would run: npm run build in $frontend_dir"
    else
        (cd "$frontend_dir" && npm run build)
    fi

    # Push frontend app
    log_info "Pushing $FRONTEND_APP_NAME..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would run: cf push $FRONTEND_APP_NAME -f $frontend_dir/manifest.yml"
    else
        (cd "$frontend_dir" && cf push "$FRONTEND_APP_NAME")
    fi

    # Get frontend URL
    local frontend_url
    if [[ "$DRY_RUN" == "true" ]]; then
        frontend_url="$FRONTEND_APP_NAME.apps.example.com"
    else
        frontend_url=$(cf app "$FRONTEND_APP_NAME" | grep -E "^routes:" | awk '{print $2}')
    fi
    state_set "FRONTEND_URL" "$frontend_url"

    log_success "Frontend running at $frontend_url"
    state_set "FRONTEND" "complete"
    log_success "Phase 4 complete"
}
```

**Step 2: Update main() to call phase_frontend**

```bash
    # Run phases
    if [[ -n "$PHASE" ]]; then
        # Single phase mode
        case "$PHASE" in
            prereqs)  phase_prereqs ;;
            env)      phase_env ;;
            backend)  phase_backend ;;
            frontend) phase_frontend ;;
            verify)   log_info "Phase verify not yet implemented" ;;
        esac
    else
        # Run all phases
        if [[ "$SKIP_PREREQS" != "true" ]]; then
            phase_prereqs
        fi
        phase_env
        phase_backend
        phase_frontend
        log_info "Remaining phases not yet implemented"
    fi
```

**Step 3: Test frontend phase (dry-run)**

First set up state with backend URL:

```bash
mkdir -p .state
echo 'STATE_BACKEND_URL="capacity-backend.apps.example.com"' > .state/deploy-state
```

Run: `./scripts/deploy.sh --phase=frontend --dry-run`
Expected: Shows dry-run messages for npm install, build, cf push

Clean up: `rm -rf .state`

**Step 4: Commit**

```bash
git add scripts/deploy.sh
git commit -m "feat: implement frontend deployment phase in deploy.sh"
```

---

## Task 11: Implement phase_verify

**Files:**

- Modify: `scripts/deploy.sh`

**Step 1: Add phase_verify function after phase_frontend**

```bash
#######################################
# Phase 5: Verify
#######################################
phase_verify() {
    CURRENT_PHASE="verify"
    log_info "Phase 5: Verifying deployment"

    if state_is_complete "VERIFY" && [[ "$FRESH" != "true" ]]; then
        log_success "Phase 5 already complete (skipping)"
        return 0
    fi

    local backend_url
    local frontend_url
    backend_url=$(state_get "BACKEND_URL")
    frontend_url=$(state_get "FRONTEND_URL")

    if [[ -z "$backend_url" ]]; then
        log_error "Backend URL not found in state. Run phase 'backend' first."
        return 1
    fi

    local failed=false

    # Test backend health endpoint
    log_info "Testing backend health endpoint..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would test: https://$backend_url/api/v1/health"
        log_success "/api/v1/health returns 200 (dry-run)"
    else
        local health_response
        local health_status
        health_status=$(curl -s -o /dev/null -w "%{http_code}" "https://$backend_url/api/v1/health" || echo "000")
        if [[ "$health_status" == "200" ]]; then
            log_success "/api/v1/health returns 200"
        else
            log_error "/api/v1/health returned $health_status (expected 200)"
            failed=true
        fi
    fi

    # Test backend dashboard endpoint
    log_info "Testing backend dashboard endpoint..."
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY-RUN] Would test: https://$backend_url/api/v1/dashboard"
        log_success "/api/v1/dashboard returns valid JSON (dry-run)"
    else
        local dashboard_response
        dashboard_response=$(curl -s "https://$backend_url/api/v1/dashboard" || echo "")
        if echo "$dashboard_response" | jq . >/dev/null 2>&1; then
            log_success "/api/v1/dashboard returns valid JSON"
        else
            log_error "/api/v1/dashboard did not return valid JSON"
            failed=true
        fi
    fi

    # Test frontend is accessible
    if [[ -n "$frontend_url" ]]; then
        log_info "Testing frontend accessibility..."
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "[DRY-RUN] Would test: https://$frontend_url"
            log_success "Frontend accessible (dry-run)"
        else
            local frontend_status
            frontend_status=$(curl -s -o /dev/null -w "%{http_code}" "https://$frontend_url" || echo "000")
            if [[ "$frontend_status" == "200" ]]; then
                log_success "Frontend accessible at https://$frontend_url"
            else
                log_error "Frontend returned $frontend_status (expected 200)"
                failed=true
            fi
        fi
    fi

    if [[ "$failed" == "true" ]]; then
        log_error "Verification failed"
        return 1
    fi

    state_set "VERIFY" "complete"
    log_success "Phase 5 complete"

    # Print summary
    echo ""
    echo "========================================"
    log_success "Deployment successful!"
    echo "     Backend:  https://$backend_url"
    if [[ -n "$frontend_url" ]]; then
        echo "     Frontend: https://$frontend_url"
    fi
    echo "========================================"
}
```

**Step 2: Update main() to call phase_verify and complete the phase orchestration**

Replace the entire main() function with the final version:

```bash
#######################################
# Main entry point
#######################################
main() {
    parse_args "$@"

    log_info "TAS Capacity Analyzer Deployment Script"
    log_info "Project root: $PROJECT_ROOT"

    if [[ "$DRY_RUN" == "true" ]]; then
        log_warn "Dry-run mode: no changes will be made"
    fi

    load_config

    if [[ "$FRESH" == "true" ]]; then
        state_clear
    fi

    state_init

    if [[ "$VERBOSE" == "true" ]]; then
        log_debug "Verbose mode enabled"
        log_debug "CF_ORG=$CF_ORG"
        log_debug "CF_SPACE=$CF_SPACE"
        log_debug "BACKEND_APP_NAME=$BACKEND_APP_NAME"
        log_debug "FRONTEND_APP_NAME=$FRONTEND_APP_NAME"
    fi

    # Run phases
    if [[ -n "$PHASE" ]]; then
        # Single phase mode
        log_info "Running single phase: $PHASE"
        case "$PHASE" in
            prereqs)  phase_prereqs ;;
            env)      phase_env ;;
            backend)  phase_backend ;;
            frontend) phase_frontend ;;
            verify)   phase_verify ;;
        esac
    else
        # Run all phases in order
        if [[ "$SKIP_PREREQS" != "true" ]]; then
            phase_prereqs
        fi
        phase_env
        phase_backend
        phase_frontend
        phase_verify
    fi

    CURRENT_PHASE=""
}

main "$@"
```

**Step 3: Test verify phase (dry-run)**

Set up state:

```bash
mkdir -p .state
cat > .state/deploy-state << 'EOF'
STATE_BACKEND_URL="capacity-backend.apps.example.com"
STATE_FRONTEND_URL="capacity-ui.apps.example.com"
EOF
```

Run: `./scripts/deploy.sh --phase=verify --dry-run`
Expected: Shows dry-run verification messages and success summary

Clean up: `rm -rf .state`

**Step 4: Commit**

```bash
git add scripts/deploy.sh
git commit -m "feat: implement verify phase and complete deploy.sh"
```

---

## Task 12: Final testing and documentation update

**Files:**

- Modify: `docs/DEPLOYMENT.md` (add reference to deploy.sh)

**Step 1: Run shellcheck on deploy.sh**

Run: `shellcheck scripts/deploy.sh`
Expected: No errors (warnings about sourcing are expected)

**Step 2: Test full dry-run flow**

```bash
# Set up minimal config
export OM_TARGET=test.example.com
export OM_USERNAME=admin
export OM_PASSWORD=secret

# Run full deployment in dry-run mode
./scripts/deploy.sh --dry-run --verbose
```

Expected: All phases show dry-run messages

**Step 3: Add deployment script reference to DEPLOYMENT.md**

Add to the beginning of docs/DEPLOYMENT.md after the title:

````markdown
## Automated Deployment

For automated deployment, use the deployment script:

```bash
# Copy and configure
cp config/deploy.conf.example config/deploy.conf
# Edit config/deploy.conf with your credentials

# Run deployment
./scripts/deploy.sh

# Or run with environment variables
OM_TARGET=opsman.example.com OM_USERNAME=admin OM_PASSWORD=secret ./scripts/deploy.sh
```
````

See `./scripts/deploy.sh --help` for all options including single-phase execution and dry-run mode.

---

## Manual Deployment

````

**Step 4: Commit all changes**

```bash
git add docs/DEPLOYMENT.md scripts/deploy.sh
git commit -m "docs: add automated deployment reference to DEPLOYMENT.md"
````

**Step 5: Final commit summary**

Run: `git log --oneline -10`
Expected: Shows all commits from this implementation

---

## Summary

Files created/modified:

- `scripts/deploy.sh` - Main deployment script (~450 lines)
- `config/deploy.conf.example` - Configuration template
- `.gitignore` - Updated with state and config ignores
- `docs/DEPLOYMENT.md` - Updated with script reference
- `docs/plans/2026-01-21-deployment-script-design.md` - Design document
- `docs/plans/2026-01-21-deployment-script-impl.md` - This implementation plan
