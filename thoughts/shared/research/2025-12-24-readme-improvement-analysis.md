---
date: 2025-12-24T09:15:00-08:00
researcher: Claude
git_commit: bb9a5e65d57a610255b5dee4310e9541ee172eed
branch: main
repository: diego-capacity-analyzer
topic: "README Improvement Analysis"
tags: [research, documentation, readme, improvements]
status: complete
last_updated: 2025-12-24
last_updated_by: Claude
---

# Research: README Improvement Analysis

**Date**: 2025-12-24T09:15:00-08:00
**Researcher**: Claude
**Git Commit**: bb9a5e65d57a610255b5dee4310e9541ee172eed
**Branch**: main
**Repository**: diego-capacity-analyzer

## Research Question

Analyze the current README.md against the actual codebase to identify gaps and suggest improvements.

## Summary

The current README is significantly outdated compared to the actual codebase. The project has evolved from a simple dashboard to a comprehensive capacity planning tool with scenario analysis, vSphere integration, and a step-based wizard. The README reflects only ~40% of current functionality.

## Key Gaps Identified

### 1. Version Badge Outdated
- **Current**: Shows `1.3.2`
- **Actual**: `v1.4.0` (from git tags)

### 2. Features Section Incomplete

**Currently Listed (4 features):**
- Real-time Capacity Monitoring
- Isolation Segment Filtering
- What-If Scenario Modeling
- Right-Sizing Recommendations

**Missing Features:**
- **Scenario Analysis Wizard** - Step-based configuration (Resources → Cell Config → Advanced)
- **vSphere Infrastructure Discovery** - Live infrastructure from vCenter
- **Capacity Planning Calculator** - N-1 HA calculations, max cell estimates
- **TPS Performance Modeling** - Estimates throughput based on cell count
- **Markdown Export** - Export analysis results as reports
- **Toast Notifications** - User feedback for operations
- **Sample Data Files** - Pre-built scenarios for testing (dev/staging/prod/enterprise)

### 3. Quick Start Section Incomplete

**Backend Environment Variables:**

Current README only shows:
```bash
CF_API_URL, CF_USERNAME, CF_PASSWORD
```

Missing variables:
```bash
# BOSH Integration (optional but important)
BOSH_ENVIRONMENT, BOSH_CLIENT, BOSH_CLIENT_SECRET, BOSH_CA_CERT, BOSH_ALL_PROXY

# vSphere Integration (optional)
VSPHERE_HOST, VSPHERE_USERNAME, VSPHERE_PASSWORD, VSPHERE_DATACENTER, VSPHERE_INSECURE

# Tuning
PORT, CACHE_TTL, DASHBOARD_CACHE_TTL, VSPHERE_CACHE_TTL, LOG_LEVEL, LOG_FORMAT
```

### 4. Project Structure Outdated

**Missing from tree:**
```text
frontend/
├── public/
│   └── samples/              # 7 sample JSON files
├── src/
│   ├── components/
│   │   └── wizard/           # Step-based wizard (7 files)
│   │       └── steps/        # Individual step components
│   ├── contexts/             # ToastContext.jsx, AuthContext.jsx
│   ├── config/               # vmPresets.js, resourceConfig.js
│   └── utils/                # exportMarkdown.js, metricsCalculations.js

backend/
├── logger/                   # Structured logging
├── middleware/               # HTTP logging middleware
└── services/
    ├── vsphere.go            # vSphere integration (missing)
    ├── scenario.go           # Scenario calculator (missing)
    └── planning.go           # Planning calculator (missing)

.github/
└── workflows/                # CI/CD (not mentioned at all)
```

### 5. API Endpoints Incomplete

**Currently Listed (3):**
```text
GET /api/health
GET /api/dashboard
POST /api/refresh
```

