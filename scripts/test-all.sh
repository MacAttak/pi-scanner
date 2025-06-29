#!/bin/bash
# Unified test runner for PI Scanner
# Runs all test suites and generates comprehensive reports

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Timing
START_TIME=$(date +%s)

# Configuration
COVERAGE_DIR="${COVERAGE_DIR:-.coverage}"
REPORT_DIR="${REPORT_DIR:-.test-reports}"
VERBOSE="${VERBOSE:-false}"
FAIL_FAST="${FAIL_FAST:-false}"
RUN_E2E="${RUN_E2E:-true}"
RUN_BDD="${RUN_BDD:-true}"
RUN_BENCHMARKS="${RUN_BENCHMARKS:-false}"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --fail-fast|-f)
            FAIL_FAST=true
            shift
            ;;
        --no-e2e)
            RUN_E2E=false
            shift
            ;;
        --no-bdd)
            RUN_BDD=false
            shift
            ;;
        --benchmarks|-b)
            RUN_BENCHMARKS=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -v, --verbose      Show detailed test output"
            echo "  -f, --fail-fast    Stop on first test failure"
            echo "  --no-e2e           Skip E2E tests"
            echo "  --no-bdd           Skip BDD tests"
            echo "  -b, --benchmarks   Run performance benchmarks"
            echo "  -h, --help         Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  COVERAGE_DIR       Directory for coverage reports (default: .coverage)"
            echo "  REPORT_DIR         Directory for test reports (default: .test-reports)"
            echo "  GITHUB_TOKEN       Required for E2E tests that scan repositories"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Create directories
mkdir -p "$COVERAGE_DIR" "$REPORT_DIR"

# Helper functions
log_section() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

run_test_suite() {
    local suite_name=$1
    local command=$2
    local timeout=${3:-10m}

    echo -e "\n${GREEN}▶ Running $suite_name${NC}"

    local suite_start
    suite_start=$(date +%s)
    local exit_code=0

    if [ "$VERBOSE" = true ]; then
        timeout "$timeout" bash -c "$command" || exit_code=$?
    else
        timeout "$timeout" bash -c "$command" > "$REPORT_DIR/${suite_name// /_}.log" 2>&1 || exit_code=$?
    fi

    local suite_end
    suite_end=$(date +%s)
    local duration=$((suite_end - suite_start))

    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✅ $suite_name passed (${duration}s)${NC}"
        ((PASSED_TESTS++))
    elif [ $exit_code -eq 124 ]; then
        echo -e "${YELLOW}⏱️  $suite_name timed out after $timeout${NC}"
        ((FAILED_TESTS++))
        if [ "$FAIL_FAST" = true ]; then
            exit 1
        fi
    else
        echo -e "${RED}❌ $suite_name failed (exit code: $exit_code)${NC}"
        if [ "$VERBOSE" = false ]; then
            echo "   See $REPORT_DIR/${suite_name// /_}.log for details"
        fi
        ((FAILED_TESTS++))
        if [ "$FAIL_FAST" = true ]; then
            exit 1
        fi
    fi

    ((TOTAL_TESTS++))
}

# Start testing
echo -e "${GREEN}🧪 PI Scanner Comprehensive Test Suite${NC}"
echo -e "${GREEN}=====================================>${NC}"
echo ""
echo "Configuration:"
echo "  Verbose: $VERBOSE"
echo "  Fail Fast: $FAIL_FAST"
echo "  Run E2E: $RUN_E2E"
echo "  Run BDD: $RUN_BDD"
echo "  Run Benchmarks: $RUN_BENCHMARKS"

# 1. Unit Tests with Coverage
log_section "1. Unit Tests with Coverage"

cd "$PROJECT_ROOT"

# Run unit tests with coverage
run_test_suite "Unit Tests" \
    "go test -v -race -coverprofile=$COVERAGE_DIR/unit.out -covermode=atomic ./pkg/... ./cmd/... ./internal/..."

# Generate coverage report
if [ -f "$COVERAGE_DIR/unit.out" ]; then
    echo -e "\n${GREEN}📊 Coverage Report:${NC}"
    go tool cover -func="$COVERAGE_DIR/unit.out" | grep total | awk '{print "Total Coverage: " $3}'
    go tool cover -html="$COVERAGE_DIR/unit.out" -o "$COVERAGE_DIR/unit.html"
fi

# 2. Integration Tests
log_section "2. Integration Tests"

# Run integration tests separately (they might have different build tags)
run_test_suite "Integration Tests" \
    "go test -v -tags=integration -timeout=15m ./pkg/detection ./pkg/scanner ./pkg/validation" \
    "15m"

