#!/bin/bash
set -euo pipefail

VERSION="${SHIPSAFE_VERSION:-v0.3.0-alpha}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
esac

BINARY="shipsafe-${OS}-${ARCH}"
BASE_URL="https://repo.toyintest.org/teey/shipsafe/releases/download/${VERSION}"

echo "Installing ShipSafe ${VERSION} (${OS}/${ARCH})..."
curl -fsSL "${BASE_URL}/${BINARY}" -o /usr/local/bin/shipsafe
chmod +x /usr/local/bin/shipsafe
echo "ShipSafe ${VERSION} installed"
