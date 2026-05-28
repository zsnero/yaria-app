package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"yaria/pkg/appconfig"
)

// corsMiddleware wraps an http.Handler with full CORS support.
// This is required because the Wails frontend runs at a different origin
// (wails://wails/ or http://wails.localhost:PORT) than the stream servers
// (http://127.0.0.1:{random_port}). Without proper CORS preflight handling,
// crossOrigin='anonymous' on <video> elements will fail, and the Web Audio
// API (used for audio boost) will silently mute audio.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range, Content-Length, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// expandTilde expands a leading ~ in a path to the user's home directory.
// Go doesn't perform shell expansion, so "~/Videos" would create a literal "~" folder.
func expandTilde(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

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
