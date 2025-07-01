# GitHub PI Scanner

[![CI Status](https://github.com/MacAttak/pi-scanner/workflows/Fixed%20CI%2FCD%20Pipeline/badge.svg)](https://github.com/MacAttak/pi-scanner/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/MacAttak/pi-scanner)](https://goreportcard.com/report/github.com/MacAttak/pi-scanner)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/doc/install)

A high-performance scanner for detecting Australian Personal Information (PI) in GitHub repositories, designed for enterprise compliance with Australian privacy regulations.

## Features

- **Australian PI Detection**: Specialized detection for TFN, ABN, Medicare numbers, BSB codes, ACN, driver licenses, passports, and credit cards
- **Banking Domain Intelligence**: AST-based analysis for Java, Scala, and Python with banking-specific risk assessment
- **Two-Phase Architecture**: Pattern detection followed by optional AI-powered validation for 100% accuracy
- **Local LLM Integration**: Code-aware validation using LM Studio for superior false positive reduction
- **Repository Structure Analysis**: Intelligent risk zone mapping based on file paths and code patterns
- **Smart Progress Tracking**: Real-time progress indicators with accurate time estimates
- **Secure Output**: Configurable masking levels to protect sensitive data in reports
- **Enterprise Ready**: Non-interactive mode for CI/CD integration with comprehensive reporting

## Prerequisites

- Go 1.21+ (for building from source)
- GitHub token with repository read access
- (Optional) LM Studio for AI-powered validation

## Quick Start

### Installation

#### Option 1: Docker (Recommended)

```bash
# Pull the latest image
docker pull ghcr.io/macattak/pi-scanner:latest

# Run with GitHub token
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN \
  ghcr.io/macattak/pi-scanner:latest https://github.com/example/repo

# Run with local output directory
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v $(pwd)/output:/home/scanner/output \
  ghcr.io/macattak/pi-scanner:latest https://github.com/example/repo
```

#### Option 2: Download Binary

Download the latest release from the [releases page](https://github.com/MacAttak/pi-scanner/releases).

```bash
# macOS/Linux
curl -LO https://github.com/MacAttak/pi-scanner/releases/download/v1.2.0/pi-scanner-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m).tar.gz
tar -xzf pi-scanner-*.tar.gz
chmod +x pi-scanner
sudo mv pi-scanner /usr/local/bin/
```

#### Option 3: Build from Source

```bash
# Clone the repository
git clone https://github.com/MacAttak/pi-scanner.git
cd pi-scanner

# Build the binary
go build -o bin/pi-scanner ./cmd/pi-scanner

# Or use Make
make build
```

### Basic Usage

The scanner provides a guided experience through two phases:

1. **Pattern-based scanning** - Fast detection using regex patterns
2. **AI validation** (optional) - Reduce false positives using LLM

```bash
# Interactive guided scan
pi-scanner https://github.com/example/repo

# The scanner will:
# 1. Clone and scan the repository for PI patterns
# 2. Save a masked report to ./reports/
# 3. Show you a summary of findings
# 4. Ask if you want to validate findings with AI
```

### Non-Interactive Mode

For automation and CI/CD pipelines:

```bash
# Pattern scan only (no AI validation)
pi-scanner https://github.com/example/repo --no-input

# Automatic high-risk validation
pi-scanner https://github.com/example/repo --no-input --validate=high

# Validate all findings
pi-scanner https://github.com/example/repo --no-input --validate=all
```

### Masking Levels

Control how PI data appears in reports:

```bash
# Partial masking (default) - Shows partial values like 123****82
pi-scanner https://github.com/example/repo --masking=partial

# Full masking - Complete redaction
pi-scanner https://github.com/example/repo --masking=full

# No masking - Shows full values (use with caution!)
pi-scanner https://github.com/example/repo --masking=none
```

## AI-Powered Validation

The scanner can use a local LLM to validate findings and reduce false positives:

### Setup LM Studio

1. Download and install [LM Studio](https://lmstudio.ai/)
2. Download a recommended model (e.g., `qwen2.5-coder-7b-instruct`)
3. Start the local server (usually on port 1234)

### Check LLM Availability

```bash
# Test if LLM service is available
pi-scanner llm-check
```

### Validation Options

During interactive scanning, you'll be presented with validation options:

```
📊 Would you like to validate these findings with AI?
This can significantly reduce false positives.

1) Validate all findings (329 items) - Est. 10-15 minutes
2) Validate HIGH + MEDIUM only (28 items) - Est. 1-2 minutes
3) Validate HIGH + CRITICAL only (5 items) - Est. < 1 minute
4) Skip validation
```

## Reports

All scan results are saved to the `./reports/` directory with the following structure:

```
reports/
└── 20250628_140000_owner_repo/
    ├── phase1_pattern_scan.json      # Pattern scan results
    ├── phase2_llm_validated.json     # AI validation results (if performed)
    └── summary.txt                   # Human-readable summary
```

## Docker Usage

The PI Scanner is available as a Docker image from GitHub Container Registry.

### Basic Docker Commands

```bash
# Pull specific version
docker pull ghcr.io/macattak/pi-scanner:1.2.0

# Run scan with GitHub token
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN \
  ghcr.io/macattak/pi-scanner:latest https://github.com/example/repo

# Save reports to local directory
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v $(pwd)/reports:/home/scanner/output \
  ghcr.io/macattak/pi-scanner:latest https://github.com/example/repo

# Run with custom config
docker run --rm -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v $(pwd)/config.yaml:/etc/pi-scanner/config/config.yaml:ro \
  ghcr.io/macattak/pi-scanner:latest https://github.com/example/repo
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  pi-scanner:
    image: ghcr.io/macattak/pi-scanner:latest
    environment:
      - GITHUB_TOKEN=${GITHUB_TOKEN}
    volumes:
      - ./reports:/home/scanner/output
      - ./config.yaml:/etc/pi-scanner/config/config.yaml:ro
    command: https://github.com/example/repo --no-input --validate=high
```

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: PI Security Scan
  run: |
    pi-scanner ${{ github.event.repository.html_url }} \
      --no-input \
      --validate=high \
      --masking=full
```

### Using Docker in CI

```yaml
- name: PI Security Scan (Docker)
  run: |
    docker run --rm \
      -e GITHUB_TOKEN=${{ secrets.GITHUB_TOKEN }} \
      -v ${{ github.workspace }}/reports:/home/scanner/output \
      ghcr.io/macattak/pi-scanner:latest \
      ${{ github.event.repository.html_url }} \
      --no-input --validate=high --masking=full
```

## Environment Variables

- `GITHUB_TOKEN` - Required for accessing private repositories
- `NO_COLOR` - Disable colored output
- `CI` - Automatically enables non-interactive mode

## Advanced Usage

### Verbose Output

```bash
# Show detailed progress and debugging information
pi-scanner https://github.com/example/repo --verbose
```

### Custom LLM Configuration

```bash
# Use a different LLM endpoint
pi-scanner llm-check --endpoint http://localhost:8080/v1 --model codellama-7b
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.
