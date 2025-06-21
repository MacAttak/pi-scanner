#!/bin/bash
# Simplified benchmark tracking for compatibility with bash 3.x

set -euo pipefail

# Configuration
BENCH_DIR="${BENCH_DIR:-.benchmarks}"
CURRENT_FILE="$BENCH_DIR/current.txt"
BASELINE_FILE="$BENCH_DIR/baseline.txt"
COMPARISON_FILE="$BENCH_DIR/comparison.md"

# Create benchmark directory
mkdir -p "$BENCH_DIR"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "🏃 Running benchmarks..."

# Critical packages to benchmark
PACKAGES=(
    "pkg/detection"
    "pkg/validation"
    "pkg/risk"
)

# Run benchmarks and combine results
: > "$CURRENT_FILE"
for pkg in "${PACKAGES[@]}"; do
    echo "  Benchmarking $pkg..."
    if go test -bench=. -benchmem -benchtime=5s -run=^$ "./$pkg" >> "$CURRENT_FILE" 2>&1; then
        echo -e "  ${GREEN}✓${NC} $pkg completed"
    else
        echo -e "  ${RED}✗${NC} $pkg failed"
    fi
done

# Create baseline if it doesn't exist
if [ ! -f "$BASELINE_FILE" ]; then
    echo "📸 Creating baseline..."
    cp "$CURRENT_FILE" "$BASELINE_FILE"
fi

# Simple comparison
echo -e "\n📊 Benchmark Results"
echo "===================="

# Show current results
echo -e "\nCurrent benchmarks:"
grep -E "^Benchmark.*ns/op" "$CURRENT_FILE" | head -10 || echo "No benchmark results found"

# Show baseline for comparison
if [ -f "$BASELINE_FILE" ]; then
    echo -e "\nBaseline benchmarks:"
    grep -E "^Benchmark.*ns/op" "$BASELINE_FILE" | head -10 || echo "No baseline results found"
fi

echo -e "\n📄 Reports:"
echo "  - Current: $CURRENT_FILE"
echo "  - Baseline: $BASELINE_FILE"

# Update baseline if requested
if [ "${UPDATE_BASELINE:-false}" = "true" ]; then
    echo -e "\n📸 Updating baseline..."
    cp "$CURRENT_FILE" "$BASELINE_FILE"
    echo "  Baseline updated!"
fi

echo -e "\n${GREEN}✅ Benchmark run complete${NC}"
exit 0