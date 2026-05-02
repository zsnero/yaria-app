#!/usr/bin/env bash
#
# Yaria Desktop App — Uninstall Script
# Usage: curl -fsSL https://yaria.xyz/uninstall.sh | bash
#
# Removes the Yaria desktop app from Linux.
#

set -euo pipefail

INSTALL_DIR="/opt/yaria"
BIN_LINK="/usr/local/bin/yaria-app"
DESKTOP_FILE="/usr/share/applications/yaria.desktop"
ICON_BASE="/usr/share/icons/hicolor"
USER_DATA_DIR="$HOME/.yaria"
USER_CONFIG_DIR="$HOME/.config/yaria"
MANTOREX_CONFIG_DIR="$HOME/.config/mantorex"

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

echo ""
echo -e "${PURPLE}╔══════════════════════════════════════╗${NC}"
echo -e "${PURPLE}║${NC}    ${CYAN}Yaria Desktop App Uninstaller${NC}    ${PURPLE}║${NC}"
echo -e "${PURPLE}╚══════════════════════════════════════╝${NC}"
echo ""

# --- Check root ---
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo &>/dev/null; then
        info "This script needs root access. Re-running with sudo..."
        exec sudo bash "$0" "$@"
    else
        echo -e "${RED}[ERROR]${NC} This script must be run as root (or with sudo)."
        exit 1
    fi
fi

# --- Remove binary ---
if [ -d "$INSTALL_DIR" ]; then
    rm -rf "$INSTALL_DIR"
    ok "Removed ${INSTALL_DIR}"
else
    info "Binary directory not found (already removed?)"
fi

# --- Remove symlink ---
if [ -L "$BIN_LINK" ] || [ -f "$BIN_LINK" ]; then
    rm -f "$BIN_LINK"
    ok "Removed ${BIN_LINK}"
fi

# --- Remove desktop entry ---
if [ -f "$DESKTOP_FILE" ]; then
    rm -f "$DESKTOP_FILE"
    ok "Removed desktop entry"
fi

# --- Remove icons ---
REMOVED_ICONS=0
for size in 16 32 48 64 128 256 512; do
    ICON="${ICON_BASE}/${size}x${size}/apps/yaria.png"
    if [ -f "$ICON" ]; then
        rm -f "$ICON"
        REMOVED_ICONS=$((REMOVED_ICONS + 1))
    fi
done
if [ "$REMOVED_ICONS" -gt 0 ]; then
    ok "Removed ${REMOVED_ICONS} icon files"
fi

# Update caches
if command -v gtk-update-icon-cache &>/dev/null; then
    gtk-update-icon-cache -f -t "$ICON_BASE" 2>/dev/null || true
fi
if command -v update-desktop-database &>/dev/null; then
    update-desktop-database /usr/share/applications 2>/dev/null || true
fi

# --- Ask about user data ---
echo ""
echo -e "${YELLOW}Do you also want to remove user data?${NC}"
echo ""
echo "  This includes:"
echo "    - Download history, media database, thumbnails"
echo "    - License file, cookies cache"
echo "    - Settings and configuration"
echo "    - Mantorex library and torrent state"
echo ""
echo "  Directories:"
echo "    ${USER_DATA_DIR}"
echo "    ${USER_CONFIG_DIR}"
echo "    ${MANTOREX_CONFIG_DIR}"
echo ""

# Check if running non-interactively (piped from curl)
if [ -t 0 ]; then
    read -p "  Remove user data? [y/N]: " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Get the actual user's home (not root's)
        REAL_USER="${SUDO_USER:-$USER}"
        REAL_HOME=$(getent passwd "$REAL_USER" | cut -d: -f6)
        
        rm -rf "${REAL_HOME}/.yaria" 2>/dev/null || true
        rm -rf "${REAL_HOME}/.config/yaria" 2>/dev/null || true
        rm -rf "${REAL_HOME}/.config/mantorex" 2>/dev/null || true
        ok "Removed user data"
    else
        info "User data preserved"
    fi
else
    info "Running non-interactively. User data preserved."
    info "To remove manually: rm -rf ~/.yaria ~/.config/yaria ~/.config/mantorex"
fi

# --- Done ---
echo ""
echo -e "${GREEN}╔══════════════════════════════════════╗${NC}"
echo -e "${GREEN}║${NC}  ${CYAN}Yaria has been uninstalled.${NC}        ${GREEN}║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════╝${NC}"
echo ""
