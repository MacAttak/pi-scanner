# GitHub PI Scanner - Design and Implementation Overview

## Executive Summary

### What is the PI Scanner?

The GitHub PI Scanner is an automated security tool designed to detect Australian Personally Identifiable Information (PI) within GitHub repositories. It employs pattern recognition, contextual analysis, and optional AI-powered validation to identify sensitive data that may have been inadvertently committed to source code repositories.

### Why was it built?

Australian organizations face strict compliance requirements under the Privacy Act 1988 and the Notifiable Data Breaches (NDB) scheme. With modern software development practices involving numerous repositories and frequent commits, manual review of code for PI exposure is no longer feasible. The PI Scanner addresses this challenge by providing automated, scalable detection of Australian PI across entire GitHub organizations.

### Key Capabilities

- **Comprehensive PI Detection**: Identifies 11+ types of Australian PI including Tax File Numbers (TFN), Medicare numbers, and Australian Business Numbers (ABN)
- **Context-Aware Analysis**: Reduces false positives by understanding code context (test data vs. production)
- **LLM-Enhanced Validation**: Optional AI validation for high-confidence results
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

The PI Scanner follows a multi-stage pipeline architecture:

```
Repository Discovery → File Processing → PI Detection → Validation → Scoring → Reporting
        ↓                    ↓               ↓              ↓           ↓           ↓
   GitHub API          Concurrency      Pattern Match    Context    Risk Level   Multiple
   Integration          Control          + Checksum      Analysis   Assignment   Formats
```

### How the Scanner Works

1. **Repository Access**
   - Connects to GitHub using personal access tokens or GitHub Apps
   - Clones repositories to temporary directories with automatic cleanup
   - Supports both public and private repositories

2. **File Discovery and Filtering**
   - Traverses repository structure efficiently
   - Filters by file size (default: <1MB) and type
   - Skips binary files and known non-risk directories

3. **Concurrent Processing**
   - Worker pool architecture for parallel file scanning
   - Configurable concurrency (default: CPU cores * 2)
   - Memory-efficient streaming for large files

4. **PI Detection Pipeline**
   - Pattern matching using optimized regular expressions
   - Checksum validation for structured PI (TFN, ABN, Medicare)
   - Context extraction (±50 lines around matches)

5. **Validation and Scoring**
   - Proximity analysis determines context (test vs. production)
   - Multi-factor confidence scoring
   - Optional LLM validation for ambiguous cases

6. **Result Aggregation**
   - Deduplication of findings
   - Risk level assignment (Critical/High/Medium/Low)
   - Statistical analysis for reporting

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

### LLM Validation Feature (Local-Only Design)

The scanner includes an optional LLM validation feature designed exclusively for **local deployment** to maintain security:

1. **Local LLM Requirement**
   - Uses LM Studio or similar local LLM servers
   - Default endpoint: `http://localhost:1234/v1`
   - No cloud LLM providers supported by design
   - Prevents PI from leaving your infrastructure

2. **How LLM Validation Works**
   - **Context Extraction**: ±50 lines of surrounding code
   - **Intelligent Analysis**: Determines if code handles real or test data
   - **Risk Refinement**: May downgrade findings in obvious test scenarios
   - **Detailed Explanation**: Documents reasoning for risk level assignment

3. **Security Safeguards**
   - Configuration validates localhost-only endpoints
   - Warning system for non-local configurations
   - Option to disable PI transmission entirely
   - Audit logging of all LLM interactions

4. **When to Use LLM Validation**
   - High false positive rates in test-heavy codebases
   - Need for detailed explanations in reports
   - Regulatory requirements for documented risk assessments
   - When local compute resources are available

**Important**: Never configure the LLM endpoint to point to external services, as this would transmit detected PI outside your control.

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

```yaml
# Example GitHub Actions integration
- name: Scan for Australian PI
  uses: pi-scanner/action@v1
  with:
    fail-on: critical
    output-format: sarif
```

Integration options:
- **GitHub Actions**: Native workflow integration
- **GitLab CI**: Docker-based scanning jobs
- **Jenkins**: Plugin or shell execution
- **Bitbucket Pipelines**: Container-based scanning

### Reporting and Remediation Workflows

1. **Detection Phase**
   - Scanner identifies potential PI
   - Results are categorized by risk level

2. **Review Phase**
   - Security team reviews high-risk findings
   - Developers validate context

3. **Remediation Phase**
   - Remove or mask PI in code
   - Update tests to use synthetic data
   - Implement preventive controls

4. **Verification Phase**
   - Re-scan to confirm remediation
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

- **Repository Size**: Performance degrades beyond 10,000 files
- **File Size**: Files over 10MB are skipped by default
- **Memory Usage**: Requires ~2GB RAM per concurrent worker
- **Scan Time**: Large repos may take 30+ minutes

## Current Implementation Status

### Completed Features
- Pattern-based detection for major Australian PI types
- Context-aware analysis to reduce false positives
- Local LLM integration for enhanced validation
- Multiple output formats (JSON, CSV, HTML)
- Docker-based deployment

### In Development
- Complete file processing implementation
- Report generation pipeline
- Repository batch processing
- Enhanced security controls
- Performance optimizations

### Planned Enhancements
- Additional PI pattern support
- Streaming architecture for large files
- Plugin system for custom validators
- Enterprise monitoring integration
- Compliance profile templates

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

The PI Scanner provides a foundation for Australian PI protection in code repositories. While currently in active development, its privacy-first design and local processing approach make it suitable for security-conscious organizations. With the planned improvements, it will evolve into a comprehensive compliance tool that balances security, performance, and usability.

## Additional Resources

- **[Developer Guide](DEVELOPER_GUIDE.md)** - Setup and development instructions
- **[Contributing Guide](CONTRIBUTING.md)** - How to contribute to the project
- **[Security Design](SECURITY_DESIGN.md)** - Detailed security architecture
- **Repository**: https://github.com/MacAttak/pi-scanner
