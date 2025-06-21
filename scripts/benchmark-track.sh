#!/bin/bash
# Track and compare performance benchmarks over time

set -euo pipefail

# Configuration
BENCH_DIR="${BENCH_DIR:-.benchmarks}"
CURRENT_FILE="$BENCH_DIR/current.txt"
BASELINE_FILE="$BENCH_DIR/baseline.txt"
HISTORY_FILE="$BENCH_DIR/history.json"
COMPARISON_FILE="$BENCH_DIR/comparison.md"

# Thresholds
REGRESSION_THRESHOLD="${REGRESSION_THRESHOLD:-10}"  # 10% slower is a regression
IMPROVEMENT_THRESHOLD="${IMPROVEMENT_THRESHOLD:-10}" # 10% faster is noteworthy

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# Create benchmark directory
mkdir -p "$BENCH_DIR"

# Run benchmarks
echo "🏃 Running benchmarks..."

# Critical packages to benchmark
PACKAGES=(
    "pkg/detection"
    "pkg/detection/patterns"
    "pkg/validation"
    "pkg/risk"
    "pkg/scanner"
)

# Run benchmarks and combine results
: > "$CURRENT_FILE"
for pkg in "${PACKAGES[@]}"; do
    echo "  Benchmarking $pkg..."
    if go test -bench=. -benchmem -benchtime=10s -run=^$ "./$pkg" >> "$CURRENT_FILE" 2>&1; then
        echo -e "  ${GREEN}✓${NC} $pkg completed"
    else
        echo -e "  ${RED}✗${NC} $pkg failed"
    fi
done

# Parse benchmark results
parse_benchmarks() {
    local file=$1
    grep -E "^Benchmark" "$file" | while IFS=$'\t' read -r name time rest; do
        # Extract benchmark name and iterations
        bench_name=$(echo "$name" | awk '{print $1}')
        iterations=$(echo "$name" | awk '{print $2}')

        # Extract ns/op
        ns_per_op=$(echo "$time" | grep -oE '[0-9.]+\s*ns/op' | awk '{print $1}')

        # Extract allocations and memory
        allocs=$(echo "$rest" | grep -oE '[0-9]+\s*allocs/op' | awk '{print $1}' || echo "0")
        bytes=$(echo "$rest" | grep -oE '[0-9]+\s*B/op' | awk '{print $1}' || echo "0")

        if [ -n "$ns_per_op" ]; then
            echo "$bench_name|$iterations|$ns_per_op|$allocs|$bytes"
        fi
    done
}

# Create baseline if it doesn't exist
if [ ! -f "$BASELINE_FILE" ]; then
    echo "📸 Creating baseline..."
    cp "$CURRENT_FILE" "$BASELINE_FILE"
fi

# Compare with baseline
echo -e "\n📊 Benchmark Comparison"
echo "====================="

# Parse current and baseline results
declare -A current_results baseline_results

while IFS='|' read -r name iters ns allocs bytes; do
    current_results["$name"]="$iters|$ns|$allocs|$bytes"
done < <(parse_benchmarks "$CURRENT_FILE")

while IFS='|' read -r name iters ns allocs bytes; do
    baseline_results["$name"]="$iters|$ns|$allocs|$bytes"
done < <(parse_benchmarks "$BASELINE_FILE")

# Generate comparison report
{
    echo "# Benchmark Comparison Report"
    echo ""
    echo "Generated: $(date)"
    echo ""
    echo "## Summary"
    echo ""

    improvements=0
    regressions=0
    unchanged=0

    echo "| Benchmark | Baseline (ns/op) | Current (ns/op) | Change | Status |"
    echo "|-----------|------------------|-----------------|--------|--------|"
} > "$COMPARISON_FILE"

# Compare each benchmark
for bench in "${!current_results[@]}"; do
    IFS='|' read -r curr_iters curr_ns curr_allocs curr_bytes <<< "${current_results[$bench]}"

    if [ -n "${baseline_results[$bench]:-}" ]; then
        IFS='|' read -r base_iters base_ns base_allocs base_bytes <<< "${baseline_results[$bench]}"

        # Calculate percentage change
        if [ "$base_ns" != "0" ]; then
            change=$(echo "scale=2; (($curr_ns - $base_ns) / $base_ns) * 100" | bc)
            change_int=${change%.*}

            # Determine status
            if [ "$change_int" -le "-$IMPROVEMENT_THRESHOLD" ]; then
                status="✅ Improved"
                symbol="${GREEN}▼${NC}"
                ((improvements++))
            elif [ "$change_int" -ge "$REGRESSION_THRESHOLD" ]; then
                status="❌ Regression"
                symbol="${RED}▲${NC}"
                ((regressions++))
            else
                status="➖ No change"
                symbol="${YELLOW}≈${NC}"
                ((unchanged++))
            fi

            # Output comparison
            printf "%-50s %10.2f ns/op %s %10.2f ns/op (%+.1f%%) %s\n" \
                "$bench" "$base_ns" "$symbol" "$curr_ns" "$change" "$status"

            # Add to markdown report
            echo "| $bench | $base_ns | $curr_ns | ${change}% | $status |" >> "$COMPARISON_FILE"
        fi
    else
        echo -e "${BLUE}NEW${NC} $bench: $curr_ns ns/op"
        echo "| $bench | - | $curr_ns | NEW | 🆕 New |" >> "$COMPARISON_FILE"
    fi
