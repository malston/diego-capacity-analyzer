#!/usr/bin/env bash
# ABOUTME: Generates .env file by deriving credentials from Ops Manager and BOSH.
# ABOUTME: Requires OM_TARGET and either username/password or client credentials.

set -euo pipefail

# Check required variable
if [[ -z "${OM_TARGET:-}" ]]; then
    echo "Error: OM_TARGET is required." >&2
    echo "" >&2
    echo "Usage with username/password:" >&2
    echo "  export OM_TARGET=opsman.example.com" >&2
    echo "  export OM_USERNAME=admin" >&2
    echo "  export OM_PASSWORD=<password>" >&2
    echo "" >&2
    echo "Usage with client credentials:" >&2
    echo "  export OM_TARGET=opsman.example.com" >&2
    echo "  export OM_CLIENT_ID=<client-id>" >&2
    echo "  export OM_CLIENT_SECRET=<client-secret>" >&2
    echo "" >&2
    echo "Optional (for non-routable BOSH networks):" >&2
    echo "  export OM_PRIVATE_KEY=~/.ssh/opsman_key" >&2
    exit 1
fi

# Check authentication: need either username/password OR client credentials
has_user_auth=false
has_client_auth=false

if [[ -n "${OM_USERNAME:-}" && -n "${OM_PASSWORD:-}" ]]; then
    has_user_auth=true
fi

if [[ -n "${OM_CLIENT_ID:-}" && -n "${OM_CLIENT_SECRET:-}" ]]; then
    has_client_auth=true
fi

if [[ "$has_user_auth" == "false" && "$has_client_auth" == "false" ]]; then
    echo "Error: Missing authentication credentials." >&2
    echo "Set either OM_USERNAME/OM_PASSWORD or OM_CLIENT_ID/OM_CLIENT_SECRET" >&2
    exit 1
fi

# Optional: set to true to skip SSL validation (not recommended for production)
export OM_SKIP_SSL_VALIDATION="${OM_SKIP_SSL_VALIDATION:-false}"

# Export required vars for om CLI
export OM_TARGET
if [[ "$has_user_auth" == "true" ]]; then
    export OM_USERNAME OM_PASSWORD
fi
if [[ "$has_client_auth" == "true" ]]; then
    export OM_CLIENT_ID OM_CLIENT_SECRET
fi

eval "$(om bosh-env 2>/dev/null)"

# Set up SSH proxy if private key is provided (for non-routable BOSH networks)
if [[ -n "${OM_PRIVATE_KEY:-}" ]]; then
    export BOSH_ALL_PROXY="ssh+socks5://ubuntu@$OM_TARGET:22?private-key=$OM_PRIVATE_KEY"
fi

export BOSH_DEPLOYMENT
BOSH_DEPLOYMENT=$(bosh deployments --json | jq -r '.Tables[0].Rows[] | select(.name | startswith("cf-")) | .name')

export CF_USERNAME=admin
export CF_PASSWORD
CF_PASSWORD=$(om credentials -p cf -c .uaa.admin_credentials -t json | jq -r '.password')
CF_SYSTEM_DOMAIN=$(om curl -s --path /api/v0/staged/products/"$BOSH_DEPLOYMENT"/properties | jq -r '.properties.".cloud_controller.system_domain"' | jq -r '.value')

VSPHERE_HOST=$(om curl -s --path /api/v0/staged/director/properties | jq -r '.iaas_configuration?.vcenter_host')
VSPHERE_DATACENTER=$(om curl -s --path /api/v0/staged/director/properties | jq -r '.iaas_configuration?.datacenter')
VSPHERE_USERNAME=$(om curl -s --path /api/v0/staged/director/properties | jq -r '.iaas_configuration?.vcenter_username')
export VSPHERE_PASSWORD
VSPHERE_PASSWORD=$(om staged-director-config --no-redact | yq '.iaas-configurations[].vcenter_password')
export VSPHERE_INSECURE
VSPHERE_INSECURE=$(om curl -s --path /api/v0/staged/director/properties | jq -r '.iaas_configuration?.vcenter_ca_certificate' | jq -r 'if . == null then "true" else "false" end')

cat > .env << EOF
BOSH_CA_CERT="$BOSH_CA_CERT"
BOSH_CLIENT=$BOSH_CLIENT
BOSH_CLIENT_SECRET=$BOSH_CLIENT_SECRET
BOSH_DEPLOYMENT=$BOSH_DEPLOYMENT
BOSH_ENVIRONMENT=$BOSH_ENVIRONMENT
EOF

# Add proxy settings if configured (for non-routable BOSH networks)
if [[ -n "${BOSH_ALL_PROXY:-}" ]]; then
    cat >> .env << EOF
BOSH_ALL_PROXY=$BOSH_ALL_PROXY
EOF
fi

cat >> .env << EOF

CF_API_URL=https://api.$CF_SYSTEM_DOMAIN
CF_USERNAME=$CF_USERNAME
CF_PASSWORD=$CF_PASSWORD

CREDHUB_CA_CERT="$CREDHUB_CA_CERT"
CREDHUB_CLIENT=$CREDHUB_CLIENT
CREDHUB_SECRET=$CREDHUB_SECRET
CREDHUB_SERVER=$CREDHUB_SERVER
EOF

if [[ -n "${BOSH_ALL_PROXY:-}" ]]; then
    cat >> .env << EOF
CREDHUB_PROXY=$BOSH_ALL_PROXY
EOF
fi

cat >> .env << EOF

OM_CONNECT_TIMEOUT=60
OM_TARGET=$OM_TARGET
OM_SKIP_SSL_VALIDATION=$OM_SKIP_SSL_VALIDATION
EOF

# Append auth credentials based on which method was used
if [[ "$has_user_auth" == "true" ]]; then
    cat >> .env << EOF
OM_USERNAME=$OM_USERNAME
OM_PASSWORD=$OM_PASSWORD
EOF
fi

if [[ "$has_client_auth" == "true" ]]; then
    cat >> .env << EOF
OM_CLIENT_ID=$OM_CLIENT_ID
OM_CLIENT_SECRET=$OM_CLIENT_SECRET
EOF
fi

cat >> .env << EOF

REACT_APP_CF_API_URL=https://api.$CF_SYSTEM_DOMAIN
REACT_APP_CF_UAA_URL=https://login.$CF_SYSTEM_DOMAIN

VITE_CF_API_URL=https://api.$CF_SYSTEM_DOMAIN
VITE_CF_UAA_URL=https://login.$CF_SYSTEM_DOMAIN
# Optional: BOSH Director URL (for diego cell metrics)
VITE_BOSH_URL=https://$BOSH_ENVIRONMENT:25555

VSPHERE_HOST=$VSPHERE_HOST
VSPHERE_DATACENTER=$VSPHERE_DATACENTER
VSPHERE_USERNAME=$VSPHERE_USERNAME
VSPHERE_PASSWORD=$VSPHERE_PASSWORD
VSPHERE_INSECURE=$VSPHERE_INSECURE
EOF

echo "Generated .env file with credentials for:"
echo "  - BOSH Director: $BOSH_ENVIRONMENT"
echo "  - CF Deployment: $BOSH_DEPLOYMENT"
echo "  - CF API: api.$CF_SYSTEM_DOMAIN"
echo "  - vSphere: $VSPHERE_HOST"
