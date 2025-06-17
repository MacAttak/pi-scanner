# GitHub PI Scanner - Implementation Roadmap

## Overview

This roadmap outlines the implementation plan for the GitHub PI Scanner, organized into 4 weekly sprints aligned with the solution design. The plan emphasizes development best practices including BDD/TDD, continuous testing, structured commits, and iterative delivery.

## Development Best Practices

### Core Practices
1. **Test-Driven Development (TDD)**: Write tests first, then implementation
2. **Behavior-Driven Development (BDD)**: Define behaviors in Gherkin format
3. **Continuous Integration**: All tests must pass before merging
4. **Structured Commits**: Follow conventional commit format
5. **Code Review**: All code requires peer review
6. **Documentation as Code**: Update docs with each feature
7. **Feature Flags**: Develop behind flags for safe rollout
8. **Observability First**: Add logging/metrics from the start
9. **Security by Design**: Continuous static analysis and vulnerability scanning
10. **Dependency Hygiene**: Regular updates and vulnerability checks

### Commit Convention
```
type(scope): subject

body

footer
```
Types: feat, fix, docs, style, refactor, test, chore

Example:
```
feat(detection): add TFN validation with modulus 11 algorithm

- Implement checksum validation for Australian Tax File Numbers
- Add unit tests covering valid and invalid TFN formats
- Update detection pipeline to use new validator

Closes #12
```

### Daily Practices
- Morning: Review yesterday's work, plan today's tasks
- Write failing tests before any new feature
- Commit at least every 2 hours with meaningful messages
- Run full test suite before pushing
- Update documentation with code changes

## Week 1: Foundation & Core Pipeline

### Day 1-2: Project Setup & CLI Framework

#### BDD Scenarios
```gherkin
Feature: CLI initialization and basic commands
  Scenario: User runs scanner without arguments
    Given the scanner is installed
    When I run "pi-scanner"
    Then I should see help text with available commands
    
  Scenario: User scans a repository
    Given a valid GitHub repository URL
    When I run "pi-scanner scan --repo <url>"
    Then the scan should initiate successfully
```

#### Tasks with Testing Requirements
- [ ] Initialize Go module: `go mod init github.com/pi-scanner/pi-scanner`
- [ ] Set up project structure with test directories:
  ```
  /cmd/pi-scanner/        - CLI entry point
  /cmd/pi-scanner/*_test.go
  /pkg/scanner/           - Core scanning logic
  /pkg/scanner/*_test.go
  /pkg/detection/         - Detection engines
  /pkg/detection/*_test.go
  /features/              - BDD feature files
  /test/                  - Integration tests
  /test/fixtures/         - Test data
  ```
- [ ] Set up CI/CD pipeline (GitHub Actions):
  ```yaml
  # .github/workflows/ci.yml
  on: [push, pull_request]
  jobs:
    test:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v3
        - uses: actions/setup-go@v4
        - run: go test -v -race -coverprofile=coverage.out ./...
        - run: go vet ./...
        - run: golangci-lint run
    
    security:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v3
        - uses: securego/gosec@master
          with:
            args: ./...
        - name: Run Trivy vulnerability scanner
          uses: aquasecurity/trivy-action@master
          with:
            scan-type: 'fs'
            scan-ref: '.'
        - name: Upload SARIF results
          uses: github/codeql-action/upload-sarif@v2
          with:
            sarif_file: 'trivy-results.sarif'
  ```
- [ ] Implement basic CLI with Cobra (TDD approach):
  1. Write tests for CLI commands
  2. Implement commands to pass tests
  3. Add integration tests
- [ ] Configure pre-commit hooks:
  ```yaml
  # .pre-commit-config.yaml
  repos:
    - repo: local
      hooks:
        - id: go-test
          name: go test
          entry: go test ./...
          language: system
          pass_filenames: false
        - id: go-fmt
          name: go fmt
          entry: go fmt ./...
          language: system
        - id: go-sec
          name: gosec
          entry: gosec -fmt json -out gosec-report.json ./...
          language: system
          pass_filenames: false
        - id: go-vulncheck
          name: govulncheck
          entry: govulncheck ./...
          language: system
          pass_filenames: false
  ```