**Actually Available (8+):**
```text
GET  /api/health                    # Health check
GET  /api/dashboard                 # Dashboard data
GET  /api/infrastructure            # Live vSphere infrastructure
POST /api/infrastructure/manual     # Manual infrastructure input
POST /api/infrastructure/state      # Set infrastructure state
GET  /api/infrastructure/status     # Data source status
POST /api/infrastructure/planning   # Calculate max deployable cells
POST /api/scenario/compare          # Compare current vs proposed scenarios
```

Note: `/api/refresh` doesn't appear to exist in the current codebase.

### 6. Technology Stack Incomplete

**Frontend - Missing:**
- Tailwind CSS 3.3.6 (styling)
- Vitest 4.0.16 (testing)
- @testing-library/react (component testing)
- Lucide React (icons)

**Backend - Missing:**
- govmomi v0.52.0 (vSphere integration)
- cloudfoundry/socks5-proxy (BOSH SSH tunneling)

### 7. No Testing Section

The README has no mention of:
- How to run frontend tests (`npm test`)
- How to run backend tests (`go test ./...`)
- Test frameworks used (Vitest, Go testing)
- CI/CD pipeline (GitHub Actions)

### 8. No CI/CD Section

The project has GitHub Actions workflows:
- `ci.yml` - Runs on PRs and pushes to main
- `release.yml` - Creates releases on version tags

### 9. Missing Documentation Links

**Currently Listed:**
- docs/UI-GUIDE.md
- docs/DEPLOYMENT.md

**Also Available:**
- AUTHENTICATION.md (root)
- backend/README.md
- docs/plans/*.md (8 design documents)

## Suggested Improvements

### Priority 1: Critical Updates

1. **Update version badge** from 1.3.2 to 1.4.0
2. **Add missing features** to Features section
3. **Expand Quick Start** with optional environment variables
4. **Fix API endpoints** list (remove non-existent, add actual)

### Priority 2: Structural Improvements

5. **Update Project Structure** tree to reflect current directories
6. **Add Testing section** with commands and coverage info
7. **Add CI/CD section** describing GitHub Actions workflows
8. **Update Technology Stack** with all dependencies

### Priority 3: Enhancements

9. **Add Architecture diagram** showing data flow
10. **Add Screenshots** of dashboard and scenario wizard
11. **Link additional documentation** (AUTHENTICATION.md, design docs)
12. **Add Contributing section** with development workflow
13. **Add Sample Data section** explaining the test scenarios

## Proposed README Structure

```markdown
# TAS Capacity Analyzer

[badges - updated version]

## Features
[expanded list with all 11+ features]

## Screenshots
[dashboard + scenario wizard images]

## Quick Start
### Prerequisites
### Backend
### Frontend
### Sample Data

## Architecture
[data flow diagram]

## API Reference
[all 8 endpoints]

## Configuration
### Required Variables
### Optional: BOSH Integration
### Optional: vSphere Integration
### Tuning Options

## Development
### Testing
### CI/CD
### Project Structure

## Documentation
[all doc links]

## License
```

## Code References

- `README.md:1-62` - Current README content
- `frontend/package.json:15-38` - Frontend dependencies
- `backend/go.mod:5-8` - Backend dependencies
- `backend/handlers/handlers.go:52-59` - API endpoint registration
- `.github/workflows/ci.yml` - CI pipeline
- `.github/workflows/release.yml` - Release pipeline
- `frontend/src/components/wizard/` - Wizard components
- `backend/services/vsphere.go` - vSphere integration

## Architecture Documentation

The current README's project structure shows a simplified view. The actual architecture includes:

- **Two-tab dashboard**: Real-time metrics + Scenario planning
- **Three data sources**: Live vSphere, JSON upload, Manual entry
- **Step-based wizard**: Resources → Cell Config → Advanced
- **Comparison engine**: Current vs Proposed with warnings
- **Export capability**: Markdown reports

## Open Questions

1. Should the README include architecture diagrams (Mermaid)?
2. Should sample data file formats be documented?
3. Should there be a separate CONTRIBUTING.md file?
4. Should design docs in `docs/plans/` be linked from README?
