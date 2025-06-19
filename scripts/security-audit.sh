#!/bin/bash
# Comprehensive Security Audit for PI Scanner
# Following 2025 best practices for Go security

set -e

echo "🔐 PI Scanner Security Audit"
echo "=========================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Results directory
AUDIT_DIR="security-audit-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$AUDIT_DIR"

# Summary counters
TOTAL_ISSUES=0
CRITICAL_ISSUES=0
HIGH_ISSUES=0

# Function to print section headers
print_section() {
    echo ""
    echo "🔍 $1"
    echo "-------------------------------------------"
}

# Function to check if command exists
check_command() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${YELLOW}Warning: $1 is not installed. Installing...${NC}"
        return 1
    fi
    return 0
}

# 1. Install/Update Security Tools
print_section "Installing/Updating Security Tools"

# gosec - Go Security Checker
if ! check_command gosec; then
    echo "Installing gosec..."
    go install github.com/securego/gosec/v2/cmd/gosec@latest
fi

# govulncheck - Go Vulnerability Scanner
if ! check_command govulncheck; then
    echo "Installing govulncheck..."
    go install golang.org/x/vuln/cmd/govulncheck@latest
fi

# nancy - Dependency Scanner
if ! check_command nancy; then
    echo "Installing nancy..."
    go install github.com/sonatype-nexus-community/nancy@latest
fi

# trivy - Comprehensive Scanner
if ! check_command trivy; then
    echo "Installing trivy..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install trivy
    else
        curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin
    fi
fi

# staticcheck - Go Static Analysis
if ! check_command staticcheck; then
    echo "Installing staticcheck..."
    go install honnef.co/go/tools/cmd/staticcheck@latest
fi

# 2. Run gosec Security Scan
print_section "Running gosec Static Security Analysis"

echo "Scanning for security issues..."
gosec -fmt=sarif -out="$AUDIT_DIR/gosec-report.sarif" -severity=low ./... 2>&1 | tee "$AUDIT_DIR/gosec-output.txt" || true

# Also generate JSON for parsing
gosec -fmt=json -out="$AUDIT_DIR/gosec-report.json" -severity=low ./... 2>/dev/null || true

# Count gosec issues
if [ -f "$AUDIT_DIR/gosec-report.json" ]; then
    GOSEC_ISSUES=$(jq '.Issues | length' "$AUDIT_DIR/gosec-report.json" 2>/dev/null || echo "0")
    GOSEC_HIGH=$(jq '[.Issues[] | select(.severity == "HIGH")] | length' "$AUDIT_DIR/gosec-report.json" 2>/dev/null || echo "0")
    GOSEC_CRITICAL=$(jq '[.Issues[] | select(.severity == "CRITICAL")] | length' "$AUDIT_DIR/gosec-report.json" 2>/dev/null || echo "0")
    
    TOTAL_ISSUES=$((TOTAL_ISSUES + GOSEC_ISSUES))
    HIGH_ISSUES=$((HIGH_ISSUES + GOSEC_HIGH))
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + GOSEC_CRITICAL))
    
    echo -e "${YELLOW}Found $GOSEC_ISSUES issues (Critical: $GOSEC_CRITICAL, High: $GOSEC_HIGH)${NC}"
fi

# 3. Run govulncheck for Known Vulnerabilities
print_section "Checking for Known Vulnerabilities (govulncheck)"

echo "Scanning dependencies and code..."
govulncheck -json ./... > "$AUDIT_DIR/govulncheck-report.json" 2>&1 || true

# Parse vulnerability results
if [ -f "$AUDIT_DIR/govulncheck-report.json" ]; then
    # Count vulnerabilities (simplified parsing)
    VULN_COUNT=$(grep -c '"type":"osv"' "$AUDIT_DIR/govulncheck-report.json" || echo "0")
    if [ "$VULN_COUNT" -gt 0 ]; then
        echo -e "${RED}Found $VULN_COUNT known vulnerabilities${NC}"
        TOTAL_ISSUES=$((TOTAL_ISSUES + VULN_COUNT))
        HIGH_ISSUES=$((HIGH_ISSUES + VULN_COUNT))
    else
        echo -e "${GREEN}No known vulnerabilities found${NC}"
    fi
fi

# 4. Run nancy for Dependency Scanning
print_section "Scanning Dependencies with Nancy"

