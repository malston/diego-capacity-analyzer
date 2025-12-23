# ABOUTME: Build and development targets for diego-capacity-analyzer
# ABOUTME: Includes backend (Go) and frontend (React) commands

.PHONY: help all build test lint check clean
.PHONY: backend-build backend-test backend-lint backend-clean backend-run
.PHONY: frontend-build frontend-test frontend-lint frontend-dev frontend-clean

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

build: backend-build frontend-build ## Build backend and frontend

test: backend-test frontend-test ## Run all tests

lint: backend-lint frontend-lint ## Run all linters

check: test lint ## Run tests and linters

clean: backend-clean frontend-clean ## Clean all build artifacts

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

backend-run: backend-build ## Build and run the backend server
	cd backend && ./capacity-backend

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

frontend-dev: ## Start frontend dev server
	cd frontend && npm run dev

frontend-clean: ## Remove frontend build artifacts
	rm -rf frontend/dist

frontend-install: ## Install frontend dependencies
	cd frontend && npm install
