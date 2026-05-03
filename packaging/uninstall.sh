#!/usr/bin/env bash
#
# Yaria Desktop App — Uninstall Script
# Usage: curl -fsSL https://yaria.live/uninstall.sh | bash
#
# Removes the Yaria desktop app from Linux.
# No root/sudo required.
#

set -euo pipefail

INSTALL_DIR="$HOME/.local/bin"
DESKTOP_DIR="$HOME/.local/share/applications"
ICON_DIR="$HOME/.local/share/icons/hicolor"
USER_DATA_DIR="$HOME/.yaria"
USER_CONFIG_DIR="$HOME/.config/yaria"
MANTOREX_CONFIG_DIR="$HOME/.config/mantorex"

# Also check legacy install location
LEGACY_DIR="/opt/yaria"

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
echo -e "  ${PURPLE}Yaria Desktop App Uninstaller${NC}"
echo ""

# --- Remove binary ---
if [ -f "$INSTALL_DIR/yaria-app" ]; then
    rm -f "$INSTALL_DIR/yaria-app"
    ok "Removed ${INSTALL_DIR}/yaria-app"
else
    info "Binary not found in ${INSTALL_DIR}"
fi

# --- Remove legacy install ---
if [ -d "$LEGACY_DIR" ]; then
    warn "Legacy install found at ${LEGACY_DIR}"
    if [ -w "$LEGACY_DIR" ]; then
        rm -rf "$LEGACY_DIR"
        ok "Removed ${LEGACY_DIR}"
    else
        warn "Cannot remove ${LEGACY_DIR} (run: sudo rm -rf ${LEGACY_DIR})"
    fi
fi

# --- Remove legacy symlink ---
if [ -L "/usr/local/bin/yaria-app" ]; then
    if [ -w "/usr/local/bin/yaria-app" ]; then
        rm -f "/usr/local/bin/yaria-app"
        ok "Removed /usr/local/bin/yaria-app"
    fi
fi

# --- Remove desktop entry ---
if [ -f "$DESKTOP_DIR/yaria.desktop" ]; then
    rm -f "$DESKTOP_DIR/yaria.desktop"
    ok "Removed desktop entry"
fi

# Remove legacy system desktop entry
if [ -f "/usr/share/applications/yaria.desktop" ] && [ -w "/usr/share/applications/yaria.desktop" ]; then
    rm -f "/usr/share/applications/yaria.desktop"
    ok "Removed system desktop entry"
fi

# --- Remove icons ---
REMOVED_ICONS=0
for size in 16 32 48 64 128 256 512; do
    ICON="${ICON_DIR}/${size}x${size}/apps/yaria.png"
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
    gtk-update-icon-cache -f -t "$ICON_DIR" 2>/dev/null || true
fi
if command -v update-desktop-database &>/dev/null; then
    update-desktop-database "$DESKTOP_DIR" 2>/dev/null || true
fi

# --- Ask about user data ---
echo ""
echo -e "${YELLOW}Do you also want to remove user data?${NC}"
echo ""
echo "  This includes:"
echo "    - Download history, media database, thumbnails"
echo "    - License file, cookies cache"
echo "    - Settings and configuration"
echo ""
echo "  Directories:"
echo "    ${USER_DATA_DIR}"
echo "    ${USER_CONFIG_DIR}"
echo "    ${MANTOREX_CONFIG_DIR}"
echo ""

if [ -t 0 ]; then
    read -p "  Remove user data? [y/N]: " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf "$USER_DATA_DIR" 2>/dev/null || true
        rm -rf "$USER_CONFIG_DIR" 2>/dev/null || true
        rm -rf "$MANTOREX_CONFIG_DIR" 2>/dev/null || true
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
echo -e "  ${GREEN}Yaria has been uninstalled.${NC}"
echo ""
