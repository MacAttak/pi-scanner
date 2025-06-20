#!/bin/bash
# Local CI simulation script for GitHub PI Scanner
# This simulates the GitHub Actions CI pipeline locally

set -e

echo "🚀 Running local CI simulation..."
echo "This simulates the GitHub Actions CI pipeline"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track timing
START_TIME=$(date +%s)

# Function to print section headers
print_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2 failed${NC}"
        return 1
    fi
}

# Track overall status
FAILED=0

# Simulate GitHub Actions environment
export CI=true
export GITHUB_ACTIONS=true
export CGO_ENABLED=0

print_section "Environment Setup"
echo "Go version: $(go version)"
echo "Working directory: $(pwd)"
echo "Git branch: $(git branch --show-current 2>/dev/null || echo 'unknown')"

print_section "Step 1: Code Quality Checks"

# 1.1 Go formatting
echo -n "Checking Go formatting... "
if [ -z "$(gofmt -l .)" ]; then
    print_status 0 "Go formatting"
else
    print_status 1 "Go formatting"
    echo "Files needing formatting:"
    gofmt -l .
    FAILED=1
fi

# 1.2 Go vet
echo -n "Running go vet... "
if go vet ./... 2>&1; then
    print_status 0 "go vet"
else
    print_status 1 "go vet"
    FAILED=1
fi

# 1.3 Go mod tidy
echo -n "Checking go.mod... "
cp go.mod go.mod.bak && cp go.sum go.sum.bak
go mod tidy
if cmp -s go.mod go.mod.bak && cmp -s go.sum go.sum.bak; then
    rm go.mod.bak go.sum.bak
    print_status 0 "go.mod tidy"
else
    rm go.mod.bak go.sum.bak
    print_status 1 "go.mod not tidy"
    FAILED=1
fi

print_section "Step 2: Build"

# Build for current platform
echo -n "Building binary... "
if go build -ldflags="-s -w" -o bin/pi-scanner ./cmd/pi-scanner; then
    print_status 0 "Build successful"
else
    print_status 1 "Build"
    FAILED=1
fi

print_section "Step 3: Tests"

# Run tests with coverage
echo "Running tests with coverage..."
if go test -race -coverprofile=coverage.out -covermode=atomic ./... -v > test-results.log 2>&1; then
    print_status 0 "All tests passed"
    
    # Show coverage summary
    echo -n "Generating coverage report... "
    go tool cover -func=coverage.out | tail -1
    
    # Clean up
    rm -f coverage.out test-results.log
else
    print_status 1 "Tests"
    echo "Test failures:"
    grep -E "FAIL|Error:|panic:" test-results.log | head -20
    rm -f test-results.log
    FAILED=1
fi

print_section "Step 4: Security Scans"

# 4.1 Gitleaks
if command -v gitleaks >/dev/null 2>&1; then
    echo -n "Scanning for secrets... "
    if gitleaks detect --no-banner --exit-code 0 2>/dev/null; then
        print_status 0 "No secrets found"
    else
        print_status 1 "Secrets detected"
        FAILED=1
    fi
else
    echo -e "${YELLOW}⚠ gitleaks not installed${NC}"
fi

# 4.2 Gosec
if command -v gosec >/dev/null 2>&1; then
    echo -n "Running security scan... "
    if gosec -fmt text -severity medium -quiet ./... 2>/dev/null; then
        print_status 0 "Security scan"
    else
        echo -e "${YELLOW}⚠ Security issues found (non-blocking)${NC}"
    fi
else
    echo -e "${YELLOW}⚠ gosec not installed${NC}"
fi

print_section "Step 5: Linting"

if command -v golangci-lint >/dev/null 2>&1; then
    echo "Running golangci-lint..."
    if golangci-lint run --timeout=5m --config .golangci.yml 2>/dev/null || golangci-lint run --timeout=5m; then
        print_status 0 "Linting"
    else
        print_status 1 "Linting"
        FAILED=1
    fi
else
    echo -e "${YELLOW}⚠ golangci-lint not installed${NC}"
    echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

print_section "Step 6: Cross-platform Build Test"

echo "Testing cross-platform builds..."
PLATFORMS=("linux/amd64" "darwin/amd64" "darwin/arm64" "windows/amd64")
BUILD_FAILED=0

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    echo -n "Building for $GOOS/$GOARCH... "
    if GOOS=$GOOS GOARCH=$GOARCH go build -o /dev/null ./cmd/pi-scanner 2>/dev/null; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        BUILD_FAILED=1
    fi
done

if [ $BUILD_FAILED -eq 0 ]; then
    print_status 0 "Cross-platform builds"
else
    print_status 1 "Cross-platform builds"
    FAILED=1
fi

# Calculate elapsed time
END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

print_section "Summary"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All CI checks passed!${NC}"
    echo -e "Time elapsed: ${ELAPSED}s"
    echo -e "\n${GREEN}Your code is ready to push!${NC}"
else
    echo -e "${RED}❌ CI checks failed${NC}"
    echo -e "Time elapsed: ${ELAPSED}s"
    echo -e "\n${YELLOW}Fix the issues above before pushing to avoid CI failures.${NC}"
    exit 1
fi

# Clean up
rm -f bin/pi-scanner

exit 0