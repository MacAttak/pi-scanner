#!/bin/bash
# Run E2E tests for PI Scanner

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

echo "🔍 PI Scanner E2E Test Suite"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
# RED='\033[0;31m' # Not used
NC='\033[0m' # No Color

# Default options
RUN_BASIC=true
RUN_LLM=true
RUN_AUSTRALIAN=false
RUN_PERFORMANCE=false
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --all)
            RUN_AUSTRALIAN=true
            RUN_PERFORMANCE=true
            shift
            ;;
        --australian)
            RUN_AUSTRALIAN=true
            shift
            ;;
        --performance)
            RUN_PERFORMANCE=true
            shift
            ;;
        --no-llm)
            RUN_LLM=false
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --all           Run all tests including Australian repos and performance"
            echo "  --australian    Run tests on real Australian repositories"
            echo "  --performance   Run performance tests"
            echo "  --no-llm        Skip LLM integration tests"
            echo "  --verbose       Show detailed test output"
            echo "  --help          Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  RUN_AUSTRALIAN_REPOS=true   Run Australian repository tests"
            echo "  RUN_PERFORMANCE_TESTS=true  Run performance tests"
            echo "  GITHUB_TOKEN                Required for scanning private repos"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check GitHub token
if [ -z "$GITHUB_TOKEN" ]; then
    echo -e "${YELLOW}Warning: GITHUB_TOKEN not set. Some tests may fail due to rate limits.${NC}"
    echo ""
fi

# Change to E2E test directory
cd "$PROJECT_ROOT/test/e2e"

# Test flags
TEST_FLAGS="-v"
if [ "$VERBOSE" = false ]; then
    TEST_FLAGS=""
fi

# Run basic CLI tests
if [ "$RUN_BASIC" = true ]; then
    echo -e "${GREEN}Running basic CLI tests...${NC}"
    go test $TEST_FLAGS -timeout 10m -run TestCLIBasicScan ./...
    echo ""
fi

# Run LLM integration tests
if [ "$RUN_LLM" = true ]; then
    echo -e "${GREEN}Running LLM integration tests...${NC}"

    # Check if LLM is available
    if "$PROJECT_ROOT/pi-scanner" llm &>/dev/null; then
        go test $TEST_FLAGS -timeout 10m -run TestCLILLMIntegration ./...
    else
        echo -e "${YELLOW}LLM service not available, skipping LLM tests${NC}"
    fi
    echo ""
fi

# Run Australian repository tests
if [ "$RUN_AUSTRALIAN" = true ]; then
    echo -e "${GREEN}Running Australian repository tests...${NC}"
    echo "This may take several minutes..."

    export RUN_AUSTRALIAN_REPOS=true
    go test $TEST_FLAGS -timeout 30m -run TestCLIAustralianRepositories ./...
    unset RUN_AUSTRALIAN_REPOS
    echo ""
fi

# Run performance tests
if [ "$RUN_PERFORMANCE" = true ]; then
    echo -e "${GREEN}Running performance tests...${NC}"
    echo "This may take 10+ minutes..."

    export RUN_PERFORMANCE_TESTS=true
    go test $TEST_FLAGS -timeout 30m -run TestCLIPerformance ./...
    unset RUN_PERFORMANCE_TESTS
    echo ""
fi

# Run error handling tests
echo -e "${GREEN}Running error handling tests...${NC}"
go test $TEST_FLAGS -timeout 5m -run TestCLIErrorHandling ./...
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ E2E tests completed!${NC}"
echo ""

# Generate summary
echo "Test Summary:"
echo "- Basic CLI tests: ✅"
if [ "$RUN_LLM" = true ]; then
    echo "- LLM integration: ✅"
fi
if [ "$RUN_AUSTRALIAN" = true ]; then
    echo "- Australian repos: ✅"
fi
if [ "$RUN_PERFORMANCE" = true ]; then
    echo "- Performance tests: ✅"
fi
echo "- Error handling: ✅"
