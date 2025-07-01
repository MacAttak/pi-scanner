# Scripts Directory

This directory contains various utility scripts for development, testing, and release management of the GitHub PI Scanner.

## Development Scripts

### setup.sh
Initial setup script for configuring the development environment.
- Installs required dependencies
- Sets up Git hooks
- Configures development tools

Usage: `./scripts/setup.sh`

### install-hooks.sh
Installs Git hooks for the project.
- Pre-commit hooks for code formatting and linting
- Commit message validation

Usage: `./scripts/install-hooks.sh`

### update-module-refs.sh
Updates module references across the codebase when the module path changes.

Usage: `./scripts/update-module-refs.sh <old-module-path> <new-module-path>`

## Testing Scripts

### test-all.sh
Runs all tests including unit tests, integration tests, and benchmarks.

Usage: `./scripts/test-all.sh`

### run-e2e-tests.sh
Runs end-to-end tests against real repositories.

Usage: `./scripts/run-e2e-tests.sh`

### coverage-report.sh
Generates a comprehensive test coverage report.

Usage: `./scripts/coverage-report.sh`

### check-quality-gates.sh
Checks if the code meets quality gates for:
- Test coverage thresholds
- Linting standards
- Security requirements

Usage: `./scripts/check-quality-gates.sh`

## Benchmark Scripts

### benchmark-track.sh
Runs performance benchmarks and tracks results over time.

Usage: `./scripts/benchmark-track.sh`

### benchmark-track-simple.sh
Simplified version of benchmark tracking for quick performance checks.

Usage: `./scripts/benchmark-track-simple.sh`

## CI/CD Scripts

### ci-local.sh
Runs the full CI pipeline locally for testing before pushing.

Usage: `./scripts/ci-local.sh`

## Security Scripts

### security-audit.sh
Performs a comprehensive security audit of the codebase and dependencies.

Usage: `./scripts/security-audit.sh`

### security-audit-summary.sh
Generates a summary report of security findings.

Usage: `./scripts/security-audit-summary.sh`

## Release Scripts

### build-release.sh
Builds release binaries for multiple platforms.

Usage: `./scripts/build-release.sh <version>`

### publish-release.sh
Publishes a new release to GitHub and other distribution channels.

Usage: `./scripts/publish-release.sh <version>`

## Script Conventions

All scripts follow these conventions:

1. **Exit on Error**: Scripts use `set -e` to exit on any command failure
2. **Undefined Variables**: Scripts use `set -u` to error on undefined variables
3. **Color Output**: Scripts use color codes for better readability
4. **Help Text**: Scripts accept `-h` or `--help` for usage information
5. **Idempotency**: Scripts can be run multiple times safely

## Environment Variables

Some scripts use these environment variables:

- `CI`: Set to "true" in CI environments
- `GITHUB_TOKEN`: GitHub authentication token for API access
- `PI_SCANNER_WORKERS`: Number of parallel workers for scanning
- `NO_COLOR`: Disable colored output when set

## Contributing

When adding new scripts:

1. Follow the naming convention: `action-target.sh`
2. Add a header comment explaining the script's purpose
3. Include help text accessible via `-h` flag
4. Update this README with documentation
5. Make the script executable: `chmod +x scripts/new-script.sh`
