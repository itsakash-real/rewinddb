#!/usr/bin/env bash
set -euo pipefail

REPO="itsakash-real/rewinddb"
BINARY="rw"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="x86_64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Fetch latest release version
VERSION=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | sed 's/.*"tag_name": "\(.*\)".*/\1/')

OS_TITLE=$(echo "$OS" | sed 's/darwin/Darwin/;s/linux/Linux/;s/windows/Windows/')
ARCHIVE="rewinddb_${VERSION#v}_${OS_TITLE}_${ARCH}"

case "$OS" in
  windows*) ARCHIVE="${ARCHIVE}.zip" ;;
  *)        ARCHIVE="${ARCHIVE}.tar.gz" ;;
esac

URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "→ Downloading RewindDB ${VERSION} for ${OS}/${ARCH}"
TMP=$(mktemp -d)
curl -sSfL "$URL" -o "${TMP}/${ARCHIVE}"

echo "→ Extracting"
cd "$TMP"
case "$ARCHIVE" in
  *.tar.gz) tar xzf "$ARCHIVE" ;;
  *.zip)    unzip -q "$ARCHIVE" ;;
esac

echo "→ Installing to ${INSTALL_DIR}/${BINARY}"
install -m 0755 "${BINARY}" "${INSTALL_DIR}/${BINARY}"
rm -rf "$TMP"

echo "✓ RewindDB ${VERSION} installed"
rw version
