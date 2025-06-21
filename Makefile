# Makefile for GitHub PI Scanner

# Variables
BINARY_NAME=pi-scanner
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_TEST=$(GO_CMD) test
GO_CLEAN=$(GO_CMD) clean
GO_GET=$(GO_CMD) get
GO_MOD=$(GO_CMD) mod
BINARY_DIR=bin
LDFLAGS=-ldflags="-s -w"

# Platform detection
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Set CGO flags
export CGO_ENABLED=1

# Targets
.PHONY: all build test clean deps run help setup install-hooks pre-commit pre-push ci-local docker-dev docker-test docker-security docker-lint docker-shell docker-build-dev

all: test build ## Run tests and build

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

setup: install-hooks ## Setup development environment
	@echo "Setting up development environment..."
	@mkdir -p $(BINARY_DIR)
	@echo "Setup complete!"

install-hooks: ## Install Git hooks for local development
	@echo "Installing Git hooks..."
	@./scripts/install-hooks.sh

deps: ## Download and verify dependencies
	$(GO_MOD) download
	$(GO_MOD) verify

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	$(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) ./cmd/pi-scanner

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/pi-scanner
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/pi-scanner
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/pi-scanner
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/pi-scanner
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/pi-scanner

test: ## Run all tests
	@echo "Running tests..."
	$(GO_TEST) -v ./...

test-short: ## Run tests in short mode (no integration tests)
	@echo "Running tests in short mode..."
	$(GO_TEST) -short -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GO_TEST) -cover -coverprofile=coverage.out ./...
	$(GO_CMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	$(GO_TEST) -race -v ./...

test-e2e: ## Run end-to-end tests
	@echo "Running E2E tests..."
	$(GO_TEST) -v ./test -run TestPIScannerE2E

benchmark-basic: ## Run basic benchmarks
	@echo "Running benchmarks..."
	$(GO_TEST) -bench=. -benchmem ./...

lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	$(GO_CMD) fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GO_CMD) vet ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GO_CLEAN)
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

run: build ## Build and run the scanner
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_DIR)/$(BINARY_NAME) $(ARGS)

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t pi-scanner:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -e GITHUB_TOKEN=$${GITHUB_TOKEN} -v $$(pwd)/output:/home/scanner/output pi-scanner:latest $(ARGS)

install: build ## Install binary to system
	@echo "Installing $(BINARY_NAME)..."
	@sudo cp $(BINARY_DIR)/$(BINARY_NAME) /usr/local/bin/

uninstall: ## Uninstall binary from system
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)

pre-commit: ## Run pre-commit checks
	@echo "Running pre-commit checks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run --all-files; \
	else \
		echo "pre-commit not installed. Run: make install-hooks"; \
		exit 1; \
	fi

pre-push: ## Run pre-push checks
	@echo "Running pre-push checks..."
	@./.githooks/pre-push

ci-local: ## Simulate CI pipeline locally
	@echo "Running local CI simulation..."
	@./scripts/ci-local.sh

# Docker Development Environment Targets
docker-build-dev: ## Build development Docker image
	@echo "Building development environment..."
	docker compose build dev

docker-shell: docker-build-dev ## Start interactive development shell
	@echo "Starting development shell..."
	docker compose run --rm dev

docker-dev: docker-build-dev ## Start development environment
	@echo "Starting development environment..."
	docker compose up -d dev
	@echo "Development environment ready. Use 'make docker-shell' to connect."

docker-test: docker-build-dev ## Run tests in Docker environment
	@echo "Running tests in Docker environment..."
	docker compose run --rm test-dev

docker-security: docker-build-dev ## Run security scans in Docker environment
	@echo "Running security scans in Docker environment..."
	docker compose run --rm security-dev

docker-lint: docker-build-dev ## Run linting in Docker environment
	@echo "Running linting in Docker environment..."
	docker compose run --rm dev golangci-lint run

