#!/bin/bash
# Check Quality Gates for GitHub PI Scanner
# This script validates code against defined quality standards

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
QUALITY_CONFIG="${QUALITY_CONFIG:-quality-gates.yaml}"
COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-70}"
BENCHMARK_TIMEOUT="${BENCHMARK_TIMEOUT:-30s}"
OUTPUT_DIR="${OUTPUT_DIR:-.quality-reports}"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Summary variables
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

# Helper functions
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((PASSED_CHECKS++))
    ((TOTAL_CHECKS++))
}

warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++))
    ((TOTAL_CHECKS++))
}

error() {
    echo -e "${RED}[FAIL]${NC} $1"
    ((FAILED_CHECKS++))
    ((TOTAL_CHECKS++))
}

header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Check if running in CI
is_ci() {
    [ "${CI:-false}" = "true" ] || [ -n "${GITHUB_ACTIONS:-}" ]
}

# 1. Code Formatting
header "Code Formatting"
if gofmt -l . | grep -q .; then
    error "Code formatting issues found. Run: gofmt -w ."
    gofmt -l .
else
    success "All Go files are properly formatted"
fi

# 2. Imports
header "Import Organization"
if command -v goimports >/dev/null 2>&1; then
    if goimports -l . | grep -q .; then
        error "Import formatting issues found. Run: goimports -w ."
        goimports -l .
    else
        success "All imports are properly organized"
    fi
else
    warning "goimports not installed, skipping import checks"
fi

# 3. Static Analysis
header "Static Analysis"
if go vet ./... 2>&1 | grep -q .; then
    error "go vet found issues"
    go vet ./... 2>&1
else
    success "go vet passed"
fi

# 4. Linting
header "Linting"
if command -v golangci-lint >/dev/null 2>&1; then
    if golangci-lint run --timeout=5m > "$OUTPUT_DIR/lint-report.txt" 2>&1; then
        success "golangci-lint passed"
    else
        error "golangci-lint found issues (see $OUTPUT_DIR/lint-report.txt)"
        head -20 "$OUTPUT_DIR/lint-report.txt"
    fi
else
    warning "golangci-lint not installed, skipping advanced linting"
fi

# 5. Test Coverage
header "Test Coverage"
info "Running tests with coverage analysis..."

# Run tests with coverage
if go test -coverprofile="$OUTPUT_DIR/coverage.out" -covermode=atomic ./... > "$OUTPUT_DIR/test-report.txt" 2>&1; then
    # Generate coverage report
    go tool cover -html="$OUTPUT_DIR/coverage.out" -o "$OUTPUT_DIR/coverage.html"

    # Check overall coverage
    COVERAGE=$(go tool cover -func="$OUTPUT_DIR/coverage.out" | grep total | awk '{print $3}' | sed 's/%//')
    COVERAGE_INT=${COVERAGE%.*}

    if [ "$COVERAGE_INT" -ge "$COVERAGE_THRESHOLD" ]; then
        success "Test coverage ${COVERAGE}% exceeds threshold of ${COVERAGE_THRESHOLD}%"
    else
        error "Test coverage ${COVERAGE}% is below threshold of ${COVERAGE_THRESHOLD}%"
    fi

    # Check package-specific coverage
    info "Package coverage breakdown:"
    go tool cover -func="$OUTPUT_DIR/coverage.out" | grep -E "^github.com/MacAttak/pi-scanner/(pkg|cmd)" | while read -r line; do
        pkg=$(echo "$line" | awk '{print $1}')
        cov=$(echo "$line" | awk '{print $3}' | sed 's/%//')
        cov_int=${cov%.*}

        case "$pkg" in
            *pkg/detection*)
                [ "$cov_int" -ge 80 ] && echo -e "  ${GREEN}✓${NC} $pkg: $cov%" || echo -e "  ${RED}✗${NC} $pkg: $cov% (need 80%)"
                ;;
            *pkg/validation*)
                [ "$cov_int" -ge 90 ] && echo -e "  ${GREEN}✓${NC} $pkg: $cov%" || echo -e "  ${RED}✗${NC} $pkg: $cov% (need 90%)"
                ;;
            *pkg/risk*)
                [ "$cov_int" -ge 75 ] && echo -e "  ${GREEN}✓${NC} $pkg: $cov%" || echo -e "  ${RED}✗${NC} $pkg: $cov% (need 75%)"
                ;;
            *)
                [ "$cov_int" -ge 50 ] && echo -e "  ${GREEN}✓${NC} $pkg: $cov%" || echo -e "  ${YELLOW}⚠${NC} $pkg: $cov%"
                ;;
        esac
    done
else
    error "Tests failed (see $OUTPUT_DIR/test-report.txt)"
    tail -20 "$OUTPUT_DIR/test-report.txt"
fi

# 6. Race Detection
header "Race Detection"
info "Running tests with race detector..."
if go test -race -short ./... > "$OUTPUT_DIR/race-report.txt" 2>&1; then
    success "No race conditions detected"
else
    error "Race conditions detected (see $OUTPUT_DIR/race-report.txt)"
    grep -A 5 "WARNING: DATA RACE" "$OUTPUT_DIR/race-report.txt" | head -20
fi

