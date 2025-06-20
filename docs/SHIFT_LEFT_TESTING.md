# Shift-Left Testing Strategy

This document describes the shift-left testing strategy implemented for the GitHub PI Scanner project to catch issues early in the development process and prevent CI failures.

## Overview

The shift-left approach moves testing earlier in the development workflow, catching issues before they reach the CI pipeline. This reduces feedback loops and improves developer productivity.

## Implementation Layers

### 1. Pre-commit Hooks (Fastest Checks)

Pre-commit hooks run automatically on `git commit` and perform quick validations:

- **Go formatting** (`go fmt`)
- **Import organization** (`goimports`)
- **Basic linting** (`go vet`)
- **Module tidiness** (`go mod tidy`)
- **File size checks**
- **Secret scanning** (gitleaks)
- **Trailing whitespace**

**Runtime**: ~5-10 seconds

### 2. Pre-push Hooks (Comprehensive Checks)

Pre-push hooks run on `git push` and perform thorough validation:

- All pre-commit checks
- **Full test suite** (`go test ./...`)
- **Advanced linting** (golangci-lint)
- **Security scanning** (gosec)
- **Vulnerability checks** (govulncheck)

**Runtime**: ~30-60 seconds

### 3. Local CI Simulation

Full CI pipeline simulation that mirrors GitHub Actions:

```bash
make ci-local
```

Includes:
- Environment verification
- Code quality checks
- Build verification
- Test execution with coverage
- Security scans
- Cross-platform build tests

**Runtime**: ~2-3 minutes

## Setup Instructions

### Initial Installation

```bash
# Clone and setup
git clone https://github.com/MacAttak/pi-scanner.git
cd pi-scanner
make setup  # Installs hooks and dependencies
```

### Manual Hook Installation

```bash
# Install pre-commit framework
pip install pre-commit

# Install hooks
pre-commit install
git config core.hooksPath .githooks

# Or use the setup script
./scripts/install-hooks.sh
```

### Installing Optional Tools

```bash
# Go tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

# Other tools
brew install gitleaks  # macOS
brew install pre-commit # macOS
```

## Usage

### Running Checks Manually

```bash
# Pre-commit checks only
make pre-commit
# or
pre-commit run --all-files

# Pre-push checks
make pre-push

# Full CI simulation
make ci-local
```

### Bypassing Hooks (Emergency Only)

```bash
# Skip pre-commit
git commit --no-verify -m "Emergency fix"

# Skip pre-push
git push --no-verify
```

**Note**: Only bypass hooks in emergencies. Always run `make ci-local` before creating a PR.

## Configuration

### Pre-commit Configuration

Edit `.pre-commit-config.yaml` to customize checks:

```yaml
repos:
  - repo: local
    hooks:
      - id: go-fmt
        name: Check Go formatting
        entry: bash -c '...'
        language: system
        types: [go]
```

### Pre-push Configuration

Edit `.githooks/pre-push` to customize checks.

## Troubleshooting

### Common Issues

#### 1. Pre-commit failing on formatting

```bash
# Auto-fix formatting
gofmt -w .
go mod tidy

# Then retry commit
git add -u
git commit
```

#### 2. Tests failing in pre-push

```bash
# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test -v ./pkg/detection/...
```

#### 3. Tool not found errors

```bash
# Check which tools are missing
make install-hooks

# Install missing tools manually
go install <tool>@latest
```

#### 4. Hooks not running

```bash
# Verify hook installation
ls -la .git/hooks/
git config core.hooksPath

# Reinstall
make setup
```

### Performance Issues

If hooks are too slow:

1. Use `make pre-commit` for quick checks during development
2. Run `make ci-local` before pushing
3. Consider upgrading hardware or using cloud development

## CI/CD Integration

The shift-left strategy complements the GitHub Actions CI pipeline:

1. **Local checks** catch 90% of issues
2. **CI pipeline** provides final validation
3. **Consistent tooling** between local and CI

## Benefits

1. **Faster feedback** - Issues caught in seconds, not minutes
2. **Reduced CI failures** - Most issues fixed before push
3. **Lower costs** - Fewer CI runs needed
4. **Better DX** - Developers stay in flow

## Metrics

Track effectiveness with:

- CI failure rate (should decrease)
- Time to fix issues (should decrease)
- Developer satisfaction (should increase)

## Future Improvements

1. **IDE integration** - Real-time linting
2. **Incremental testing** - Only test changed files
3. **Parallel execution** - Speed up checks
4. **Custom rules** - Project-specific validations

## Summary

The shift-left testing strategy provides multiple safety nets:

```
Edit → Save → Commit → Push → PR → Merge
      ↓       ↓         ↓      ↓     ↓
     IDE   Pre-commit Pre-push CI  Deploy
```

Each stage catches different issues, ensuring high code quality while maintaining developer productivity.