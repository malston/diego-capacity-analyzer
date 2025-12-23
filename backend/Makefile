# ABOUTME: Build and development targets for the backend service
# ABOUTME: Includes test, lint, and build commands

.PHONY: build test lint clean run

# Build the backend binary
build:
	go build -o capacity-backend .

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run linter (staticcheck)
lint:
	staticcheck ./...

# Run tests and linter
check: test lint

# Clean build artifacts
clean:
	rm -f capacity-backend

# Run the backend (requires environment variables)
run: build
	./capacity-backend
