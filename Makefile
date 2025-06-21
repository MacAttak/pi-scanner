# GitHub PI Scanner - Development Makefile
# All commands run in Docker to ensure environment parity with CI

.PHONY: help build test test-race lint format vet clean dev shell deps coverage security bench

# Default target
help: ## Show this help message
	@echo "GitHub PI Scanner Development Commands"
	@echo "======================================"
	@echo "All commands run in Docker for environment parity with CI"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Environment Validation:"
	@echo "  - Docker and Docker Compose are required"
	@echo "  - All commands run in containerized environment"

# Environment validation
check-docker:
	@which docker >/dev/null || (echo "❌ Docker is required but not installed" && exit 1)
	@which docker-compose >/dev/null || docker compose version >/dev/null || (echo "❌ Docker Compose is required but not installed" && exit 1)
	@echo "✅ Docker environment validated"

# Build development environment
build: check-docker ## Build development Docker environment
	@echo "🏗️  Building development environment..."
	docker compose build --no-cache dev

# Development shell
dev: check-docker ## Start interactive development shell in Docker
	@echo "🐚 Starting development shell..."
	docker compose run --rm dev bash

shell: dev ## Alias for dev command

# Testing commands
test: check-docker ## Run tests with CI tags (no race detection)
	@echo "🧪 Running tests in Docker..."
	docker compose run --rm dev bash -c "CGO_ENABLED=1 go test -tags ci -v -coverprofile=coverage.out -covermode=atomic \$$(go list ./... | grep -v '/test\$$')"

test-race: check-docker ## Run tests with race detection (for debugging)
	@echo "🏃 Running tests with race detection in Docker..."
	docker compose run --rm dev bash -c "CGO_ENABLED=1 go test -tags ci -v -race \$$(go list ./... | grep -v '/test\$$')"

test-all: check-docker ## Run all tests including E2E (no build tags)
	@echo "🧪 Running all tests (including E2E) in Docker..."
	docker compose run --rm dev bash -c "CGO_ENABLED=1 go test -v ./..."

# Code quality commands
lint: check-docker ## Run golangci-lint
	@echo "🔍 Running linter in Docker..."
	docker compose run --rm dev bash -c "GOROOT=/usr/local/go golangci-lint run --timeout=5m"

format: check-docker ## Format Go code
	@echo "✨ Formatting code in Docker..."
	docker compose run --rm dev bash -c "gofmt -w ."

vet: check-docker ## Run go vet
	@echo "🔍 Running go vet in Docker..."
	docker compose run --rm dev bash -c "go vet ./..."

# Coverage and reporting
coverage: check-docker ## Generate test coverage report
	@echo "📊 Generating coverage report in Docker..."
	docker compose run --rm dev bash -c "CGO_ENABLED=1 go test -tags ci -coverprofile=coverage.out -covermode=atomic \$$(go list ./... | grep -v '/test\$$') && go tool cover -html=coverage.out -o coverage.html"
	@echo "📄 Coverage report generated: coverage.html"

# Security scanning
security: check-docker ## Run security scans
	@echo "🔒 Running security scans in Docker..."
	docker compose run --rm dev bash -c "gosec -fmt sarif -out gosec-results.sarif ./... || echo 'Security issues found - check gosec-results.sarif'"

# Performance benchmarks
bench: check-docker ## Run performance benchmarks
	@echo "⚡ Running benchmarks in Docker..."
	docker compose run --rm dev bash -c "go test -tags ci -bench=. -benchmem ./..."

# Dependency management
deps: check-docker ## Download and tidy dependencies
	@echo "📦 Managing dependencies in Docker..."
	docker compose run --rm dev bash -c "go mod download && go mod tidy"

# Quality gates (same as pre-push hooks)
quality-gates: check-docker ## Run all quality gates
	@echo "🚪 Running quality gates in Docker..."
	docker compose run --rm dev bash -c "./scripts/check-quality-gates.sh"

# Cleanup
clean: check-docker ## Clean build artifacts and Docker resources
	@echo "🧹 Cleaning up..."
	docker compose down --volumes --remove-orphans
	docker system prune -f

# CI simulation - runs the exact same commands as CI
ci-local: check-docker ## Simulate CI pipeline locally
	@echo "🤖 Running CI pipeline simulation in Docker..."
	docker compose run --rm dev bash -c "set -e && \
		echo '=== Running Go Format Check ===' && \
		if [ -n \"\$$(gofmt -l .)\" ]; then echo 'Code needs formatting:' && gofmt -l . && exit 1; fi && \
		echo '=== Running Go Vet ===' && \
		go vet ./... && \
		echo '=== Running Tests with Coverage ===' && \
		CGO_ENABLED=1 go test -tags ci -v -coverprofile=coverage.out -covermode=atomic \$$(go list ./... | grep -v '/test\$$') && \
		echo '=== Running Linter ===' && \
		GOROOT=/usr/local/go golangci-lint run --timeout=5m && \
		echo '✅ CI simulation completed successfully'"

# Warning for direct Go commands - this target gets called if someone tries direct go commands
go-warning:
	@echo "⚠️  WARNING: Direct go commands detected!"
	@echo "   For environment parity, use 'make test' instead of 'go test'"
	@echo "   For environment parity, use 'make dev' for interactive development"
	@echo "   All commands should run in Docker to match CI environment"

# Build commands (also in Docker for consistency)
build-scanner: check-docker ## Build pi-scanner binary
	@echo "🔨 Building pi-scanner in Docker..."
	docker compose run --rm dev bash -c "CGO_ENABLED=0 go build -ldflags='-s -w' -o bin/pi-scanner ./cmd/pi-scanner"

build-all: check-docker ## Build for all platforms
	@echo "🔨 Building for all platforms in Docker..."
	docker compose run --rm dev bash -c "mkdir -p bin && \
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o bin/pi-scanner-linux-amd64 ./cmd/pi-scanner && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags='-s -w' -o bin/pi-scanner-darwin-amd64 ./cmd/pi-scanner && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags='-s -w' -o bin/pi-scanner-darwin-arm64 ./cmd/pi-scanner && \
		CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags='-s -w' -o bin/pi-scanner-windows-amd64.exe ./cmd/pi-scanner"

# Legacy support with warnings (gradually migrate users to Docker commands)
legacy-test: go-warning
	@echo "🚨 Use 'make test' instead for environment parity!"
	@exit 1

legacy-lint: go-warning
	@echo "🚨 Use 'make lint' instead for environment parity!"
	@exit 1

# Makefile magic to catch common Go commands and redirect to Docker versions
%:
	@if echo "$@" | grep -E "^(go|golangci-lint|gosec|govulncheck)" >/dev/null; then \
		echo "⚠️  Direct $@ command detected!"; \
		echo "   Use 'make dev' for interactive development"; \
		echo "   Use 'make test' for testing"; \
		echo "   Use 'make lint' for linting"; \
		echo "   All commands should run in Docker for environment parity"; \
		exit 1; \
	fi
