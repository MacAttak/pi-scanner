# PI Scanner Distribution Strategy

## Overview

This document outlines the distribution strategy for the PI Scanner artifacts, including binaries, Docker images, and documentation.

## Release Artifacts

### 1. Binary Releases

#### Cross-Platform Builds
- **Targets**:
  - macOS Intel: `pi-scanner-darwin-amd64.tar.gz` (~3.8 MB)
  - Linux x64: `pi-scanner-linux-amd64.tar.gz` (~3.7 MB)
  - Linux ARM64: `pi-scanner-linux-arm64.tar.gz` (~3.4 MB)
  - Windows x64: `pi-scanner-windows-amd64.zip` (~3.9 MB)
- **Features**: Pure Go builds with full context validation capabilities
- **Note**: Built with CGO_ENABLED=0 for maximum portability

### 2. Docker Images

#### Multi-Architecture Image
```dockerfile
# Supports linux/amd64 and linux/arm64
docker pull ghcr.io/MacAttak/pi-scanner:latest
docker pull ghcr.io/MacAttak/pi-scanner:1.0.0
```

#### Features:
- Built on Alpine Linux for minimal size
- Pure Go implementation with no external dependencies
- Non-root user execution for security
- Volume mounts for reports and configuration

### 3. Source Distribution

#### GitHub Release
- Source code archive (automatically created by GitHub)
- Includes all source files, tests, and documentation
- Ready to build with standard Go toolchain

## Distribution Channels

### 1. GitHub Releases (Primary)

**URL**: https://github.com/MacAttak/pi-scanner/releases

**Process**:
1. Create GitHub Release with tag `v1.0.0`
2. Upload all binary artifacts from `releases/v1.0.0/`
3. Include `RELEASE_NOTES.md` as release description
4. Mark as "Latest Release"
5. Include checksums.txt for verification

**Automation**:
```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            releases/v${{ github.ref_name }}/*.tar.gz
            releases/v${{ github.ref_name }}/*.zip
            releases/v${{ github.ref_name }}/checksums.txt
          body_path: releases/v${{ github.ref_name }}/RELEASE_NOTES.md
```

### 2. GitHub Container Registry (Docker)

**URL**: ghcr.io/MacAttak/pi-scanner

**Process**:
1. Build multi-architecture images
2. Tag with version and latest
3. Push to GHCR with proper labels

**Commands**:
```bash
# Build and push multi-arch image
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag ghcr.io/MacAttak/pi-scanner:1.0.0 \
  --tag ghcr.io/MacAttak/pi-scanner:latest \
  --push .
```

### 3. Homebrew (Future)

**Formula**: pi-scanner.rb
```ruby
class PiScanner < Formula
  desc "GitHub PI Scanner for Australian regulatory compliance"
  homepage "https://github.com/MacAttak/pi-scanner"
  version "1.0.0"
  
  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/MacAttak/pi-scanner/releases/download/v1.0.0/pi-scanner-darwin-arm64-ml.tar.gz"
    sha256 "75a85aa628b4a17492e5967ee707f07675b5f24f26df0804c8318cc5637cdc49"
  elsif OS.mac?
    url "https://github.com/MacAttak/pi-scanner/releases/download/v1.0.0/pi-scanner-darwin-amd64.tar.gz"
    sha256 "c6128d91b593ab5bc6df42bddaced8bbbfcfd2453d492235f3539a4f79b7ee80"
  elsif OS.linux?
    url "https://github.com/MacAttak/pi-scanner/releases/download/v1.0.0/pi-scanner-linux-amd64.tar.gz"
    sha256 "a3a10ae7002fe6c8469fd368de0dda9d2103451c8a83201525f8a53bd6c0257c"
  end
  
  def install
    bin.install "pi-scanner"
  end
end
```

### 4. Package Managers (Future)

#### APT/DEB (Debian/Ubuntu)
- Create .deb package with proper dependencies
- Host on GitHub Releases or dedicated APT repository

#### RPM (RedHat/Fedora)
- Create .rpm package
- Submit to EPEL or host on GitHub

