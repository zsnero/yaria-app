#!/usr/bin/env bash
#
# Yaria Desktop App — Install Script
# Usage: curl -fsSL https://yaria.live/install.sh | bash
#
# Installs the Yaria desktop app on Linux (amd64/arm64).
# Requires: wget or curl, tar, and root/sudo access.
#

set -euo pipefail

APP_NAME="yaria"
APP_DISPLAY="Yaria"
INSTALL_DIR="/opt/yaria"
BIN_LINK="/usr/local/bin/yaria-app"
DESKTOP_FILE="/usr/share/applications/yaria.desktop"
ICON_BASE="/usr/share/icons/hicolor"

# Download from yaria.live (self-hosted pro binary)
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
echo -e "${PURPLE}╔══════════════════════════════════════╗${NC}"
echo -e "${PURPLE}║${NC}     ${CYAN}Yaria Desktop App Installer${NC}     ${PURPLE}║${NC}"
echo -e "${PURPLE}║${NC}     Video Downloader + Media Center  ${PURPLE}║${NC}"
echo -e "${PURPLE}╚══════════════════════════════════════╝${NC}"
echo ""

# --- Check root ---
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo &>/dev/null; then
        info "This script needs root access. Re-running with sudo..."
        exec sudo bash "$0" "$@"
    else
        error "This script must be run as root (or with sudo)."
    fi
fi

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
info "URL: ${DOWNLOAD_URL}"

# --- Download and extract ---
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

$DL_OUT "$TMP_DIR/yaria.tar.gz" "$DOWNLOAD_URL"
info "Extracting..."
tar -xzf "$TMP_DIR/yaria.tar.gz" -C "$TMP_DIR"

# Find the binary (it might be in a subdirectory)
BINARY=$(find "$TMP_DIR" -name "yaria-app" -o -name "YariaApp" -o -name "yaria" -type f -executable | head -1)
if [ -z "$BINARY" ]; then
    # Try any executable
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

# Create symlink
ln -sf "$INSTALL_DIR/yaria-app" "$BIN_LINK"
ok "Binary installed: ${INSTALL_DIR}/yaria-app"

# --- Install desktop entry ---
info "Installing desktop entry..."
cat > "$DESKTOP_FILE" << 'EOF'
[Desktop Entry]
Name=Yaria
GenericName=Media Center & Video Downloader
Comment=Download videos, manage local media, stream torrents, and serve to your devices
Exec=/opt/yaria/yaria-app
Icon=yaria
Terminal=false
Type=Application
Categories=AudioVideo;Video;Network;
Keywords=video;download;torrent;media;stream;
StartupWMClass=Yaria
MimeType=x-scheme-handler/magnet;
EOF
chmod 644 "$DESKTOP_FILE"
ok "Desktop entry installed"

# --- Install icons ---
info "Installing icons..."

# Check if icons are bundled in the archive
ICON_SRC=$(find "$TMP_DIR" -name "yaria-256.png" -o -name "icon.png" | head -1)

if [ -n "$ICON_SRC" ]; then
    ICON_DIR=$(dirname "$ICON_SRC")
    for size in 16 32 48 64 128 256 512; do
        ICON_FILE="${ICON_DIR}/yaria-${size}.png"
        if [ -f "$ICON_FILE" ]; then
            DEST_DIR="${ICON_BASE}/${size}x${size}/apps"
            mkdir -p "$DEST_DIR"
            cp "$ICON_FILE" "$DEST_DIR/yaria.png"
        fi
    done
else
    # Download icons from yaria.live
    warn "Icon files not found in archive. Downloading..."
    for size in 48 128 256; do
        DEST_DIR="${ICON_BASE}/${size}x${size}/apps"
        mkdir -p "$DEST_DIR"
        $DL_OUT "$DEST_DIR/yaria.png" "${BASE_URL}/icons/yaria-${size}.png" 2>/dev/null || true
    done
fi

# Update icon cache
if command -v gtk-update-icon-cache &>/dev/null; then
    gtk-update-icon-cache -f -t "$ICON_BASE" 2>/dev/null || true
fi
if command -v update-desktop-database &>/dev/null; then
    update-desktop-database /usr/share/applications 2>/dev/null || true
fi
ok "Icons installed"

# --- Install WebKitGTK dependency hint ---
echo ""
info "Checking WebKitGTK..."
if ldconfig -p 2>/dev/null | grep -q "libwebkit2gtk-4.0"; then
    ok "WebKitGTK found"
elif ldconfig -p 2>/dev/null | grep -q "libwebkit2gtk-4.1"; then
    ok "WebKitGTK 4.1 found"
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
echo -e "${GREEN}╔══════════════════════════════════════╗${NC}"
echo -e "${GREEN}║${NC}    ${CYAN}Yaria installed successfully!${NC}     ${GREEN}║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════╝${NC}"
echo ""
echo -e "  Launch from your app drawer or run: ${CYAN}yaria-app${NC}"
echo ""
echo -e "  ${PURPLE}Activate Pro:${NC} Enter your license key in Settings"
echo -e "  ${PURPLE}Get a key:${NC}    https://yaria.live"
echo ""
