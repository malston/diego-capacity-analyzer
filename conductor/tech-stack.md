# Technology Stack: TAS Capacity Analyzer

## Backend

### Language and Runtime
- **Go 1.23** - Primary backend language
- **Standard library HTTP server** - No external web framework; uses `net/http`

### External Dependencies
| Package | Purpose |
|---------|---------|
| `github.com/vmware/govmomi` | vSphere/vCenter API client for infrastructure discovery |
| `github.com/cloudfoundry/socks5-proxy` | SSH tunneling for BOSH Director connections through Ops Manager |

### API Integrations
- **BOSH Director API** - VM inventory, vitals, deployment info (OAuth via UAA)
- **Cloud Foundry API** - Application and process metadata
- **Log Cache API** - Container memory metrics
- **vSphere/vCenter API** - Infrastructure discovery (clusters, hosts, VMs)

## Frontend

### Core Framework
- **React 18** - UI component framework
- **Vite 5** - Build tool and development server

### Styling
- **Tailwind CSS 3** - Utility-first CSS framework
- **PostCSS** - CSS processing

### UI Libraries
| Package | Purpose |
|---------|---------|
| `recharts` | Data visualization (charts, graphs) |
| `lucide-react` | Icon library |

### Testing
- **Vitest** - Test runner (Vite-native)
- **Testing Library** - React component testing (`@testing-library/react`)
- **jsdom** - DOM simulation for tests

### Development Tools
- **ESLint 9** - Code linting
- **TypeScript types** - Type definitions for React (`@types/react`, `@types/react-dom`)

## Deployment

### Cloud Foundry
- `manifest.yml` in both `backend/` and `frontend/` directories
- Frontend deployed as static files
- Backend deployed as Go binary

### CI/CD
- **GitHub Actions** - Automated pipelines
  - CI workflow: lint, test, build on PRs and main branch
  - Release workflow: cross-compile binaries on version tags

### Supported Platforms (Release Builds)
- linux/amd64
- linux/arm64
- darwin/amd64
- darwin/arm64

## Development Environment

### Prerequisites
- Go 1.23+
- Node.js (for frontend development)
- npm (package manager)

### Local Development
```bash
# Backend
cd backend && go run main.go

# Frontend
cd frontend && npm install && npm run dev
```

## Architecture Notes

### Monorepo Structure
Single repository containing both backend and frontend code with independent build processes.

### API Design
RESTful JSON API with endpoints for:
- Health checks
- Dashboard data aggregation
- Infrastructure discovery and state management
- Scenario comparison and planning calculations

### Caching
In-memory caching with configurable TTL for:
- Dashboard data
- vSphere infrastructure data
- BOSH/CF API responses
