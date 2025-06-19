#!/bin/bash
# Build release binaries for PI Scanner

set -e

# Version information
VERSION=${VERSION:-"1.0.0"}
BUILD_DATE=$(date -u '+%Y-%m-%d %H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS="-s -w -extldflags '-L./lib'"
LDFLAGS="${LDFLAGS} -X 'main.version=${VERSION}'"
LDFLAGS="${LDFLAGS} -X 'main.commit=${GIT_COMMIT}'"
LDFLAGS="${LDFLAGS} -X 'main.buildDate=${BUILD_DATE}'"

# Create release directory
RELEASE_DIR="releases/v${VERSION}"
mkdir -p ${RELEASE_DIR}

echo "🚀 Building PI Scanner Release v${VERSION}"
echo "   Build: ${GIT_COMMIT}"
echo "   Date: ${BUILD_DATE}"
echo ""

# Function to build for a specific platform
build_platform() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT_NAME=$3
    
    echo "Building for ${GOOS}/${GOARCH}..."
    
    # Set environment
    export GOOS=${GOOS}
    export GOARCH=${GOARCH}
    export CGO_ENABLED=0  # Disable CGO for cross-platform builds
    
    # Build
    go build -ldflags="${LDFLAGS}" -o "${RELEASE_DIR}/${OUTPUT_NAME}" ./cmd/pi-scanner
    
    # Create archive
    if [ "${GOOS}" = "windows" ]; then
        # Create ZIP for Windows
        cd ${RELEASE_DIR}
        zip "${OUTPUT_NAME%.exe}.zip" "${OUTPUT_NAME}"
        rm "${OUTPUT_NAME}"
        cd - > /dev/null
    else
        # Create tar.gz for Unix
        cd ${RELEASE_DIR}
        tar -czf "${OUTPUT_NAME}.tar.gz" "${OUTPUT_NAME}"
        rm "${OUTPUT_NAME}"
        cd - > /dev/null
    fi
    
    echo "✓ Created ${RELEASE_DIR}/${OUTPUT_NAME}.tar.gz" || echo "✓ Created ${RELEASE_DIR}/${OUTPUT_NAME}.zip"
}

# Build for native platform with CGO (includes ML support)
echo "Building native binary with ML support..."
export CGO_ENABLED=1
export CGO_LDFLAGS="-L${PWD}/lib"
export CGO_CFLAGS="-I${PWD}/lib/tokenizers-src"

NATIVE_OS=$(go env GOOS)
NATIVE_ARCH=$(go env GOARCH)
NATIVE_BINARY="pi-scanner-${NATIVE_OS}-${NATIVE_ARCH}-ml"
NATIVE_BINARY_FILE="${NATIVE_BINARY}-bin"

go build -ldflags="${LDFLAGS}" -o "${RELEASE_DIR}/${NATIVE_BINARY_FILE}" ./cmd/pi-scanner

# Create archive with libraries
echo "Creating native release with ML libraries..."
NATIVE_RELEASE_DIR="${RELEASE_DIR}/pi-scanner-${NATIVE_OS}-${NATIVE_ARCH}-ml"
rm -rf "${NATIVE_RELEASE_DIR}"
mkdir -p "${NATIVE_RELEASE_DIR}"
cp "${RELEASE_DIR}/${NATIVE_BINARY_FILE}" "${NATIVE_RELEASE_DIR}/pi-scanner"
cp lib/libtokenizers.a "${NATIVE_RELEASE_DIR}/"
cp pkg/config/default_config.yaml "${NATIVE_RELEASE_DIR}/"

# Create README for native release
cat > "${NATIVE_RELEASE_DIR}/README.txt" << EOF
PI Scanner v${VERSION} - Native Build with ML Support

This build includes machine learning support for enhanced PI detection.

Requirements:
- ONNX Runtime (optional, for ML features)
- libtokenizers.a is included

Usage:
  ./pi-scanner scan --repo https://github.com/org/repo

For more information:
  ./pi-scanner help
EOF

cd ${RELEASE_DIR}
tar -czf "${NATIVE_BINARY}.tar.gz" "pi-scanner-${NATIVE_OS}-${NATIVE_ARCH}-ml"
rm -rf "pi-scanner-${NATIVE_OS}-${NATIVE_ARCH}-ml" "${NATIVE_BINARY_FILE}"
cd - > /dev/null

echo "✓ Created ${RELEASE_DIR}/${NATIVE_BINARY}.tar.gz (with ML support)"

# Build cross-platform binaries (without ML)
echo ""
echo "Building cross-platform binaries (without ML support)..."

# macOS AMD64
build_platform "darwin" "amd64" "pi-scanner-darwin-amd64"

# Linux AMD64
build_platform "linux" "amd64" "pi-scanner-linux-amd64"

# Linux ARM64
build_platform "linux" "arm64" "pi-scanner-linux-arm64"

# Windows AMD64
build_platform "windows" "amd64" "pi-scanner-windows-amd64.exe"

# Create checksums
echo ""
echo "Creating checksums..."
cd ${RELEASE_DIR}
shasum -a 256 *.tar.gz *.zip > checksums.txt
cd - > /dev/null

# Create release notes
cat > "${RELEASE_DIR}/RELEASE_NOTES.md" << EOF
# PI Scanner v${VERSION}

Release Date: ${BUILD_DATE}
Git Commit: ${GIT_COMMIT}

## Release Assets

### With ML Support (Recommended)
- \`pi-scanner-${NATIVE_OS}-${NATIVE_ARCH}-ml.tar.gz\` - Native build with ML support

### Cross-Platform Builds (No ML)
- \`pi-scanner-darwin-amd64.tar.gz\` - macOS Intel
- \`pi-scanner-linux-amd64.tar.gz\` - Linux x64
- \`pi-scanner-linux-arm64.tar.gz\` - Linux ARM64
- \`pi-scanner-windows-amd64.zip\` - Windows x64

## Installation

### Native Build (with ML)
\`\`\`bash
tar -xzf pi-scanner-${NATIVE_OS}-${NATIVE_ARCH}-ml.tar.gz
cd pi-scanner-${NATIVE_OS}-${NATIVE_ARCH}-ml
./pi-scanner version
\`\`\`

### Cross-Platform Build
\`\`\`bash
# Linux/macOS
tar -xzf pi-scanner-<platform>.tar.gz
./pi-scanner version

# Windows
unzip pi-scanner-windows-amd64.zip
pi-scanner.exe version
\`\`\`

## Features
- Australian PI detection (TFN, ABN, Medicare, BSB)
- Multi-stage validation pipeline
- Risk scoring with regulatory compliance
- Multiple report formats (HTML, CSV, SARIF, JSON)

## Notes
- ML features require the native build with included libraries
- Cross-platform builds have pattern matching only (no ML validation)
EOF

echo ""
echo "✅ Release build complete!"
echo "   Output: ${RELEASE_DIR}"
echo ""
ls -la ${RELEASE_DIR}/