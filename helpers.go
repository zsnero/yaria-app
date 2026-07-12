package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
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
	return expandTilde(dataDir)
}

// binaryName returns name with .exe suffix on Windows.
func binaryName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// appDataDir returns the per-user data directory for Yaria (~/.yaria).
func appDataDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".yaria")
}

// openWithDefaultApp opens a file with the OS default handler.
func openWithDefaultApp(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// Use rundll32 so paths with spaces work without a shell.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	setProcAttrDetached(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

// openFolder opens a directory in the system file manager.
func openFolder(dir string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	setProcAttrDetached(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

// openMediaFile launches a media file in a known player, or the system default.
// Returns the player name used ("system" for the OS default handler).
func openMediaFile(filePath string) (string, error) {
	var players []string
	switch runtime.GOOS {
	case "darwin":
		players = []string{"mpv", "vlc", "iina"}
	case "windows":
		players = []string{"mpv", "vlc"}
	default:
		players = []string{"mpv", "vlc", "celluloid", "totem"}
	}
	for _, name := range players {
		if p, err := exec.LookPath(name); err == nil {
			cmd := exec.Command(p, filePath)
			setProcAttrDetached(cmd)
			if err := cmd.Start(); err == nil {
				go cmd.Wait()
				return name, nil
			}
		}
	}
	if err := openWithDefaultApp(filePath); err != nil {
		return "", fmt.Errorf("no media player found: %w", err)
	}
	return "system", nil
}

// isSafeToDeleteDir rejects filesystem roots so we never RemoveAll("/") or "C:\".
func isSafeToDeleteDir(dir string) bool {
	if dir == "" || dir == "." {
		return false
	}
	clean := filepath.Clean(dir)
	if clean == string(filepath.Separator) {
		return false
	}
	// Windows drive root: "C:\" or "C:"
	if vol := filepath.VolumeName(clean); vol != "" {
		rest := strings.TrimPrefix(clean, vol)
		if rest == "" || rest == string(filepath.Separator) {
			return false
		}
	}
	return true
}
