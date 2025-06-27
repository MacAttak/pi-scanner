# Contributing to GitHub PI Scanner

Thank you for your interest in contributing to the GitHub PI Scanner! This guide provides everything you need to know to contribute effectively to this security-focused project.

## Table of Contents

- [Development Setup](#development-setup)
- [Development Workflow](#development-workflow)
- [Testing Strategy](#testing-strategy)
- [Code Standards](#code-standards)
- [Security Guidelines](#security-guidelines)
- [Pull Request Process](#pull-request-process)

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Git 2.40 or higher
- Make
- Docker (optional, for containerized testing)

### Initial Setup

1. Clone the repository:
```bash
git clone https://github.com/MacAttak/pi-scanner.git
cd pi-scanner
```

2. Install dependencies and setup hooks:
```bash
make setup  # Installs all dependencies and git hooks
```

This command will:
- Install required Go tools
- Setup pre-commit and pre-push hooks
- Configure your development environment
- Install security scanning tools

### Development Tools

The following tools are automatically installed:
- `golangci-lint` - Comprehensive linting
- `gosec` - Security analysis
- `govulncheck` - Vulnerability scanning
- `gofumpt` - Enhanced formatting
- `gitleaks` - Secret detection

## Development Workflow

### Git Workflow

We use a feature branch workflow:

1. **Branch from `develop`** (or `main` for hotfixes)
2. **Use descriptive branch names**:
   - `feature/add-bank-validation`
   - `bugfix/fix-memory-leak`
   - `security/patch-cve-2024-xxx`
   - `refactor/improve-performance`

3. **Follow conventional commits**:
```
<type>(<scope>): <subject>

<body>

<footer>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `security`

### Test-Driven Development (TDD)

**Always follow TDD practices:**

1. Write tests first
2. Run tests (they should fail)
3. Implement minimal code to pass
4. Refactor while keeping tests green
5. Repeat

Example workflow:
```bash
# 1. Write test
vim pkg/validation/newfeature_test.go

# 2. Run test (should fail)
go test ./pkg/validation -run TestNewFeature

# 3. Implement feature
vim pkg/validation/newfeature.go

# 4. Run test (should pass)
go test ./pkg/validation -run TestNewFeature

# 5. Run all tests
make test
```

## Testing Strategy

### Testing Pyramid

We follow a testing pyramid approach:

- **60% Unit Tests** - Fast, isolated, comprehensive
- **30% Integration Tests** - Component interaction
- **10% E2E Tests** - Full system validation

### Shift-Left Testing

Our shift-left approach catches issues early:

#### 1. Pre-commit Hooks (5-10 seconds)

Automatically runs on `git commit`:
- Go formatting (`gofumpt`)
- Import organization
- Basic linting
- Secret scanning
- File size checks

#### 2. Pre-push Hooks (30-60 seconds)

Automatically runs on `git push`:
- Full test suite
- Advanced linting (`golangci-lint`)
- Security scanning (`gosec`)
- Vulnerability checks (`govulncheck`)

#### 3. Local CI Simulation (2-3 minutes)

Run the full CI pipeline locally:
```bash
make ci-local
```

### Writing Tests

#### Unit Tests

```go
func TestValidatePassport(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {"valid old format", "N1234567", true},
        {"valid new format", "PA1234567", true},
        {"invalid format", "123456", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := ValidatePassport(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### BDD Feature Tests

```gherkin
Feature: Secure Output
  As a security-conscious developer
  I want to ensure PI data is properly masked
  So that sensitive information is never exposed

  Scenario: Partial masking for verification
    Given a TFN "123456789"
    When I apply partial masking
    Then the output should be "123****89"
```

### Test Requirements

- **Minimum 80% code coverage**
- **All edge cases covered**
- **Performance benchmarks for critical paths**
- **Security test cases**
- **Concurrent operation testing**
- **Context cancellation testing**

## Code Standards

### Go Standards

Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and:

- Use meaningful variable names
- Keep functions small and focused
- Document all exported types and functions
- Handle all errors explicitly
- Use context for cancellation

### Project-Specific Standards

1. **Interface Design**
```go
// Small, focused interfaces
type Validator interface {
    Validate(value string) (bool, error)
    Type() string
}
```

2. **Error Handling**
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("validating passport %s: %w", value, err)
}
```

3. **Logging**
```go
// NEVER log sensitive data
logger.Info("Processing file",
    slog.String("file", filename),
    slog.Int("size", size))
// NOT: slog.String("content", content)
```

4. **Input Validation**
```go
// Always validate inputs
if len(value) == 0 {
    return ErrEmptyInput
}
```

## Security Guidelines

### Core Principles

1. **Never expose PI data** in logs, errors, or external communications
2. **Validate all inputs** before processing
3. **Use secure defaults** (masking enabled by default)
4. **Audit security operations** (access, modifications, exports)
5. **Minimize external dependencies** and audit them regularly

### Security Checklist

Before submitting code, ensure:

- [ ] No PI data in logs or error messages
- [ ] All inputs validated and sanitized
- [ ] Proper context cancellation handling
- [ ] Resource limits enforced
- [ ] Security tests included
- [ ] No hardcoded secrets or credentials
- [ ] Dependencies scanned for vulnerabilities

### Sensitive Operations

When working with PI data:

```go
// Good: Masked logging
logger.Info("Found PI",
    slog.String("type", "TFN"),
    slog.String("masked", MaskValue(value)))

// Bad: Exposed PI
logger.Info("Found TFN: " + value)
```

## Pull Request Process

### Before Creating a PR

1. **Ensure all tests pass**:
```bash
make test
```

2. **Check coverage**:
```bash
make coverage
```

3. **Run security scan**:
```bash
make security
```

4. **Update documentation** if needed

### PR Requirements

1. **Complete PR template** with all sections
2. **Pass all automated checks**:
   - Build successful
   - Tests passing (with race detection)
   - Coverage ≥ 80%
   - No linting issues
   - Security scan clean
   - Valid commit messages

3. **Obtain reviews**:
   - Minimum 2 approvals required
   - Security team review for `security` labeled PRs
   - Documentation review for API changes

### PR Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Security fix
- [ ] Performance improvement
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Security tests included
- [ ] Performance impact assessed

## Security Impact
- [ ] No PI exposure risk
- [ ] Input validation added
- [ ] Security scan clean

## Checklist
- [ ] Code follows style guide
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No commented-out code
```

### Emergency Procedures

For critical security fixes only:

```bash
# Bypass hooks (use sparingly!)
git commit --no-verify -m "security: emergency fix for CVE-XXXX"
git push --no-verify
```

Document why hooks were bypassed in the PR description.

## Getting Help

- **Documentation**: Check `/docs` directory
- **Issues**: Search existing issues before creating new ones
- **Discussions**: Use GitHub Discussions for questions
- **Security**: Report security issues privately via security@example.com

## Recognition

Contributors are recognized in:
- Release notes
- CONTRIBUTORS.md file
- Security hall of fame (for security fixes)

Thank you for helping make GitHub PI Scanner more secure!
