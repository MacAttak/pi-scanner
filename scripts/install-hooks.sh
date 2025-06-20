#!/bin/bash
# Install script for Git hooks and pre-commit

set -e

echo "🔧 Installing Git hooks for GitHub PI Scanner..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "Error: Not in a git repository"
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

echo -e "${BLUE}Installing pre-commit framework...${NC}"

# Install pre-commit if not already installed
if ! command -v pre-commit >/dev/null 2>&1; then
    echo "pre-commit not found. Installing..."
    
    # Try pip first
    if command -v pip3 >/dev/null 2>&1; then
        pip3 install --user pre-commit
    elif command -v pip >/dev/null 2>&1; then
        pip install --user pre-commit
    else
        echo -e "${YELLOW}Warning: pip not found. Please install pre-commit manually:${NC}"
        echo "  brew install pre-commit  # on macOS"
        echo "  pip install pre-commit   # with pip"
        echo ""
    fi
fi

# Install pre-commit hooks
if command -v pre-commit >/dev/null 2>&1; then
    echo -e "${BLUE}Installing pre-commit hooks...${NC}"
    pre-commit install
    echo -e "${GREEN}✓ Pre-commit hooks installed${NC}"
else
    echo -e "${YELLOW}⚠ pre-commit not available, skipping hook installation${NC}"
fi

# Set up custom Git hooks directory
echo -e "${BLUE}Setting up custom Git hooks...${NC}"
git config core.hooksPath .githooks

# Make hooks executable
if [ -d ".githooks" ]; then
    chmod +x .githooks/* 2>/dev/null || true
    echo -e "${GREEN}✓ Custom Git hooks configured${NC}"
fi

# Install optional tools
echo -e "\n${BLUE}Checking optional tools...${NC}"

# Check for Go tools
check_tool() {
    local tool=$1
    local install_cmd=$2
    
    if command -v "$tool" >/dev/null 2>&1; then
        echo -e "${GREEN}✓ $tool is installed${NC}"
    else
        echo -e "${YELLOW}⚠ $tool is not installed${NC}"
        echo "  Install with: $install_cmd"
    fi
}

check_tool "goimports" "go install golang.org/x/tools/cmd/goimports@latest"
check_tool "golangci-lint" "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
check_tool "gosec" "go install github.com/securego/gosec/v2/cmd/gosec@latest"
check_tool "gitleaks" "brew install gitleaks # or download from GitHub"
check_tool "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest"

# Create local config file if needed
if [ ! -f ".git/hooks/pre-push" ] && [ -f ".githooks/pre-push" ]; then
    echo -e "\n${BLUE}Symlinking pre-push hook...${NC}"
    ln -sf ../../.githooks/pre-push .git/hooks/pre-push
    echo -e "${GREEN}✓ Pre-push hook linked${NC}"
fi

echo -e "\n${GREEN}✅ Git hooks installation complete!${NC}"
echo ""
echo "Available commands:"
echo "  pre-commit run --all-files  # Run all pre-commit checks"
echo "  make ci-local              # Simulate CI pipeline locally"
echo "  make pre-commit            # Run pre-commit checks"
echo "  make pre-push              # Run pre-push checks"
echo ""
echo "Hooks will run automatically on:"
echo "  - git commit (pre-commit hooks)"
echo "  - git push (pre-push hooks)"
echo ""
echo -e "${YELLOW}Note: Install missing tools for full functionality${NC}"