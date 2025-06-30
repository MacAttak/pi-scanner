# GitHub PI Scanner - Design and Implementation Overview

## Executive Summary

### What is the PI Scanner?

The GitHub PI Scanner is an automated security tool designed to detect Australian Personally Identifiable Information (PI) within GitHub repositories. It employs pattern recognition, contextual analysis, and optional AI-powered validation to identify sensitive data that may have been inadvertently committed to source code repositories.

### Why was it built?

Australian organizations face strict compliance requirements under the Privacy Act 1988 and the Notifiable Data Breaches (NDB) scheme. With modern software development practices involving numerous repositories and frequent commits, manual review of code for PI exposure is no longer feasible. The PI Scanner addresses this challenge by providing automated, scalable detection of Australian PI across entire GitHub organizations.

### Key Capabilities

- **Comprehensive PI Detection**: Identifies 12+ types of Australian PI including Tax File Numbers (TFN), Medicare numbers, credit cards (Luhn validated), and Australian Business Numbers (ABN)
- **Two-Phase Architecture**: Fast pattern detection followed by optional AI validation for 100% accuracy
- **Banking Domain Intelligence**: AST-based code analysis using Tree-sitter for Java, Scala, and Python with automated detection of high-risk zones (customer data, transactions, integrations)
- **Context-Aware Analysis**: Repository structure analysis and code pattern recognition to minimize false positives
- **Local LLM Integration**: Code-aware validation using on-premise models (LM Studio) with context extraction and intelligent false positive reduction
- **Enterprise Integration**: Supports CI/CD pipelines with multiple output formats
- **Privacy-First Design**: Operates without creating new disclosure risks

## Problem Statement

### The Challenge of Australian PI in Code Repositories

Modern software development creates unprecedented risks for PI exposure:

- **Scale**: Large organizations maintain hundreds of repositories with millions of lines of code
- **Velocity**: Continuous deployment means thousands of commits daily
- **Complexity**: PI can appear in configuration files, test data, logs, and documentation
- **Human Error**: Developers may inadvertently commit real PI when testing integrations

### Compliance Requirements

Australian organizations must comply with:

- **Privacy Act 1988**: Requires protection of personal information
- **Notifiable Data Breaches (NDB) Scheme**: Mandates notification within 72 hours of discovering eligible breaches
- **Industry Standards**: PCI-DSS, ISO 27001, and sector-specific requirements

### Why Manual Review Doesn't Scale

Traditional approaches fail because:
- Code review processes miss PI in large pull requests
- Security teams lack resources to audit every repository
- Point-in-time audits miss newly introduced PI
- Manual patterns can't keep pace with code velocity

## Solution Design

### High-Level Architecture

The PI Scanner employs a sophisticated two-phase approach with intelligent resource management:

```
Resource Initialization
ScanResourceManager → Repository Clone → Context Setup
       ↓                    ↓                 ↓
 Lifecycle Control    GitHub API        AST Analysis
                     Integration         for Java/Scala/Python

Phase 1: Pattern-Based Detection with AST Intelligence
File Discovery → AST Analysis → Pattern Matching → Risk Mapping → Initial Report
      ↓              ↓               ↓                ↓              ↓
 Concurrent      Tree-sitter    Regex + Checksum  Banking Domain  Masked JSON
 Processing      Parsing        + Luhn Algorithm   Intelligence    Output

Interactive Decision Point
       ↓
User Reviews Findings → Choose Validation Scope → Proceed or Skip

Phase 2: AI-Powered Validation (Optional)
Load Raw Findings → Context Extraction → LLM Analysis → Enhanced Report
       ↓                   ↓                  ↓               ↓
 Unmasked Data      Full Code Context    Local LLM      Risk-Adjusted
 From Phase 1       for Accuracy      (LM Studio)       Final Report

Resource Cleanup (Automatic)
       ↓
Repository Cleanup → Memory Clear → Report Finalization
```

### How the Scanner Works

**Phase 1: Pattern-Based Detection (Always Runs)**

1. **Repository Access**
   - Connects to GitHub using personal access tokens
   - Clones repository to temporary directory with automatic cleanup
   - Validates authentication before proceeding

2. **File Discovery and Processing**
   - Traverses repository structure efficiently
   - Filters by file size (default: <1MB) and type
   - Skips binary files and known non-risk directories
   - Concurrent processing with worker pools

3. **Pattern Detection**
   - Pattern matching using optimized regular expressions
   - Checksum validation for structured PI (TFN, ABN, Medicare)
   - Luhn algorithm validation for credit card numbers
   - Context extraction (±50 lines around matches)
   - Initial risk scoring based on patterns and context
   - AST analysis integration for code-aware risk assessment

