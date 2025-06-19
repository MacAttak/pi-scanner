# Developer Guide for GitHub PI Scanner

## Table of Contents
- [Prerequisites](#prerequisites)
- [Setting Up Development Environment](#setting-up-development-environment)
- [Building the Application](#building-the-application)
- [Running Tests](#running-tests)
- [ML/AI Components](#mlai-components)
- [Docker Development](#docker-development)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements
- Go 1.23+ (required for generics support)
- Git
- macOS, Linux, or Windows (with WSL recommended)

### Required Dependencies

#### macOS
```bash
# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install required dependencies
brew install go
brew install git
brew install libomp
brew install onnxruntime
```

#### Linux (Ubuntu/Debian)
```bash
# Update package list
sudo apt update

# Install Go (if not already installed)
sudo snap install go --classic

# Install Git
sudo apt install git

# Download ONNX Runtime
wget https://github.com/microsoft/onnxruntime/releases/download/v1.22.0/onnxruntime-linux-x64-1.22.0.tgz
tar -xvf onnxruntime-linux-x64-1.22.0.tgz
sudo cp onnxruntime-linux-x64-1.22.0/lib/* /usr/local/lib/
sudo ldconfig
```

#### Windows
1. Install Go from https://go.dev/dl/
2. Install Git from https://git-scm.com/download/win
3. Download ONNX Runtime from https://github.com/microsoft/onnxruntime/releases
4. Extract and add the DLL location to your PATH

## Setting Up Development Environment

### 1. Clone the Repository
```bash
git clone https://github.com/your-org/github-pi-scanner.git
cd github-pi-scanner
```

### 2. Install Go Dependencies
```bash
go mod download
go mod verify
```

### 3. Set Up Pre-commit Hooks
```bash
# Install pre-commit
pip install pre-commit

# Install the git hook scripts
pre-commit install
```

### 4. Configure GitHub Authentication
```bash
# Set your GitHub personal access token
export GITHUB_TOKEN="your-github-personal-access-token"

# Or use GitHub CLI
gh auth login
```

## Building the Application

### Standard Build
```bash
# Build the binary
go build -o pi-scanner ./cmd/pi-scanner

# Build with optimizations
go build -ldflags="-s -w" -o pi-scanner ./cmd/pi-scanner
```

### Cross-Platform Build
```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o pi-scanner-linux ./cmd/pi-scanner

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o pi-scanner.exe ./cmd/pi-scanner

# Build for macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o pi-scanner-darwin-amd64 ./cmd/pi-scanner

# Build for macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o pi-scanner-darwin-arm64 ./cmd/pi-scanner
```

## Running Tests

### Unit Tests
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Integration Tests
```bash
# Run integration tests (requires GitHub authentication)
go test -tags=integration ./test/integration/...
```

### End-to-End Tests
```bash
# Run E2E tests (requires network access and GitHub authentication)
go test ./test -run TestPIScannerE2E

# Run specific E2E test
go test ./test -run TestPIScannerE2E_AustralianRepositories
```

### Benchmarks
```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkPatternDetection ./pkg/detection/...
```

## ML/AI Components

### ML/AI Dependencies

#### ONNX Runtime Setup

The ML inference component uses ONNX Runtime for running DeBERTa models. The library paths are automatically detected, but you can also set them manually:

```bash
# macOS
export ONNX_RUNTIME_PATH="/opt/homebrew/lib/libonnxruntime.dylib"

# Linux
export ONNX_RUNTIME_PATH="/usr/local/lib/libonnxruntime.so"

# Windows
set ONNX_RUNTIME_PATH="C:\onnxruntime\lib\onnxruntime.dll"
```

#### HuggingFace Tokenizers

The ML component uses the `daulet/tokenizers` Go bindings for HuggingFace tokenizers:

1. **Build Requirements**:
   ```bash
   # The library requires libtokenizers.a
   # Option 1: Use prebuilt binaries from releases
   wget https://github.com/daulet/tokenizers/releases/download/v1.20.2/libtokenizers.a
   
   # Option 2: Build from source
   git clone https://github.com/daulet/tokenizers
   cd tokenizers
   make build
   ```

2. **Linking Configuration**:
   ```bash
   # Add to your environment
   export CGO_LDFLAGS="-L/path/to/libtokenizers/directory"
   
   # Or use with go run
   go run -ldflags="-extldflags '-L/path/to/libtokenizers/directory'" .
   ```

3. **Supported Models**:
   - All HuggingFace models with `tokenizer.json`
   - DeBERTa v3 models for PI validation
   - BERT, RoBERTa, and other transformer models

### Downloading ML Models

Models will be automatically downloaded on first use, but you can pre-download them:

```bash
# Download DeBERTa model for PI validation
./scripts/download-models.sh
```

### ML Development

To work on ML components:
1. Ensure ONNX Runtime is properly installed
2. Models are stored in `~/.pi-scanner/models/`
3. Use the mock runtime for unit tests: `go test ./pkg/ml/inference -short`

## Docker Development

### Building Docker Image
```bash
# Build the Docker image
docker build -t pi-scanner:latest .

# Build with specific ONNX Runtime version
docker build --build-arg ONNX_VERSION=1.22.0 -t pi-scanner:latest .
```

### Running with Docker
```bash
# Run scan with Docker
docker run --rm \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -v $(pwd)/output:/output \
  pi-scanner:latest scan --repo github/docs

# Interactive shell
docker run --rm -it \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  pi-scanner:latest /bin/bash
```

### Docker Compose for Development
```yaml
version: '3.8'
services:
  pi-scanner:
    build: .
    environment:
      - GITHUB_TOKEN=${GITHUB_TOKEN}
    volumes:
      - ./output:/output
      - ./config:/config
    command: scan --repo github/docs --output /output/results.json
```

## Troubleshooting

### ONNX Runtime Issues

#### Library Not Found
```
Error: failed to initialize ONNX runtime environment: Error loading ONNX shared library
```

**Solution:**
- macOS: `brew install onnxruntime`
- Linux: Download from GitHub releases and install to `/usr/local/lib`
- Windows: Add ONNX Runtime DLL to PATH

#### Version Mismatch
```
Error: ONNX Runtime version mismatch
```

**Solution:**
Ensure you have ONNX Runtime 1.22.0 installed to match the Go bindings.

### Build Issues

#### CGO Errors
```
# Ensure CGO is enabled
export CGO_ENABLED=1

# Set proper compiler flags for macOS
export CGO_CFLAGS="-I/opt/homebrew/include"
export CGO_LDFLAGS="-L/opt/homebrew/lib"
```

### GitHub API Rate Limits
```
Error: GitHub API rate limit exceeded
```

**Solution:**
- Use authenticated requests: `export GITHUB_TOKEN=your-token`
- Implement caching for repeated scans
- Use GitHub Enterprise with higher limits

### Memory Issues
For large repositories:
```bash
# Increase memory limit
GOGC=200 ./pi-scanner scan --repo large/repository

# Use worker pool size adjustment
./pi-scanner scan --repo large/repository --workers 2
```

## Development Workflow

### 1. Create Feature Branch
```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes
- Follow TDD approach
- Write tests first
- Implement features
- Ensure all tests pass

### 3. Run Quality Checks
```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Run tests
go test ./...

# Check coverage
go test -cover ./...
```

### 4. Commit Changes
```bash
git add .
git commit -m "feat: add new detection pattern for Australian driver licenses"
```

### 5. Push and Create PR
```bash
git push origin feature/your-feature-name
# Create PR on GitHub
```

## Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [ONNX Runtime Documentation](https://onnxruntime.ai/docs/)
- [GitHub API Documentation](https://docs.github.com/en/rest)
- [Australian PI Standards](https://www.oaic.gov.au/)

## Contributing

Please read [CONTRIBUTING.md](../CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.