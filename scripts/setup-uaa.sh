#!/usr/bin/env bash
# ABOUTME: Configures UAA groups and a dedicated OAuth client for diego-analyzer.
# ABOUTME: Requires OM_TARGET and either username/password or client credentials.

set -euo pipefail

# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS] <username> [username...]

Creates UAA groups (diego-analyzer.viewer, diego-analyzer.operator) and a
dedicated OAuth client, then assigns the specified users to both groups.

Prerequisites:
  - om CLI    (https://github.com/pivotal-cf/om)
  - uaac CLI  (gem install cf-uaac)
  - jq

Environment variables (required):
  OM_TARGET                Ops Manager hostname (e.g. opsman.example.com)
  OM_USERNAME / OM_PASSWORD   -or-
  OM_CLIENT_ID / OM_CLIENT_SECRET

Environment variables (optional):
  OM_SKIP_SSL_VALIDATION   Skip TLS verification (default: false)
  OAUTH_CLIENT_SECRET      Pre-set client secret (otherwise generated)

Options:
  -h, --help       Show this help
  --skip-client    Skip OAuth client creation (groups and members only)
  --dry-run        Print commands without executing

Examples:
  $(basename "$0") admin
  $(basename "$0") admin operator-user
  $(basename "$0") --skip-client admin
EOF
    exit 1
}

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

info()  { echo "==> $*"; }
warn()  { echo "WARNING: $*" >&2; }
error() { echo "ERROR: $*" >&2; }

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

require_cmd() {
    command -v "$1" >/dev/null 2>&1 || {
        error "Required command not found: $1"
        exit 1
    }
}

# Run a uaac command, or print it in dry-run mode
run_uaac() {
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "[dry-run] uaac $*"
    else
        uaac "$@"
    fi
}

# Create a UAA group if it does not already exist
ensure_group() {
    local group="$1"
    info "Creating group: $group"
    if run_uaac group add "$group" 2>/dev/null; then
        info "  Created $group"
    else
        info "  Group $group already exists (skipped)"
    fi
}

# Add a user to a UAA group (idempotent)
ensure_member() {
    local group="$1"
    local user="$2"
    info "Adding $user to $group"
    if run_uaac member add "$group" "$user" 2>/dev/null; then
        info "  Added $user to $group"
    else
        info "  $user is already a member of $group (skipped)"
    fi
}

# Generate a random secret (32 bytes, base64url, no padding)
generate_secret() {
    openssl rand -base64 32 | tr -d '=' | tr '+/' '-_'
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------

DRY_RUN=false
SKIP_CLIENT=false
USERS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)     usage ;;
        --dry-run)     DRY_RUN=true; shift ;;
        --skip-client) SKIP_CLIENT=true; shift ;;
        -*)            error "Unknown option: $1"; usage ;;
        *)             USERS+=("$1"); shift ;;
    esac
done