4. **Phase 1 Report Generation**
   - Masked PI values by default (configurable)
   - JSON output saved to `./reports/<timestamp>_<repo>/phase1_pattern_scan.json`
   - Human-readable summary displayed to user

**Interactive Decision Point**

5. **User Review and Choice**
   - Scanner presents findings summary with counts by risk level
   - User chooses validation scope:
     - Validate all findings (comprehensive but slower)
     - Validate HIGH + MEDIUM only (balanced approach)
     - Validate HIGH + CRITICAL only (fast, focused)
     - Skip validation (pattern results only)
   - Time estimates provided for each option

**Phase 2: AI-Powered Validation (Optional)**

6. **LLM-Enhanced Analysis**
   - Loads findings from Phase 1
   - Sends context to local LLM for analysis
   - LLM determines if PI is real or test data
   - Updates risk scores based on deeper context understanding

7. **Final Report Generation**
   - Enhanced JSON with LLM explanations
   - Refined risk levels with reduced false positives
   - Summary statistics comparing Phase 1 vs Phase 2 results

### Key Design Decisions and Trade-offs

- **Local Processing**: All scanning happens locally to prevent PI transmission
- **Pattern-First Approach**: Regex detection provides speed; validation adds accuracy
- **Modular Architecture**: Allows easy addition of new PI types and validators
- **Default-Secure**: Conservative risk assignment; assumes PI is real unless proven otherwise

## PI Detection Methodology

### What Types of Australian PI are Detected

The scanner detects these Australian PI types:

1. **Tax File Number (TFN)**: 9-digit identifier with modulo 11 checksum
2. **Australian Business Number (ABN)**: 11-digit identifier with modulo 89 checksum
3. **Medicare Number**: 10-11 digit health identifier with position-based checksum
4. **Bank State Branch (BSB)**: 6-digit bank routing codes with state validation
5. **Australian Company Number (ACN)**: 9-digit company identifier with checksum
6. **Phone Numbers**: Australian mobile and landline numbers
7. **Email Addresses**: Standard email format validation
8. **Person Names**: Context-aware detection excluding code constructs
9. **Australian Addresses**: Street addresses with postcode validation
10. **Driver Licenses**: State-specific format validation
11. **Passport Numbers**: Australian passport pattern (letter + 7 digits)
12. **Credit Cards**: All major card types with Luhn algorithm validation

### How Pattern Matching Works

Pattern matching uses a three-layer approach:

1. **Initial Pattern Detection**
   ```
   TFN Pattern: \b\d{3}[\s\-]?\d{3}[\s\-]?\d{3}\b
   ```
   This finds any 9-digit sequence that could be a TFN

2. **Format Normalization**
   - Removes spaces and hyphens
   - Validates length and character types

3. **Checksum Validation**
   - Each PI type has specific validation rules
   - For TFN: Multiply each digit by weight [1,4,3,7,5,8,6,9,10], sum must be divisible by 11

### Confidence Scoring Explained

The scanner assigns confidence scores (0.0-1.0) based on multiple factors:

- **Base Score**: Pattern match (0.5) + valid checksum (0.3)
- **Context Bonus**: Label proximity (+0.2), form field (+0.1)
- **Context Penalty**: Test file (-0.4), documentation (-0.3)
- **Co-occurrence**: Multiple PI types nearby (+0.1)

Example: A TFN with valid checksum in a variable named "customerTFN" in production code would score 0.9 (High confidence)

### Context-Aware Detection

Context analysis examines surrounding code to determine if PI is real or test data:

**High-Risk Contexts**:
- Database queries: `INSERT INTO users (tfn) VALUES (?)`
- API calls: `customer.setTaxFileNumber(tfn)`
- Configuration files: `default_tfn: "123456789"`

**Low-Risk Contexts**:
- Test files: `describe("TFN validation", () => {`
- Documentation: `// Example TFN: 123-456-789`
- Mock data: `const TEST_TFN = "123456789"`

### Banking Domain Intelligence with AST Analysis

The scanner includes specialized intelligence for financial services codebases:

**AST-Based Code Analysis**:
- Uses Tree-sitter for language-agnostic parsing
- Supports Java, Scala, and Python (optimized for Spark pipelines)
- Understands code structure beyond simple pattern matching
- Identifies function calls, variable assignments, and data flows

**Banking-Specific Risk Zones**:
1. **Customer Data Processing**:
   - Functions/classes with names like `Customer`, `Account`, `Profile`
   - Database models and ORM entities
   - API endpoints handling user data

