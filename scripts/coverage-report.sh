#!/bin/bash
# Generate detailed coverage report with trend tracking

set -euo pipefail

# Configuration
COVERAGE_DIR="${COVERAGE_DIR:-.coverage}"
HISTORY_FILE="$COVERAGE_DIR/coverage-history.json"
BADGE_FILE="$COVERAGE_DIR/coverage-badge.svg"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Create coverage directory
mkdir -p "$COVERAGE_DIR"

# Run tests with coverage
echo "Running tests with coverage analysis..."
go test -coverprofile="$COVERAGE_DIR/coverage.out" -covermode=atomic ./... > "$COVERAGE_DIR/test.log" 2>&1

# Generate reports
go tool cover -html="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.html"
go tool cover -func="$COVERAGE_DIR/coverage.out" -o "$COVERAGE_DIR/coverage.txt"

# Extract overall coverage
TOTAL_COVERAGE=$(grep "total:" "$COVERAGE_DIR/coverage.txt" | awk '{print $3}' | sed 's/%//')

# Generate detailed package report
echo -e "\n📊 Coverage Report"
echo "=================="
echo -e "Total Coverage: ${TOTAL_COVERAGE}%\n"

# Package breakdown with visual indicators
echo "Package Coverage:"
while IFS= read -r line; do
    if [[ $line == *"github.com/MacAttak/pi-scanner"* ]] && [[ $line != *"total:"* ]]; then
        pkg=$(echo "$line" | awk '{print $1}')
        coverage=$(echo "$line" | awk '{print $3}' | sed 's/%//')
        coverage_int=${coverage%.*}

        # Visual indicator
        if [ "$coverage_int" -ge 80 ]; then
            color=$GREEN
            symbol="✅"
        elif [ "$coverage_int" -ge 60 ]; then
            color=$YELLOW
            symbol="⚠️ "
        else
            color=$RED
            symbol="❌"
        fi

        # Progress bar
        filled=$((coverage_int / 5))
        empty=$((20 - filled))
        bar="["
        for ((i=0; i<filled; i++)); do bar="${bar}█"; done
        for ((i=0; i<empty; i++)); do bar="${bar}░"; done
        bar="${bar}]"

        printf "%s %-50s %s %s%6s%%%s\n" "$symbol" "$pkg" "$bar" "$color" "$coverage" "$NC"
    fi
done < "$COVERAGE_DIR/coverage.txt"

# Track coverage history
if [ ! -f "$HISTORY_FILE" ]; then
    echo "[]" > "$HISTORY_FILE"
fi

# Add current coverage to history
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)
jq --arg timestamp "$TIMESTAMP" \
   --arg coverage "$TOTAL_COVERAGE" \
   --arg commit "${GITHUB_SHA:-$(git rev-parse HEAD 2>/dev/null || echo 'unknown')}" \
   '. += [{timestamp: $timestamp, coverage: ($coverage | tonumber), commit: $commit}] | .[-30:]' \
   "$HISTORY_FILE" > "$HISTORY_FILE.tmp" && mv "$HISTORY_FILE.tmp" "$HISTORY_FILE"

# Generate coverage badge SVG
generate_badge() {
    local coverage=$1
    local color

    if [ "${coverage%.*}" -ge 80 ]; then
        color="#4c1"  # Green
    elif [ "${coverage%.*}" -ge 60 ]; then
        color="#dfb317"  # Yellow
    else
        color="#e05d44"  # Red
    fi

    cat > "$BADGE_FILE" << EOF
<svg xmlns="http://www.w3.org/2000/svg" width="116" height="20">
  <linearGradient id="b" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="116" height="20" rx="3" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h63v20H0z"/>
    <path fill="$color" d="M63 0h53v20H63z"/>
    <path fill="url(#b)" d="M0 0h116v20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="31.5" y="15" fill="#010101" fill-opacity=".3">coverage</text>
    <text x="31.5" y="14">coverage</text>
    <text x="88.5" y="15" fill="#010101" fill-opacity=".3">${coverage}%</text>
    <text x="88.5" y="14">${coverage}%</text>
  </g>
</svg>
EOF
}

generate_badge "$TOTAL_COVERAGE"

# Coverage trend
echo -e "\n📈 Coverage Trend (last 5 entries):"
jq -r '.[-5:] | reverse | .[] | "\(.timestamp): \(.coverage)% (\(.commit[0:7]))"' "$HISTORY_FILE"

# Find uncovered code
echo -e "\n🔍 Uncovered Code (top 10 files by missing coverage):"
go tool cover -func="$COVERAGE_DIR/coverage.out" | \
    grep -E "github.com/MacAttak/pi-scanner" | \
    grep -v "100.0%" | \
    grep -v "total:" | \
    awk '{print $3 " " $1}' | \
    sed 's/%//' | \
    sort -n | \
    head -10 | \
    while read coverage file; do
        missing=$(echo "100 - $coverage" | bc)
        printf "  %-50s %5.1f%% missing\n" "$file" "$missing"
    done

# Generate recommendations
echo -e "\n💡 Recommendations:"
if [ "${TOTAL_COVERAGE%.*}" -lt 70 ]; then
    echo "  - Overall coverage is below 70%. Focus on adding tests for core packages."
fi

# Check critical packages
for pkg in "pkg/detection" "pkg/validation" "pkg/risk"; do
    pkg_coverage=$(grep "$pkg" "$COVERAGE_DIR/coverage.txt" | head -1 | awk '{print $3}' | sed 's/%//')
    if [ -n "$pkg_coverage" ]; then
        pkg_cov_int=${pkg_coverage%.*}
        case "$pkg" in
            *detection*) min=80 ;;
            *validation*) min=90 ;;
            *risk*) min=75 ;;
            *) min=70 ;;
        esac

        if [ "$pkg_cov_int" -lt "$min" ]; then
            echo "  - $pkg has ${pkg_coverage}% coverage (need ${min}%)"
        fi
    fi
done

# Summary
echo -e "\n📄 Reports Generated:"
echo "  - HTML Report: $COVERAGE_DIR/coverage.html"
echo "  - Text Report: $COVERAGE_DIR/coverage.txt"
echo "  - Coverage Badge: $BADGE_FILE"
echo "  - History: $HISTORY_FILE"

# Exit code based on threshold
THRESHOLD="${COVERAGE_THRESHOLD:-70}"
if [ "${TOTAL_COVERAGE%.*}" -lt "$THRESHOLD" ]; then
    echo -e "\n${RED}❌ Coverage ${TOTAL_COVERAGE}% is below threshold of ${THRESHOLD}%${NC}"
    exit 1
else
    echo -e "\n${GREEN}✅ Coverage ${TOTAL_COVERAGE}% meets threshold of ${THRESHOLD}%${NC}"
    exit 0
fi