if [ -f "go.sum" ]; then
    echo "Analyzing go.sum for vulnerable dependencies..."
    go list -json -deps ./... | nancy sleuth -o "$AUDIT_DIR/nancy-report.txt" 2>&1 || true
    
    # Check if vulnerabilities were found
    if grep -q "Vulnerable Packages" "$AUDIT_DIR/nancy-report.txt"; then
        NANCY_VULNS=$(grep -c "Vulnerable Packages" "$AUDIT_DIR/nancy-report.txt" || echo "0")
        echo -e "${RED}Found dependency vulnerabilities${NC}"
        TOTAL_ISSUES=$((TOTAL_ISSUES + NANCY_VULNS))
    else
        echo -e "${GREEN}No vulnerable dependencies found${NC}"
    fi
else
    echo -e "${YELLOW}No go.sum file found${NC}"
fi

# 5. Run trivy for Comprehensive Scanning
print_section "Running Trivy Comprehensive Scan"

echo "Scanning filesystem for vulnerabilities..."
trivy fs --scanners vuln,misconfig,secret . \
    --format json \
    --output "$AUDIT_DIR/trivy-report.json" 2>&1 | tee "$AUDIT_DIR/trivy-output.txt" || true

# Also generate human-readable report
trivy fs --scanners vuln,misconfig,secret . \
    --format table \
    --output "$AUDIT_DIR/trivy-report.txt" 2>/dev/null || true

# 6. Check for Hardcoded Secrets
print_section "Scanning for Hardcoded Secrets"

echo "Checking for potential secrets..."
# Custom patterns for Australian PI
grep -r -E "(TFN|tfn|ABN|abn|Medicare|medicare).*[0-9]{8,11}" . \
    --include="*.go" \
    --include="*.yaml" \
    --include="*.yml" \
    --include="*.json" \
    --exclude-dir=".git" \
    --exclude-dir="vendor" \
    --exclude-dir="security-audit-*" \
    --exclude="*_test.go" > "$AUDIT_DIR/hardcoded-secrets.txt" 2>/dev/null || true

if [ -s "$AUDIT_DIR/hardcoded-secrets.txt" ]; then
    echo -e "${YELLOW}Potential hardcoded secrets found - review manually${NC}"
else
    echo -e "${GREEN}No obvious hardcoded secrets found${NC}"
fi

# 7. Check Binary Permissions
print_section "Checking Binary Security"

if [ -f "bin/pi-scanner" ]; then
    echo "Analyzing compiled binary..."
    
    # Check for stack protection
    if otool -hv bin/pi-scanner 2>/dev/null | grep -q "PIE" || \
       readelf -h bin/pi-scanner 2>/dev/null | grep -q "DYN"; then
        echo -e "${GREEN}✓ Position Independent Executable (PIE) enabled${NC}"
    else
        echo -e "${YELLOW}⚠ PIE not detected${NC}"
    fi
    
    # Check binary size and stripping
    BINARY_SIZE=$(ls -lh bin/pi-scanner | awk '{print $5}')
    echo "Binary size: $BINARY_SIZE"
fi

# 8. License and Dependency Analysis
print_section "License Compliance Check"

echo "Analyzing dependency licenses..."
go-licenses report ./... --ignore github.com/pi-scanner > "$AUDIT_DIR/licenses.csv" 2>/dev/null || {
    echo "Installing go-licenses..."
    go install github.com/google/go-licenses@latest
    go-licenses report ./... --ignore github.com/pi-scanner > "$AUDIT_DIR/licenses.csv" 2>/dev/null || true
}

# 9. OWASP Dependency Check (if available)
print_section "OWASP Dependency Analysis"

if command -v dependency-check &> /dev/null; then
    echo "Running OWASP dependency check..."
    dependency-check --project "PI Scanner" \
        --scan . \
        --format JSON \
        --out "$AUDIT_DIR/owasp-report.json" \
        --suppression .dependency-check-suppressions.xml 2>/dev/null || true
else
    echo -e "${YELLOW}OWASP Dependency Check not installed - skipping${NC}"
fi

# 10. Custom Security Checks
print_section "PI Scanner Specific Security Checks"

echo "Checking for PI data handling issues..."

# Check for proper data masking
echo -n "Checking data masking implementation... "
if grep -r "MaskValue\|MaskPI\|Redact" pkg/ --include="*.go" > /dev/null; then
    echo -e "${GREEN}✓ Data masking found${NC}"
