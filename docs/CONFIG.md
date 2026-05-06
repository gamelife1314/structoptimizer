# Release Infrastructure Configuration

This document describes the one-time setup required for publishing to package managers.

## GitHub Pages (APT/YUM Repository)

The publish workflow pushes `.deb` and `.rpm` packages to the `gh-pages` branch.

**One-time setup**: Enable GitHub Pages:

1. Go to repository **Settings → Pages**
2. Under **Branch**, select `gh-pages` and `/ (root)`, click **Save**
3. Wait 1-2 minutes for the site to deploy

**Verification**: Visit `https://gamelife1314.github.io/structoptimizer/` - you should see the package repository page.

## GitHub Release Permissions

The existing `release.yml` workflow needs `contents: write` permission (already configured).

If you see permission errors, go to **Settings → Actions → General → Workflow permissions** and select **"Read and write permissions"**.

## Homebrew

No additional setup needed. The formula in `Formula/structoptimizer.rb` is auto-updated by the `publish.yml` workflow on each release.

Users install via:
```bash
brew tap gamelife1314/structoptimizer
brew install structoptimizer
```

## Scoop (Windows)

To add Scoop support, create a separate bucket repository:
1. Create repo `scoop-structoptimizer`
2. Add `structoptimizer.json` manifest
3. Submit to official Scoop buckets or use as custom bucket

The scoop manifest template is:
```json
{
  "version": "1.8.0",
  "description": "Go struct alignment optimization tool",
  "homepage": "https://github.com/gamelife1314/structoptimizer",
  "license": "MIT",
  "architecture": {
    "64bit": {
      "url": "https://github.com/gamelife1314/structoptimizer/releases/download/v1.8.2/structoptimizer-windows-amd64.zip"
    }
  },
  "bin": "structoptimizer.exe",
  "checkver": { "github": "https://github.com/gamelife1314/structoptimizer" }
}
```
