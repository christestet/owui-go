#!/bin/sh
set -e

REPO="christestet/owui-go"
BINARY="owui"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

log() {
    echo "$@" >&2
}

fail() {
    log "ERROR: $1"
    exit 1
}

# Check dependencies
command -v curl >/dev/null 2>&1 || fail "curl is required but not installed"
command -v tar >/dev/null 2>&1 || fail "tar is required but not installed"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)  OS="linux" ;;
    darwin) OS="darwin" ;;
    *)      fail "unsupported operating system: $OS (supported: linux, darwin)" ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)   ARCH="amd64" ;;
    aarch64|arm64)   ARCH="arm64" ;;
    *)               fail "unsupported architecture: $ARCH (supported: amd64, arm64)" ;;
esac

log "Detected platform: ${OS}/${ARCH}"

# Fetch latest version
log "Fetching latest release..."
RELEASE_URL="https://api.github.com/repos/${REPO}/releases/latest"

if [ -n "$GITHUB_TOKEN" ]; then
    RELEASE_INFO=$(curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" "$RELEASE_URL")
else
    RELEASE_INFO=$(curl -fsSL "$RELEASE_URL")
fi

TAG=$(echo "$RELEASE_INFO" | grep '"tag_name"' | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')
[ -z "$TAG" ] && fail "could not determine latest version"

# Strip v prefix for archive naming (goreleaser uses Version without v)
VERSION="${TAG#v}"
log "Latest version: ${VERSION}"

# Construct download URLs
ARCHIVE="${BINARY}_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"

# Create temp directory with cleanup
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Download archive and checksums
log "Downloading ${ARCHIVE}..."
curl -fsSL -o "${TMP_DIR}/${ARCHIVE}" "$DOWNLOAD_URL" || fail "failed to download ${ARCHIVE}"
curl -fsSL -o "${TMP_DIR}/checksums.txt" "$CHECKSUMS_URL" || fail "failed to download checksums"

# Verify checksum
log "Verifying checksum..."
EXPECTED=$(grep "${ARCHIVE}" "${TMP_DIR}/checksums.txt" | awk '{print $1}')
[ -z "$EXPECTED" ] && fail "checksum not found for ${ARCHIVE}"

if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    ACTUAL=$(shasum -a 256 "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')
else
    fail "no SHA256 tool found (need sha256sum or shasum)"
fi

if [ "$EXPECTED" != "$ACTUAL" ]; then
    fail "checksum mismatch: expected ${EXPECTED}, got ${ACTUAL}"
fi
log "Checksum verified."

# Extract
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "${TMP_DIR}"

# Install
log "Installing ${BINARY} to ${INSTALL_DIR}..."
if install -m 755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}" 2>/dev/null; then
    :
elif command -v sudo >/dev/null 2>&1; then
    log "Permission denied, retrying with sudo..."
    sudo install -m 755 "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    fail "permission denied and sudo not available. Try: INSTALL_DIR=~/.local/bin sh install.sh"
fi

log ""
log "${BINARY} ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
log "Run '${BINARY} version' to verify."
