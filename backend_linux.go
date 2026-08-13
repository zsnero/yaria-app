//go:build linux

package main

import (
	"os"
	"os/exec"
	"strings"
)

// x11BackendActive reports whether GTK will use the X11 backend first.
// GDK_BACKEND is a comma-separated priority list (e.g. "wayland,x11,*"), so the
// first entry decides which backend GTK tries first.
func x11BackendActive() bool {
	v := os.Getenv("GDK_BACKEND")
	if v == "" {
		return false
	}
	first := strings.TrimSpace(strings.Split(v, ",")[0])
	return first == "x11"
}

// stripEnv removes all entries with the given key (avoid duplicate GDK_BACKEND
// values — getenv may return the first one).
func stripEnv(env []string, key string) []string {
	prefix := key + "="
	out := env[:0]
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return out
}

// ensureX11ForMpv re-executes the process with GDK_BACKEND=x11 when running on
// a Wayland session. The libmpv embed path is X11-only (it looks the window up
// in _NET_CLIENT_LIST and creates an X11 child window), so a native Wayland
// window can never host it. Called before any GUI/wails initialization.
func ensureX11ForMpv() {
	if x11BackendActive() {
		return
	}
	if os.Getenv("WAYLAND_DISPLAY") == "" {
		return // not a Wayland session — GTK will use X11 anyway
	}

	exe, err := os.Executable()
	if err != nil {
		return
	}
	env := stripEnv(os.Environ(), "GDK_BACKEND")
	env = append(env, "GDK_BACKEND=x11")
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	code := 0
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			code = 1
		}
	}
	os.Exit(code)
}