if [[ ${#USERS[@]} -eq 0 ]]; then
    error "At least one username is required."
    echo ""
    usage
fi

# ---------------------------------------------------------------------------
# Validate environment
# ---------------------------------------------------------------------------

require_cmd om
require_cmd uaac
require_cmd jq
require_cmd openssl

if [[ -z "${OM_TARGET:-}" ]]; then
    error "OM_TARGET is required."
    echo ""
    usage
fi

has_user_auth=false
has_client_auth=false

if [[ -n "${OM_USERNAME:-}" && -n "${OM_PASSWORD:-}" ]]; then
    has_user_auth=true
fi
if [[ -n "${OM_CLIENT_ID:-}" && -n "${OM_CLIENT_SECRET:-}" ]]; then
    has_client_auth=true
fi

if [[ "$has_user_auth" == "false" && "$has_client_auth" == "false" ]]; then
    error "Missing authentication credentials."
    error "Set either OM_USERNAME/OM_PASSWORD or OM_CLIENT_ID/OM_CLIENT_SECRET"
    exit 1
fi

export OM_SKIP_SSL_VALIDATION="${OM_SKIP_SSL_VALIDATION:-false}"

# ---------------------------------------------------------------------------
# Step 1: Get UAA admin client secret from Ops Manager
# ---------------------------------------------------------------------------

info "Retrieving UAA admin client secret from Ops Manager..."

if [[ "$DRY_RUN" == "true" ]]; then
    echo "[dry-run] om credentials -p cf -c .uaa.admin_client_credentials -t json"
    UAA_ADMIN_SECRET="<dry-run-secret>"
else
    UAA_ADMIN_SECRET=$(om credentials -p cf -c .uaa.admin_client_credentials -t json | jq -r '.password')
    if [[ -z "$UAA_ADMIN_SECRET" || "$UAA_ADMIN_SECRET" == "null" ]]; then
        error "Failed to retrieve UAA admin client secret from Ops Manager."
        exit 1
    fi
    info "  Retrieved UAA admin secret."
fi

# Derive UAA URL from CF system domain
if [[ "$DRY_RUN" == "true" ]]; then
    echo "[dry-run] om curl -s --path /api/v0/staged/products/.../properties"
    UAA_URL="https://uaa.sys.example.com"
else
    CF_DEPLOYMENT=$(om curl -s --path /api/v0/deployed/products | jq -r '.[] | select(.type == "cf") | .installation_name')
    if [[ -z "$CF_DEPLOYMENT" ]]; then
        error "Could not find a deployed CF product in Ops Manager."
        exit 1
    fi
    CF_SYSTEM_DOMAIN=$(om curl -s --path "/api/v0/staged/products/${CF_DEPLOYMENT}/properties" \
        | jq -r '.properties.".cloud_controller.system_domain".value')
    if [[ -z "$CF_SYSTEM_DOMAIN" || "$CF_SYSTEM_DOMAIN" == "null" ]]; then
        error "Could not determine CF system domain from Ops Manager."
        exit 1
    fi
    UAA_URL="https://uaa.${CF_SYSTEM_DOMAIN}"
    info "  UAA endpoint: $UAA_URL"
fi

# ---------------------------------------------------------------------------
# Step 2: Authenticate with UAA and create groups
# ---------------------------------------------------------------------------

info "Targeting UAA at $UAA_URL..."
run_uaac target "$UAA_URL" --skip-ssl-validation

info "Authenticating as UAA admin client..."
run_uaac token client get admin -s "$UAA_ADMIN_SECRET"

ensure_group "diego-analyzer.viewer"
ensure_group "diego-analyzer.operator"

for user in "${USERS[@]}"; do
    ensure_member "diego-analyzer.viewer" "$user"
    ensure_member "diego-analyzer.operator" "$user"
done

# ---------------------------------------------------------------------------
# Step 3: Create dedicated OAuth client
# ---------------------------------------------------------------------------

if [[ "$SKIP_CLIENT" == "true" ]]; then
    info "Skipping OAuth client creation (--skip-client)."
else
    CLIENT_SECRET="${OAUTH_CLIENT_SECRET:-$(generate_secret)}"

    info "Creating OAuth client: diego-analyzer"
    if run_uaac client add diego-analyzer \
        --name "Diego Capacity Analyzer" \
        --scope "openid diego-analyzer.operator diego-analyzer.viewer" \
        --authorized_grant_types "password,refresh_token" \
        --access_token_validity 7200 \
        --refresh_token_validity 1209600 \
        --secret "$CLIENT_SECRET" 2>/dev/null; then
        info "  Created diego-analyzer client."
    else
        warn "Client diego-analyzer may already exist. To update it, run:"
        warn "  uaac client update diego-analyzer --scope 'openid diego-analyzer.operator diego-analyzer.viewer'"
        warn "  uaac secret set diego-analyzer -s <new-secret>"
    fi

    echo ""
    echo "============================================="
    echo "  OAuth client created successfully"
    echo "============================================="
    echo ""
    echo "Add these to your .env file:"
    echo ""
    echo "  OAUTH_CLIENT_ID=diego-analyzer"
    echo "  OAUTH_CLIENT_SECRET=$CLIENT_SECRET"
    echo ""
    echo "See docs/AUTHENTICATION.md for details."
    echo "============================================="
fi

info "Done."
