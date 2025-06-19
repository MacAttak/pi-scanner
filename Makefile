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
LDFLAGS=-ldflags="-s -w -extldflags '-L./lib'"

# Platform detection
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

# Set CGO flags
export CGO_ENABLED=1
export CGO_LDFLAGS=-L${PWD}/lib
export CGO_CFLAGS=-I${PWD}/lib/tokenizers-src

# For runtime linking on macOS
export DYLD_LIBRARY_PATH:=${PWD}/lib:$(DYLD_LIBRARY_PATH)
# For Linux  
export LD_LIBRARY_PATH:=${PWD}/lib:$(LD_LIBRARY_PATH)

# Targets
.PHONY: all build test clean deps run help setup

all: test build ## Run tests and build

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

setup: ## Setup development environment
	@echo "Setting up development environment..."
	@mkdir -p $(BINARY_DIR)
	@mkdir -p lib
	@echo "Note: tokenizers library must be built from source for v1.20.2"
	@echo "Please build the tokenizers library manually:"
	@echo "  1. cd lib/tokenizers-src"
	@echo "  2. make build"
	@echo "  3. cd ../.. && cp lib/tokenizers-src/libtokenizers.a lib/"
	@echo "Setup complete!"

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

test-no-ml: ## Run tests excluding ML packages
	@echo "Running tests (excluding ML)..."
	$(GO_TEST) -v $$(go list ./... | grep -v '/ml')

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

benchmark: ## Run benchmarks
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

.DEFAULT_GOAL := help