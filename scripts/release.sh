#!/usr/bin/env bash
# Release script: sync version number across all files before tagging
# Usage: ./scripts/release.sh X.Y.Z
#
# Updates version in:
#   - reporter/reporter_i18n.go  (const Version)
#   - design.md                  (header)
#   - README.md                  (install examples)
#   - README.zh-CN.md            (install examples)
#   - docs/CONFIG.md             (scoop example)
#
# Files auto-updated by CI (publish.yml):
#   - Formula/structoptimizer.rb

set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 X.Y.Z"
    echo "Example: $0 1.8.2"
    exit 1
fi

# Get previous version from git tag
PREV=$(git tag --sort=-v:refname | head -1 | sed 's/^v//')
echo "Previous version: ${PREV:-none}"
echo "New version: $VERSION"
echo ""

# Validate version format
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "Error: version must be in X.Y.Z format"
    exit 1
fi

echo "Syncing version $VERSION across all files..."
echo ""

# 1. Core version constant
sed -i "s/const Version = \"[0-9.]*\"/const Version = \"${VERSION}\"/" reporter/reporter_i18n.go
echo "  ✓ reporter/reporter_i18n.go"

# 2. Design document
sed -i "s/\*\*Version:\*\* [0-9.]*/\*\*Version:\*\* ${VERSION}/" design.md
echo "  ✓ design.md"

# 3. README - specific version example (match any previous version)
sed -i "s|VERSION=v[0-9]\+\.[0-9]\+\.[0-9]\+|VERSION=v${VERSION}|g" README.md
sed -i "s|VERSION=v[0-9]\+\.[0-9]\+\.[0-9]\+|VERSION=v${VERSION}|g" README.zh-CN.md
sed -i "s|download/v[0-9]\+\.[0-9]\+\.[0-9]\+/|download/v${VERSION}/|g" docs/CONFIG.md
echo "  ✓ README.md"
echo "  ✓ README.zh-CN.md"
echo "  ✓ docs/CONFIG.md"

echo ""
echo "Done. Next steps:"
echo "  git diff                 # review changes"
echo "  git add -u scripts"
echo "  git commit -m 'chore: bump version to ${VERSION}'"
echo "  git tag -a v${VERSION} -m 'v${VERSION}'"
echo "  git push origin main --tags"
