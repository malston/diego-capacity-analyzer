#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# List of projects to keep
KEEP=(
  "-Users-markalston-workspace-bosh-mcp-server"
  "-Users-markalston-workspace-bosh-mock-director"
  "-Users-markalston-workspace-chrome-tabs"
  "-Users-markalston-workspace-claudeup"
  "-Users-markalston-workspace-diego-capacity-analyzer"
  "-Users-markalston-workspace-diego-capacity-analyzer-backend"
  "-Users-markalston-workspace-diego-capacity-analyzer-frontend"
  "-Users-markalston-workspace-expense-report"
  "-Users-markalston-workspace-github-malston-website"
  "-Users-markalston-workspace-homelab"
  "-Users-markalston-workspace-shared-isoseg-demo"
  "-Users-markalston-workspace-tanzu-cf-architect"
  "-Users-markalston-workspace-tanzu-homelab"
  "-Users-markalston-workspace-tanzu-platform-sbom-service"
  "-Users-markalston-workspace-tas-vcf"
  "-Users-markalston-workspace-tile-diff"
  "-Users-markalston-workspace-vcf-9x-in-box"
  "-Users-markalston-workspace-vcf-offline-depot"
)

# Delete everything NOT in the keep list
cd ~/.claude/projects
for dir in */; do
  dir_name="${dir%/}"
  keep=false
  for proj in "${KEEP[@]}"; do
    if [[ "$dir_name" == "$proj" ]]; then
      keep=true
      break
    fi
  done
  if [[ "$keep" == false ]]; then
    echo "Deleting: $dir_name"
    rm -rf -- "$dir_name"  # Use -- to stop processing flags
  fi
done

# Now clean up build artifacts from what we're keeping
echo "Cleaning build artifacts..."
find ~/.claude/projects -type d \( \
  -name node_modules \
  -o -name .next \
  -o -name dist \
  -o -name build \
  -o -name __pycache__ \
  -o -name .pytest_cache \
  -o -name .venv \
  -o -name venv \
  -o -name .gradle \
  -o -name target \
\) -exec rm -rf {} + 2>/dev/null

echo "Done. New size:"
du -sh ~/.claude/projects
