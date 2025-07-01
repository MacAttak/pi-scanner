# Contributing to GitHub PI Scanner

Thank you for your interest in contributing to the GitHub PI Scanner! This guide will help you get started.

## Quick Links

- [Development Environment Setup](DEVELOPMENT.md) - Docker-based development environment
- [Developer Guide](docs/DEVELOPER_GUIDE.md) - Architecture and API reference
- [Design Overview](docs/DESIGN_OVERVIEW.md) - System design and architecture

## Getting Started

### Prerequisites

- Go 1.21+ (for local development)
- Docker and Docker Compose (for containerized development)
- GitHub account
- Git

### Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR-USERNAME/pi-scanner.git
   cd pi-scanner
   ```
3. Set up pre-commit hooks:
   ```bash
   ./scripts/install-hooks.sh
   ```

### Development Workflow

We use a containerized development environment to ensure consistency:

```bash
# Start development environment
make dev

# Run tests
make test

# Run linting
make lint

# Build the application
make build
```

## Making Contributions

### Code Standards

- Follow Go best practices and idioms
- Use `gofmt` for formatting (enforced by pre-commit hooks)
- Add tests for new functionality
- Maintain or improve code coverage
- Document exported functions and types

### Testing Strategy

- Unit tests: Cover individual functions and methods
- Integration tests: Test component interactions
- E2E tests: Validate complete workflows
- Minimum 80% code coverage for new code

### Commit Messages

Follow conventional commit format:
```
type(scope): description

[optional body]

[optional footer]
```

Types: feat, fix, docs, style, refactor, test, chore

### Pull Request Process

1. Create a feature branch from `main`
2. Make your changes following our standards
3. Ensure all tests pass locally
4. Push your branch and create a PR
5. Address reviewer feedback
6. Squash commits if requested

### Security Guidelines

- Never commit sensitive data (tokens, credentials, PI)
- Review code for security vulnerabilities
- Follow secure coding practices
- Report security issues privately via GitHub Security tab

## Architecture Overview

The PI Scanner uses a two-phase architecture:

1. **Phase 1**: Pattern-based detection using regex
2. **Phase 2**: Optional LLM validation for accuracy

Key packages:
- `pkg/detection`: Core detection logic
- `pkg/processing`: File processing pipeline
- `pkg/llm`: LLM integration
- `pkg/ast`: Code structure analysis

## Getting Help

- **Issues**: Search existing issues before creating new ones
- **Discussions**: Use GitHub Discussions for questions
- **Documentation**: Check docs/ directory for detailed guides

## Code of Conduct

We are committed to providing a welcoming and inclusive environment. Please read and follow our Code of Conduct.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
