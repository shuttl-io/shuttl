#!/usr/bin/env bash
set -euo pipefail

# Get version from git tag if available, otherwise use short commit hash
get_version() {
    local tag
    tag=$(git tag --points-at HEAD 2>/dev/null | head -1)
    if [ -n "$tag" ]; then
        echo "$tag"
    else
        git rev-parse --short HEAD 2>/dev/null || echo 'dev'
    fi
}

VERSION=$(get_version)
echo "Publishing CLI version: $VERSION"

# Platforms to publish
PLATFORMS=(
    "shuttl"
    "shuttl-linux-amd64"
    "shuttl-linux-arm64"
    "shuttl-darwin-amd64"
    "shuttl-darwin-arm64"
    "shuttl-windows-amd64.exe"
    "shuttl-windows-arm64.exe"
)

# Create zip files
echo "Creating zip archives..."
for platform in "${PLATFORMS[@]}"; do
    # Remove .exe suffix for zip filename
    zip_name="shuttl-cli-$VERSION-${platform%.exe}"
    # Handle the default build (no platform suffix)
    if [ "$platform" = "shuttl" ]; then
        zip_name="shuttl-cli-$VERSION"
    fi
    
    if [ -f "dist/$platform" ]; then
        zip -r "dist/${zip_name}.zip" "dist/$platform"
        echo "  Created: dist/${zip_name}.zip"
    else
        echo "  Warning: dist/$platform not found, skipping"
    fi
done

# Upload to GitHub release
echo "Uploading to GitHub release: $VERSION"
for platform in "${PLATFORMS[@]}"; do
    zip_name="shuttl-cli-$VERSION-${platform%.exe}"
    if [ "$platform" = "shuttl" ]; then
        zip_name="shuttl-cli-$VERSION"
    fi
    
    zip_file="dist/${zip_name}.zip"
    if [ -f "$zip_file" ]; then
        gh release upload "$VERSION" "$zip_file" --clobber || {
            echo "  Warning: Failed to upload $zip_file"
        }
        echo "  Uploaded: $zip_file"
    fi
done

echo "Done!"

