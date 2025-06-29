# End-to-End Tests

This directory contains end-to-end tests for the PI Scanner project.

## Separate Go Module

The e2e tests use a separate `go.mod` file for the following reasons:

1. **Dependency Isolation**: E2E tests may require additional dependencies (like test utilities or external service clients) that we don't want to include in the main module.

2. **Build Performance**: Keeping e2e tests separate prevents them from being built during normal development builds, improving build times.

3. **CI/CD Flexibility**: Allows running e2e tests in a different environment or stage of the CI/CD pipeline without affecting the main build.

4. **Version Independence**: E2E tests can use different versions of dependencies if needed for testing compatibility.

## Running E2E Tests

From this directory:

```bash
# Run all e2e tests
go test -v ./...

# Run with specific timeout
go test -v -timeout 30m ./...

# Run specific test
go test -v -run TestPublicRepoScan ./...
```

## Test Structure

- `local_scan_test.go` - Tests scanning local test data
- `public_repo_test.go` - Tests scanning public repositories
- `performance_test.go` - Performance and load tests

## Adding New Tests

When adding new e2e tests:

1. Place them in this directory
2. Use the `e2e` package name
3. Ensure tests are idempotent and can run in parallel
4. Use appropriate timeouts for long-running tests
5. Clean up any resources created during tests