## Installation Instructions

### Binary Installation

#### macOS/Linux (with ML)
```bash
# Download and extract
curl -L https://github.com/MacAttak/pi-scanner/releases/download/v1.0.0/pi-scanner-$(uname -s)-$(uname -m)-ml.tar.gz | tar xz
cd pi-scanner-*-ml

# Install
sudo cp pi-scanner /usr/local/bin/
sudo cp libtokenizers.a /usr/local/lib/
sudo cp default_config.yaml /etc/pi-scanner/

# Verify
pi-scanner version
```

#### Cross-Platform (no ML)
```bash
# Download appropriate binary
curl -L https://github.com/MacAttak/pi-scanner/releases/download/v1.0.0/pi-scanner-$(uname -s)-$(uname -m).tar.gz | tar xz

# Install
sudo mv pi-scanner /usr/local/bin/
pi-scanner version
```

### Docker Installation
```bash
# Pull image
docker pull ghcr.io/MacAttak/pi-scanner:1.0.0

# Run scan
docker run --rm \
  -v $(pwd)/reports:/app/reports \
  ghcr.io/MacAttak/pi-scanner:1.0.0 \
  scan --repo https://github.com/org/repo
```

## Security Considerations

### Signing Artifacts
1. GPG sign all release artifacts
2. Include signature files (.asc) with releases
3. Publish public key on keyserver and GitHub

### Verification
```bash
# Import public key
gpg --keyserver keys.openpgp.org --recv-keys <KEY_ID>

# Verify signature
gpg --verify pi-scanner-darwin-arm64-ml.tar.gz.asc

# Verify checksum
shasum -a 256 -c checksums.txt
```

### Container Security
1. Sign container images with cosign
2. Generate SBOM with syft
3. Scan for vulnerabilities with grype

## Release Process

### 1. Pre-Release
- [ ] Run security audit: `make security-audit`
- [ ] Update version in scripts/build-release.sh
- [ ] Update CHANGELOG.md
- [ ] Create release branch: `git checkout -b release/v1.0.0`

### 2. Build Artifacts
```bash
# Clean build
rm -rf releases/
VERSION=1.0.0 ./scripts/build-release.sh

# Build Docker images
docker buildx build --platform linux/amd64,linux/arm64 -t pi-scanner:1.0.0 .
```

### 3. Test Artifacts
- [ ] Test native binary with ML
- [ ] Test cross-platform binaries
- [ ] Test Docker image on different platforms
- [ ] Verify checksums

### 4. Create Release
- [ ] Tag release: `git tag -s v1.0.0 -m "Release v1.0.0"`
- [ ] Push tag: `git push origin v1.0.0`
- [ ] Create GitHub Release via UI or CLI
- [ ] Upload artifacts
- [ ] Publish Docker images

### 5. Post-Release
- [ ] Update documentation
- [ ] Announce release (if applicable)
- [ ] Monitor for issues

## Support Matrix

| Platform | Architecture | ML Support | Distribution Method |
|----------|-------------|------------|-------------------|
| macOS    | ARM64       | ✅ Full    | Binary, Homebrew  |
| macOS    | AMD64       | ❌ Pattern | Binary, Homebrew  |
| Linux    | AMD64       | ❌ Pattern | Binary, Docker    |
| Linux    | ARM64       | ❌ Pattern | Binary, Docker    |
| Windows  | AMD64       | ❌ Pattern | Binary            |

## Future Enhancements

1. **ML Support for All Platforms**
   - Build tokenizers for each platform
   - Bundle ONNX Runtime libraries
   - Increase binary size but provide full functionality

2. **Package Manager Integration**
   - Submit to Homebrew core
   - Create APT/YUM repositories
   - Windows Package Manager (winget)

3. **Cloud Distribution**
   - AWS S3 bucket with CloudFront
   - Azure Blob Storage
   - Google Cloud Storage

4. **Auto-Update Mechanism**
   - Built-in version checking
   - Self-update capability
   - Update notifications