#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$ROOT/bin"
cd /tmp
OS="$(uname -s)"
ARCH="$(uname -m)"
ASSET=""
case "$OS-$ARCH" in
  Darwin-arm64) ASSET="Xray-macos-arm64-v8a.zip" ;;
  Darwin-x86_64) ASSET="Xray-macos-64.zip" ;;
  Linux-x86_64|Linux-amd64) ASSET="Xray-linux-64.zip" ;;
  Linux-aarch64|Linux-arm64) ASSET="Xray-linux-arm64-v8a.zip" ;;
  *) echo "unsupported platform $OS $ARCH"; exit 1 ;;
esac
VER="${XRAY_VERSION:-v25.8.3}"
URL="https://github.com/XTLS/Xray-core/releases/download/${VER}/${ASSET}"
echo "downloading $URL"
curl -fsSL -o xray.zip "$URL"
rm -rf xray-extract && mkdir xray-extract && unzip -o xray.zip -d xray-extract
install -m 755 xray-extract/xray "$ROOT/bin/xray"
cp xray-extract/geoip.dat xray-extract/geosite.dat "$ROOT/bin/" 2>/dev/null || true
rm -rf xray.zip xray-extract
echo "installed $ROOT/bin/xray"
"$ROOT/bin/xray" version
