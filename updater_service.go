package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	AppVersion    = "1.5.2"
	UpdateBaseURL = "https://yaria.live/download"
)

// UpdaterService handles checking for and applying app updates.
type UpdaterService struct {
	ctx       context.Context
	mu        sync.Mutex
	updating  bool
	progress  int
	statusMsg string
}

func NewUpdaterService() *UpdaterService {
	return &UpdaterService{}
}

func (u *UpdaterService) startup(ctx context.Context) {
	u.ctx = ctx
}

// GetVersion returns the current app version.
func (u *UpdaterService) GetVersion() string {
	return AppVersion
}

// CheckUpdate checks if a newer version is available.
func (u *UpdaterService) CheckUpdate() map[string]interface{} {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(UpdateBaseURL + "/latest.txt")
	if err != nil {
		return map[string]interface{}{"error": "Could not reach update server"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{"error": "Failed to read version info"}
	}

	latest := strings.TrimSpace(string(body))
	if latest == "" {
		return map[string]interface{}{"error": "No version info available"}
	}

	return map[string]interface{}{
		"current":   AppVersion,
		"latest":    latest,
		"available": latest != AppVersion,
	}
}

// Update downloads and installs the latest version.
func (u *UpdaterService) Update() map[string]interface{} {
	u.mu.Lock()
	if u.updating {
		u.mu.Unlock()
		return map[string]interface{}{"error": "Update already in progress"}
	}
	u.updating = true
	u.progress = 0
	u.statusMsg = "Starting update..."
	u.mu.Unlock()

	go u.runUpdate()
	return map[string]interface{}{"status": "started"}
}

// UpdateStatus returns current update progress.
func (u *UpdaterService) UpdateStatus() map[string]interface{} {
	u.mu.Lock()
	defer u.mu.Unlock()
	return map[string]interface{}{
		"updating": u.updating,
		"progress": u.progress,
		"message":  u.statusMsg,
	}
}

func (u *UpdaterService) runUpdate() {
	defer func() {
		u.mu.Lock()
		u.updating = false
		u.mu.Unlock()
	}()

	// Determine architecture
	arch := "amd64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}

	downloadURL := fmt.Sprintf("%s/yaria-app-linux-%s.tar.gz", UpdateBaseURL, arch)

	u.setStatus(5, "Downloading update...")

	// Download to temp file
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		u.setStatus(0, "Download failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		u.setStatus(0, fmt.Sprintf("Download failed: HTTP %d", resp.StatusCode))
		return
	}

	tmpFile, err := os.CreateTemp("", "yaria-update-*.tar.gz")
	if err != nil {
		u.setStatus(0, "Failed to create temp file")
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Download with progress
	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			tmpFile.Write(buf[:n])
			downloaded += int64(n)
			if totalSize > 0 {
				pct := int(float64(downloaded) / float64(totalSize) * 70) // 0-70% for download
				u.setStatus(5+pct, fmt.Sprintf("Downloading... %s / %s",
					formatUpdateBytes(downloaded), formatUpdateBytes(totalSize)))
			}
		}
		if readErr != nil {
			break
		}
	}
	tmpFile.Close()

	u.setStatus(75, "Extracting update...")

	// Extract tarball
	tmpDir, err := os.MkdirTemp("", "yaria-update-extract-")
	if err != nil {
		u.setStatus(0, "Failed to create extract directory")
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(tmpPath, tmpDir); err != nil {
		u.setStatus(0, "Extract failed: "+err.Error())
		return
	}

	// Find the new binary
	var newBinary string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if name == "yaria-app" || name == "YariaApp" {
			newBinary = path
			return filepath.SkipAll
		}
		return nil
	})

	if newBinary == "" {
		u.setStatus(0, "Binary not found in update package")
		return
	}

	u.setStatus(85, "Installing update...")

	// Get current binary path
	currentBin, err := os.Executable()
	if err != nil {
		u.setStatus(0, "Could not determine current binary path")
		return
	}
	currentBin, _ = filepath.EvalSymlinks(currentBin)

	// Try direct replace first (works if user owns the binary)
	oldBin := currentBin + ".old"
	os.Remove(oldBin)

	updated := false

	if err := os.Rename(currentBin, oldBin); err == nil {
		if err := copyFileForUpdate(newBinary, currentBin); err != nil {
			os.Rename(oldBin, currentBin)
			u.setStatus(0, "Failed to install new binary: "+err.Error())
			return
		}
		os.Chmod(currentBin, 0755)
		os.Remove(oldBin)
		updated = true
	}

	// If direct replace failed, install to user-local path
	if !updated {
		u.setStatus(85, "Installing to user directory...")
		home, _ := os.UserHomeDir()
		localBin := filepath.Join(home, ".local", "bin")
		os.MkdirAll(localBin, 0755)
		userBinary := filepath.Join(localBin, "yaria-app")

		if err := copyFileForUpdate(newBinary, userBinary); err != nil {
			u.setStatus(0, "Failed to install: "+err.Error())
			return
		}
		os.Chmod(userBinary, 0755)

		// Update desktop entry to point to user binary
		desktopFile := filepath.Join(home, ".local", "share", "applications", "yaria.desktop")
		os.MkdirAll(filepath.Dir(desktopFile), 0755)
		desktop := fmt.Sprintf(`[Desktop Entry]
Name=Yaria
GenericName=Media Center & Video Downloader
Comment=Download videos, manage local media, stream torrents
Exec=%s
Icon=yaria
Terminal=false
Type=Application
Categories=AudioVideo;Video;Network;
StartupWMClass=Yaria
`, userBinary)
		os.WriteFile(desktopFile, []byte(desktop), 0644)
		updated = true
	}

	if !updated {
		u.setStatus(0, "Update failed: could not replace binary")
		return
	}

	u.setStatus(100, "Update complete! Restart the app to use the new version.")

	// Emit event to frontend
	if u.ctx != nil {
		wailsRuntime.EventsEmit(u.ctx, "update-complete", nil)
	}
}

