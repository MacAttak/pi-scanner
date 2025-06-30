# E2E Test Guide for PI Scanner

## Overview

This guide covers the end-to-end (E2E) test suite for PI Scanner, which tests the complete CLI functionality against real repositories.

## Test Categories

### 1. Basic CLI Tests (`TestCLIBasicScan`)
- Tests core scanning functionality
- Verifies output formats and masking levels
- Validates command-line arguments
- Quick to run (~2 minutes)

### 2. LLM Integration Tests (`TestCLILLMIntegration`)
- Tests LLM service connectivity
- Validates LLM-enhanced detection
- Requires LM Studio running locally
- Runtime: ~3 minutes

### 3. Australian Repository Tests (`TestCLIAustralianRepositories`)
- Scans real Australian repositories
- Tests against healthcare, financial, and government repos
- Requires `RUN_AUSTRALIAN_REPOS=true`
- Runtime: ~10-15 minutes

### 4. Performance Tests (`TestCLIPerformance`)
- Tests scanning large repositories
- Measures files/second and MB/second throughput
- Requires `RUN_PERFORMANCE_TESTS=true`
- Runtime: ~10+ minutes

### 5. Error Handling Tests (`TestCLIErrorHandling`)
- Tests various error conditions
- Validates error messages and recovery
- Quick to run (~1 minute)

## Running Tests

### Quick Test (Basic + Error Handling)
```bash
cd test/e2e
go test -v ./...
```

### Using the Test Script
```bash
# Basic tests only
./scripts/run-e2e-tests.sh

# Include Australian repos
./scripts/run-e2e-tests.sh --australian

# All tests including performance
./scripts/run-e2e-tests.sh --all

# Skip LLM tests
./scripts/run-e2e-tests.sh --no-llm
```

### Individual Test Suites
```bash
# Just basic CLI tests
go test -v -run TestCLIBasicScan ./...

# Just LLM tests
go test -v -run TestCLILLMIntegration ./...

# Australian repos (set env var first)
RUN_AUSTRALIAN_REPOS=true go test -v -run TestCLIAustralianRepositories ./...

# Performance tests
RUN_PERFORMANCE_TESTS=true go test -v -run TestCLIPerformance ./...
```

## Prerequisites

### 1. GitHub Token
Set `GITHUB_TOKEN` to avoid rate limits:
```bash
export GITHUB_TOKEN="your-token-here"
```

### 2. LM Studio (for LLM tests)
1. Install LM Studio from https://lmstudio.ai/
2. Download the model: `qwen2.5-coder-7b-instruct`
3. Start the local server on port 1234

### 3. Build the Binary
The tests automatically build the binary, but you can pre-build:
```bash
go build -o pi-scanner ./cmd/pi-scanner
```

## Test Repositories

### Test Data Repository
- URL: `https://github.com/MacAttak/pi-scanner-test-data`
- Contains known test PI patterns
- Used for validation testing

### Australian Repositories (samples)
- Healthcare: `https://github.com/hl7au/au-fhir-test-data`
- Government: `https://github.com/govau/design-system-components`
- Financial: `https://github.com/CommBank/CommBank-API-Samples`

## Debugging Tests

### Verbose Output
```bash
# See detailed command output
go test -v -run TestName ./...

# Or with the script
./scripts/run-e2e-tests.sh --verbose
```

### Check Scan Results
Test results are saved in temporary directories. To keep them:
```bash
# Modify the test to use a fixed directory
tempDir := "/tmp/pi-scanner-test"
os.MkdirAll(tempDir, 0755)
```

### LLM Service Issues
Check LLM availability:
```bash
./pi-scanner llm
```

## Performance Benchmarks

Expected performance on modern hardware:
- Files/second: >50 for small files
- MB/second: >5 for text processing
- Memory usage: <500MB for typical repos

## CI/CD Integration

### GitHub Actions
```yaml
- name: Run E2E Tests
  run: |
    ./scripts/run-e2e-tests.sh
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Docker
```bash
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v $(pwd):/app \
  pi-scanner-test \
  ./scripts/run-e2e-tests.sh
```

## Troubleshooting

### Common Issues

1. **Rate Limits**
   - Set `GITHUB_TOKEN`
   - Use smaller repos for testing

2. **Timeouts**
   - Increase timeout: `go test -timeout 30m`
   - Check network connectivity

3. **LLM Not Available**
   - Start LM Studio server
   - Check port 1234 is free
   - Verify model is loaded

4. **Build Failures**
   - Update Go modules: `go mod tidy`
   - Check Go version: requires 1.21+

## Adding New Tests

1. Add test to `cli_test.go`
2. Follow naming convention: `TestCLI<Feature>`
3. Use table-driven tests for multiple scenarios
4. Always clean up resources
5. Document any special requirements
