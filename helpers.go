package main

import (
	"os"
	"path/filepath"
	"strings"

	"yaria/pkg/appconfig"
)

// getDataDir returns the configured Mantorex data directory, expanding ~ if needed.
// Shared between settings_service.go and stream_service.go (pro build).
func getDataDir() string {
	dataDir := appconfig.MantorexDataDir()
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, "Downloads", "Mantorex")
	}
	if strings.HasPrefix(dataDir, "~/") {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, dataDir[2:])
	}
	return dataDir
}
