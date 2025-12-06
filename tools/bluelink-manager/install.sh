#!/bin/sh
# Bluelink bootstrap installer
#
# This script downloads and runs the bluelink-manager binary,
# which handles the full installation of Bluelink components.
#
# Usage:
#   curl --proto '=https' --tlsv1.2 -sSf https://manager-sh.bluelink.dev | sh
#
# Environment variables:
#   BLUELINK_INSTALL_DIR - Installation directory (default: ~/.bluelink)

set -e

GITHUB_REPO="newstack-cloud/bluelink"

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    BLUE=''
    NC=''
fi

info() {
    printf "${BLUE}info${NC}: %s\n" "$1"
}

success() {
    printf "${GREEN}success${NC}: %s\n" "$1"
}

error() {
    printf "${RED}error${NC}: %s\n" "$1" >&2
    exit 1
}

# Compute SHA256 hash of a file
compute_sha256() {
    file="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$file" | awk '{print $1}'
    elif command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 "$file" | awk '{print $NF}'
    else
        echo ""
    fi
}

# Verify checksum of downloaded file
verify_checksum() {
    file="$1"
    archive_name="$2"
    checksums_url="$3"

    # Download checksums file
    checksums_file="$TMP_DIR/checksums.txt"
    if ! curl -sL --fail -o "$checksums_file" "$checksums_url" 2>/dev/null; then
        info "Checksums not available, skipping verification"
        return 0
    fi

    # Find expected hash
    expected_hash=$(grep "$archive_name" "$checksums_file" | awk '{print $1}')
    if [ -z "$expected_hash" ]; then
        info "Checksum not found for $archive_name, skipping verification"
        return 0
    fi

    # Compute actual hash
    actual_hash=$(compute_sha256 "$file")
    if [ -z "$actual_hash" ]; then
        info "No SHA256 tool available, skipping verification"
        return 0
    fi

    # Compare
    if [ "$expected_hash" != "$actual_hash" ]; then
        error "Checksum verification failed for $archive_name
  Expected: $expected_hash
  Got:      $actual_hash"
    fi

    success "Checksum verified"
}

# Detect OS and architecture
detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux*)  OS="linux" ;;
        Darwin*) OS="darwin" ;;
        *)
            error "Unsupported operating system: $OS. For Windows, download the installer from https://github.com/$GITHUB_REPO/releases"
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *)
            error "Unsupported architecture: $ARCH"
            ;;
    esac

    PLATFORM="${OS}_${ARCH}"
}

# Get latest manager version from GitHub releases
get_latest_version() {
    releases_url="https://api.github.com/repos/$GITHUB_REPO/releases"

    version=$(curl -sL "$releases_url" | \
        grep -o '"tag_name": "tools/bluelink-manager/v[^"]*"' | \
        head -1 | \
        sed 's/.*\/v\([^"]*\)".*/\1/')

    if [ -z "$version" ]; then
        error "Could not determine latest bluelink-manager version"
    fi

    echo "$version"
}

main() {
    info "Detecting platform..."
    detect_platform
    info "Platform: $PLATFORM"

    # Setup directories
    INSTALL_DIR="${BLUELINK_INSTALL_DIR:-$HOME/.bluelink}"
    BIN_DIR="$INSTALL_DIR/bin"
    mkdir -p "$BIN_DIR"

    # Get latest version
    info "Fetching latest bluelink-manager version..."
    VERSION=$(get_latest_version)
    info "Latest version: v$VERSION"

    # Download manager binary
    TAG="tools/bluelink-manager/v${VERSION}"
    ARCHIVE="bluelink-manager_${VERSION}_${PLATFORM}.tar.gz"
    URL="https://github.com/$GITHUB_REPO/releases/download/$TAG/$ARCHIVE"

    info "Downloading bluelink-manager..."

    TMP_DIR=$(mktemp -d)
    trap "rm -rf '$TMP_DIR'" EXIT

    if ! curl -sL --fail -o "$TMP_DIR/$ARCHIVE" "$URL"; then
        error "Failed to download bluelink-manager from $URL"
    fi

    # Verify checksum
    CHECKSUMS_URL="https://github.com/$GITHUB_REPO/releases/download/$TAG/checksums.txt"
    verify_checksum "$TMP_DIR/$ARCHIVE" "$ARCHIVE" "$CHECKSUMS_URL"

    # Extract
    tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"

    if [ ! -f "$TMP_DIR/bluelink-manager" ]; then
        error "bluelink-manager binary not found in archive"
    fi

    mv "$TMP_DIR/bluelink-manager" "$BIN_DIR/bluelink-manager"
    chmod +x "$BIN_DIR/bluelink-manager"

    success "Downloaded bluelink-manager v$VERSION"

    # Run the manager to complete installation
    info "Running bluelink-manager install..."
    exec "$BIN_DIR/bluelink-manager" install "$@"
}

main "$@"