2. **Transaction Handling**:
   - Payment processing code
   - Transaction validation logic
   - Financial calculation modules

3. **Integration Points**:
   - External API calls
   - Data export/import functions
   - Third-party service integrations

4. **Data Pipelines**:
   - Spark/ETL job definitions
   - Data transformation functions
   - Batch processing scripts

**Risk Scoring Enhancement**:
- Code in high-risk zones receives elevated base scores
- Test files are marked but still scanned (never skipped)
- Production paths (`src/main/`) weighted higher than test paths
- API routes and database queries flagged for extra scrutiny

### LLM Validation Feature (Phase 2 - Local-Only Design)

The scanner's Phase 2 includes optional LLM validation designed exclusively for **local deployment** to maintain security:

1. **Integration in Two-Phase Architecture**
   - Phase 1 performs initial pattern detection
   - Interactive decision point allows validation scope selection
   - Phase 2 applies LLM analysis to selected findings only
   - Significantly reduces processing time by filtering first

2. **Local LLM Requirement**
   - Uses LM Studio or similar local LLM servers
   - Default endpoint: `http://localhost:1234/v1`
   - No cloud LLM providers supported by design
   - Prevents PI from leaving your infrastructure

3. **How LLM Validation Works**
   - **Smart Selection**: User chooses which risk levels to validate
   - **Context Extraction**: ±50 lines of surrounding code
   - **Intelligent Analysis**: Determines if code handles real or test data
   - **Risk Refinement**: Updates confidence scores based on context
   - **Detailed Explanation**: Documents reasoning for each decision

4. **Interactive Validation Options**
   - **All Findings**: Comprehensive but time-intensive
   - **HIGH + MEDIUM**: Balanced approach for most use cases
   - **HIGH + CRITICAL Only**: Quick validation of highest risks
   - **Skip**: Use Phase 1 results only (pattern matching)

5. **Security Safeguards**
   - Configuration validates localhost-only endpoints
   - Warning system for non-local configurations
   - All PI processing remains on local infrastructure
   - Audit logging of all LLM interactions

6. **Performance Optimization**
   - Time estimates based on finding count
   - Progress tracking during validation
   - Concurrent processing with configurable limits
   - Smart filtering to reduce validation workload

**Important**: Never configure the LLM endpoint to point to external services, as this would transmit detected PI outside your control.

### Resource Lifecycle Management

The scanner implements robust resource management:

**ScanResourceManager Architecture**:
- Centralized lifecycle control for all scan resources
- Context-based cancellation and timeout handling
- Automatic cleanup on errors or completion
- Prevents resource leaks between phases

**Key Features**:
1. **Repository Persistence**: Files remain available throughout both phases
2. **Memory Efficiency**: Streaming for large files, cleanup of processed data
3. **Error Recovery**: Graceful handling of failures with guaranteed cleanup
4. **Context Propagation**: Cancellation signals flow through all operations

## Security and Privacy by Design

### How Sensitive Data is Handled

The scanner implements multiple layers of protection:

1. **Memory Management**
   - PI is processed in memory without persistence
   - Automatic cleanup of temporary files
   - No caching of sensitive data

2. **Access Control**
   - Requires explicit GitHub authentication
   - Respects repository permissions
   - No elevation of privileges

3. **Data Minimization**
   - Only necessary context is extracted
   - Large files are streamed, not loaded entirely
   - Results can be configured to exclude actual PI values

4. **Dual Data Handling**
   - Raw findings stored in memory for LLM validation
   - Masked findings written to reports
   - Clear separation between processing and output data
   - No raw PI data persisted to disk unless explicitly configured

### Why the Scanner Itself Doesn't Create Disclosure Risks

The scanner is designed to identify risks without creating new ones:

- **Local Processing**: All scanning happens on the user's infrastructure
- **No External Transmission**: PI is never sent to external services
- **Local-Only LLM**: LLM validation uses only local servers by design
- **Secure Deletion**: Temporary files are removed after scanning
- **Encrypted Output**: Results can be encrypted at rest
- **Mandatory Redaction**: Default configuration masks PI in outputs

### Output Redaction and Reporting

The scanner supports multiple levels of output redaction:

- **Full Redaction**: Only PI type and location, no actual values
- **Partial Masking**: Shows partial PI for verification (e.g., `XXX-XXX-789`)
- **Full Disclosure**: Complete PI values (only for secure environments)

## Usage and Integration

### Typical Use Cases

1. **Pre-Commit Scanning**: Developers scan before pushing code
2. **CI/CD Integration**: Automated scanning in build pipelines
3. **Periodic Audits**: Scheduled scans of entire organizations
4. **Incident Response**: Rapid scanning after suspected exposure
5. **Compliance Reporting**: Regular reports for audit requirements

