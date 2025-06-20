# Developer Guide

This guide provides instructions for developers working on the GitHub PI Scanner project.

## Table of Contents

- [Development Setup](#development-setup)
- [Git Hooks and CI](#git-hooks-and-ci)
- [Testing](#testing)
- [Code Quality](#code-quality)
- [Troubleshooting](#troubleshooting)

## Development Setup

### Prerequisites

- Go 1.23 or later
- Git
- Make
- Docker (optional, for containerized testing)

### Initial Setup

1. Clone the repository:
```bash
git clone https://github.com/MacAttak/pi-scanner.git
cd pi-scanner
```

2. Install dependencies and set up development environment:
```bash
make setup
```

This will:
- Install Git hooks for pre-commit and pre-push checks
- Set up the development environment
- Create necessary directories

### Installing Optional Tools

For the best development experience, install these optional tools:

```bash
# Go tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

# Other tools
brew install pre-commit  # macOS
brew install gitleaks    # macOS

# Or use pip for pre-commit
pip install pre-commit
```

## Git Hooks and CI

### Shift-Left Strategy

We use a "shift-left" approach to catch issues early in the development process:

1. **Pre-commit hooks**: Quick checks that run before each commit
2. **Pre-push hooks**: Comprehensive tests that run before pushing
3. **Local CI simulation**: Full CI pipeline simulation

### Pre-commit Checks

Pre-commit hooks run automatically on `git commit`. They include:
- Go formatting checks
- Import organization
- Basic linting
- Secret scanning

Run manually:
```bash
make pre-commit
# or
pre-commit run --all-files
```

### Pre-push Checks

Pre-push hooks run automatically on `git push`. They include:
- All pre-commit checks
- Full test suite
- Security scanning
- Build verification

Run manually:
```bash
make pre-push
```

### Local CI Simulation

Simulate the full CI pipeline locally before pushing:

```bash
make ci-local
```

This runs:
- Code quality checks
- Build verification
- Full test suite with coverage
- Security scans
- Cross-platform build tests

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests in short mode (no E2E tests)
make test-short

# Run with race detector
make test-race

# Run with coverage
make test-coverage

# Run specific package tests
go test ./pkg/detection/...

# Run E2E tests only
make test-e2e
```

### Writing Tests

- Follow table-driven test patterns
- Aim for >80% code coverage
- Include edge cases and error scenarios
- Use meaningful test names

Example:
```go
func TestFeature(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "TEST",
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := MyFunction(tc.input)
            assert.Equal(t, tc.expected, result)
        })
    }
}
```

## Code Quality

### Formatting

Code must be formatted with `gofmt`:

```bash
# Check formatting
gofmt -l .

# Format all files
gofmt -w .
```

### Linting

We use `golangci-lint` for comprehensive linting:

```bash
make lint
```

### Security

Security scanning with `gosec`:

```bash
gosec ./...
```

### Vulnerability Scanning

Check for known vulnerabilities:

```bash
govulncheck ./...
```

## Troubleshooting

### Common Issues

#### Pre-commit hooks failing

1. Ensure all tools are installed:
```bash
make install-hooks
```

2. Run checks manually to see detailed output:
```bash
make pre-commit
```

#### Tests failing locally but not in CI

1. Ensure Go version matches CI:
```bash
go version  # Should be 1.23 or later
```

2. Run with CGO disabled (like CI):
```bash
CGO_ENABLED=0 go test ./...
```

#### Build issues

1. Clean and rebuild:
```bash
make clean
make build
```

2. Update dependencies:
```bash
go mod tidy
make deps
```

### Getting Help

1. Check existing issues: https://github.com/MacAttak/pi-scanner/issues
2. Ask in discussions: https://github.com/MacAttak/pi-scanner/discussions
3. Review CI logs for similar failures

## Best Practices

1. **Always run `make ci-local` before pushing** to catch issues early
2. **Keep commits focused** - one logical change per commit
3. **Write meaningful commit messages** following conventional commits
4. **Update tests** when changing functionality
5. **Document complex logic** with comments
6. **Follow Go idioms** and effective Go guidelines

## Quick Reference

```bash
# Development workflow
make setup          # Initial setup
make test           # Run tests
make build          # Build binary
make ci-local       # Simulate CI

# Git hooks
make pre-commit     # Run pre-commit checks
make pre-push       # Run pre-push checks
make install-hooks  # Install/update hooks

# Code quality
make fmt            # Format code
make lint           # Run linter
make vet            # Run go vet

# Testing
make test           # All tests
make test-short     # Quick tests
make test-coverage  # With coverage
make test-race      # With race detector
```