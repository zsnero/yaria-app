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

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var zipOpen = zip.OpenReader
var gzipNewReader = gzip.NewReader

// DepsService manages application-level dependencies (FFmpeg binary).
// Downloads static binaries from GitHub, no package manager needed.
type DepsService struct {
	ctx     context.Context
	mu      sync.Mutex
	depsDir string
}

func NewDepsService() *DepsService {
	home, _ := os.UserHomeDir()
	return &DepsService{
		depsDir: filepath.Join(home, ".yaria", "dependencies"),
	}
}

func (d *DepsService) startup(ctx context.Context) {
	d.ctx = ctx
	os.MkdirAll(d.depsDir, 0755)
}

// FFmpegPath returns the path to the bundled FFmpeg binary, or empty if not installed.
func (d *DepsService) FFmpegPath() string {
	p := filepath.Join(d.depsDir, "ffmpeg")
	if runtime.GOOS == "windows" {
		p += ".exe"
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	// Also check system PATH
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path
	}
	return ""
}

// CheckDeps returns the status of all app dependencies.
func (d *DepsService) CheckDeps() map[string]interface{} {
	ffmpegPath := d.FFmpegPath()
	ffmpegInstalled := ffmpegPath != ""

	ytdlpInstalled := false
	if _, err := exec.LookPath("yt-dlp"); err == nil {
		ytdlpInstalled = true
	} else if _, err := os.Stat(filepath.Join(d.depsDir, "yt-dlp")); err == nil {
		ytdlpInstalled = true
	}

	allReady := ffmpegInstalled && ytdlpInstalled

	return map[string]interface{}{
		"all_ready": allReady,
		"deps": []map[string]interface{}{
			{
				"name":      "FFmpeg",
				"desc":      "Video transcoding for full codec support",
				"installed": ffmpegInstalled,
				"path":      ffmpegPath,
			},
			{
				"name":      "yt-dlp",
				"desc":      "Video download engine",
				"installed": ytdlpInstalled,
			},
		},
	}
}

// InstallFFmpeg downloads a static FFmpeg binary from GitHub.
// Emits "deps-install-progress" events with {name, status, percent, message}
func (d *DepsService) InstallFFmpeg() map[string]interface{} {
	if d.FFmpegPath() != "" {
		return map[string]interface{}{"status": "already_installed", "path": d.FFmpegPath()}
	}
	go d.downloadFFmpeg()
	return map[string]interface{}{"status": "downloading"}
}

func (d *DepsService) downloadFFmpeg() {
	emit := func(status string, percent int, message string) {
		wailsRuntime.EventsEmit(d.ctx, "deps-install-progress", map[string]interface{}{
			"name":    "FFmpeg",
			"status":  status,
			"percent": percent,
			"message": message,
		})
	}

	emit("downloading", 0, "Fetching FFmpeg static build...")

	var downloadURL string
	arch := runtime.GOARCH
	goos := runtime.GOOS

	switch {
	case goos == "linux" && arch == "amd64":
		downloadURL = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz"
	case goos == "linux" && arch == "arm64":
		downloadURL = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linuxarm64-gpl.tar.xz"
	case goos == "windows" && arch == "amd64":
		downloadURL = "https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"
	case goos == "darwin" && arch == "amd64":
		downloadURL = "https://github.com/eugeneware/ffmpeg-static/releases/latest/download/ffmpeg-darwin-x64.gz"
	case goos == "darwin" && arch == "arm64":
		downloadURL = "https://github.com/eugeneware/ffmpeg-static/releases/latest/download/ffmpeg-darwin-arm64.gz"
	default:
		emit("error", 0, fmt.Sprintf("Unsupported platform: %s/%s", goos, arch))
		return
	}

	emit("downloading", 5, "Downloading FFmpeg (~80MB)...")

	resp, err := http.Get(downloadURL)
	if err != nil {
		emit("error", 0, fmt.Sprintf("Download failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		emit("error", 0, fmt.Sprintf("Download failed: HTTP %d", resp.StatusCode))
		return
	}

	tmpFile := filepath.Join(d.depsDir, "ffmpeg_download.tmp")
	out, err := os.Create(tmpFile)
	if err != nil {
		emit("error", 0, fmt.Sprintf("Failed to create temp file: %v", err))
		return
	}

	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 256*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)
			if totalSize > 0 {
				pct := int(float64(downloaded)/float64(totalSize)*75) + 5
				emit("downloading", pct, fmt.Sprintf("Downloading... %dMB / %dMB", downloaded/(1024*1024), totalSize/(1024*1024)))
			}
		}
		if readErr != nil {
			break
		}
	}
	out.Close()

	emit("extracting", 82, "Extracting FFmpeg binary...")

	ffmpegBin := filepath.Join(d.depsDir, "ffmpeg")
	if goos == "windows" {
		ffmpegBin += ".exe"
	}

	var extractErr error
	switch {
	case strings.HasSuffix(downloadURL, ".tar.xz"):
		extractErr = d.extractFromTarXz(tmpFile, ffmpegBin)
	case strings.HasSuffix(downloadURL, ".zip"):
		extractErr = d.extractFromZip(tmpFile, ffmpegBin)
	case strings.HasSuffix(downloadURL, ".gz"):
		extractErr = d.extractFromGz(tmpFile, ffmpegBin)
	default:
		// Direct binary
		extractErr = os.Rename(tmpFile, ffmpegBin)
	}

	os.Remove(tmpFile)

	if extractErr != nil {
		emit("error", 0, fmt.Sprintf("Extraction failed: %v", extractErr))
		return
	}

	os.Chmod(ffmpegBin, 0755)
	emit("complete", 100, "FFmpeg installed!")
}

func (d *DepsService) extractFromTarXz(archivePath, destPath string) error {
	// xz decompress -> tar extract
	xzCmd := exec.Command("xz", "-d", "-c", archivePath)
	stdout, err := xzCmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := xzCmd.Start(); err != nil {
		return fmt.Errorf("xz not found, install xz-utils")
	}

	tr := tar.NewReader(stdout)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			xzCmd.Wait()
			return err
		}
		base := filepath.Base(hdr.Name)
		if base == "ffmpeg" && !hdr.FileInfo().IsDir() {
			out, err := os.Create(destPath)
			if err != nil {
				xzCmd.Wait()
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			xzCmd.Wait()
			return err
		}
	}
	xzCmd.Wait()
	return fmt.Errorf("ffmpeg binary not found in archive")
}

func (d *DepsService) extractFromZip(archivePath, destPath string) error {
	// On Windows we need to extract ffmpeg.exe from the zip
	// The archive structure is: ffmpeg-master-latest-win64-gpl/bin/ffmpeg.exe
	dir := filepath.Dir(destPath)

	// Try Go's archive/zip
	r, err := zipOpen(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	target := "ffmpeg"
	if runtime.GOOS == "windows" {
		target = "ffmpeg.exe"
	}

	for _, f := range r.File {
		if filepath.Base(f.Name) == target && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			out, err := os.Create(filepath.Join(dir, target))
			if err != nil {
				rc.Close()
				return err
			}
			_, err = io.Copy(out, rc)
			out.Close()
			rc.Close()
			return err
		}
	}
	return fmt.Errorf("ffmpeg not found in zip archive")
}

func (d *DepsService) extractFromGz(archivePath, destPath string) error {
	// .gz is just gzip-compressed single file (macOS ffmpeg-static builds)
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzipNewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, gz)
	out.Close()
	return err
}