### Day 3-4: Pattern Detection Engine

#### Test-First Implementation
1. **Write validator tests first**:
   ```go
   func TestTFNPattern(t *testing.T) {
       validTFNs := []string{"123456789", "123 456 789", "123-456-789"}
       invalidTFNs := []string{"12345678", "1234567890", "abc456789"}
       // Test implementation
   }
   ```

2. **Create BDD features**:
   ```gherkin
   Feature: Australian PI Detection
     Scenario: Detect valid TFN
       Given a file containing "TFN: 123456789"
       When I scan the file
       Then a TFN should be detected
   ```

3. **Implement to pass tests**

#### Tasks
- [ ] Write comprehensive test suite for pattern detection
- [ ] Integrate Gitleaks as a library with tests
- [ ] Create custom TOML rules with validation tests
- [ ] Implement file discovery with edge case tests
- [ ] Add worker pool with concurrency tests
- [ ] Benchmark pattern matching performance

### Day 5: Repository Management

#### Tasks with Quality Gates
- [ ] Write tests for GitHub repository operations
- [ ] Implement repository cloning with error handling tests
- [ ] Add cleanup tests (including interrupt scenarios)
- [ ] Create integration tests with mock repositories
- [ ] Document repository management API

## Week 2: ML Integration & Advanced Detection

### Day 1-2: ML Model Integration

#### BDD Approach
```gherkin
Feature: ML-based PI validation
  Scenario: Validate detected pattern with ML
    Given a text "My TFN is 123456789"
    When the ML validator processes it
    Then confidence should be above 0.85
    And PI type should be "TAX_ID"
```

#### Testing Strategy
- [ ] Unit tests for tokenizer
- [ ] Integration tests for ONNX runtime
- [ ] Performance benchmarks for inference
- [ ] Fallback tests when ML unavailable

### Day 3: Australian PI Validators

#### TDD Implementation Order
1. Write failing tests for each validator
2. Implement validators incrementally
3. Add property-based tests for edge cases
4. Create benchmark tests

#### Validator Test Coverage
```go
// Each validator needs:
func TestValidateTFN_ValidInput(t *testing.T)
func TestValidateTFN_InvalidInput(t *testing.T)
func TestValidateTFN_EdgeCases(t *testing.T)
func BenchmarkValidateTFN(b *testing.B)
```

### Day 4-5: Context Analysis

#### Test Scenarios
- [ ] Proximity detection with various layouts
- [ ] Test data detection accuracy
- [ ] Confidence scoring calibration
- [ ] Integration tests for full pipeline

## Week 3: Risk Scoring & Reporting

### Day 1-2: Risk Analysis Engine

#### BDD Features
```gherkin
Feature: Risk scoring
  Scenario: Critical risk for multiple high-value PI
    Given findings contain Name, TFN, and Bank Account
    And findings are within 5 lines
    When risk scoring runs
    Then risk level should be "CRITICAL"
```

#### Implementation with Tests
- [ ] Risk matrix unit tests (all combinations)
- [ ] Co-occurrence detection tests
- [ ] Environment detection tests
- [ ] Integration tests with real findings

### Day 3-4: Report Generation

#### Quality Requirements
- [ ] Visual regression tests for HTML reports
- [ ] CSV format validation tests
- [ ] SARIF compliance tests
- [ ] Performance tests for large reports
- [ ] Accessibility tests for HTML output

### Day 5: Configuration System

#### Test Coverage
- [ ] Config parsing with invalid YAML
- [ ] Default config validation
- [ ] Override mechanism tests
- [ ] Integration with CLI flags

## Week 4: Polish & Production Readiness

### Day 1-2: Performance Optimization

#### Performance Testing Framework
```go
func BenchmarkScanSmallRepo(b *testing.B)
func BenchmarkScanLargeRepo(b *testing.B)
func BenchmarkMemoryUsage(b *testing.B)
```

#### Optimization Process
1. Profile current performance
2. Identify bottlenecks with pprof
3. Implement optimizations
4. Verify improvements with benchmarks
5. Ensure no regression in accuracy

### Day 3: Comprehensive Testing