func (u *UpdaterService) setStatus(progress int, msg string) {
	u.mu.Lock()
	u.progress = progress
	u.statusMsg = msg
	u.mu.Unlock()
}

func sudoUpdate(src, dst string) error {
	// Write a small shell script that copies and sets permissions
	script := fmt.Sprintf("#!/bin/sh\ncp '%s' '%s' && chmod 755 '%s'\n", src, dst, dst)
	scriptPath := filepath.Join(os.TempDir(), "yaria-update.sh")
	os.WriteFile(scriptPath, []byte(script), 0755)
	defer os.Remove(scriptPath)

	// Try pkexec with DISPLAY set for graphical prompt
	if path, err := exec.LookPath("pkexec"); err == nil {
		cmd := exec.Command(path, "sh", scriptPath)
		cmd.Env = append(os.Environ(),
			"DISPLAY="+os.Getenv("DISPLAY"),
			"XAUTHORITY="+os.Getenv("XAUTHORITY"),
			"WAYLAND_DISPLAY="+os.Getenv("WAYLAND_DISPLAY"),
		)
		if out, err := cmd.CombinedOutput(); err == nil {
			return nil
		} else {
			_ = out
		}
	}

	// Try sudo non-interactive
	if path, err := exec.LookPath("sudo"); err == nil {
		if out, err := exec.Command(path, "-n", "sh", scriptPath).CombinedOutput(); err == nil {
			return nil
		} else {
			_ = out
		}
	}

	// Try direct copy (works if user owns the install dir)
	if err := copyFileForUpdate(src, dst); err == nil {
		os.Chmod(dst, 0755)
		return nil
	}

	return fmt.Errorf("permission denied - run manually:\n  sudo cp %s %s && sudo chmod 755 %s", src, dst, dst)
}

func copyFileForUpdate(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func extractTarGz(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, hdr.Name)
		// Security: prevent path traversal
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0o755)
			out, err := os.Create(target)
			if err != nil {
				return err
			}
			io.Copy(out, tr)
			out.Close()
			os.Chmod(target, os.FileMode(hdr.Mode))
		}
	}
	return nil
}

func formatUpdateBytes(b int64) string {
	if b >= 1073741824 {
		return fmt.Sprintf("%.1f GB", float64(b)/1073741824)
	}
	if b >= 1048576 {
		return fmt.Sprintf("%.1f MB", float64(b)/1048576)
	}
	if b >= 1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%d B", b)
}
