# GitHub PI Scanner

A high-performance scanner for detecting Australian Personal Information (PI) in GitHub repositories, designed for enterprise compliance with Australian privacy regulations.

## Features

- **Australian PI Detection**: Specialized detection for TFN, ABN, Medicare numbers, BSB codes, ACN, and driver licenses
- **Context-Aware Detection**: Pattern matching with intelligent context validation and confidence scoring
- **LLM-Enhanced Validation**: Optional AI-powered validation using local LLMs to reduce false positives
- **High Performance**: Concurrent processing with worker pools
- **Enterprise Ready**: Batch processing, comprehensive reporting, and CI/CD integration
- **Compliance Focused**: Designed for Australian Privacy Act and Notifiable Data Breach compliance

## Prerequisites

- Go 1.21+ (for building from source)
- Docker (for containerized development and deployment)
- GitHub token with repository read access

## Quick Start

### Using Docker (Recommended)

```bash
# Set your GitHub token
export GITHUB_TOKEN="your-github-token"

# Run with Docker
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN \
  ghcr.io/macattak/pi-scanner:latest \
  scan --repo octocat/Hello-World
```

### Installation

#### Building from Source

```bash
# Clone the repository
git clone https://github.com/MacAttak/pi-scanner.git
cd pi-scanner

# Setup development environment
make setup

# Build the binary
make build

# Or use Go directly
go build -o bin/pi-scanner ./cmd/pi-scanner

# Install to system PATH
make install
```

## Usage

### Basic Scan

```bash
# Scan a single repository
pi-scanner scan --repo github/docs --output results.json

# Scan with verbose output
pi-scanner scan --repo github/docs --output results.json --verbose

# Scan with LLM validation (requires LM Studio)
pi-scanner scan --repo github/docs --enable-llm --output results.json
```

### Batch Scanning

```bash
# Scan multiple repositories from a file
pi-scanner scan --repo-list repos.txt --output batch-results.json

# Example repos.txt:
# govau/design-system-components
# qld-gov-au/qgds-qol-mvp
# TerriaJS/nationalmap
```

### Configuration

```bash
# Use custom configuration
pi-scanner scan --repo github/docs --config custom-config.yaml

# Generate default configuration
pi-scanner config generate > config.yaml
```

### Reporting

```bash
# Generate HTML report
pi-scanner report --input results.json --format html --output report.html

# Generate CSV report
pi-scanner report --input results.json --format csv --output findings.csv

# Generate SARIF for CI/CD integration
pi-scanner report --input results.json --format sarif --output results.sarif
```

## Configuration Options

Create a `config.yaml` file:

```yaml
# Detection settings
detection:
  patterns:
    enabled: true
    confidence_threshold: 0.8
  gitleaks:
    enabled: true
    config_path: "gitleaks.toml"
  context:
    enabled: true
    proximity_distance: 10

# Performance settings
performance:
  workers: 8
  file_queue_size: 10000
  max_file_size: 10485760  # 10MB

# Repository settings
repository:
  clone_depth: 1
  timeout: 300s

# Risk scoring
risk:
  high_threshold: 0.9
  medium_threshold: 0.7

# LLM validation (optional)
llm:
  enabled: false
  provider: "lmstudio"
  endpoint: "http://localhost:1234/v1"
  model: "qwen2.5-coder-7b-instruct"
  validate_risks:
    - HIGH
    - MEDIUM
```

## Output Format

The scanner produces detailed JSON output:

```json
{
  "repository": {
    "url": "github/docs",
    "file_count": 1234,
    "size": 45678900
  },
  "scan_duration": "45.2s",
  "findings": [
    {
      "type": "TFN",
      "file": "docs/example.md",
      "line": 42,
      "confidence": 0.95,
      "risk_level": "HIGH",
      "context": "Example TFN: [REDACTED]"
    }
  ],
  "statistics": {
    "total_files": 1234,
    "scanned_files": 1200,
    "findings_by_type": {
      "TFN": 5,
      "ABN": 12,
      "MEDICARE": 3
    }
  }
}
```

## LLM-Enhanced Detection

The scanner supports optional LLM-based validation to reduce false positives by analyzing the context around detected patterns.

### Setup

1. **Install LM Studio**: Download from [lmstudio.ai](https://lmstudio.ai/)
2. **Download a Model**: Recommended models:
   - Qwen2.5-Coder-7B-Instruct (best for code analysis)
   - DeepSeek-Coder-6.7B
   - Llama-3.2-3B-Instruct (lighter option)
3. **Start the Server**: In LM Studio, go to the Developer tab and start the local server

### Usage

```bash
# Enable LLM validation
pi-scanner scan --repo github/docs --enable-llm

# Use a different model
pi-scanner scan --repo github/docs --enable-llm --llm-model "deepseek-coder-6.7b"

# Custom endpoint
pi-scanner scan --repo github/docs --enable-llm --llm-endpoint "http://192.168.1.100:1234/v1"
```

### Benefits

- **Reduced False Positives**: LLM analyzes context to identify test data vs. real PI
- **Risk Refinement**: Adjusts risk levels based on code context
- **Clear Explanations**: Provides reasoning for each validation decision

## CI/CD Integration

### GitHub Actions

```yaml
name: PI Scan
on: [push, pull_request]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run PI Scanner
        uses: your-org/pi-scanner-action@v1
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          output-format: sarif

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: pi-scan-results.sarif
```

### GitLab CI

```yaml
pi-scan:
  image: ghcr.io/your-org/pi-scanner:latest
  script:
    - pi-scanner scan --repo $CI_PROJECT_PATH --output results.json
  artifacts:
    reports:
      sast: results.sarif
```

## Performance

- Processes ~1,300 files/second on modern hardware
- Concurrent detection pipeline with configurable workers
- Memory efficient streaming for large repositories
- Automatic binary file detection and skipping

## Supported PI Types

| Type | Description | Example Pattern |
|------|-------------|-----------------|
| TFN | Tax File Number | XXX XXX XXX |
| ABN | Australian Business Number | XX XXX XXX XXX |
| Medicare | Medicare Card Number | XXXX XXXXX X |
| BSB | Bank State Branch | XXX-XXX |
| ACN | Australian Company Number | XXX XXX XXX |
| ARBN | Australian Registered Body Number | XXX XXX XXX |
| Passport | Australian Passport Number | X1234567, PA1234567 |
| Bank Account | Bank Account Number | 6-10 digits |
| Driver License | State-based licenses | Various formats |
| Phone | Australian Phone Numbers | 04XX XXX XXX |
| Email | Email Addresses | user@example.com |

## Development

For development setup and guidelines:
- [DEVELOPMENT.md](DEVELOPMENT.md) - Docker-based development environment
- [Developer Guide](docs/DEVELOPER_GUIDE.md) - Architecture and API reference
- [Contributing](CONTRIBUTING.md) - Contribution guidelines

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE) for details.

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/MacAttak/pi-scanner/issues)
- **Security**: Please report security vulnerabilities via GitHub Security tab

## Acknowledgments

- Gitleaks by Zachary Rice
- Australian Government for PI validation algorithms
