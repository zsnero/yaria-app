//go:build !linux

package main

// ensureX11ForMpv is a no-op on non-Linux platforms (the libmpv embed is
// Linux/X11 only).
func ensureX11ForMpv() {}
