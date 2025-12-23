# ABOUTME: Build and development targets for diego-capacity-analyzer
# ABOUTME: Includes backend (Go) and frontend (React) commands

.PHONY: all build test lint check clean
.PHONY: backend-build backend-test backend-lint backend-clean backend-run
.PHONY: frontend-build frontend-test frontend-lint frontend-dev frontend-clean

# Default target
all: check build

#
# Combined targets
#

# Build both backend and frontend
build: backend-build frontend-build

# Run all tests
test: backend-test frontend-test

# Run all linters
lint: backend-lint frontend-lint

# Run tests and linters
check: test lint

# Clean all build artifacts
clean: backend-clean frontend-clean

#
# Backend targets (Go)
#

backend-build:
	cd backend && go build -o capacity-backend .

backend-test:
	cd backend && go test ./...

backend-test-verbose:
	cd backend && go test -v ./...

backend-lint:
	cd backend && staticcheck ./...

backend-clean:
	rm -f backend/capacity-backend

backend-run: backend-build
	cd backend && ./capacity-backend

#
# Frontend targets (React/Vite)
#

frontend-build:
	cd frontend && npm run build

frontend-test:
	cd frontend && npm test

frontend-test-watch:
	cd frontend && npm run test:watch

frontend-test-coverage:
	cd frontend && npm run test:coverage

frontend-lint:
	cd frontend && npm run lint

frontend-dev:
	cd frontend && npm run dev

frontend-clean:
	rm -rf frontend/dist

# Install frontend dependencies
frontend-install:
	cd frontend && npm install
