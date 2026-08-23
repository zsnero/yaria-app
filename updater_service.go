package main

import (
	"archive/tar"
	"archive/zip"
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
	AppVersion    = "2.3.62"
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

func updateArch() string {
	switch runtime.GOARCH {
	case "arm64":
		return "arm64"
	default:
		return "amd64"
	}
}

// updatePackageURL returns the download URL for the current platform package.
func updatePackageURL() string {
	arch := updateArch()
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s/yaria-app-windows-%s.zip", UpdateBaseURL, arch)
	case "darwin":
		return fmt.Sprintf("%s/yaria-app-darwin-%s.tar.gz", UpdateBaseURL, arch)
	default:
		return fmt.Sprintf("%s/yaria-app-linux-%s.tar.gz", UpdateBaseURL, arch)
	}
}

func (u *UpdaterService) runUpdate() {
	defer func() {
		u.mu.Lock()
		u.updating = false
		u.mu.Unlock()
	}()

	downloadURL := updatePackageURL()
	u.setStatus(5, "Downloading update...")

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

	isZip := strings.HasSuffix(downloadURL, ".zip")
	suffix := "*.tar.gz"
	if isZip {
		suffix = "*.zip"
	}
	tmpFile, err := os.CreateTemp("", "yaria-update-"+suffix)
	if err != nil {
		u.setStatus(0, "Failed to create temp file")
		return
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			tmpFile.Write(buf[:n])
			downloaded += int64(n)
			if totalSize > 0 {
				pct := int(float64(downloaded) / float64(totalSize) * 70)
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

	tmpDir, err := os.MkdirTemp("", "yaria-update-extract-")
	if err != nil {
		u.setStatus(0, "Failed to create extract directory")
		return
	}
	defer os.RemoveAll(tmpDir)

	if isZip {
		if err := extractZip(tmpPath, tmpDir); err != nil {
			u.setStatus(0, "Extract failed: "+err.Error())
			return
		}
	} else {
		if err := extractTarGz(tmpPath, tmpDir); err != nil {
			u.setStatus(0, "Extract failed: "+err.Error())
			return
		}
	}

	newBinary := findUpdateBinary(tmpDir)
	if newBinary == "" {
		u.setStatus(0, "Binary not found in update package")
		return
	}

	u.setStatus(85, "Installing update...")

	currentBin, err := os.Executable()
	if err != nil {
		u.setStatus(0, "Could not determine current binary path")
		return
	}
	currentBin, _ = filepath.EvalSymlinks(currentBin)

	// Linux launcher layout: real binary is yaria-app.bin next to shell yaria-app
	if runtime.GOOS == "linux" {
		if err := installLinuxUpdatePackage(tmpDir, newBinary, currentBin); err != nil {
			u.setStatus(0, "Failed to install: "+err.Error())
			return
		}
	} else {
		if err := installUpdateBinary(newBinary, currentBin); err != nil {
			u.setStatus(0, "Failed to install: "+err.Error())
			return
		}
	}

	u.setStatus(100, "Update complete! Restart the app to use the new version.")

	if u.ctx != nil {
		wailsRuntime.EventsEmit(u.ctx, "update-complete", nil)
	}
}

// findUpdateBinary locates the app binary inside an extracted update package.
// Prefers yaria-app.bin (ELF) over the shell launcher named yaria-app.
func findUpdateBinary(root string) string {
	preferred := []string{"yaria-app.bin", "yaria-app.exe", "YariaApp.exe", "YariaApp", "yaria-app"}
	found := map[string]string{}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		for _, p := range preferred {
			if name == p {
				// Skip shell scripts mistakenly named like the app
				if isProbablyShellScript(path) && !strings.HasSuffix(strings.ToLower(name), ".exe") {
					return nil
				}
				found[name] = path
			}
		}
		return nil
	})
	for _, p := range preferred {
		if path, ok := found[p]; ok {
			return path
		}
	}
	return ""
}

func isProbablyShellScript(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 2)
	n, _ := f.Read(buf)
	return n >= 2 && buf[0] == '#' && buf[1] == '!'
}