done

# Memory usage comparison
echo -e "\n💾 Memory Usage"
echo "==============="

{
    echo ""
    echo "## Memory Usage"
    echo ""
    echo "| Benchmark | Baseline (B/op) | Current (B/op) | Baseline (allocs/op) | Current (allocs/op) |"
    echo "|-----------|-----------------|----------------|---------------------|-------------------|"
} >> "$COMPARISON_FILE"

for bench in "${!current_results[@]}"; do
    IFS='|' read -r curr_iters curr_ns curr_allocs curr_bytes <<< "${current_results[$bench]}"

    if [ -n "${baseline_results[$bench]:-}" ]; then
        IFS='|' read -r base_iters base_ns base_allocs base_bytes <<< "${baseline_results[$bench]}"

        printf "%-50s %10s B/op → %10s B/op | %6s → %6s allocs\n" \
            "$bench" "$base_bytes" "$curr_bytes" "$base_allocs" "$curr_allocs"

        echo "| $bench | $base_bytes | $curr_bytes | $base_allocs | $curr_allocs |" >> "$COMPARISON_FILE"
    fi
done

# Add summary to report
{
    echo ""
    echo "## Results"
    echo ""
    echo "- ✅ Improvements: $improvements"
    echo "- ❌ Regressions: $regressions"
    echo "- ➖ Unchanged: $unchanged"
    echo ""
} >> "$COMPARISON_FILE"

# Update history
if [ ! -f "$HISTORY_FILE" ]; then
    echo "[]" > "$HISTORY_FILE"
fi

# Create history entry
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
COMMIT="${GITHUB_SHA:-$(git rev-parse HEAD 2>/dev/null || echo 'unknown')}"

# Calculate average performance
total_ns=0
count=0
for bench in "${!current_results[@]}"; do
    IFS='|' read -r iters ns allocs bytes <<< "${current_results[$bench]}"
    total_ns=$(echo "$total_ns + $ns" | bc)
    ((count++))
done

if [ "$count" -gt 0 ]; then
    avg_ns=$(echo "scale=2; $total_ns / $count" | bc)
else
    avg_ns=0
fi

# Add to history
jq --arg timestamp "$TIMESTAMP" \
   --arg commit "$COMMIT" \
   --arg avg_ns "$avg_ns" \
   --argjson improvements "$improvements" \
   --argjson regressions "$regressions" \
   '. += [{
     timestamp: $timestamp,
     commit: $commit,
     avg_ns_per_op: ($avg_ns | tonumber),
     improvements: $improvements,
     regressions: $regressions
   }] | .[-50:]' \
   "$HISTORY_FILE" > "$HISTORY_FILE.tmp" && mv "$HISTORY_FILE.tmp" "$HISTORY_FILE"

# Show performance trend
echo -e "\n📈 Performance Trend (last 5 runs):"
jq -r '.[-5:] | reverse | .[] |
  "\(.timestamp | split("T")[0]): \(.avg_ns_per_op) ns/op avg (\(.improvements) ↑ \(.regressions) ↓)"' \
  "$HISTORY_FILE"

# Critical benchmark checks
echo -e "\n🎯 Critical Benchmarks:"
critical_checks=(
    "BenchmarkDetector:1000:Detection must process >1000 files/sec"
    "BenchmarkPatternMatch:50000:Pattern matching must handle >50k ops/sec"
    "BenchmarkValidation:100000:Validation must handle >100k ops/sec"
)

for check in "${critical_checks[@]}"; do
    IFS=':' read -r bench_pattern threshold message <<< "$check"

    for bench in "${!current_results[@]}"; do
        if [[ "$bench" == *"$bench_pattern"* ]]; then
            IFS='|' read -r iters ns allocs bytes <<< "${current_results[$bench]}"
            ops_per_sec=$(echo "scale=0; 1000000000 / $ns" | bc)

            if [ "$ops_per_sec" -ge "$threshold" ]; then
                echo -e "  ${GREEN}✓${NC} $bench: $ops_per_sec ops/sec - $message"
            else
                echo -e "  ${RED}✗${NC} $bench: $ops_per_sec ops/sec - $message"
            fi
        fi
    done
done

# Summary
echo -e "\n📄 Reports:"
echo "  - Current Results: $CURRENT_FILE"
echo "  - Baseline: $BASELINE_FILE"
echo "  - Comparison: $COMPARISON_FILE"
echo "  - History: $HISTORY_FILE"

# Update baseline if requested
if [ "${UPDATE_BASELINE:-false}" = "true" ]; then
    echo -e "\n📸 Updating baseline..."
    cp "$CURRENT_FILE" "$BASELINE_FILE"
    echo "  Baseline updated!"
fi

# Exit code based on regressions
if [ "$regressions" -gt 0 ]; then
    echo -e "\n${RED}❌ Performance regressions detected!${NC}"
    exit 1
else
    echo -e "\n${GREEN}✅ No performance regressions${NC}"
    exit 0
fi