#### Test Pyramid
```
         /\
        /  \    E2E Tests (10%)
       /----\   
      /      \  Integration Tests (30%)
     /--------\
    /          \ Unit Tests (60%)
   /____________\
```

#### Coverage Requirements
- [ ] Unit test coverage > 80%
- [ ] Integration test coverage > 70%
- [ ] E2E test coverage for critical paths
- [ ] Mutation testing to verify test quality

### Day 4: Documentation & Examples

#### Documentation Standards
- [ ] README with quick start guide
- [ ] API documentation (godoc)
- [ ] Architecture decision records (ADRs)
- [ ] Troubleshooting guide with common issues
- [ ] Performance tuning guide

### Day 5: Security Review & Packaging

#### Security Checklist
- [ ] Static analysis with gosec (automated in CI)
- [ ] Dependency vulnerability scan with Trivy (automated in CI)
- [ ] SAST scan with CodeQL (automated in CI)
- [ ] Fuzzing for parser components
- [ ] Security-focused code review
- [ ] Verify no secrets in code with gitleaks
- [ ] Check for common Go security anti-patterns
- [ ] Validate all user inputs are sanitized
- [ ] Ensure secure file operations (path traversal prevention)
- [ ] Memory safety verification for ML operations

## Continuous Practices Throughout Development

### Daily Routines
1. **Morning Standup** (even if solo):
   - What was completed yesterday?
   - What will be done today?
   - Any blockers?

2. **TDD Cycle** (Red-Green-Refactor):
   - Write failing test
   - Write minimal code to pass
   - Refactor with confidence

3. **Commit Practices**:
   - Commit after each test passes
   - Use conventional commit format
   - Include issue references

### Weekly Routines
1. **Code Review**:
   - Review all code before merge
   - Check test coverage
   - Verify documentation updates
   - Security-focused review for sensitive operations

2. **Performance Review**:
   - Run benchmarks
   - Check for regressions
   - Update performance metrics

3. **Security Review**:
   - Review gosec findings
   - Check dependency vulnerabilities
   - Update security allowlist if needed
   - Review any new external dependencies

4. **Retrospective**:
   - What went well?
   - What could improve?
   - Action items for next week

## Quality Gates

### Definition of Done
- [ ] Feature has unit tests (>80% coverage)
- [ ] Feature has integration tests
- [ ] All tests pass
- [ ] Code reviewed and approved
- [ ] Documentation updated
- [ ] No security vulnerabilities
- [ ] Performance benchmarks pass
- [ ] Conventional commit messages used

### Merge Requirements
```yaml
# .github/branch-protection.yml
protection_rules:
  main:
    required_reviews: 1
    dismiss_stale_reviews: true
    require_code_owner_reviews: true
    required_status_checks:
      - test
      - security
      - gosec
      - trivy
      - codeql
      - coverage/coveralls
    enforce_admins: true
    restrictions:
      dismiss_stale_reviews: true
```

## Metrics & Monitoring

### Development Metrics
- Test coverage percentage
- Build success rate
- Average PR review time
- Defect escape rate
- Performance benchmark trends
- Security vulnerability count (must be 0)
- Static analysis findings trend
- Dependency freshness score

### Runtime Metrics (built into app)
- Scan duration by repo size
- Memory usage patterns
- Detection accuracy rates
- Error rates and types

## Risk Mitigation

### Technical Risks
1. **ML Model Performance**: 
   - Mitigation: Continuous benchmarking
   - Fallback: Regex-only mode

2. **Memory Usage**:
   - Mitigation: Memory profiling from day 1
   - Fallback: Streaming mode for large files

3. **Accuracy Issues**:
   - Mitigation: Comprehensive test dataset
   - Fallback: Conservative detection mode

## Success Criteria

- [ ] 100% of features have corresponding tests
- [ ] All commits follow conventional format
- [ ] CI/CD pipeline catches issues before merge
- [ ] Performance benchmarks tracked over time
- [ ] Documentation stays in sync with code
- [ ] Security scans pass on every build
- [ ] >95% detection accuracy maintained
- [ ] <5% false positive rate achieved

## Next Steps

1. Set up development environment with tools
2. Configure CI/CD pipeline
3. Create initial test fixtures
4. Begin TDD implementation
5. Daily commits with progress updates