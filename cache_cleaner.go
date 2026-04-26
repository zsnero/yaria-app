package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cleanOldCache removes torrent data files older than maxAge from the data directory.
// Runs silently in background -- errors are ignored.
func cleanOldCache(maxAge time.Duration) {
	dataDir := getDataDir()
	if dataDir == "" {
		return
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-maxAge)

	for _, entry := range entries {
		path := filepath.Join(dataDir, entry.Name())

		if entry.IsDir() {
			// Check if directory is old and contains only media/torrent files
			info, err := entry.Info()
			if err != nil || info.ModTime().After(cutoff) {
				continue
			}
			// Remove old torrent data directories
			cleanOldDirectory(path, cutoff)
		} else {
			// Remove old .part files, .torrent files, etc.
			info, err := entry.Info()
			if err != nil {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			isTemp := ext == ".part" || ext == ".torrent" || strings.HasSuffix(entry.Name(), ".torrent.db") ||
				strings.HasSuffix(entry.Name(), ".torrent.bolt.db")
			if isTemp && info.ModTime().Before(cutoff) {
				os.Remove(path)
			}
		}
	}
}

// cleanOldDirectory removes a directory if all its files are older than cutoff.
func cleanOldDirectory(dir string, cutoff time.Time) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	allOld := true
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(cutoff) {
			allOld = false
			break
		}
	}

	if allOld && len(entries) > 0 {
		os.RemoveAll(dir)
	}
}

// cleanStreamCache removes the data from the just-finished stream.
// Called after user stops watching.
func cleanStreamCache(torrentName string) {
	if torrentName == "" {
		return
	}
	dataDir := getDataDir()
	if dataDir == "" {
		return
	}
	torrentDir := filepath.Join(dataDir, torrentName)
	if info, err := os.Stat(torrentDir); err == nil && info.IsDir() {
		os.RemoveAll(torrentDir)
	}
}

// startCacheCleaner runs cache cleanup on startup and periodically.
func startCacheCleaner() {
	// Clean on startup: remove files older than 24 hours
	go func() {
		time.Sleep(5 * time.Second) // wait for app to fully start
		cleanOldCache(24 * time.Hour)
	}()

	// Periodic cleanup every 6 hours
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			cleanOldCache(24 * time.Hour)
		}
	}()
}
