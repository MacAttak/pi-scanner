# PI Scanner Publishing Guide

## Current Configuration

The PI Scanner has been configured to publish under your personal GitHub account.

### GitHub Repository
- **URL**: https://github.com/MacAttak/pi-scanner
- **Status**: ✅ Created and code pushed
- **Visibility**: Public

### Docker Registry
- **Registry**: GitHub Container Registry (ghcr.io)
- **Image**: `ghcr.io/MacAttak/pi-scanner`
- **Note**: NOT Docker Hub - uses GitHub's container registry

## Publishing Steps

### 1. Create GitHub Release

```bash
# This will create a draft release on GitHub
./scripts/publish-release.sh 1.0.0
```

This script will:
- Create git tag `v1.0.0`
- Push tag to GitHub
- Create draft release
- Upload all binaries and checksums
- You can review at: https://github.com/MacAttak/pi-scanner/releases

### 2. Publish Docker Images

First, authenticate with GitHub Container Registry:
```bash
# Create a GitHub Personal Access Token with 'write:packages' scope
# Go to: https://github.com/settings/tokens/new

# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u MacAttak --password-stdin
```

Build and push multi-architecture images:
```bash
# Create buildx builder (one time)
docker buildx create --name mybuilder --use

# Build and push
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag ghcr.io/MacAttak/pi-scanner:1.0.0 \
  --tag ghcr.io/MacAttak/pi-scanner:latest \
  --push .
```

### 3. Make Release Public

1. Go to https://github.com/MacAttak/pi-scanner/releases
2. Find your draft release
3. Review all artifacts
4. Click "Publish release"

## What Gets Published

### GitHub Release Assets
- `pi-scanner-darwin-arm64-ml.tar.gz` - macOS ARM64 with ML (12.9 MB)
- `pi-scanner-darwin-amd64.tar.gz` - macOS Intel without ML (3.8 MB)
- `pi-scanner-linux-amd64.tar.gz` - Linux x64 without ML (3.7 MB)
- `pi-scanner-linux-arm64.tar.gz` - Linux ARM64 without ML (3.4 MB)
- `pi-scanner-windows-amd64.zip` - Windows x64 without ML (3.9 MB)
- `checksums.txt` - SHA256 checksums for verification

### Docker Images
- `ghcr.io/MacAttak/pi-scanner:1.0.0` - Version tagged
- `ghcr.io/MacAttak/pi-scanner:latest` - Latest tag
- Multi-architecture: linux/amd64 and linux/arm64
- Includes full ML support

## Usage After Publishing

### Installing from GitHub Release
```bash
# Download for your platform
curl -L https://github.com/MacAttak/pi-scanner/releases/download/v1.0.0/pi-scanner-darwin-arm64-ml.tar.gz | tar xz
cd pi-scanner-darwin-arm64-ml
sudo cp pi-scanner /usr/local/bin/
```

### Using Docker Image
```bash
# Pull image
docker pull ghcr.io/MacAttak/pi-scanner:1.0.0

# Run scan
docker run --rm \
  -v $(pwd)/reports:/app/reports \
  ghcr.io/MacAttak/pi-scanner:1.0.0 \
  scan --repo https://github.com/org/repo
```

## Future Considerations

### Moving to Organization
If you later want to move to an organization:
1. Create organization on GitHub
2. Transfer repository (Settings → Transfer ownership)
3. Update Docker image references
4. Rebuild and republish

### Distribution Channels
Consider adding:
- Homebrew tap for macOS users
- AUR package for Arch Linux
- Snap or Flatpak for Linux
- Windows Package Manager (winget)

### Signing Releases
For production use:
1. GPG sign your commits and tags
2. Sign release artifacts
3. Provide public key for verification

## Troubleshooting

### Docker Push Fails
- Ensure you have `write:packages` scope in GitHub token
- Check you're logged in: `docker login ghcr.io`
- Verify image exists locally: `docker images`

### GitHub Release Fails
- Check you have push access to repository
- Ensure tag doesn't already exist
- Verify `gh` CLI is authenticated: `gh auth status`

---

Ready to publish! Start with `./scripts/publish-release.sh 1.0.0`