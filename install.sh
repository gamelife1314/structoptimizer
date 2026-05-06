#!/usr/bin/env bash
# StructOptimizer Universal Installer
# Supports: Homebrew, APT, YUM/DNF, direct binary download
# Usage: curl -fsSL https://raw.githubusercontent.com/gamelife1314/structoptimizer/main/install.sh | bash

set -euo pipefail

REPO="gamelife1314/structoptimizer"
BINARY="structoptimizer"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

detect_os() {
    case "$(uname -s)" in
        Linux)  echo "linux" ;;
        Darwin) echo "darwin" ;;
        *)      echo "unsupported" ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unsupported" ;;
    esac
}

# ---- Installation Methods ----

install_via_homebrew() {
    if command -v brew &>/dev/null; then
        info "Homebrew detected, installing via brew..."
        brew tap "$REPO"
        brew install "$BINARY"
        return 0
    elif command -v gh &>/dev/null; then
        info "Installing via GitHub release..."
        install_via_github
        return 0
    else
        return 1
    fi
}

install_via_apt() {
    if command -v apt-get &>/dev/null; then
        info "APT detected, installing via direct download (no apt repo yet)..."
        install_via_github
        return 0
    else
        return 1
    fi
}

install_via_yum_dnf() {
    if command -v dnf &>/dev/null || command -v yum &>/dev/null; then
        info "YUM/DNF detected, installing via direct download (no yum repo yet)..."
        install_via_github
        return 0
    else
        return 1
    fi
}

install_via_github() {
    OS=$(detect_os)
    ARCH=$(detect_arch)

    if [ "$OS" = "unsupported" ] || [ "$ARCH" = "unsupported" ]; then
        error "Unsupported platform: $(uname -s) $(uname -m)"
    fi

    if [ "$VERSION" = "latest" ]; then
        LATEST_URL="https://api.github.com/repos/$REPO/releases/latest"
        TAG=$(curl -sL "$LATEST_URL" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)".*/\1/')
        if [ -z "$TAG" ]; then
            error "Failed to get latest version from GitHub"
        fi
    else
        TAG="$VERSION"
    fi

    ARCHIVE="structoptimizer-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        error "Windows is not supported by this installer. Use: go install or scoop"
        ARCHIVE="${ARCHIVE}.zip"
    else
        ARCHIVE="${ARCHIVE}.tar.gz"
    fi

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/${TAG}/${ARCHIVE}"

    info "Downloading $BINARY $TAG for $OS/$ARCH..."
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    if ! curl -sL "$DOWNLOAD_URL" -o "$ARCHIVE"; then
        error "Failed to download $DOWNLOAD_URL"
    fi

    if [[ "$ARCHIVE" == *.tar.gz ]]; then
        tar xzf "$ARCHIVE"
    elif [[ "$ARCHIVE" == *.zip ]]; then
        unzip -q "$ARCHIVE"
    fi

    info "Installing to $INSTALL_DIR..."
    sudo mkdir -p "$INSTALL_DIR"
    sudo cp "$BINARY" "$INSTALL_DIR/$BINARY"
    sudo chmod +x "$INSTALL_DIR/$BINARY"

    cd - > /dev/null
    rm -rf "$TMP_DIR"

    info "✅ $BINARY $TAG installed to $INSTALL_DIR/$BINARY"
}

# ---- Main ----

info "StructOptimizer Installer"
info "========================"

OS=$(detect_os)
ARCH=$(detect_arch)
info "Detected: $OS/$ARCH"

# Try Homebrew first on macOS
if [ "$OS" = "darwin" ]; then
    if install_via_homebrew; then
        exit 0
    fi
fi

# Try package managers on Linux
if [ "$OS" = "linux" ]; then
    if install_via_apt; then
        exit 0
    fi
    if install_via_yum_dnf; then
        exit 0
    fi
fi

# Final fallback: direct GitHub download
install_via_github

info ""
info "Verifying installation..."
if command -v "$BINARY" &>/dev/null; then
    "$BINARY" -version
else
    warn "Binary not found in PATH. You may need to add $INSTALL_DIR to your PATH."
fi
