# PI Scanner v1.0.0

Release Date: 2025-06-19 05:34:41
Git Commit: 7b8aaa9

## Release Assets

### Cross-Platform Builds
- `pi-scanner-darwin-amd64.tar.gz` - macOS Intel
- `pi-scanner-linux-amd64.tar.gz` - Linux x64
- `pi-scanner-linux-arm64.tar.gz` - Linux ARM64
- `pi-scanner-windows-amd64.zip` - Windows x64

## Installation

```bash
# Linux/macOS
tar -xzf pi-scanner-<platform>.tar.gz
./pi-scanner version

# Windows
unzip pi-scanner-windows-amd64.zip
pi-scanner.exe version
```

## Features
- Australian PI detection (TFN, ABN, Medicare, BSB)
- Multi-stage validation pipeline with context analysis
- Risk scoring with regulatory compliance
- Multiple report formats (HTML, CSV, SARIF, JSON)
