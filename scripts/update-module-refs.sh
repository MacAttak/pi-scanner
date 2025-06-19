#!/bin/bash
# Update all module references from pi-scanner/pi-scanner to MacAttak/pi-scanner

set -e

echo "Updating module references to use MacAttak/pi-scanner..."

# Update Go imports in all .go files
find . -name "*.go" -type f | while read file; do
    if grep -q "github.com/pi-scanner/pi-scanner" "$file"; then
        echo "Updating: $file"
        sed -i '' 's|github.com/pi-scanner/pi-scanner|github.com/MacAttak/pi-scanner|g' "$file"
    fi
done

# Update build script
echo "Updating build script..."
sed -i '' "s|github.com/pi-scanner/pi-scanner|github.com/MacAttak/pi-scanner|g" scripts/build-release.sh

# Update config files
echo "Updating config files..."
sed -i '' "s|github.com/pi-scanner/pi-scanner|github.com/MacAttak/pi-scanner|g" pkg/config/default_config.yaml

# Update distribution docs
echo "Updating documentation..."
sed -i '' "s|github.com/pi-scanner/pi-scanner|github.com/MacAttak/pi-scanner|g" docs/DISTRIBUTION.md
sed -i '' "s|ghcr.io/pi-scanner/pi-scanner|ghcr.io/MacAttak/pi-scanner|g" docs/DISTRIBUTION.md

# Update publish script
echo "Updating publish script..."
sed -i '' "s|github.com/pi-scanner/pi-scanner|github.com/MacAttak/pi-scanner|g" scripts/publish-release.sh

# Update docker-compose
if [ -f "docker-compose.yml" ]; then
    echo "Updating docker-compose..."
    sed -i '' "s|pi-scanner/pi-scanner|MacAttak/pi-scanner|g" docker-compose.yml
fi

echo "Done! Module references updated to use MacAttak/pi-scanner"