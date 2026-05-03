#!/usr/bin/env bash
#
# Yaria Desktop App — Install Script
# Usage: curl -fsSL https://yaria.live/install.sh | bash
#
# Installs the Yaria desktop app on Linux (amd64/arm64).
# No root/sudo required -- installs to ~/.local/
#

set -euo pipefail

APP_NAME="yaria"
INSTALL_DIR="$HOME/.local/bin"
DESKTOP_DIR="$HOME/.local/share/applications"
ICON_DIR="$HOME/.local/share/icons/hicolor"

# Download from yaria.live
BASE_URL="https://yaria.live/download"
VERSION_URL="${BASE_URL}/latest.txt"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC} $1"; }
ok()    { echo -e "${GREEN}[OK]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# --- Header ---
echo ""
echo -e "  ${PURPLE}Yaria Desktop App Installer${NC}"
echo -e "  ${CYAN}Video Downloader + Media Center${NC}"
echo ""

# --- Detect architecture ---
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH_SUFFIX="amd64" ;;
    aarch64) ARCH_SUFFIX="arm64" ;;
    arm64)   ARCH_SUFFIX="arm64" ;;
    *)       error "Unsupported architecture: $ARCH. Yaria supports x86_64 and aarch64." ;;
esac

info "Detected architecture: ${ARCH} (${ARCH_SUFFIX})"

# --- Detect download tool ---
if command -v curl &>/dev/null; then
    DL="curl -fsSL"
    DL_OUT="curl -fsSL -o"
elif command -v wget &>/dev/null; then
    DL="wget -qO-"
    DL_OUT="wget -qO"
else
    error "Neither curl nor wget found. Please install one and try again."
fi

# --- Get latest version ---
info "Checking latest version..."

VERSION=$($DL "$VERSION_URL" 2>/dev/null | tr -d '[:space:]') || true

if [ -z "$VERSION" ]; then
    VERSION="latest"
fi

DOWNLOAD_URL="${BASE_URL}/yaria-app-linux-${ARCH_SUFFIX}.tar.gz"

info "Downloading Yaria ${VERSION}..."

# --- Download and extract ---
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

$DL_OUT "$TMP_DIR/yaria.tar.gz" "$DOWNLOAD_URL"
info "Extracting..."
tar -xzf "$TMP_DIR/yaria.tar.gz" -C "$TMP_DIR"

# Find the binary
BINARY=$(find "$TMP_DIR" -name "yaria-app" -type f -executable | head -1)
if [ -z "$BINARY" ]; then
    BINARY=$(find "$TMP_DIR" -type f -executable ! -name "*.sh" | head -1)
fi

if [ -z "$BINARY" ]; then
    error "Could not find the Yaria binary in the archive."
fi

# --- Install binary ---
info "Installing to ${INSTALL_DIR}..."
mkdir -p "$INSTALL_DIR"
cp "$BINARY" "$INSTALL_DIR/yaria-app"
chmod 755 "$INSTALL_DIR/yaria-app"
ok "Binary installed: ${INSTALL_DIR}/yaria-app"

# --- Ensure ~/.local/bin is in PATH ---
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    warn "$HOME/.local/bin is not in your PATH"
    echo ""
    echo "  Add it by running:"
    echo -e "    ${CYAN}echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc${NC}"
    echo -e "    ${CYAN}source ~/.bashrc${NC}"
    echo ""
fi

# --- Install desktop entry ---
info "Installing desktop entry..."
mkdir -p "$DESKTOP_DIR"
cat > "$DESKTOP_DIR/yaria.desktop" << EOF
[Desktop Entry]
Name=Yaria
GenericName=Media Center & Video Downloader
Comment=Download videos, manage local media, stream torrents, and serve to your devices
Exec=$INSTALL_DIR/yaria-app
Icon=yaria
Terminal=false
Type=Application
Categories=AudioVideo;Video;Network;
Keywords=video;download;torrent;media;stream;
StartupWMClass=Yaria
MimeType=x-scheme-handler/magnet;
EOF
chmod 644 "$DESKTOP_DIR/yaria.desktop"
ok "Desktop entry installed"

# --- Install icons ---
info "Installing icons..."

ICON_SRC=$(find "$TMP_DIR" -name "yaria-256.png" | head -1)

if [ -n "$ICON_SRC" ]; then
    ICON_SRC_DIR=$(dirname "$ICON_SRC")
    for size in 16 32 48 64 128 256 512; do
        ICON_FILE="${ICON_SRC_DIR}/yaria-${size}.png"
        if [ -f "$ICON_FILE" ]; then
            DEST_DIR="${ICON_DIR}/${size}x${size}/apps"
            mkdir -p "$DEST_DIR"
            cp "$ICON_FILE" "$DEST_DIR/yaria.png"
        fi
    done
else
    for size in 48 128 256; do
        DEST_DIR="${ICON_DIR}/${size}x${size}/apps"
        mkdir -p "$DEST_DIR"
        $DL_OUT "$DEST_DIR/yaria.png" "${BASE_URL}/icons/yaria-${size}.png" 2>/dev/null || true
    done
fi

# Update icon cache
if command -v gtk-update-icon-cache &>/dev/null; then
    gtk-update-icon-cache -f -t "$ICON_DIR" 2>/dev/null || true
fi
if command -v update-desktop-database &>/dev/null; then
    update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
fi
ok "Icons installed"

# --- Check WebKitGTK ---
echo ""
info "Checking WebKitGTK..."
if ldconfig -p 2>/dev/null | grep -q "libwebkit2gtk-4"; then
    ok "WebKitGTK found"
else
    warn "WebKitGTK not detected. Yaria requires WebKitGTK to run."
    echo ""
    echo -e "  Install it with:"
    echo -e "    ${CYAN}Arch:${NC}     sudo pacman -S webkit2gtk"
    echo -e "    ${CYAN}Debian:${NC}   sudo apt install libwebkit2gtk-4.0-dev"
    echo -e "    ${CYAN}Fedora:${NC}   sudo dnf install webkit2gtk4.0-devel"
    echo -e "    ${CYAN}openSUSE:${NC} sudo zypper install libwebkit2gtk3-devel"
    echo ""
fi

# --- Done ---
echo ""
echo -e "  ${GREEN}Yaria installed successfully!${NC}"
echo ""
echo -e "  Launch from your app drawer or run: ${CYAN}yaria-app${NC}"
echo ""
echo -e "  ${PURPLE}Activate Pro:${NC} Enter your license key in Settings"
echo -e "  ${PURPLE}Get a key:${NC}    https://yaria.live"
echo ""