### How it Fits into CI/CD Pipelines

The scanner provides a single command interface optimized for automation:

```yaml
# Example GitHub Actions integration
- name: Scan for Australian PI
  run: |
    # Pattern scan only (fastest)
    pi-scanner ${{ github.event.repository.html_url }} \
      --no-input \
      --masking=full

    # Or with automatic high-risk validation
    pi-scanner ${{ github.event.repository.html_url }} \
      --no-input \
      --validate=high \
      --masking=full
```

```bash
# GitLab CI example
pi-scan:
  script:
    - pi-scanner $CI_PROJECT_URL --no-input --validate=high-medium
  artifacts:
    paths:
      - reports/
```

Integration options:
- **Non-Interactive Mode**: `--no-input` flag enables CI/CD usage
- **Validation Control**: `--validate` parameter sets automatic validation scope
- **Exit Codes**: Non-zero on critical/high findings (configurable)
- **Report Artifacts**: JSON outputs in `./reports/` directory

### Guided Scanning Workflow

The scanner provides an intuitive, guided experience that helps users make informed decisions:

#### Interactive Mode (Default)

1. **Welcome & Authentication**
   ```
   🔍 GitHub PI Scanner
   Welcome! Let's scan your repository for Australian Personal Information.
   ✓ GitHub authentication verified
   ```

2. **Phase 1: Pattern Scanning**
   ```
   📊 Phase 1: Pattern-based scanning
   Scanning 1,234 files...
   [████████████████████] 100% | 1234/1234 files | Time: 00:45
   ```

3. **Results Presentation**
   ```
   ✅ Pattern scan complete! Found 329 potential PI items:

   Risk Level    Count    Types Found
   ----------    -----    -----------
   CRITICAL      2        TFN (2)
   HIGH          3        Medicare (2), ABN (1)
   MEDIUM        23       Email (15), Phone (8)
   LOW           301      Names (250), Addresses (51)
   ```

4. **Interactive Decision Point**
   ```
   📊 Would you like to validate these findings with AI?
   This can significantly reduce false positives.

   1) Validate all findings (329 items) - Est. 10-15 minutes
   2) Validate HIGH + MEDIUM only (28 items) - Est. 1-2 minutes
   3) Validate HIGH + CRITICAL only (5 items) - Est. < 1 minute
   4) Skip validation

   Your choice:
   ```

5. **Phase 2: AI Validation (If Selected)**
   ```
   🤖 Phase 2: AI-powered validation
   Validating 28 findings with local LLM...
   [████████████████████] 100% | 28/28 items | Time: 01:23
   ```

6. **Final Results**
   ```
   ✅ Validation complete! Results refined:

   Before AI    After AI    Reduction
   ---------    --------    ---------
   28 items  →  12 items    57% fewer false positives

   Reports saved to: ./reports/20250628_140000_myrepo/
   ```

#### Non-Interactive Mode (CI/CD)

For automated pipelines, all decisions are made via command-line flags:

```bash
# Quick pattern scan only
pi-scanner https://github.com/org/repo --no-input

# Full scan with high-risk validation
pi-scanner https://github.com/org/repo --no-input --validate=high

# Maximum security with full validation
pi-scanner https://github.com/org/repo --no-input --validate=all --masking=full
```

### Report Structure and Outputs

The scanner generates comprehensive reports in a structured directory format:

```
./reports/
└── 20250628_140000_owner_repo/
    ├── phase1_pattern_scan.json      # Pattern detection results
    ├── phase2_llm_validated.json     # AI validation results (if performed)
    └── summary.txt                   # Human-readable summary
```

**Report Contents**:
- **phase1_pattern_scan.json**: All findings from pattern matching with initial risk scores
- **phase2_llm_validated.json**: Refined findings after LLM analysis with explanations
- **summary.txt**: Executive summary with statistics and key findings

### Reporting and Remediation Workflows

1. **Detection Phase**
   - Scanner identifies potential PI
   - Results are categorized by risk level
   - Reports generated automatically

2. **Review Phase**
   - Security team reviews high-risk findings
   - Developers validate context
   - LLM explanations guide decision-making

3. **Remediation Phase**
   - Remove or mask PI in code
   - Update tests to use synthetic data
   - Implement preventive controls

4. **Verification Phase**
   - Re-scan to confirm remediation
   - Compare before/after reports
   - Update compliance records

## Limitations and Boundaries

### What the Scanner CANNOT Do

