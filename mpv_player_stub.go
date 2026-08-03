//go:build !linux

package main

import (
	"context"
	"fmt"
)

func mpvPlatformAvailable() (bool, string) {
	return false, "Native libmpv player is only available on Linux in this build"
}

func mpvPlatformStart(ctx context.Context) error {
	return fmt.Errorf("libmpv not available on this platform")
}

func mpvPlatformLoad(pathOrURL string) error {
	return fmt.Errorf("libmpv not available")
}

func mpvPlatformSetBounds(x, y, w, h float64) error { return nil }

func mpvPlatformSetVisible(visible bool) {}

func mpvPlatformSetPause(pause bool) error { return fmt.Errorf("libmpv not available") }

func mpvPlatformTogglePause() error { return fmt.Errorf("libmpv not available") }

func mpvPlatformSeek(seconds float64) error { return fmt.Errorf("libmpv not available") }

func mpvPlatformSetVolume(vol float64) error { return fmt.Errorf("libmpv not available") }

func mpvPlatformGetTime() float64 { return 0 }

func mpvPlatformGetDuration() float64 { return 0 }

func mpvPlatformIsPaused() bool { return true }

func mpvPlatformStop() {}
