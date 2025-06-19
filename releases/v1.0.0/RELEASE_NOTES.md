# PI Scanner v1.0.0

Release Date: 2025-06-19 05:34:41
Git Commit: 7b8aaa9

## Release Assets

### With ML Support (Recommended)
- `pi-scanner-darwin-arm64-ml.tar.gz` - Native build with ML support

### Cross-Platform Builds (No ML)
- `pi-scanner-darwin-amd64.tar.gz` - macOS Intel
- `pi-scanner-linux-amd64.tar.gz` - Linux x64
- `pi-scanner-linux-arm64.tar.gz` - Linux ARM64
- `pi-scanner-windows-amd64.zip` - Windows x64

## Installation

### Native Build (with ML)
```bash
tar -xzf pi-scanner-darwin-arm64-ml.tar.gz
cd pi-scanner-darwin-arm64-ml
./pi-scanner version
```

### Cross-Platform Build
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
- Multi-stage validation pipeline
- Risk scoring with regulatory compliance
- Multiple report formats (HTML, CSV, SARIF, JSON)

## Notes
- ML features require the native build with included libraries
- Cross-platform builds have pattern matching only (no ML validation)