- **Decrypt Encrypted Data**: Cannot detect PI in properly encrypted content
- **Analyze Compiled Code**: Only works with source code and text files
- **Prevent Future Commits**: Detection only, not prevention
- **Guarantee 100% Detection**: Some obfuscated PI may be missed
- **Replace Human Judgment**: Cannot understand business context fully

### What it Should NOT Be Used For

- **General Data Classification**: Designed specifically for Australian PI
- **Real-time Prevention**: Not suitable for commit hooks due to performance
- **Production Data Scanning**: Only for code repositories, not databases
- **International PI**: Patterns are Australian-specific

### Known Limitations

**File Types**:
- Limited to text-based files
- No support for Office documents or PDFs
- Cannot scan inside archives without extraction

**Languages**:
- Optimized for common programming languages
- May have reduced accuracy in domain-specific languages

**PI Types**:
- Some Australian PI types not yet supported
- International PI requires different patterns

### Performance Constraints

- **Repository Size**: Optimized for repos up to 10,000 files
- **File Size**: Files over 1MB are skipped by default (configurable)
- **Memory Usage**: ~500MB base + ~100MB per concurrent worker
- **Scan Time**:
  - Pattern scan: ~1-2 minutes per 1,000 files
  - LLM validation: ~2-3 seconds per finding
- **Concurrency**: Default 2x CPU cores, adjustable via config
- **Single Repository**: Processes one repository at a time (batch processing not supported)

## Current Implementation Status

### Completed Features ✅

**Core Functionality**
- Two-phase scanning architecture with interactive decision point
- Pattern-based detection for 12+ Australian PI types including credit cards
- Context-aware analysis with confidence scoring
- Checksum validation for structured identifiers (TFN, ABN, Medicare)
- Luhn algorithm validation for credit card numbers
- AST-based code analysis for Java, Scala, and Python
- Banking domain intelligence with automated risk zone detection
- File processing with concurrent worker pools
- Comprehensive report generation (JSON format)
- Resource lifecycle management with ScanResourceManager

**User Experience**
- Guided interactive workflow with progress tracking
- Real-time progress bars with time estimates
- Clear phase separation and decision points
- Non-interactive mode for CI/CD integration
- Configurable masking levels (none/partial/full)

**AI Integration**
- Local LLM integration via LM Studio
- Context-based validation to reduce false positives
- Detailed explanations for risk assessments
- Secure local-only processing
- Smart filtering to optimize validation performance
- Progress tracking with time estimates
- Concurrent validation with configurable rate limiting

**Security & Privacy**
- All processing performed locally
- Automatic cleanup of temporary files
- Default masking of PI in outputs
- Secure handling of GitHub credentials
- Separate storage of raw and masked findings
- In-memory processing of sensitive data
- No external API calls for PI data
- SkipTestFiles hardcoded to false for compliance

### In Active Development 🚧

- Performance optimizations for repositories >10,000 files
- Additional output formats (CSV, SARIF)
- Extended PI pattern support
- Enhanced reporting dashboards

### Not Currently Supported ❌

- Repository batch processing (removed in new architecture)
- GitHub Actions marketplace integration
- Cloud-based LLM providers (by design)
- Real-time commit prevention
- Scanning of compiled/binary files

## Future Considerations

### Potential Enhancements

1. **Extended PI Support**
   - International PI patterns
   - Custom pattern definitions
   - Industry-specific identifiers
   - Encrypted PI detection

2. **Advanced Detection**
   - Machine learning models for context
   - Behavioral analysis of code patterns
   - Cross-file correlation
   - Obfuscated PI detection

3. **Integration Expansion**
   - IDE plugins for real-time detection
   - Git hook support for prevention
   - Cloud repository scanning (with encryption)
   - SIEM integration

### Scalability Considerations

For enterprise deployment:
- **Distributed Architecture**: Scanner nodes for parallel processing
- **Results Database**: Centralized findings storage with encryption
- **API Gateway**: RESTful API for integrations
- **Monitoring**: Metrics and alerting infrastructure
- **High Availability**: Redundant scanner instances
- **Compliance Dashboard**: Real-time compliance status

The PI Scanner provides enterprise-grade Australian PI protection for code repositories. Its privacy-first design, banking domain intelligence, and local processing approach make it ideal for financial institutions and security-conscious organizations. The two-phase architecture with optional AI validation delivers both speed and accuracy, while the robust resource management ensures reliable operation at scale.

## Additional Resources

- **[Developer Guide](DEVELOPER_GUIDE.md)** - Setup and development instructions
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to the project
- **[Security Design](SECURITY_DESIGN.md)** - Detailed security architecture
- **Repository**: https://github.com/MacAttak/pi-scanner
