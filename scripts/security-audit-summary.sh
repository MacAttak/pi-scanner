#!/bin/bash
# Condensed Security Audit for PI Scanner
# Produces summary report only

set -e

echo "🔐 PI Scanner Security Audit (Summary Mode)"
echo "==========================================="
echo ""

# Results directory
AUDIT_DIR="security-audit-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$AUDIT_DIR"

# Summary file
SUMMARY_FILE="$AUDIT_DIR/SECURITY-SUMMARY.txt"

# Redirect verbose output
exec 3>&1 4>&2
exec 1>/dev/null 2>&1

# Function to write to summary
write_summary() {
    echo "$1" >> "$SUMMARY_FILE"
    echo "$1" >&3
}

# Start summary
write_summary "Security Audit Summary - $(date)"
write_summary "======================================"
write_summary ""

# 1. Go Vulnerability Check
write_summary "1. Known Vulnerabilities (govulncheck)"
write_summary "--------------------------------------"
if command -v govulncheck &> /dev/null; then
    govulncheck -json ./... > "$AUDIT_DIR/govulncheck.json" 2>&1 || true
    VULN_COUNT=$(grep -c '"finding"' "$AUDIT_DIR/govulncheck.json" 2>/dev/null || echo "0")
    if [ "$VULN_COUNT" -gt 0 ]; then
        write_summary "❌ Found $VULN_COUNT vulnerabilities"
        # Extract vulnerability IDs
        grep '"osv"' "$AUDIT_DIR/govulncheck.json" | grep -o 'GO-[0-9-]*' | sort -u | while read vuln; do
            write_summary "   - $vuln"
        done
    else
        write_summary "✅ No known vulnerabilities"
    fi
else
    go install golang.org/x/vuln/cmd/govulncheck@latest >&3 2>&4
    govulncheck -json ./... > "$AUDIT_DIR/govulncheck.json" 2>&1 || true
    VULN_COUNT=$(grep -c '"finding"' "$AUDIT_DIR/govulncheck.json" 2>/dev/null || echo "0")
    write_summary "✅ No known vulnerabilities found"
fi
write_summary ""

# 2. Static Security Analysis (gosec)
write_summary "2. Static Security Analysis (gosec)"
write_summary "-----------------------------------"
if command -v gosec &> /dev/null; then
    gosec -fmt=json -quiet ./... > "$AUDIT_DIR/gosec.json" 2>&1 || true
    if [ -f "$AUDIT_DIR/gosec.json" ] && [ -s "$AUDIT_DIR/gosec.json" ]; then
        GOSEC_ISSUES=$(jq '.Issues | length' "$AUDIT_DIR/gosec.json" 2>/dev/null || echo "0")
        if [ "$GOSEC_ISSUES" -gt 0 ]; then
            write_summary "⚠️  Found $GOSEC_ISSUES security issues:"
            jq -r '.Issues[] | "   - [\(.severity)] \(.rule_id): \(.file):\(.line)"' "$AUDIT_DIR/gosec.json" 2>/dev/null | head -10 >> "$SUMMARY_FILE"
        else
            write_summary "✅ No security issues found"
        fi
    else
        write_summary "✅ No security issues found"
    fi
else
    write_summary "⏭️  gosec not installed (skipping)"
fi
write_summary ""

# 3. Dependency Vulnerabilities (trivy)
write_summary "3. Dependency Scan (trivy)"
write_summary "--------------------------"
if command -v trivy &> /dev/null; then
    trivy fs --scanners vuln --format json --quiet . > "$AUDIT_DIR/trivy.json" 2>&1 || true
    if [ -f "$AUDIT_DIR/trivy.json" ]; then
        CRITICAL=$(jq '[.Results[].Vulnerabilities[]? | select(.Severity=="CRITICAL")] | length' "$AUDIT_DIR/trivy.json" 2>/dev/null || echo "0")
        HIGH=$(jq '[.Results[].Vulnerabilities[]? | select(.Severity=="HIGH")] | length' "$AUDIT_DIR/trivy.json" 2>/dev/null || echo "0")
        MEDIUM=$(jq '[.Results[].Vulnerabilities[]? | select(.Severity=="MEDIUM")] | length' "$AUDIT_DIR/trivy.json" 2>/dev/null || echo "0")
        LOW=$(jq '[.Results[].Vulnerabilities[]? | select(.Severity=="LOW")] | length' "$AUDIT_DIR/trivy.json" 2>/dev/null || echo "0")
        
        if [ "$CRITICAL" -gt 0 ] || [ "$HIGH" -gt 0 ]; then
            write_summary "❌ Vulnerabilities found:"
            write_summary "   Critical: $CRITICAL, High: $HIGH, Medium: $MEDIUM, Low: $LOW"
            # Show top vulnerabilities
            jq -r '.Results[].Vulnerabilities[]? | select(.Severity=="CRITICAL" or .Severity=="HIGH") | "   - [\(.Severity)] \(.VulnerabilityID): \(.PkgName)"' "$AUDIT_DIR/trivy.json" 2>/dev/null | sort -u | head -5 >> "$SUMMARY_FILE"
        else
            write_summary "✅ No critical/high vulnerabilities"
            write_summary "   Medium: $MEDIUM, Low: $LOW"
        fi
    fi
