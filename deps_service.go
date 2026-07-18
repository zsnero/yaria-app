package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
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
	return &DepsService{
		depsDir: filepath.Join(appDataDir(), "dependencies"),
	}
}

func (d *DepsService) startup(ctx context.Context) {
	d.ctx = ctx
	os.MkdirAll(d.depsDir, 0755)
}

// ListDirectories returns subdirectories of a given path.
// Used by the in-app file picker.
func (d *DepsService) ListDirectories(path string) []map[string]interface{} {
	path = expandTilde(path)

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var dirs []map[string]interface{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip hidden directories
		if strings.HasPrefix(name, ".") {
			continue
		}
		dirs = append(dirs, map[string]interface{}{
			"name": name,
			"path": filepath.Join(path, name),
		})
	}
	return dirs
}

// ListEntries returns directories and optionally files under path.
// fileExt is a comma-separated list of extensions without dots (e.g. "json,toml").
// Empty fileExt returns directories only (same as ListDirectories).
func (d *DepsService) ListEntries(path, fileExt string) []map[string]interface{} {
	path = expandTilde(path)
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	extSet := map[string]bool{}
	for _, e := range strings.Split(fileExt, ",") {
		e = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(e, ".")))
		if e != "" {
			extSet[e] = true
		}
	}
	includeFiles := len(extSet) > 0

	var out []map[string]interface{}
	// Directories first
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			out = append(out, map[string]interface{}{
				"name":  name,
				"path":  filepath.Join(path, name),
				"is_dir": true,
			})
		}
	}
	if includeFiles {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
			if !extSet[ext] {
				continue
			}
			out = append(out, map[string]interface{}{
				"name":   name,
				"path":   filepath.Join(path, name),
				"is_dir": false,
			})
		}
	}
	return out
}

// ReadTextFile reads a UTF-8 text file (used for library import, etc.).
func (d *DepsService) ReadTextFile(path string) map[string]interface{} {
	path = expandTilde(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	// Cap at 50MB to avoid blowing memory
	if len(data) > 50*1024*1024 {
		return map[string]interface{}{"error": "file too large"}
	}
	return map[string]interface{}{"data": string(data), "path": path}
}

// WriteTextFile writes UTF-8 text to a path (used for library export, etc.).
func (d *DepsService) WriteTextFile(path, content string) map[string]interface{} {
	path = expandTilde(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "saved", "path": path}
}

// FFmpegPath returns the path to the bundled FFmpeg binary, or empty if not installed.
func (d *DepsService) FFmpegPath() string {
	p := filepath.Join(d.depsDir, binaryName("ffmpeg"))
	if _, err := os.Stat(p); err == nil {
		return p
	}
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path
	}
	return ""
}

// FFprobePath returns the path to the bundled FFprobe binary, or empty if not installed.
func (d *DepsService) FFprobePath() string {
	p := filepath.Join(d.depsDir, binaryName("ffprobe"))
	if _, err := os.Stat(p); err == nil {
		return p
	}
	// Same directory as ffmpeg (system install or partial extract)
	if ff := d.FFmpegPath(); ff != "" {
		probe := filepath.Join(filepath.Dir(ff), binaryName("ffprobe"))
		if _, err := os.Stat(probe); err == nil {
			return probe
		}
	}
	if path, err := exec.LookPath("ffprobe"); err == nil {
		return path
	}
	return ""
}

// CheckDeps returns the status of all app dependencies.
func (d *DepsService) CheckDeps() map[string]interface{} {
	ffmpegPath := d.FFmpegPath()
	ffmpegInstalled := ffmpegPath != ""

	ytdlpInstalled := false
	if _, err := exec.LookPath(binaryName("yt-dlp")); err == nil {
		ytdlpInstalled = true
	} else if _, err := os.Stat(filepath.Join(d.depsDir, binaryName("yt-dlp"))); err == nil {
		ytdlpInstalled = true
	} else if _, err := exec.LookPath("yt-dlp"); err == nil {
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

// GetStreamDetails extracts video/audio stream information from a file.
func (d *DepsService) GetStreamDetails(filePath string) map[string]interface{} {
	ffprobePath := d.FFprobePath()
	if ffprobePath == "" {
		return map[string]interface{}{"error": "ffprobe not found"}
	}

	cmd := exec.Command(ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		filePath,
	)
	hideConsole(cmd)
	out, err := cmd.Output()
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	var result map[string]interface{}
	json.Unmarshal(out, &result)
	return result
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

	ffmpegBin := filepath.Join(d.depsDir, binaryName("ffmpeg"))
	ffprobeBin := filepath.Join(d.depsDir, binaryName("ffprobe"))

	var extractErr error
	switch {
	case strings.HasSuffix(downloadURL, ".tar.xz"):
		extractErr = d.extractFFmpegTools(tmpFile, "tar.xz", ffmpegBin, ffprobeBin)
	case strings.HasSuffix(downloadURL, ".zip"):
		extractErr = d.extractFFmpegTools(tmpFile, "zip", ffmpegBin, ffprobeBin)
	case strings.HasSuffix(downloadURL, ".gz"):
		// macOS static builds are single-file ffmpeg only
		extractErr = d.extractFromGz(tmpFile, ffmpegBin)
	default:
		extractErr = os.Rename(tmpFile, ffmpegBin)
	}

	os.Remove(tmpFile)

	if extractErr != nil {
		emit("error", 0, fmt.Sprintf("Extraction failed: %v", extractErr))
		return
	}

	os.Chmod(ffmpegBin, 0755)
	if _, err := os.Stat(ffprobeBin); err == nil {
		os.Chmod(ffprobeBin, 0755)
	}
	emit("complete", 100, "FFmpeg installed!")
}

// extractFFmpegTools extracts ffmpeg and ffprobe from tar.xz or zip archives.
func (d *DepsService) extractFFmpegTools(archivePath, kind, ffmpegDest, ffprobeDest string) error {
	want := map[string]string{
		filepath.Base(ffmpegDest):  ffmpegDest,
		filepath.Base(ffprobeDest): ffprobeDest,
	}
	found := map[string]bool{}

	extractOne := func(base string, r io.Reader) error {
		dest, ok := want[base]
		if !ok || found[base] {
			return nil
		}
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		_, err = io.Copy(out, r)
		out.Close()
		if err != nil {
			return err
		}
		found[base] = true
		return nil
	}

	switch kind {
	case "tar.xz":
		xzCmd := exec.Command("xz", "-d", "-c", archivePath)
		hideConsole(xzCmd)
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
			if hdr.FileInfo().IsDir() {
				continue
			}
			base := filepath.Base(hdr.Name)
			if err := extractOne(base, tr); err != nil {
				xzCmd.Wait()
				return err
			}
			if found[filepath.Base(ffmpegDest)] && found[filepath.Base(ffprobeDest)] {
				break
			}
		}
		xzCmd.Wait()
	case "zip":
		r, err := zipOpen(archivePath)
		if err != nil {
			return err
		}
		defer r.Close()
		for _, f := range r.File {
			if f.FileInfo().IsDir() {
				continue
			}
			base := filepath.Base(f.Name)
			if _, ok := want[base]; !ok || found[base] {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			err = extractOne(base, rc)
			rc.Close()
			if err != nil {
				return err
			}
			if found[filepath.Base(ffmpegDest)] && found[filepath.Base(ffprobeDest)] {
				break
			}
		}
	}

	if !found[filepath.Base(ffmpegDest)] {
		return fmt.Errorf("ffmpeg binary not found in archive")
	}
	// ffprobe is optional for macOS single-binary builds; required when present in archive
	return nil
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