# 3. BDD Tests
if [ "$RUN_BDD" = true ]; then
    log_section "3. BDD Tests (Cucumber/Godog)"

    cd "$PROJECT_ROOT/test/bdd"

    # Install dependencies if needed
    if [ ! -f "go.sum" ]; then
        echo "Installing BDD test dependencies..."
        go mod tidy
    fi

    # Run BDD tests
    run_test_suite "BDD Features" \
        "go test -v" \
        "10m"

    cd "$PROJECT_ROOT"
fi

# 4. E2E Tests
if [ "$RUN_E2E" = true ]; then
    log_section "4. End-to-End Tests"

    if [ -z "${GITHUB_TOKEN:-}" ]; then
        echo -e "${YELLOW}⚠️  Warning: GITHUB_TOKEN not set. E2E tests may fail due to rate limits.${NC}"
    fi

    # Basic E2E tests
    run_test_suite "E2E Basic" \
        "$SCRIPT_DIR/run-e2e-tests.sh" \
        "20m"

    # Run comprehensive E2E if requested
    if [ "${RUN_COMPREHENSIVE_E2E:-false}" = true ]; then
        run_test_suite "E2E Comprehensive" \
            "$SCRIPT_DIR/run-e2e-tests.sh --all" \
            "45m"
    fi
fi

# 5. Specific Test Suites
log_section "5. Specialized Test Suites"

# PI Format Tests
run_test_suite "PI Format Tests" \
    "go test -v -run TestComprehensivePIFormats ./pkg/detection"

# False Positive Tests
run_test_suite "False Positive Tests" \
    "go test -v -run TestFalsePositives ./pkg/detection"

# Context-Aware Tests
run_test_suite "Context-Aware Tests" \
    "go test -v -run TestContextAware ./pkg/detection"

# 6. Performance Benchmarks
if [ "$RUN_BENCHMARKS" = true ]; then
    log_section "6. Performance Benchmarks"

    # Run benchmarks
    run_test_suite "Detection Benchmarks" \
        "go test -bench=. -benchmem -run=^$ ./pkg/detection -benchtime=10s" \
        "5m"

    # Save benchmark results
    if [ "$VERBOSE" = false ]; then
        cp "$REPORT_DIR/Detection_Benchmarks.log" "$REPORT_DIR/benchmarks.txt"
    fi
fi

# 7. Static Analysis
log_section "7. Static Analysis"

# Go vet
run_test_suite "Go Vet" \
    "go vet ./..."

# Golangci-lint (if available)
if command -v golangci-lint &> /dev/null; then
    run_test_suite "Golangci-lint" \
        "golangci-lint run --timeout=5m"
else
    echo -e "${YELLOW}⚠️  golangci-lint not found, skipping${NC}"
    ((SKIPPED_TESTS++))
fi

# 8. Security Checks
log_section "8. Security Checks"

# Check for hardcoded secrets
run_test_suite "Secret Detection" \
    "go test -v -run TestNoHardcodedSecrets ./pkg/testing"

# 9. Generate Reports
log_section "9. Test Reports"

END_TIME=$(date +%s)
TOTAL_DURATION=$((END_TIME - START_TIME))

# Create summary report
cat > "$REPORT_DIR/summary.txt" << EOF
PI Scanner Test Summary
======================
Date: $(date)
Duration: ${TOTAL_DURATION}s

Test Results:
- Total Tests: $TOTAL_TESTS
- Passed: $PASSED_TESTS
- Failed: $FAILED_TESTS
- Skipped: $SKIPPED_TESTS

Coverage:
$(if [ -f "$COVERAGE_DIR/unit.out" ]; then go tool cover -func="$COVERAGE_DIR/unit.out" | grep total; else echo "N/A"; fi)

Reports Generated:
- Coverage HTML: $COVERAGE_DIR/unit.html
- Test Logs: $REPORT_DIR/
- Summary: $REPORT_DIR/summary.txt
EOF

# Display summary
echo ""
cat "$REPORT_DIR/summary.txt"

# Quality Gates Check
log_section "10. Quality Gates"

# Check coverage threshold
if [ -f "$COVERAGE_DIR/unit.out" ]; then
    COVERAGE=$(go tool cover -func="$COVERAGE_DIR/unit.out" | grep total | awk '{print $3}' | sed 's/%//')
    THRESHOLD=70

    if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
        echo -e "${RED}❌ Coverage ${COVERAGE}% is below threshold of ${THRESHOLD}%${NC}"
        ((FAILED_TESTS++))
    else
        echo -e "${GREEN}✅ Coverage ${COVERAGE}% meets threshold of ${THRESHOLD}%${NC}"
    fi
fi

# Final Status
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed! 🎉${NC}"
    exit 0
else
    echo -e "${RED}❌ $FAILED_TESTS test suite(s) failed${NC}"
    echo -e "${YELLOW}Check the logs in $REPORT_DIR for details${NC}"
    exit 1
fi