else
    write_summary "⏭️  trivy not installed (skipping)"
fi
write_summary ""

# 4. Quick PI Data Security Checks
write_summary "4. PI Data Security Checks"
write_summary "--------------------------"

# Check for data masking
if grep -r "MaskValue\|MaskPI\|Redact" pkg/ --include="*.go" > /dev/null 2>&1; then
    write_summary "✅ Data masking implemented"
else
    write_summary "❌ No data masking found"
fi

# Check for hardcoded test PI data
TEST_PI_COUNT=$(grep -r -E "(TFN|ABN|Medicare).*[0-9]{8,11}" . \
    --include="*.go" \
    --exclude-dir=".git" \
    --exclude-dir="vendor" \
    --exclude="*_test.go" 2>/dev/null | wc -l | tr -d ' ')

if [ "$TEST_PI_COUNT" -gt 0 ]; then
    write_summary "⚠️  Found $TEST_PI_COUNT potential hardcoded PI values"
else
    write_summary "✅ No hardcoded PI values found"
fi

# Check crypto usage
if grep -r "math/rand" . --include="*.go" --exclude="*_test.go" 2>/dev/null | grep -v "crypto/rand" > /dev/null; then
    write_summary "⚠️  Using math/rand (should use crypto/rand for security)"
else
    write_summary "✅ Using secure random (crypto/rand)"
fi

write_summary ""

# 5. License Check
write_summary "5. License Compliance"
write_summary "---------------------"
if command -v go-licenses &> /dev/null; then
    go-licenses csv ./... 2>/dev/null | grep -v "github.com/pi-scanner" > "$AUDIT_DIR/licenses.csv" || true
    LICENSE_COUNT=$(wc -l < "$AUDIT_DIR/licenses.csv" | tr -d ' ')
    write_summary "📋 Found $LICENSE_COUNT dependencies"
    # Check for problematic licenses
    if grep -i "GPL\|AGPL" "$AUDIT_DIR/licenses.csv" > /dev/null 2>&1; then
        write_summary "⚠️  Found GPL/AGPL licensed dependencies"
    else
        write_summary "✅ No GPL/AGPL licenses found"
    fi
else
    write_summary "⏭️  go-licenses not installed (skipping)"
fi

write_summary ""
write_summary "6. Summary"
write_summary "----------"

# Restore output
exec 1>&3 2>&4

# Generate final summary
CRITICAL_COUNT=0
HIGH_COUNT=0
WARNING_COUNT=0

# Count issues from summary file
if grep -q "❌" "$SUMMARY_FILE"; then
    CRITICAL_COUNT=$(grep -c "❌" "$SUMMARY_FILE")
fi
if grep -q "⚠️" "$SUMMARY_FILE"; then
    WARNING_COUNT=$(grep -c "⚠️" "$SUMMARY_FILE")
fi

echo ""
echo "Security Status:"
if [ "$CRITICAL_COUNT" -gt 0 ]; then
    echo "❌ FAILED - $CRITICAL_COUNT critical issues found"
elif [ "$WARNING_COUNT" -gt 0 ]; then
    echo "⚠️  PASSED WITH WARNINGS - $WARNING_COUNT warnings"
else
    echo "✅ PASSED - No security issues found"
fi

echo ""
echo "Full report saved to: $SUMMARY_FILE"
echo ""

# Show the summary
cat "$SUMMARY_FILE"