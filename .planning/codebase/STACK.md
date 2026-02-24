# Technology Stack

**Analysis Date:** 2026-02-24

## Languages

**Primary:**

- Go 1.24.0 - Backend HTTP service (`backend/`)
- Go 1.25.5 - CLI tool (`cli/`)
- JavaScript (ES2020+) - Frontend React app (`frontend/src/`)

**Secondary:**

- JSX - React component templates (`frontend/src/components/`, `frontend/src/`)
- YAML - OpenAPI spec (`backend/handlers/openapi.yaml`), CF manifests (`backend/manifest.yml`, `frontend/manifest.yml`)

## Runtime

**Environment:**

- Go standard toolchain for backend and CLI
- Node.js (version not pinned; no `.nvmrc` present) for frontend build and dev server

**Package Manager:**

- Go modules for backend and CLI (lockfile: `backend/go.sum`, `cli/go.sum`)
- npm for frontend (lockfile: `frontend/package-lock.json`)
  - Note: `package-lock.json` present, not bun lockfile. `npm` is used explicitly in `Makefile`.

## Frameworks

**Backend:**

- Go standard library `net/http` (Go 1.22+ method+path routing pattern) - HTTP server
- `log/slog` (Go standard library) - Structured logging

**Frontend:**

- React 18.2.0 - UI framework (`frontend/src/`)
- Vite 5.0.8 - Build tool and dev server (`frontend/vite.config.js`)
- Tailwind CSS 3.3.6 - Utility-first styling (`frontend/tailwind.config.js`)
- Recharts 2.10.3 - Data visualization charts
- `swagger-ui-react` 5.31.0 - Embedded OpenAPI documentation viewer

**CLI:**

- Cobra 1.10.2 - CLI command framework (`cli/cmd/`)
- Bubbletea 1.3.10 - TUI framework
- Bubbles 0.21.1 - TUI components
- Huh 0.8.0 - Interactive TUI forms
- Lipgloss 1.1.0 - TUI styling

**Testing (Backend):**

- Go standard `testing` package
- `stretchr/testify` 1.10.0 - Assertions and test helpers

**Testing (Frontend):**

- Vitest 4.0.16 - Test runner (`frontend/vite.config.js` test config)
- jsdom 27.3.0 - DOM simulation environment
- `@testing-library/react` 16.3.1 - React component testing
- `@testing-library/user-event` 14.6.1 - User interaction simulation
- `@testing-library/jest-dom` 6.9.1 - Custom DOM matchers

**Build/Dev:**

- `staticcheck` - Go static analysis linter (external tool, invoked in `Makefile`)
- ESLint 9.39.2 with flat config (`frontend/eslint.config.js`)
- PostCSS 8.4.32 + autoprefixer 10.4.16 (`frontend/postcss.config.js`)
- watchexec or air (optional) - Backend auto-reload during development

## Key Dependencies

**Backend Critical:**

- `github.com/vmware/govmomi` v0.52.0 - vSphere/vCenter API client (`backend/services/vsphere.go`)
- `github.com/cloudfoundry/socks5-proxy` v0.2.101 - SSH+SOCKS5 tunnel for BOSH on non-routable networks (`backend/services/boshapi.go`)
- `github.com/joho/godotenv` v1.5.1 - `.env` file loading at startup (`backend/main.go`)
- `golang.org/x/sync` v0.19.0 - `singleflight` for JWKS thundering herd prevention (`backend/services/jwks.go`)

**Backend Infrastructure:**

- `github.com/google/uuid` v1.6.0 - UUID generation (indirect)
- `golang.org/x/crypto` v0.14.0 - Cryptographic primitives (indirect; used via SSH key handling)

**Frontend Critical:**

- `lucide-react` 0.294.0 - Icon library
- `recharts` 2.10.3 - All chart rendering

**CLI Critical:**

- `github.com/charmbracelet/bubbletea` v1.3.10 - TUI event loop
- `github.com/charmbracelet/huh` v0.8.0 - Interactive form flows
- `github.com/spf13/cobra` v1.10.2 - Command/subcommand structure

## Configuration

**Environment:**

- Loaded from `.env` file at startup via `godotenv` (optional; falls back gracefully)
- Backend searches current directory then `../` for `.env`
- All config via environment variables; see `backend/config/config.go` for full list
- Template: `.env.example` at project root

**Required env vars:**

- `CF_API_URL` - Cloud Foundry API endpoint (e.g., `https://api.sys.example.com`)
- `CF_USERNAME` - CF admin username
- `CF_PASSWORD` - CF admin password

**Optional env vars (with defaults):**

- `PORT` (default: `8080`) - Backend HTTP server port
- `AUTH_MODE` (default: `optional`) - `disabled` | `optional` | `required`
- `CACHE_TTL` (default: `300`) - General cache TTL in seconds
- `DASHBOARD_CACHE_TTL` (default: `30`) - BOSH/CF data cache TTL in seconds
- `LOG_LEVEL` (default: `info`) - `debug` | `info` | `warn` | `error`
- `LOG_FORMAT` (default: `text`) - `text` | `json`
- `CORS_ALLOWED_ORIGINS` - Comma-separated allowed origins (empty blocks all cross-origin)
- `RATE_LIMIT_ENABLED` (default: `true`) - Enable/disable rate limiting
- `COOKIE_SECURE` (default: `true`) - Set `false` for local HTTP dev

**Build:**

- `backend/go.mod`, `backend/go.sum` - Backend Go modules
- `cli/go.mod`, `cli/go.sum` - CLI Go modules
- `frontend/package.json`, `frontend/package-lock.json` - Frontend npm
- `frontend/vite.config.js` - Vite build and dev server config
- `Makefile` - All build/test/lint targets (canonical entry point)

## Platform Requirements

**Development:**

- Go 1.24+ (backend), Go 1.25+ (CLI)
- Node.js + npm (frontend)
- `staticcheck` (optional, needed for `make lint`)
- watchexec or air (optional, needed for `make backend-dev` auto-reload)
- Ops Manager CLI (`om`) if using `generate-env.sh`

**Production:**

- Single Go binary: `backend/capacity-backend` (compiled via `make backend-build`)
- Frontend static files: `frontend/dist/` (compiled via `make frontend-build`)
- Can be deployed to Cloud Foundry using `backend/manifest.yml` and `frontend/manifest.yml`
- CLI binary: `cli/diego-capacity` (compiled via `make cli-build`)

---

_Stack analysis: 2026-02-24_