// installLinuxUpdatePackage installs ELF as yaria-app.bin and refreshes the launcher.
func installLinuxUpdatePackage(extractRoot, newBinary, currentBin string) error {
	dir := filepath.Dir(currentBin)
	// If we're running the ELF (.bin) or a bare binary in ~/.local/bin
	targetBin := currentBin
	if strings.HasSuffix(currentBin, ".bin") {
		targetBin = currentBin
	} else if filepath.Base(currentBin) == "yaria-app" {
		// Prefer dedicated .bin path next to launcher
		targetBin = filepath.Join(dir, "yaria-app.bin")
	}

	if err := installUpdateBinary(newBinary, targetBin); err != nil {
		// Fallback: replace current path
		if err2 := installUpdateBinary(newBinary, currentBin); err2 != nil {
			return err
		}
		targetBin = currentBin
	}

	// Refresh launcher + deps helper from package when present
	var launcherSrc, depsSrc string
	filepath.Walk(extractRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		switch info.Name() {
		case "yaria-launcher.sh":
			launcherSrc = path
		case "linux-deps.sh":
			depsSrc = path
		}
		return nil
	})
	launcherDst := filepath.Join(filepath.Dir(targetBin), "yaria-app")
	if launcherSrc != "" {
		_ = copyFileForUpdate(launcherSrc, launcherDst)
		_ = os.Chmod(launcherDst, 0755)
	}
	if depsSrc != "" {
		_ = copyFileForUpdate(depsSrc, filepath.Join(filepath.Dir(targetBin), "linux-deps.sh"))
	}
	// Ensure .bin exists if we only updated launcher path historically
	if targetBin != currentBin && filepath.Base(currentBin) == "yaria-app" && !isProbablyShellScript(currentBin) {
		// current was a real ELF named yaria-app — keep a copy as .bin too
		_ = copyFileForUpdate(newBinary, filepath.Join(dir, "yaria-app.bin"))
		_ = os.Chmod(filepath.Join(dir, "yaria-app.bin"), 0755)
	}
	return nil
}

// installUpdateBinary replaces the running binary, with platform-specific fallbacks.
func installUpdateBinary(newBinary, currentBin string) error {
	// On Windows, a running .exe cannot always be overwritten; rename then copy.
	oldBin := currentBin + ".old"
	os.Remove(oldBin)

	if err := os.Rename(currentBin, oldBin); err == nil {
		if err := copyFileForUpdate(newBinary, currentBin); err != nil {
			os.Rename(oldBin, currentBin)
			return err
		}
		os.Chmod(currentBin, 0755)
		// Best-effort cleanup; Windows may keep the lock until exit
		os.Remove(oldBin)
		return nil
	}

	// Fallback: install to a user-writable location
	userBin, err := userInstallBinaryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(userBin), 0755); err != nil {
		return err
	}
	if err := copyFileForUpdate(newBinary, userBin); err != nil {
		return err
	}
	os.Chmod(userBin, 0755)
	writeUserLauncher(userBin)
	return nil
}

func userInstallBinaryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "Yaria", "yaria-app.exe"), nil
	case "darwin":
		return filepath.Join(home, "Applications", "yaria-app"), nil
	default:
		return filepath.Join(home, ".local", "bin", "yaria-app"), nil
	}
}

// writeUserLauncher creates a desktop entry on Linux after user-local install.
func writeUserLauncher(userBinary string) {
	if runtime.GOOS != "linux" {
		return
	}
	home, _ := os.UserHomeDir()
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
}

// RestartApp relaunches the application binary and exits the current process.
func (u *UpdaterService) RestartApp() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe)
	hideConsole(cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	setProcAttrDetached(cmd)
	cmd.Start()
	os.Exit(0)
}

func (u *UpdaterService) setStatus(progress int, msg string) {
	u.mu.Lock()
	u.progress = progress
	u.statusMsg = msg
	u.mu.Unlock()
}

func sudoUpdate(src, dst string) error {
	// Linux-only privilege escalation helper (kept for manual/system installs).
	if runtime.GOOS != "linux" {
		return copyFileForUpdate(src, dst)
	}
	script := fmt.Sprintf("#!/bin/sh\ncp '%s' '%s' && chmod 755 '%s'\n", src, dst, dst)
	scriptPath := filepath.Join(os.TempDir(), "yaria-update.sh")
	os.WriteFile(scriptPath, []byte(script), 0755)
	defer os.Remove(scriptPath)

	if path, err := exec.LookPath("pkexec"); err == nil {
		cmd := exec.Command(path, "sh", scriptPath)
		hideConsole(cmd)
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

	if path, err := exec.LookPath("sudo"); err == nil {
		if out, err := exec.Command(path, "-n", "sh", scriptPath).CombinedOutput(); err == nil {
			return nil
		} else {
			_ = out
		}
	}

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

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	cleanDest := filepath.Clean(destDir)
	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)
		// Security: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(target), cleanDest+string(os.PathSeparator)) &&
			filepath.Clean(target) != cleanDest {
			continue
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0o755)
			continue
		}
		os.MkdirAll(filepath.Dir(target), 0o755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
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
