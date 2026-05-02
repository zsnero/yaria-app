#!/usr/bin/env bash
#
# Build Yaria Desktop App release tarballs.
# Run this locally (where pro files exist) to produce distributable packages.
#
# Usage: ./packaging/build-release.sh
#
# Output: packaging/dist/
#   yaria-app-linux-amd64.tar.gz
#   yaria-app-linux-arm64.tar.gz
#

set -euo pipefail

cd "$(dirname "$0")/.."
PROJECT_DIR="$(pwd)"
DIST_DIR="${PROJECT_DIR}/packaging/dist"
VERSION="1.0.0"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

info() { echo -e "${CYAN}[BUILD]${NC} $1"; }
ok()   { echo -e "${GREEN}[OK]${NC} $1"; }

echo ""
echo -e "${CYAN}Building Yaria Desktop App v${VERSION}${NC}"
echo ""

# Check wails is installed
if ! command -v wails &>/dev/null; then
    echo -e "${RED}[ERROR]${NC} Wails CLI not found. Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    exit 1
fi

mkdir -p "$DIST_DIR"

# --- Build for Linux amd64 ---
build_linux() {
    local arch=$1
    local goarch=$2
    
    info "Building linux/${arch}..."
    
    local build_dir=$(mktemp -d)
    local binary_name="yaria-app"
    
    # Build with Wails (pro tag)
    CGO_ENABLED=1 GOOS=linux GOARCH=${goarch} \
        wails build -tags pro -o "${binary_name}" 2>&1 | tail -3
    
    # Find the built binary
    local built_bin="${PROJECT_DIR}/build/bin/${binary_name}"
    if [ ! -f "$built_bin" ]; then
        echo -e "${RED}[ERROR]${NC} Build failed for linux/${arch}"
        rm -rf "$build_dir"
        return 1
    fi
    
    # Create tarball with binary + icons
    local tarball="${DIST_DIR}/yaria-app-linux-${arch}.tar.gz"
    
    mkdir -p "${build_dir}/yaria-app"
    cp "$built_bin" "${build_dir}/yaria-app/${binary_name}"
    chmod 755 "${build_dir}/yaria-app/${binary_name}"
    cp -r "${PROJECT_DIR}/packaging/icons" "${build_dir}/yaria-app/icons"
    
    tar -czf "$tarball" -C "$build_dir" "yaria-app"
    rm -rf "$build_dir"
    
    local size=$(du -h "$tarball" | cut -f1)
    ok "Built: yaria-app-linux-${arch}.tar.gz (${size})"
}

# Build amd64
build_linux "amd64" "amd64"

# Build arm64 (cross-compile -- needs cross-compilation toolchain)
# Uncomment if you have aarch64-linux-gnu-gcc installed:
# CC=aarch64-linux-gnu-gcc build_linux "arm64" "arm64"

# --- Write version file ---
echo "$VERSION" > "${DIST_DIR}/latest.txt"
ok "Version file: ${VERSION}"

# --- Summary ---
echo ""
echo -e "${GREEN}Release artifacts:${NC}"
ls -lh "${DIST_DIR}/"
echo ""
echo "Upload these to yaria.live/download/"
echo ""
echo "Server directory structure:"
echo "  yaria.live/download/"
echo "    ├── latest.txt"
echo "    ├── yaria-app-linux-amd64.tar.gz"
echo "    ├── yaria-app-linux-arm64.tar.gz"
echo "    ├── install.sh  (copy from packaging/install.sh)"
echo "    ├── uninstall.sh  (copy from packaging/uninstall.sh)"
echo "    └── icons/"
echo "        ├── yaria-48.png"
echo "        ├── yaria-128.png"
echo "        └── yaria-256.png"
echo ""
