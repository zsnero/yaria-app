#!/usr/bin/env bash
# Shared Linux dependency helpers for Yaria (sourced by install.sh + yaria-launcher.sh).
# Covers Arch, Debian/Ubuntu, Fedora/RHEL, openSUSE, and common alternatives.

yaria_have_lib() {
  local needle="$1"
  if ldconfig -p 2>/dev/null | grep -q "$needle"; then
    return 0
  fi
  shift || true
  local p
  for p in "$@"; do
    [ -n "$p" ] && [ -e "$p" ] && return 0
  done
  return 1
}

yaria_have_webkit() {
  yaria_have_lib 'libwebkit2gtk-4\.1\.so' \
    /usr/lib/libwebkit2gtk-4.1.so.0 \
    /usr/lib64/libwebkit2gtk-4.1.so.0 \
    /usr/lib/x86_64-linux-gnu/libwebkit2gtk-4.1.so.0 \
    /usr/lib/aarch64-linux-gnu/libwebkit2gtk-4.1.so.0 \
    /usr/lib/arm-linux-gnueabihf/libwebkit2gtk-4.1.so.0 \
    /lib/x86_64-linux-gnu/libwebkit2gtk-4.1.so.0 \
    /lib64/libwebkit2gtk-4.1.so.0
}

yaria_have_gtk3() {
  yaria_have_lib 'libgtk-3\.so' \
    /usr/lib/libgtk-3.so.0 \
    /usr/lib64/libgtk-3.so.0 \
    /usr/lib/x86_64-linux-gnu/libgtk-3.so.0
}

yaria_run_admin() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
    return $?
  fi
  if command -v pkexec >/dev/null 2>&1; then
    pkexec "$@" && return 0
  fi
  if command -v sudo >/dev/null 2>&1; then
    # -n fails fast if passwordless not configured; then retry interactive
    sudo -n "$@" 2>/dev/null && return 0
    sudo "$@" && return 0
  fi
  return 1
}

yaria_os_id() {
  if [ -r /etc/os-release ]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    echo "${ID:-unknown}"
    return
  fi
  echo "unknown"
}

yaria_os_like() {
  if [ -r /etc/os-release ]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    echo "${ID_LIKE:-} ${ID:-}"
    return
  fi
  echo ""
}

# Install WebKitGTK 4.1 + GTK3 for the current distro family.
# Returns 0 on success.
yaria_install_webkit() {
  echo "Yaria needs WebKitGTK to display the interface."
  echo "Installing required system packages (one-time)…"

  local LIKE
  LIKE="$(yaria_os_like | tr '[:upper:]' '[:lower:]')"

  # --- Arch family (Arch, CachyOS, Manjaro, EndeavourOS, Garuda, …) ---
  if command -v pacman >/dev/null 2>&1; then
    yaria_run_admin pacman -S --noconfirm --needed webkit2gtk-4.1 gtk3 && return 0
    yaria_run_admin pacman -S --noconfirm --needed webkit2gtk gtk3 && return 0
  fi

  # --- Debian family (Debian, Ubuntu, Mint, Pop!, elementary, Zorin, …) ---
  if command -v apt-get >/dev/null 2>&1; then
    yaria_run_admin apt-get update -y || true
    # Prefer 4.1 (matches our binary); fall back to transitional names
    yaria_run_admin apt-get install -y libwebkit2gtk-4.1-0 libgtk-3-0 && return 0
    yaria_run_admin apt-get install -y libwebkit2gtk-4.1-0 libgtk-3-0t64 && return 0
    yaria_run_admin apt-get install -y gir1.2-webkit2-4.1 libgtk-3-0 && return 0
    # Last resort older packages (may not match soname — still try)
    yaria_run_admin apt-get install -y libwebkit2gtk-4.0-37 libgtk-3-0 && return 0
  fi

  # --- Fedora / RHEL / CentOS / Rocky / Alma / Nobara ---
  if command -v dnf >/dev/null 2>&1; then
    yaria_run_admin dnf install -y webkit2gtk4.1 gtk3 && return 0
    yaria_run_admin dnf install -y webkit2gtk4.1-devel gtk3 && return 0
  fi
  if command -v microdnf >/dev/null 2>&1; then
    yaria_run_admin microdnf install -y webkit2gtk4.1 gtk3 && return 0
  fi
  if command -v yum >/dev/null 2>&1; then
    yaria_run_admin yum install -y webkit2gtk4.1 gtk3 && return 0
  fi

  # --- openSUSE / SLES ---
  if command -v zypper >/dev/null 2>&1; then
    yaria_run_admin zypper --non-interactive install \
      libwebkit2gtk-4_1-0 typelib-1_0-WebKit2-4_1 libgtk-3-0 && return 0
    yaria_run_admin zypper --non-interactive install webkit2gtk3-soup2 libgtk-3-0 && return 0
    yaria_run_admin zypper --non-interactive install libwebkit2gtk-4_0-37 libgtk-3-0 && return 0
  fi

  # --- Solus ---
  if command -v eopkg >/dev/null 2>&1; then
    yaria_run_admin eopkg install -y libwebkit-gtk gtk3 && return 0
  fi

  # --- Void Linux ---
  if command -v xbps-install >/dev/null 2>&1; then
    yaria_run_admin xbps-install -Sy webkit2gtk gtk+3 && return 0
  fi

  # --- Gentoo (slow; best-effort) ---
  if command -v emerge >/dev/null 2>&1 && echo "$LIKE" | grep -q gentoo; then
    yaria_run_admin emerge --ask=n net-libs/webkit-gtk:4.1 x11-libs/gtk+:3 && return 0
  fi

  # --- Nix (user profile; no root) ---
  if command -v nix-env >/dev/null 2>&1; then
    nix-env -iA nixpkgs.webkitgtk_4_1 nixpkgs.gtk3 2>/dev/null && return 0
  fi

  return 1
}

yaria_webkit_install_help() {
  cat <<'EOF'
Yaria could not install WebKitGTK automatically.

Please install it, then open Yaria again:

  Arch / CachyOS / Manjaro:
    sudo pacman -S webkit2gtk-4.1 gtk3

  Debian / Ubuntu / Mint / Pop!_OS:
    sudo apt update
    sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0

  Fedora / RHEL / Nobara:
    sudo dnf install webkit2gtk4.1 gtk3

  openSUSE:
    sudo zypper install libwebkit2gtk-4_1-0 libgtk-3-0

  Void:
    sudo xbps-install -S webkit2gtk gtk+3
EOF
}

yaria_show_error_gui() {
  local msg="$1"
  if command -v zenity >/dev/null 2>&1; then
    zenity --error --title="Yaria" --width=480 --text="$msg" 2>/dev/null || true
  elif command -v kdialog >/dev/null 2>&1; then
    kdialog --error "$msg" 2>/dev/null || true
  elif command -v notify-send >/dev/null 2>&1; then
    notify-send -u critical "Yaria" "Missing WebKitGTK — open a terminal for install help"
  fi
}
