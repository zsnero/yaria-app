package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// minFreeDiskBytes is the minimum free space required before starting
// downloads or streams (500 MB). Below this, operations fail with a clear message.
const minFreeDiskBytes = 500 * 1024 * 1024

// diskFullUserMsg is shown when the disk is full or nearly full.
const diskFullUserMsg = "Disk is full — free up space and try again. Unable to download or stream."

// ensureDiskSpace checks that path (or its nearest existing parent) has at least
// minBytes free. Returns a user-facing error message if not enough space.
func ensureDiskSpace(path string, minBytes uint64) error {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = home
	}
	checkPath := path
	for {
		if st, err := os.Stat(checkPath); err == nil && st.IsDir() {
			break
		}
		parent := filepath.Dir(checkPath)
		if parent == checkPath {
			break
		}
		checkPath = parent
	}
	free, err := diskFreeBytes(checkPath)
	if err != nil {
		// Don't block if we can't measure (permissions, exotic FS)
		return nil
	}
	if free < minBytes {
		return fmt.Errorf("%s (only %s free on %s)", diskFullUserMsg, formatBytes(int64(free)), checkPath)
	}
	return nil
}

// formatBytes formats a byte count for UI (shared by downloads, torrents, disk checks).
func formatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// isDiskFullError reports whether err looks like ENOSPC / disk full.
func isDiskFullError(err error) bool {
	if err == nil {
		return false
	}
	return isDiskFullMessage(err.Error())
}

func isDiskFullMessage(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "no space left") ||
		strings.Contains(m, "not enough space") ||
		strings.Contains(m, "disk full") ||
		strings.Contains(m, "enospc") ||
		strings.Contains(m, "there is not enough space") ||
		strings.Contains(m, "disk quota exceeded")
}
