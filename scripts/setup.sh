#!/bin/bash
# Setup script for PI Scanner development environment

set -e

echo "🔧 PI Scanner Development Setup"
echo "=============================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check for required tools
check_tool() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}❌ $1 is not installed${NC}"
        echo "   Please install $1 and run this script again"
        exit 1
    else
        echo -e "${GREEN}✓ $1 is installed${NC}"
    fi
}

echo "Checking required tools..."
check_tool go
check_tool git
check_tool make

# Optional tools
if command -v docker &> /dev/null; then
    echo -e "${GREEN}✓ Docker is installed (optional)${NC}"
else
    echo -e "${YELLOW}⚠ Docker is not installed (optional but recommended)${NC}"
fi

# Set up environment variables
echo ""
echo "Setting up environment variables..."
if [ -f .envrc ]; then
    source .envrc
    echo -e "${GREEN}✓ Environment variables loaded${NC}"
else
    echo -e "${RED}❌ .envrc file not found${NC}"
    exit 1
fi

# Check for tokenizers library
echo ""
echo "Checking for tokenizers library..."
if [ -f lib/libtokenizers.a ]; then
    echo -e "${GREEN}✓ Tokenizers library found${NC}"
else
    echo -e "${YELLOW}⚠ Tokenizers library not found${NC}"
    echo "  Building tokenizers library..."
    if [ -d lib/tokenizers-src ]; then
        cd lib/tokenizers-src
        make build
        cp libtokenizers.a ../
        cd ../..
        echo -e "${GREEN}✓ Tokenizers library built${NC}"
    else
        echo -e "${RED}❌ Tokenizers source not found in lib/tokenizers-src${NC}"
        echo "   Please ensure the tokenizers source is available"
        exit 1
    fi
fi

# Check for ONNX Runtime
echo ""
echo "Checking for ONNX Runtime..."
if [ -f /usr/local/lib/libonnxruntime.so ] || [ -f /usr/local/lib/libonnxruntime.dylib ]; then
    echo -e "${GREEN}✓ ONNX Runtime found${NC}"
else
    echo -e "${YELLOW}⚠ ONNX Runtime not found${NC}"
    echo "  Please install ONNX Runtime manually:"
    echo "  - Download from: https://github.com/microsoft/onnxruntime/releases"
    echo "  - Or use Docker for a complete environment"
fi

# Download Go dependencies
echo ""
echo "Downloading Go dependencies..."
go mod download
echo -e "${GREEN}✓ Go dependencies downloaded${NC}"

# Run basic tests
echo ""
echo "Running basic tests (excluding ML)..."
if make test-no-ml > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Basic tests passed${NC}"
else
    echo -e "${YELLOW}⚠ Some tests failed - check with 'make test-no-ml'${NC}"
fi

# Build the binary
echo ""
echo "Building PI Scanner..."
if make build; then
    echo -e "${GREEN}✓ Build successful${NC}"
    echo "  Binary location: build/pi-scanner"
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

# Setup complete
echo ""
echo -e "${GREEN}🎉 Setup complete!${NC}"
echo ""
echo "Next steps:"
echo "  1. To run tests: make test-no-ml"
echo "  2. To run the scanner: ./build/pi-scanner --help"
echo "  3. To use Docker: docker-compose up pi-scanner"
echo ""
echo "For ML features:"
echo "  - Install ONNX Runtime locally, or"
echo "  - Use Docker: docker-compose run pi-scanner-test"