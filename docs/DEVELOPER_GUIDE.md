# Developer Guide for GitHub PI Scanner

## Table of Contents
- [Prerequisites](#prerequisites)
- [Setting Up Development Environment](#setting-up-development-environment)
- [Building the Application](#building-the-application)
- [Running Tests](#running-tests)
- [Docker Development](#docker-development)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements
- Go 1.23+ (required for generics support)
- Git
- macOS, Linux, or Windows (with WSL recommended)

### Required Dependencies

#### macOS
```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install required dependencies
brew install go
brew install git
```

#### Linux (Ubuntu/Debian)
```bash
# Update package list
sudo apt update

# Install Go (if not already installed)
sudo snap install go --classic

# Install Git
sudo apt install git

# No additional runtime dependencies needed
```

#### Windows
1. Install Go from https://go.dev/dl/
2. Install Git from https://git-scm.com/download/win
3. Ready to go - no additional dependencies

## Setting Up Development Environment

### 1. Clone the Repository
```bash
git clone https://github.com/MacAttak/pi-scanner.git
cd pi-scanner
```

### 2. Install Go Dependencies
```bash
go mod download
go mod verify
```

### 3. Set Up Pre-commit Hooks
```bash
# Install pre-commit
pip install pre-commit

# Install the git hook scripts
pre-commit install
```

### 4. Configure GitHub Authentication
```bash
# Set your GitHub personal access token
export GITHUB_TOKEN="your-github-personal-access-token"

# Or use GitHub CLI
gh auth login
```

## Building the Application

### Using Make (Recommended)
```bash
# Setup development environment
make setup

# Build the binary
make build

# Install to system PATH
make install
```

### Standard Build
```bash
# Build the binary
go build -o bin/pi-scanner ./cmd/pi-scanner

# Build with optimizations
go build -ldflags="-s -w" -o bin/pi-scanner ./cmd/pi-scanner
```

### Cross-Platform Build
```bash
# Build for all platforms using Make
make build-all

# Or build manually for specific platforms
GOOS=linux GOARCH=amd64 go build -o bin/pi-scanner-linux ./cmd/pi-scanner
GOOS=windows GOARCH=amd64 go build -o bin/pi-scanner.exe ./cmd/pi-scanner
GOOS=darwin GOARCH=amd64 go build -o bin/pi-scanner-darwin-amd64 ./cmd/pi-scanner
GOOS=darwin GOARCH=arm64 go build -o bin/pi-scanner-darwin-arm64 ./cmd/pi-scanner
```

## Running Tests

### Unit Tests
```bash
# Run all tests using Make
make test

# Run tests with coverage
make test-coverage

# Run tests with race detector
make test-race

# Or use Go directly
go test ./...
go test -cover ./...
```

### Integration Tests
```bash
# Run integration tests (requires GitHub authentication)
go test -tags=integration ./test/integration/...
```

### End-to-End Tests
```bash
# Run E2E tests (requires network access and GitHub authentication)
go test ./test -run TestPIScannerE2E

# Run specific E2E test
go test ./test -run TestPIScannerE2E_AustralianRepositories
```

### Benchmarks
```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkPatternDetection ./pkg/detection/...
```


## Docker Development

### Building Docker Image
```bash
# Build using Make
make docker-build

# Or build directly
docker build -t pi-scanner:latest .
```

### Running with Docker
```bash
# Run using Make
make docker-run ARGS="scan --repo github/docs"

# Or run directly
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v $(pwd)/output:/home/scanner/output \
  pi-scanner:latest scan --repo github/docs
```

### Docker Compose for Development
```yaml
version: '3.8'
services:
  pi-scanner:
    build: .
    environment:
      - GITHUB_TOKEN=${GITHUB_TOKEN}
    volumes:
      - ./output:/output
      - ./config:/config
    command: scan --repo github/docs --output /output/results.json
```

## Troubleshooting

### Build Issues

#### Go Module Issues
```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download
go mod verify
```

#### Binary Not Found
```bash
# Ensure binary directory exists
mkdir -p bin

# Build with full path
make build
```

### GitHub API Rate Limits
```
Error: GitHub API rate limit exceeded
```

**Solution:**
- Use authenticated requests: `export GITHUB_TOKEN=your-token`
- Implement caching for repeated scans
- Use GitHub Enterprise with higher limits

### Memory Issues
For large repositories:
```bash
# Increase memory limit
GOGC=200 ./pi-scanner scan --repo large/repository

# Use worker pool size adjustment
./pi-scanner scan --repo large/repository --workers 2
```

## Development Workflow

### 1. Create Feature Branch
```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes
- Follow TDD approach
- Write tests first
- Implement features
- Ensure all tests pass

### 3. Run Quality Checks
```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Check coverage
make test-coverage
```

### 4. Commit Changes
```bash
git add .
git commit -m "feat: add new detection pattern for Australian driver licenses"
```

### 5. Push and Create PR
```bash
git push origin feature/your-feature-name
# Create PR on GitHub
```

## Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [GitHub API Documentation](https://docs.github.com/en/rest)
- [Australian PI Standards](https://www.oaic.gov.au/)
- [Gitleaks Documentation](https://github.com/gitleaks/gitleaks)

## Contributing

Please read [CONTRIBUTING.md](../CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.