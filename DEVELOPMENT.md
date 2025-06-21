# Development Guide - GitHub PI Scanner

## Environment Parity Philosophy

**All development commands run in Docker to ensure 100% environment parity with CI.**

This project enforces containerized development to eliminate "works on my machine" issues and ensure that local development exactly matches our CI/CD pipeline.

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Make (optional but recommended)

### Development Commands

```bash
# Show all available commands
make help

# Start interactive development shell
make dev

# Run tests (same as CI)
make test

# Run linting (same as CI)
make lint

# Generate test coverage
make coverage

# Simulate entire CI pipeline locally
make ci-local
```

## Why Docker-Enforced Development?

### The Problem We Solved
Our CI/CD pipeline was failing with issues that didn't reproduce locally:
- Race conditions in containerized vs host environments
- Go toolchain version mismatches (1.23.9 vs 1.24.0)
- Different package resolution behavior
- Environment variable handling differences

### The Solution
**Every development command runs in the same Docker container as CI:**
- Same Go version (1.23.9)
- Same Ubuntu base (24.04)
- Same installed tools (golangci-lint, gosec, govulncheck)
- Same environment variables and paths

## Development Workflow

### 1. Interactive Development
```bash
# Start development shell (runs in Docker)
make dev

# Inside the container, you have access to:
# - Go 1.23.9
# - All development tools
# - Your source code (mounted volume)
# - Same environment as CI
```

### 2. Testing
```bash
# Run tests exactly as CI does
make test

# Run with race detection (for debugging)
make test-race

# Run E2E tests (includes network tests)
make test-all
```

### 3. Code Quality
```bash
# Format code
make format

# Run linter
make lint

# Run security scans
make security

# Run all quality gates
make quality-gates
```

### 4. Pre-commit Hooks
Pre-commit hooks now run in Docker for environment parity:

```bash
# Install pre-commit hooks (they use Docker)
pre-commit install

# All hooks run in Docker automatically
git commit -m "your changes"
```

## Environment Alignment Features

### 1. Docker-Based Pre-commit Hooks
All pre-commit hooks use `docker compose run --rm dev` to ensure:
- Same Go version as CI
- Same tool versions as CI
- Same environment variables as CI

### 2. Makefile Command Interception
The Makefile intercepts direct Go commands and redirects to Docker:
```bash
# This will show a warning and suggest alternatives:
go test ./...

# Use this instead:
make test
```

### 3. CI Simulation
```bash
# Run the exact same commands as CI
make ci-local

# This runs:
# - Go format check
# - Go vet
# - Tests with coverage
# - Linting
# - All in the same Docker environment as CI
```

## Troubleshooting

### Docker Issues
```bash
# Rebuild development environment
make build

# Clean up Docker resources
make clean

# Check Docker installation
make help  # This validates Docker automatically
```

### Performance
Docker development might be slower than host development, but it ensures reliability:
- First build takes longer (downloads dependencies)
- Subsequent builds use cached layers
- Volume mounting provides real-time file sync

### Legacy Commands
If you accidentally use direct Go commands, you'll see:
```
⚠️  Direct go test command detected!
   Use 'make test' instead for environment parity!
   Use 'make dev' for interactive development
```

## Advanced Usage

### Custom Commands in Development Shell
```bash
# Start development shell
make dev

# Inside the container, run any Go commands:
go test -v ./pkg/detection
go build -o debug-binary ./cmd/pi-scanner
golangci-lint run --config .custom-config.yml
```

### Debugging Race Conditions
```bash
# Run tests with race detection (disabled in CI temporarily)
make test-race

# If race detected locally, it will also fail in CI
```

### Building for Production
```bash
# Build binaries for all platforms (in Docker)
make build-all

# Binaries are created in bin/ directory
ls bin/
```

## Migration from Host Development

### Old Workflow (Host-based)
```bash
go test ./...           # ❌ Different environment than CI
golangci-lint run       # ❌ Different tool version than CI
go vet ./...           # ❌ Different Go version than CI
```

### New Workflow (Docker-based)
```bash
make test              # ✅ Same environment as CI
make lint              # ✅ Same tools as CI
make vet               # ✅ Same Go version as CI
```

## Quality Gates

All quality gates run in Docker and match CI exactly:

1. **Formatting**: `gofmt` checks
2. **Static Analysis**: `go vet` and `golangci-lint`
3. **Testing**: Unit tests with coverage
4. **Security**: `gosec` and `govulncheck`
5. **Dependencies**: Module tidiness
6. **Documentation**: README and LICENSE checks

## Performance Benchmarks

```bash
# Run performance benchmarks
make bench

# Track benchmark history (uses scripts/benchmark-track-simple.sh)
./scripts/benchmark-track-simple.sh
```

## CI/CD Integration

The local `make ci-local` command runs the exact same sequence as our GitHub Actions:

1. Format check
2. Static analysis
3. Tests with coverage
4. Linting
5. Security scanning

This ensures no surprises when pushing to CI.

## Best Practices

1. **Always use `make` commands** instead of direct Go commands
2. **Use `make dev`** for interactive development
3. **Run `make ci-local`** before pushing
4. **Keep Docker images updated** with `make build`
5. **Use `make test-race`** when debugging concurrency issues

## Environment Variables

CI-specific environment variables are automatically handled:
- `GITHUB_TOKEN`: Passed to Docker container in CI
- `CGO_ENABLED=1`: Set for race detection support
- `GOROOT=/usr/local/go`: Ensures correct Go toolchain

## Support

If you encounter issues with the Docker-based development workflow:

1. Check Docker installation: `make help`
2. Rebuild environment: `make build`
3. Clean and retry: `make clean && make build`
4. Verify with CI simulation: `make ci-local`

This environment-aligned approach ensures that what works locally will work in CI, eliminating the most common source of CI/CD failures.
