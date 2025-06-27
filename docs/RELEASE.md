# Release and Distribution Guide

This guide covers the release process, distribution channels, and publishing procedures for the GitHub PI Scanner.

## Table of Contents

- [Release Process](#release-process)
- [Distribution Channels](#distribution-channels)
- [Release Artifacts](#release-artifacts)
- [Publishing Procedures](#publishing-procedures)
- [Post-Release](#post-release)

## Release Process

### Pre-Release Checklist

Before creating a release, ensure:

- [ ] All tests passing
- [ ] Security scan clean
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Version bumped in code
- [ ] Release notes drafted

### Versioning

We follow [Semantic Versioning](https://semver.org/):
- **Major**: Breaking changes
- **Minor**: New features (backward compatible)
- **Patch**: Bug fixes

Examples: `v1.0.0`, `v1.1.0`, `v1.1.1`

## Distribution Channels

### Primary Channel: GitHub Releases

- **URL**: `https://github.com/MacAttak/pi-scanner/releases`
- **Assets**: Binaries, checksums, source code
- **Visibility**: Public

### Docker Images: GitHub Container Registry

- **Registry**: `ghcr.io/MacAttak/pi-scanner`
- **Tags**:
  - Version tags: `v1.0.0`, `v1.1.0`
  - Latest tag: `latest`
- **Architectures**: `linux/amd64`, `linux/arm64`

### Future Channels

Planned for future releases:
- Homebrew (macOS/Linux)
- APT/DEB packages (Debian/Ubuntu)
- RPM packages (RHEL/Fedora)

## Release Artifacts

### Binary Releases

Platform-specific binaries with no external dependencies:

| Platform | Architecture | Binary Name | Notes |
|----------|-------------|-------------|-------|
| macOS | Intel | `pi-scanner-darwin-amd64` | ML support included |
| macOS | ARM64 | `pi-scanner-darwin-arm64` | ML support included |
| Linux | x64 | `pi-scanner-linux-amd64` | No ML support |
| Linux | ARM64 | `pi-scanner-linux-arm64` | No ML support |
| Windows | x64 | `pi-scanner-windows-amd64.exe` | No ML support |

All binaries are built with:
- `CGO_ENABLED=0` for maximum portability
- Static linking
- Stripped symbols for smaller size

### Docker Images

Multi-architecture images supporting:
- `linux/amd64`
- `linux/arm64`

Features:
- Minimal Alpine base
- Non-root user execution
- Security scanning in CI

### Checksums

SHA256 checksums for all artifacts in `checksums.txt`:
```
abc123... pi-scanner-darwin-amd64
def456... pi-scanner-linux-amd64
...
```

## Publishing Procedures

### 1. Prepare Release

```bash
# Set version
export VERSION=v1.0.0

# Update version in code
vim internal/version/version.go  # Update Version constant

# Update CHANGELOG
vim CHANGELOG.md  # Add release notes

# Run quality gates to ensure release readiness
./scripts/check-quality-gates.sh

# Commit changes
git add -A
git commit -m "chore: prepare release $VERSION"
git push
```

### 2. Build Release Artifacts

First, build all platform binaries:
```bash
# Build release artifacts using the build script
./scripts/build-release.sh $VERSION
```

This will:
1. Create a `releases/$VERSION` directory
2. Build native binary with ML support (macOS only, CGO enabled)
3. Build cross-platform binaries (CGO disabled):
   - `darwin/amd64`
   - `linux/amd64`
   - `linux/arm64`
   - `windows/amd64`
4. Create compressed archives (.tar.gz for Unix, .zip for Windows)
5. Generate SHA256 checksums

### 3. Create GitHub Release

Use the publish script:
```bash
./scripts/publish-release.sh $VERSION
```

This script will:
1. Create and push Git tag
2. Create draft GitHub release using `gh` CLI
3. Upload all artifacts from `releases/$VERSION`
4. Upload checksums file
5. Set release as draft for review

### 4. Build Docker Images

Currently, Docker images must be built and pushed manually:

```bash
# Authenticate to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USER --password-stdin

# Build and tag using docker-compose
docker-compose build pi-scanner

# Tag for release
docker tag pi-scanner:latest ghcr.io/macattak/pi-scanner:$VERSION
docker tag pi-scanner:latest ghcr.io/macattak/pi-scanner:latest

# Push to registry
docker push ghcr.io/macattak/pi-scanner:$VERSION
docker push ghcr.io/macattak/pi-scanner:latest

# For multi-architecture support (if using buildx)
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag ghcr.io/macattak/pi-scanner:$VERSION \
  --tag ghcr.io/macattak/pi-scanner:latest \
  --push .
```

### 5. Finalize Release

1. Go to GitHub releases page
2. Review the draft release
3. Add detailed release notes
4. Mark as pre-release if beta
5. Publish release

### Release Notes Template

```markdown
## What's New
- Feature: Added bank account validation
- Fix: Improved memory handling
- Security: Patched CVE-XXXX

## Breaking Changes
None

## Installation

### Binary
\```bash
# macOS/Linux
curl -L https://github.com/MacAttak/pi-scanner/releases/download/$VERSION/pi-scanner-$(uname -s)-$(uname -m) -o pi-scanner
chmod +x pi-scanner
\```

### Docker
\```bash
docker pull ghcr.io/macattak/pi-scanner:$VERSION
\```

## Checksums
See checksums.txt for SHA256 verification

## Contributors
Thanks to @user1, @user2 for contributions!
```

## Post-Release

### 1. Verification

After release:
- [ ] Download and test each binary
- [ ] Verify checksums match
- [ ] Test Docker images on both architectures
- [ ] Check documentation links work

### 2. Announcements

Announce the release:
- GitHub Discussions
- Security mailing list (for security releases)
- Social media (if applicable)

### 3. Monitor

Watch for:
- Issue reports
- Download statistics
- Docker pull counts
- Security vulnerabilities

### 4. Hotfix Process

For critical fixes:
1. Create hotfix branch from release tag
2. Apply minimal fix
3. Fast-track testing
4. Release as patch version
5. Cherry-pick to main branch

## Security Releases

For security vulnerabilities:

1. **Coordinate disclosure** - Work with reporter
2. **Prepare fix** - Develop in private
3. **Release simultaneously** with:
   - Security advisory
   - Patched version
   - Mitigation steps
4. **Credit reporter** in release notes

## CI/CD Integration

### GitHub Actions Workflow

The main CI pipeline (`.github/workflows/ci.yml`) runs on:
- Push to `main` and `develop` branches
- Pull requests to `main`

**CI Jobs**:
1. **Test & Quality**: Unit tests, coverage, linting
2. **Security Scans**: gosec, govulncheck, Trivy
3. **Cross-Platform Build**: Builds for all platforms
4. **Performance**: Benchmarks (PR only)
5. **CodeQL**: Security analysis
6. **E2E Tests**: Integration testing

All CI runs in Docker containers for consistency with local development.

### Quality Gates

The `scripts/check-quality-gates.sh` script enforces:
- Code formatting (gofmt/gofumpt)
- Import organization (goimports)
- Static analysis (go vet)
- Linting (golangci-lint)
- Test coverage (70% minimum)
- Security scanning
- Multi-platform builds

### Release Scripts

Key scripts in `/scripts`:
- `build-release.sh` - Builds all platform binaries
- `publish-release.sh` - Creates GitHub releases
- `check-quality-gates.sh` - Validates release readiness

### Docker Development

All development happens in Docker:
- `make dev` - Development shell
- `make test` - Run tests in container
- `make ci-local` - Simulate full CI locally
- `make build-all` - Build all platforms

## Troubleshooting

### Common Issues

1. **Build failures**
   - Check Go version compatibility
   - Verify dependencies with `go mod tidy`

2. **Docker push fails**
   - Verify GHCR authentication
   - Check repository permissions

3. **Missing artifacts**
   - Ensure all platforms built
   - Check upload logs

### Rollback Procedure

If issues found post-release:
1. Yank the release (mark as draft)
2. Delete Docker tags if pushed
3. Fix issues
4. Re-release with incremented patch version

## Future Improvements

Planned enhancements:
- Automated changelog generation
- GPG signing for binaries
- Homebrew formula updates
- Package manager integration
- Release metrics dashboard