docker-fmt: docker-build-dev ## Format code in Docker environment
	@echo "Formatting code in Docker environment..."
	docker compose run --rm dev go fmt ./...

docker-clean: ## Clean Docker development environment
	@echo "Cleaning Docker development environment..."
	docker compose down --volumes --remove-orphans
	docker volume prune -f

docker-ci: docker-build-dev ## Run full CI pipeline in Docker
	@echo "Running full CI pipeline in Docker environment..."
	docker compose run --rm dev bash -c " \
		echo 'Running go fmt...' && \
		go fmt ./... && \
		echo 'Running go vet...' && \
		go vet ./... && \
		echo 'Running tests...' && \
		go test -v ./... && \
		echo 'Running golangci-lint...' && \
		golangci-lint run && \
		echo 'Running gosec...' && \
		gosec ./... && \
		echo 'Running govulncheck...' && \
		govulncheck ./... && \
		echo 'Building binaries...' && \
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o bin/pi-scanner-linux-amd64 ./cmd/pi-scanner && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags='-s -w' -o bin/pi-scanner-darwin-amd64 ./cmd/pi-scanner && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags='-s -w' -o bin/pi-scanner-darwin-arm64 ./cmd/pi-scanner && \
		echo 'All checks passed!'"

# Quality Gates
quality-check: ## Run all quality gate checks
	@echo "Running quality gate checks..."
	@./scripts/check-quality-gates.sh

quality-report: ## Generate quality reports
	@echo "Generating quality reports..."
	@mkdir -p .quality-reports
	@./scripts/check-quality-gates.sh || true
	@echo "Reports generated in .quality-reports/"

coverage: ## Generate coverage report with visualization
	@echo "Generating coverage report..."
	@./scripts/coverage-report.sh

coverage-html: ## Open coverage HTML report
	@echo "Opening coverage report..."
	@./scripts/coverage-report.sh
	@open .coverage/coverage.html 2>/dev/null || xdg-open .coverage/coverage.html 2>/dev/null || echo "Open .coverage/coverage.html manually"

benchmark: ## Run and track benchmarks
	@echo "Running benchmarks..."
	@./scripts/benchmark-track-simple.sh

benchmark-compare: ## Compare benchmarks with baseline
	@echo "Comparing benchmarks..."
	@./scripts/benchmark-track-simple.sh
	@[ -f .benchmarks/comparison.md ] && cat .benchmarks/comparison.md || echo "No comparison available"

benchmark-update: ## Update benchmark baseline
	@echo "Updating benchmark baseline..."
	@UPDATE_BASELINE=true ./scripts/benchmark-track-simple.sh

quality-install: ## Install quality gate pre-commit hooks
	@echo "Installing enhanced pre-commit hooks..."
	@cp .pre-commit-config-enhanced.yaml .pre-commit-config.yaml
	@pre-commit install
	@pre-commit install --hook-type pre-push
	@echo "Quality gates installed!"

quality-dashboard: ## Show quality metrics dashboard
	@echo "Quality Metrics Dashboard"
	@echo "========================"
	@echo ""
	@echo "📊 Test Coverage:"
	@go test -cover ./... 2>/dev/null | grep -E "coverage:|ok" | tail -10
	@echo ""
	@echo "📈 Recent Benchmarks:"
	@[ -f .benchmarks/history.json ] && jq -r '.[-3:] | reverse | .[] | "\(.timestamp | split("T")[0]): \(.avg_ns_per_op) ns/op"' .benchmarks/history.json || echo "No benchmark history"
	@echo ""
	@echo "✅ Quality Score:"
	@[ -f .quality-reports/quality-summary.json ] && jq -r '"Score: \(.score)% (Passed: \(.passed), Failed: \(.failed), Warnings: \(.warnings))"' .quality-reports/quality-summary.json || echo "Run 'make quality-check' first"

.DEFAULT_GOAL := help
