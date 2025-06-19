#!/bin/bash
# Publish release to GitHub

set -e

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "Error: GitHub CLI (gh) is not installed"
    echo "Install with: brew install gh"
    exit 1
fi

# Check arguments
if [ $# -ne 1 ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

VERSION=$1
RELEASE_DIR="releases/v${VERSION}"

# Verify release directory exists
if [ ! -d "${RELEASE_DIR}" ]; then
    echo "Error: Release directory ${RELEASE_DIR} does not exist"
    echo "Run: VERSION=${VERSION} ./scripts/build-release.sh"
    exit 1
fi

# Check if we're in a git repo
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not in a git repository"
    exit 1
fi

# Check if tag already exists
if git rev-parse "v${VERSION}" >/dev/null 2>&1; then
    echo "Error: Tag v${VERSION} already exists"
    exit 1
fi

echo "Publishing PI Scanner v${VERSION} to GitHub..."
echo ""

# Create and push tag
echo "Creating tag v${VERSION}..."
git tag -a "v${VERSION}" -m "Release v${VERSION}"
git push origin "v${VERSION}"

# Create GitHub release
echo ""
echo "Creating GitHub release..."

# Check if release already exists
if gh release view "v${VERSION}" >/dev/null 2>&1; then
    echo "Release v${VERSION} already exists. Updating..."
    RELEASE_CMD="edit"
else
    RELEASE_CMD="create"
fi

# Create/update release
gh release ${RELEASE_CMD} "v${VERSION}" \
    --title "PI Scanner v${VERSION}" \
    --notes-file "${RELEASE_DIR}/RELEASE_NOTES.md" \
    --draft

# Upload artifacts
echo ""
echo "Uploading release artifacts..."

# Upload all artifacts
gh release upload "v${VERSION}" \
    "${RELEASE_DIR}"/*.tar.gz \
    "${RELEASE_DIR}"/*.zip \
    "${RELEASE_DIR}"/checksums.txt \
    --clobber

echo ""
echo "✅ Release draft created successfully!"
echo ""
echo "Next steps:"
echo "1. Review the release at: https://github.com/MacAttak/pi-scanner/releases/tag/v${VERSION}"
echo "2. Test download links and checksums"
echo "3. Publish the release (remove draft status)"
echo ""
echo "To publish Docker images:"
echo "  docker buildx build --platform linux/amd64,linux/arm64 --tag ghcr.io/MacAttak/pi-scanner:${VERSION} --tag ghcr.io/MacAttak/pi-scanner:latest --push ."