# 7. Performance Benchmarks
header "Performance Benchmarks"
if ! is_ci || [ "${RUN_BENCHMARKS:-false}" = "true" ]; then
    info "Running performance benchmarks..."

    # Run benchmarks for critical packages
    for pkg in "pkg/detection" "pkg/validation" "pkg/detection/patterns"; do
        if go test -bench=. -benchmem -benchtime="$BENCHMARK_TIMEOUT" -run=^$ "./$pkg" > "$OUTPUT_DIR/bench-${pkg//\//-}.txt" 2>&1; then
            success "Benchmarks passed for $pkg"
            # Extract key metrics
            grep -E "Benchmark|ns/op|allocs/op" "$OUTPUT_DIR/bench-${pkg//\//-}.txt" | head -10
        else
            warning "Benchmarks failed for $pkg"
        fi
    done
else
    info "Skipping benchmarks in CI (set RUN_BENCHMARKS=true to enable)"
fi

# 8. Security Scanning
header "Security Scanning"
if command -v gosec >/dev/null 2>&1; then
    if gosec -fmt json -out "$OUTPUT_DIR/security-report.json" -severity medium ./... 2>/dev/null; then
        success "Security scan passed"
    else
        warning "Security issues found (see $OUTPUT_DIR/security-report.json)"
        # Parse and display summary
        if command -v jq >/dev/null 2>&1; then
            jq -r '.Issues[] | "\(.severity): \(.file):\(.line) - \(.rule_id)"' "$OUTPUT_DIR/security-report.json" | head -10
        fi
    fi
else
    warning "gosec not installed, skipping security scan"
fi

# 9. Vulnerability Check
header "Vulnerability Check"
if command -v govulncheck >/dev/null 2>&1; then
    if govulncheck ./... > "$OUTPUT_DIR/vuln-report.txt" 2>&1; then
        success "No known vulnerabilities found"
    else
        error "Vulnerabilities detected (see $OUTPUT_DIR/vuln-report.txt)"
        grep -A 5 "Vulnerability" "$OUTPUT_DIR/vuln-report.txt" | head -20
    fi
else
    warning "govulncheck not installed, skipping vulnerability check"
fi

# 10. Build Validation
header "Build Validation"
info "Testing multi-platform builds..."

PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
BUILD_FAILED=0

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r os arch <<< "$platform"
    if GOOS="$os" GOARCH="$arch" go build -o /dev/null ./cmd/pi-scanner 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $platform"
    else
        echo -e "  ${RED}✗${NC} $platform"
        ((BUILD_FAILED++))
    fi
done

if [ "$BUILD_FAILED" -eq 0 ]; then
    success "All platform builds successful"
else
    error "$BUILD_FAILED platform builds failed"
fi

# 11. Module Validation
header "Module Validation"
if go mod verify > "$OUTPUT_DIR/mod-verify.txt" 2>&1; then
    success "Go modules verified"
else
    error "Go module verification failed"
    cat "$OUTPUT_DIR/mod-verify.txt"
fi

# Check for module tidiness
cp go.mod go.mod.bak
cp go.sum go.sum.bak
go mod tidy
if diff -q go.mod go.mod.bak >/dev/null && diff -q go.sum go.sum.bak >/dev/null; then
    success "go.mod is tidy"
else
    warning "go.mod is not tidy. Run: go mod tidy"
fi
rm -f go.mod.bak go.sum.bak

# 12. Documentation Check
header "Documentation"
MISSING_DOCS=0

for doc in "README.md" "LICENSE" "CONTRIBUTING.md"; do
    if [ -f "$doc" ]; then
        echo -e "  ${GREEN}✓${NC} $doc exists"
    else
        echo -e "  ${RED}✗${NC} $doc missing"
        ((MISSING_DOCS++))
    fi
done

if [ "$MISSING_DOCS" -eq 0 ]; then
    success "All required documentation present"
else
    error "$MISSING_DOCS required documents missing"
fi

# Generate Summary Report
header "Quality Gate Summary"

TOTAL_SCORE=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))

echo -e "\nTotal Checks: $TOTAL_CHECKS"
echo -e "${GREEN}Passed: $PASSED_CHECKS${NC}"
echo -e "${YELLOW}Warnings: $WARNINGS${NC}"
echo -e "${RED}Failed: $FAILED_CHECKS${NC}"
echo -e "\nQuality Score: ${TOTAL_SCORE}%"

# Generate JSON report
cat > "$OUTPUT_DIR/quality-summary.json" << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "total_checks": $TOTAL_CHECKS,
  "passed": $PASSED_CHECKS,
  "warnings": $WARNINGS,
  "failed": $FAILED_CHECKS,
  "score": $TOTAL_SCORE,
  "coverage": ${COVERAGE:-0},
  "reports": {
    "lint": "$OUTPUT_DIR/lint-report.txt",
    "test": "$OUTPUT_DIR/test-report.txt",
    "coverage": "$OUTPUT_DIR/coverage.html",
    "security": "$OUTPUT_DIR/security-report.json",
    "vulnerabilities": "$OUTPUT_DIR/vuln-report.txt"
  }
}
EOF

info "Full reports available in: $OUTPUT_DIR/"

# Exit with appropriate code
if [ "$FAILED_CHECKS" -gt 0 ]; then
    exit 1
elif [ "$WARNINGS" -gt 0 ]; then
    exit 0  # Warnings don't fail the build
else
    exit 0
fi
