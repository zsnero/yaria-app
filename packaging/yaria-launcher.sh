#!/usr/bin/env bash
# Yaria launcher — ensures WebKitGTK is present on any Linux distro, then starts the app.
# Installed as ~/.local/bin/yaria-app ; real binary is yaria-app.bin
set -euo pipefail

SELF="$(readlink -f "${BASH_SOURCE[0]}" 2>/dev/null || realpath "${BASH_SOURCE[0]}" 2>/dev/null || echo "$0")"
DIR="$(cd "$(dirname "$SELF")" && pwd)"
BIN=""
# Prefer real ELF binary; never exec a shell script as the app
for cand in "$DIR/yaria-app.bin" "$DIR/yaria-app-real"; do
  if [ -x "$cand" ] && ! head -1 "$cand" 2>/dev/null | grep -q '^#!'; then
    BIN="$cand"
    break
  fi
done

# Load shared helpers (packaged next to us, or alongside install.sh)
for cand in \
  "$DIR/linux-deps.sh" \
  "$DIR/../share/yaria/linux-deps.sh" \
  "$(dirname "$SELF")/linux-deps.sh"
do
  if [ -f "$cand" ]; then
    # shellcheck disable=SC1090
    . "$cand"
    break
  fi
done

# Inline fallback if linux-deps.sh missing (older tarballs)
if ! command -v yaria_have_webkit >/dev/null 2>&1; then
  yaria_have_webkit() {
    ldconfig -p 2>/dev/null | grep -q 'libwebkit2gtk-4\.1\.so' && return 0
    for p in /usr/lib/libwebkit2gtk-4.1.so.0 /usr/lib64/libwebkit2gtk-4.1.so.0 \
      /usr/lib/x86_64-linux-gnu/libwebkit2gtk-4.1.so.0; do
      [ -e "$p" ] && return 0
    done
    return 1
  }
  yaria_install_webkit() { return 1; }
  yaria_webkit_install_help() { echo "Install libwebkit2gtk-4.1 via your package manager."; }
  yaria_show_error_gui() { :; }
fi

if ! yaria_have_webkit; then
  if ! yaria_install_webkit; then
    MSG="$(yaria_webkit_install_help)"
    yaria_show_error_gui "$MSG"
    echo "$MSG" >&2
    exit 1
  fi
  if [ "$(id -u)" -eq 0 ]; then
    ldconfig 2>/dev/null || true
  fi
  if ! yaria_have_webkit; then
    MSG="$(yaria_webkit_install_help)"
    yaria_show_error_gui "$MSG"
    echo "$MSG" >&2
    exit 1
  fi
fi

if [ -z "$BIN" ] || [ ! -x "$BIN" ]; then
  MSG="Yaria binary (yaria-app.bin) is missing next to the launcher.
Reinstall with:
  curl -fsSL https://yaria.live/install.sh | bash"
  yaria_show_error_gui "$MSG" 2>/dev/null || true
  echo "$MSG" >&2
  exit 1
fi

exec "$BIN" "$@"
