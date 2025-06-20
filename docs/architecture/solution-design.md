# GitHub PI Scanner - Solution Design Document

## Executive Summary

The GitHub PI Scanner is a CLI-first tool designed to detect and risk-score personally identifiable information (PII) in Commonwealth Bank code repositories. It employs a three-stage detection pipeline combining pattern matching, context validation, and algorithmic verification to achieve >95% accuracy with <5% false positives.

The solution prioritizes:
- **High accuracy** through multi-layer validation
- **Minimal configuration** with intelligent defaults
- **Australian regulatory compliance** (APRA CPS 234, Privacy Act 1988)
- **Performance** on standard corporate hardware
- **Clear risk reporting** for security and audit teams

## Architecture Overview

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Interface                             │
│  Commands: scan, report, configure, validate                    │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                    Orchestration Layer                           │
│  • Repository Manager (GitHub CLI integration)                   │
│  • Scan Coordinator                                              │
│  • Progress Tracker                                              │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                 Multi-Stage Detection Pipeline                   │
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐     │
│  │   Stage 1    │    │   Stage 2    │    │   Stage 3    │     │
│  │Pattern Match │───▶│Context Valid │───▶│  AU Verify   │     │
│  │ (Gitleaks+   │    │(Code-Aware)  │    │ (Checksums)  │     │
│  │   Regex)     │    │              │    │              │     │
│  └──────────────┘    └──────────────┘    └──────────────┘     │
│                                                                  │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                  Risk Analysis Engine                            │
│  • Context Scoring (co-occurrence, proximity)                   │
│  • Environment Detection (test vs production)                    │
│  • Confidence Calculation                                        │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                    Reporting Layer                               │
│  • Summary Report (HTML/PDF for executives)                     │
│  • Detailed Findings (CSV for analysts)                         │
│  • SARIF Export (for tool integration)                          │
└─────────────────────────────────────────────────────────────────┘
```

## Detailed Component Design

### 1. CLI Interface

**Technology**: Go with Cobra CLI framework

**Core Commands**:
```bash
# Basic scan
pi-scanner scan --repo https://github.com/org/repo

# Scan multiple repos
pi-scanner scan --repo-list repos.txt

# Custom configuration
pi-scanner scan --repo https://github.com/org/repo --config scanner.yaml

# Generate report only
pi-scanner report --input scan-results.json --format html
```

**Key Features**:
- Progress bars with ETA
- Interrupt handling (Ctrl+C) with graceful shutdown
- Verbose logging modes
- Dry-run capability

### 2. Repository Manager

**GitHub Integration**:
- Uses GitHub CLI (`gh`) for authentication
- Supports both public and private repositories
- Handles large repos through shallow cloning
- Automatic cleanup after scanning

**Processing Flow**:
1. Authenticate via `gh auth status`
2. Clone repository to temporary directory
3. Build file list excluding binaries and large files
4. Clean up temporary files after scan

### 3. Detection Pipeline

#### Stage 1: Pattern Detection

**Primary Tool**: Gitleaks with custom rules
**Secondary**: Go regex engine for Australian patterns

**Configuration** (embedded defaults):
```toml
[[rules]]
id = "australian-tfn"
description = "Australian Tax File Number"
regex = '''(?i)\b(\d{3}[\s-]?\d{3}[\s-]?\d{3})\b'''
keywords = ["tfn", "tax file", "taxfile"]

[[rules]]
id = "australian-medicare"
description = "Medicare Number"
regex = '''\b([2-6]\d{3}[\s-]?\d{5}[\s-]?\d{1})\b'''
keywords = ["medicare", "health card"]

# ... more rules for ABN, BSB, licenses, etc.
```

**Performance**: Parallel file processing with worker pool

#### Stage 2: Context Validation

**Approach**: Code-aware context analysis
**Implementation**: Pure Go without external dependencies

**Implementation**:
```go
type ContextValidator struct {
    codePatterns    *CodePatternAnalyzer
    proximityEngine *ProximityAnalyzer  
    syntaxAnalyzer  *SyntaxContextAnalyzer
}

func (v *ContextValidator) Validate(finding Finding, fileContent string) ValidationResult {
    // Check if in test/mock context
    // Analyze code proximity patterns
    // Verify syntax context (comments, strings)
    // Return confidence and validation result
}
```

**Context Window**: ±10 lines around detected pattern

#### Stage 3: Australian Validation

**Validates**:
- TFN: Modulus 11 checksum
- ABN: Modulus 89 checksum
- Medicare: Modulus 10 check digit
- BSB: Valid bank/branch lookup

**Example Implementation**:
```go
func ValidateTFN(tfn string) bool {
    weights := []int{1, 4, 3, 7, 5, 8, 6, 9, 10}
    tfn = regexp.MustCompile(`[^\d]`).ReplaceAllString(tfn, "")
    
    if len(tfn) != 9 {
        return false
    }
    
    sum := 0
    for i, weight := range weights {
        digit := int(tfn[i] - '0')
        sum += weight * digit
    }
    
    return sum%11 == 0
}
```

### 4. Risk Analysis Engine

**Risk Scoring Matrix**:

| Risk Level | Criteria | Examples |
|------------|----------|----------|
| **Critical** | Multiple validated high-risk PI in proximity | Name + TFN + Bank Account |
| **High** | Single validated high-risk PI OR multiple medium-risk | Valid TFN alone, Name + Address + DOB |
| **Medium** | Single medium-risk PI OR multiple low-risk | Email + Phone, Address alone |
| **Low** | Single low-risk PI | Email alone, IP address |

**Context Factors**:
```go
type ContextScore struct {
    BaseScore        int
    ProximityBonus   int  // PI elements within 5 lines
    EnvironmentMult  float32  // 0.1 for test, 1.0 for prod
    ConfidenceScore  float32  // ML model confidence
}
```

**Test Data Detection**:
- Path patterns: `/test/`, `/spec/`, `/mock/`, `/__tests__/`
- File patterns: `*.test.*`, `*.spec.*`, `*_test.go`
- Content patterns: Mock data generators, sequential numbers
- Variable names: `testUser`, `mockCustomer`, `fakeData`

### 5. Reporting System

**Summary Report** (HTML/PDF):
```
GitHub PI Scanner Report
========================
Scan Date: 2024-01-15 14:30:00
Repository: github.com/cba/customer-api

