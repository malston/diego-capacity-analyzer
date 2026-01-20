# ABOUTME: Build and development targets for diego-capacity-analyzer
# ABOUTME: Includes backend (Go) and frontend (React) commands

# Configurable ports (override with: make backend-run BACKEND_PORT=9090)
BACKEND_PORT ?= 8080
FRONTEND_PORT ?= 5173

.PHONY: help all build test lint check clean
.PHONY: backend-build backend-test backend-lint backend-clean backend-run backend-dev backend-air
.PHONY: frontend-build frontend-test frontend-lint frontend-dev frontend-preview frontend-clean
.PHONY: cli-build cli-test cli-lint cli-clean cli-install
.PHONY: openapi-serve openapi-validate openapi-clean

.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

all: check build ## Run checks and build everything

#
# Combined targets
#

build: backend-build frontend-build cli-build ## Build backend, frontend, and CLI

test: backend-test frontend-test cli-test ## Run all tests

lint: backend-lint frontend-lint cli-lint ## Run all linters

check: test lint ## Run tests and linters

clean: backend-clean frontend-clean cli-clean ## Clean all build artifacts

#
# Backend targets (Go)
#

backend-build: ## Build Go backend binary
	cd backend && go build -o capacity-backend .

backend-test: ## Run backend tests
	cd backend && go test ./...

backend-test-verbose: ## Run backend tests with verbose output
	cd backend && go test -v ./...

backend-lint: ## Run staticcheck on backend
	cd backend && staticcheck ./...

backend-clean: ## Remove backend build artifacts
	rm -f backend/capacity-backend

backend-run: backend-build ## Build and run the backend server (PORT=$(BACKEND_PORT))
	cd backend && PORT=$(BACKEND_PORT) ./capacity-backend

backend-dev: ## Run backend with auto-reload (PORT=$(BACKEND_PORT))
	@if command -v watchexec >/dev/null 2>&1; then \
		cd backend && PORT=$(BACKEND_PORT) watchexec -r -e go -- go run .; \
	elif command -v air >/dev/null 2>&1; then \
		cd backend && PORT=$(BACKEND_PORT) air; \
	else \
		echo "No auto-reload tool found. Install watchexec or air for auto-reload."; \
		echo "Falling back to 'go run' (manual restart required)"; \
		cd backend && PORT=$(BACKEND_PORT) go run .; \
	fi

backend-air: ## Run backend with air (explicit choice over watchexec)
	@if command -v air >/dev/null 2>&1; then \
		cd backend && PORT=$(BACKEND_PORT) air; \
	else \
		echo "Error: air not found. Install with: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

#
# Frontend targets (React/Vite)
#

frontend-build: ## Build frontend for production
	cd frontend && npm run build

frontend-test: ## Run frontend tests
	cd frontend && npm test

frontend-test-watch: ## Run frontend tests in watch mode
	cd frontend && npm run test:watch

frontend-test-coverage: ## Run frontend tests with coverage
	cd frontend && npm run test:coverage

frontend-lint: ## Run ESLint on frontend
	cd frontend && npm run lint

frontend-dev: ## Start frontend dev server (PORT=$(FRONTEND_PORT))
	cd frontend && npm run dev -- --port $(FRONTEND_PORT)

frontend-preview: frontend-build ## Build and preview production build locally
	cd frontend && npm run preview

frontend-clean: ## Remove frontend build artifacts
	rm -rf frontend/dist

frontend-install: ## Install frontend dependencies
	cd frontend && npm install

#
# CLI targets (diego-capacity)
#

cli-build: ## Build CLI binary (diego-capacity)
	cd cli && go build -o diego-capacity .

cli-test: ## Run CLI tests
	cd cli && go test ./...

cli-test-verbose: ## Run CLI tests with verbose output
	cd cli && go test -v ./...

cli-lint: ## Run staticcheck on CLI
	cd cli && staticcheck ./...

cli-clean: ## Remove CLI build artifacts
	rm -f cli/diego-capacity

cli-install: cli-build ## Install CLI to $GOPATH/bin
	@GOPATH_BIN=$${GOPATH:-$$(go env GOPATH)}/bin; \
	mkdir -p $$GOPATH_BIN; \
	cp cli/diego-capacity $$GOPATH_BIN/

#
# OpenAPI Documentation targets
#

OPENAPI_PORT ?= 8090
OPENAPI_SPEC := openapi.yaml
OPENAPI_SERVER_DIR := .openapi-server

openapi-serve: ## Serve OpenAPI docs via Swagger UI (PORT=$(OPENAPI_PORT))
	@if [ ! -f "$(OPENAPI_SPEC)" ]; then \
		echo "Error: $(OPENAPI_SPEC) not found. Run 'make openapi-generate' first."; \
		exit 1; \
	fi
	@mkdir -p $(OPENAPI_SERVER_DIR)
	@cp $(OPENAPI_SPEC) $(OPENAPI_SERVER_DIR)/
	@if [ ! -f "$(OPENAPI_SERVER_DIR)/swagger-ui-bundle.js" ]; then \
		echo "Downloading Swagger UI assets..."; \
		curl -sL https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js -o $(OPENAPI_SERVER_DIR)/swagger-ui-bundle.js; \
		curl -sL https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js -o $(OPENAPI_SERVER_DIR)/swagger-ui-standalone-preset.js; \
		curl -sL https://unpkg.com/swagger-ui-dist@5/swagger-ui.css -o $(OPENAPI_SERVER_DIR)/swagger-ui.css; \
	fi
	@if [ ! -f "$(OPENAPI_SERVER_DIR)/index.html" ]; then \
		echo '<!DOCTYPE html><html><head><title>Diego Capacity Analyzer API</title><link rel="stylesheet" href="swagger-ui.css"></head><body><div id="swagger-ui"></div><script src="swagger-ui-bundle.js"></script><script src="swagger-ui-standalone-preset.js"></script><script>SwaggerUIBundle({url:"openapi.yaml",dom_id:"#swagger-ui",presets:[SwaggerUIBundle.presets.apis,SwaggerUIStandalonePreset],layout:"StandaloneLayout"});</script></body></html>' > $(OPENAPI_SERVER_DIR)/index.html; \
	fi
	@echo "Serving OpenAPI docs at http://localhost:$(OPENAPI_PORT)"
	@cd $(OPENAPI_SERVER_DIR) && python3 -m http.server $(OPENAPI_PORT)

openapi-validate: ## Validate OpenAPI spec syntax
	@if [ ! -f "$(OPENAPI_SPEC)" ]; then \
		echo "Error: $(OPENAPI_SPEC) not found."; \
		exit 1; \
	fi
	@if command -v swagger-cli >/dev/null 2>&1; then \
		swagger-cli validate $(OPENAPI_SPEC); \
	elif command -v npx >/dev/null 2>&1; then \
		npx @apidevtools/swagger-cli validate $(OPENAPI_SPEC); \
	else \
		echo "Validating YAML syntax only (install swagger-cli for full validation)..."; \
		python3 -c "import yaml; yaml.safe_load(open('$(OPENAPI_SPEC)'))" && echo "$(OPENAPI_SPEC) is valid YAML"; \
	fi

openapi-clean: ## Remove OpenAPI server artifacts
	rm -rf $(OPENAPI_SERVER_DIR)