else
    echo -e "${RED}✗ No data masking found${NC}"
    TOTAL_ISSUES=$((TOTAL_ISSUES + 1))
fi

# Check for secure temp file handling
echo -n "Checking temporary file security... "
if grep -r "ioutil\.TempFile\|os\.CreateTemp" . --include="*.go" | grep -v "0600\|0700" > "$AUDIT_DIR/insecure-temp.txt"; then
    if [ -s "$AUDIT_DIR/insecure-temp.txt" ]; then
        echo -e "${YELLOW}⚠ Potential insecure temp files${NC}"
    else
        echo -e "${GREEN}✓ Temp files use secure permissions${NC}"
    fi
else
    echo -e "${GREEN}✓ No temp file issues${NC}"
fi

# Check for secure random usage
echo -n "Checking cryptographic randomness... "
if grep -r "math/rand" . --include="*.go" --exclude="*_test.go" | grep -v "crypto/rand" > "$AUDIT_DIR/weak-random.txt"; then
    if [ -s "$AUDIT_DIR/weak-random.txt" ]; then
        echo -e "${YELLOW}⚠ Using math/rand instead of crypto/rand${NC}"
    fi
else
    echo -e "${GREEN}✓ Using secure random${NC}"
fi

# 11. Generate Summary Report
print_section "Security Audit Summary"

cat > "$AUDIT_DIR/SECURITY-AUDIT-SUMMARY.md" << EOF
# PI Scanner Security Audit Report

**Date**: $(date)
**Commit**: $(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

## Executive Summary

Total Issues Found: **$TOTAL_ISSUES**
- Critical: **$CRITICAL_ISSUES**
- High: **$HIGH_ISSUES**

## Scan Results

### 1. Static Application Security Testing (SAST)
- **Tool**: gosec
- **Report**: gosec-report.sarif
- **Issues**: $GOSEC_ISSUES

### 2. Known Vulnerability Scanning
- **Tool**: govulncheck
- **Report**: govulncheck-report.json
- **Vulnerabilities**: $VULN_COUNT

### 3. Dependency Scanning
- **Tool**: nancy (Sonatype)
- **Report**: nancy-report.txt
- **Status**: See report for details

### 4. Comprehensive Security Scan
- **Tool**: trivy
- **Report**: trivy-report.json
- **Categories**: Vulnerabilities, Misconfigurations, Secrets

### 5. License Compliance
- **Report**: licenses.csv
- **Status**: Review for compliance with your policies

## Recommendations

1. **Immediate Actions**:
   - Fix all CRITICAL and HIGH severity issues
   - Update vulnerable dependencies
   - Remove any hardcoded secrets

2. **Best Practices**:
   - Run security scans on every CI/CD build
   - Keep dependencies up to date
   - Regular security audits (monthly)

3. **PI Data Protection**:
   - Ensure all PI data is masked in logs/reports
   - Use secure random for any cryptographic operations
   - Implement proper access controls

## Next Steps

1. Review all reports in detail
2. Create tickets for security fixes
3. Implement automated security scanning in CI/CD
4. Schedule regular security reviews

---
*Generated by security-audit.sh*
EOF

# Display summary
echo ""
echo "======================================"
echo -e "Security Audit Complete!"
echo "======================================"
echo ""
echo -e "Total Issues: ${TOTAL_ISSUES}"
echo -e "Critical: ${RED}${CRITICAL_ISSUES}${NC}"
echo -e "High: ${YELLOW}${HIGH_ISSUES}${NC}"
echo ""
echo "Reports saved to: $AUDIT_DIR/"
echo ""
echo "Key reports to review:"
echo "  - $AUDIT_DIR/SECURITY-AUDIT-SUMMARY.md"
echo "  - $AUDIT_DIR/gosec-report.sarif"
echo "  - $AUDIT_DIR/trivy-report.txt"
echo "  - $AUDIT_DIR/govulncheck-report.json"
echo ""

# Exit with error if critical issues found
if [ "$CRITICAL_ISSUES" -gt 0 ]; then
    echo -e "${RED}⚠️  Critical security issues found!${NC}"
    exit 1
fi

if [ "$HIGH_ISSUES" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  High severity issues found - review recommended${NC}"
    exit 0
fi

echo -e "${GREEN}✅ No critical security issues found${NC}"