Risk Summary:
- Critical: 2 findings
- High: 5 findings  
- Medium: 12 findings
- Low: 45 findings

Top Risks:
1. [CRITICAL] customer_export.go:142 - Full customer record exposed
   - Name, TFN, Address, Bank Account found in proximity
   
2. [CRITICAL] config/prod.yaml:23 - Hardcoded credentials with PI
   - Database connection string contains real customer data
```

**Detailed CSV Export**:
```csv
file_path,line_number,risk_level,pi_types,matched_text,confidence,context
src/export.go,142,CRITICAL,"NAME,TFN,ADDRESS",John Smith 123456789,0.98,"Full customer record"
config/prod.yaml,23,CRITICAL,"CREDENTIALS,EMAIL",admin@bank.com:pass123,0.95,"Database config"
```

## Performance Optimization

### Concurrency Model
```go
type Scanner struct {
    workers    int // Default: runtime.NumCPU()
    bufferSize int // Default: 100 files
    maxMemory  int64 // Default: 2GB
}
```

### Caching Strategy
- Compiled regex patterns cached
- ML model loaded once, reused
- File hash cache to skip unchanged files
- 15-minute result cache for repeated scans

### Memory Management
- Stream processing for large files
- Bounded channels to prevent memory explosion
- Garbage collection tuning for long scans

## Configuration

### Default Configuration (Minimal)
```yaml
# pi-scanner.yaml - ALL FIELDS OPTIONAL
scanner:
  # Override default exclude paths
  exclude_paths:
    - vendor/
    - node_modules/
    - .git/
  
  # Additional test path patterns
  test_paths:
    - "**/testdata/**"
    - "**/fixtures/**"
  
  # Risk score overrides (rarely needed)
  risk_weights:
    tfn: 100      # Default: 100
    medicare: 90  # Default: 90
    email: 20     # Default: 20
```

### Advanced Configuration (Power Users)
```yaml
# Advanced options - not recommended for most users
performance:
  workers: 8
  max_file_size: 100MB
  
ml_model:
  threshold: 0.85
  context_window: 100
  
output:
  format: sarif
  include_code_snippet: true
  max_snippet_length: 200
```

## Security Considerations

1. **No Data Persistence**: All processing in-memory
2. **Secure Cleanup**: Automatic removal of cloned repos
3. **No Network Calls**: ML model runs locally
4. **Access Control**: Relies on GitHub CLI authentication
5. **Output Security**: Reports marked with classification warnings

## Error Handling

### Resilience Patterns
```go
// Circuit breaker for ML inference
type CircuitBreaker struct {
    failures    int
    maxFailures int
    cooldown    time.Duration
}

// Graceful degradation
if mlValidator.IsAvailable() {
    confidence = mlValidator.Validate(context, match)
} else {
    confidence = regexOnlyConfidence(match)
}
```

### Error Categories
1. **Fatal**: Can't authenticate, no disk space
2. **Warning**: ML model unavailable, partial repo access
3. **Info**: Skipped binary files, excluded paths

## Future Extensibility

While not the initial focus, the architecture supports:

1. **Plugin System**: Additional validators via interface
2. **CI/CD Integration**: GitHub Actions, Jenkins plugins
3. **API Mode**: REST endpoints for integration
4. **Real-time Monitoring**: File watcher mode
5. **Multi-language Models**: Beyond English names/addresses

## Implementation Phases

### Phase 1: Core Pipeline (Week 1)
- Basic CLI structure
- Gitleaks integration
- Australian regex patterns
- Simple risk scoring

### Phase 2: ML Integration (Week 2)
- Context validation integration
- Context extraction
- Confidence scoring
- Test data detection

### Phase 3: Polish & Testing (Week 3)
- Full AU validation algorithms
- Advanced risk scoring
- Report generation
- Performance optimization

### Phase 4: Production Ready (Week 4)
- Comprehensive testing
- Documentation
- Security review
- Pilot deployment

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Accuracy | >95% detection rate | Manual validation sample |
| False Positives | <5% | Review of Critical/High findings |
| Performance | <5 min for 1GB repo | Benchmark suite |
| Memory Usage | <2GB for standard scan | Resource monitoring |
| User Satisfaction | >4/5 rating | Pilot user survey |

## Conclusion

This solution design provides a robust, accurate, and user-friendly PI scanner that meets Commonwealth Bank's regulatory requirements while maintaining flexibility for future enhancements. The multi-stage detection pipeline ensures high accuracy, while smart defaults minimize configuration overhead.

The architecture is optimized for:
- **Immediate value**: Works out-of-box with zero configuration
- **Accuracy**: Multi-layer validation reduces false positives
- **Performance**: Efficient on standard corporate hardware
- **Compliance**: Aligned with APRA and Privacy Act requirements
- **Usability**: Clear reports for both technical and executive audiences