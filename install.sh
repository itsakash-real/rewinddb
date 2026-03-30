#!/usr/bin/env bash
set -euo pipefail

REPO="itsakash-real/nimbi"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS_LABEL="linux" ;;
  darwin) OS_LABEL="darwin" ;;
  *)      echo "❌ Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        ARCH_LABEL="amd64" ;;
  aarch64|arm64) ARCH_LABEL="arm64" ;;
  *)             echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Fetch latest release tag
echo "→ Fetching latest release..."
VERSION=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | sed 's/.*"tag_name": "\(.*\)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "❌ Could not determine latest version"
  exit 1
fi

# Build download URL — release assets use nimbi-{os}-{arch} naming
ASSET="nimbi-${OS_LABEL}-${ARCH_LABEL}"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"

echo "→ Downloading Nimbi ${VERSION} for ${OS_LABEL}/${ARCH_LABEL}..."
TMP=$(mktemp -d)
curl -sSfL "$URL" -o "${TMP}/rw"

if [ ! -f "${TMP}/rw" ]; then
  echo "❌ Download failed"
  rm -rf "$TMP"
  exit 1
fi

echo "→ Installing to ${INSTALL_DIR}/rw"
chmod +x "${TMP}/rw"
sudo install -m 0755 "${TMP}/rw" "${INSTALL_DIR}/rw"
rm -rf "$TMP"

echo ""
echo "✅ Nimbi installed successfully — run: rw init"
