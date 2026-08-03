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

	"github.com/bodgit/sevenzip"
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

	mpvPath, mpvOK := d.MpvLibOrBinary()
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
			{
				"name":      "libmpv",
				"desc":      "Native video player (optional)",
				"installed": mpvOK,
				"path":      mpvPath,
			},
		},
	}
}

// MpvLibOrBinary returns a path to libmpv or mpv binary if found (system or deps dir).
func (d *DepsService) MpvLibOrBinary() (string, bool) {
	// Bundled / previously downloaded — prefer executable for Windows subprocess embed
	for _, name := range []string{
		"mpv.exe", "mpv.com", binaryName("mpv"), "mpv",
		"libmpv.so", "libmpv.so.2", "libmpv.so.1",
		"mpv-2.dll", "mpv.dll",
		"libmpv.dylib",
	} {
		p := filepath.Join(d.depsDir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	// Nested extract folder
	if entries, err := os.ReadDir(d.depsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(d.depsDir, e.Name(), "mpv.exe")
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, true
			}
		}
	}
	// System library locations
	for _, p := range []string{
		"/usr/lib/libmpv.so", "/usr/lib/libmpv.so.2",
		"/usr/lib64/libmpv.so", "/usr/lib64/libmpv.so.2",
		"/usr/lib/x86_64-linux-gnu/libmpv.so.2",
		"/usr/lib/aarch64-linux-gnu/libmpv.so.2",
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	if path, err := exec.LookPath("mpv"); err == nil {
		return path, true
	}
	return "", false
}

// EnsureAllDeps downloads any missing core deps in the background (FFmpeg, libmpv).
// Emits "setup-progress" events: {phase, name, percent, message, done, error}.
func (d *DepsService) EnsureAllDeps() map[string]interface{} {
	go d.ensureAllDeps()
	return map[string]interface{}{"status": "starting"}
}

func (d *DepsService) ensureAllDeps() {
	emit := func(phase, name string, percent int, message string, done bool, errMsg string) {
		if d.ctx == nil {
			return
		}
		wailsRuntime.EventsEmit(d.ctx, "setup-progress", map[string]interface{}{
			"phase":   phase,
			"name":    name,
			"percent": percent,
			"message": message,
			"done":    done,
			"error":   errMsg,
		})
	}

	needFFmpeg := d.FFmpegPath() == ""
	_, mpvOK := d.MpvLibOrBinary()
	needMpv := !mpvOK

	// Everything already present — stay silent (no banner on every launch)
	if !needFFmpeg && !needMpv {
		return
	}

	emit("start", "", 0, "Setting up missing dependencies…", false, "")

	// FFmpeg (needed for WebView transcode / probes)
	if needFFmpeg {
		emit("install", "FFmpeg", 5, "Downloading FFmpeg…", false, "")
		d.downloadFFmpeg()
		if d.FFmpegPath() == "" {
			emit("install", "FFmpeg", 40, "FFmpeg install may have failed — will retry later", false, "")
		} else {
			emit("install", "FFmpeg", 50, "FFmpeg ready", false, "")
		}
	}

	// libmpv / mpv (optional native player)
	if needMpv {
		emit("install", "libmpv", 55, "Downloading native player (libmpv)…", false, "")
		d.downloadMpv()
		if _, ok := d.MpvLibOrBinary(); ok {
			emit("install", "libmpv", 95, "Native player ready", false, "")
		} else {
			// Optional — don't treat as hard failure
			emit("install", "libmpv", 95, "Native player skipped (optional)", false, "")
		}
	}

	emit("complete", "", 100, "Setup complete", true, "")
}

// InstallMpv downloads portable libmpv/mpv into the app dependencies folder.
func (d *DepsService) InstallMpv() map[string]interface{} {
	if p, ok := d.MpvLibOrBinary(); ok {
		return map[string]interface{}{"status": "already_installed", "path": p}
	}
	go d.downloadMpv()
	return map[string]interface{}{"status": "downloading"}
}

func (d *DepsService) downloadMpv() {
	emit := func(status string, percent int, message string) {
		if d.ctx == nil {
			return
		}
		wailsRuntime.EventsEmit(d.ctx, "deps-install-progress", map[string]interface{}{
			"name":    "libmpv",
			"status":  status,
			"percent": percent,
			"message": message,
		})
	}

	if _, ok := d.MpvLibOrBinary(); ok {
		emit("complete", 100, "libmpv already available")
		return
	}

	emit("downloading", 5, "Fetching native player…")
	os.MkdirAll(d.depsDir, 0755)

	switch runtime.GOOS {
	case "windows":
		if err := d.downloadMpvWindows(emit); err != nil {
			emit("error", 0, err.Error())
			return
		}
	case "linux":
		if err := d.downloadMpvLinux(emit); err != nil {
			emit("error", 0, err.Error())
			return
		}
	case "darwin":
		emit("error", 0, "Auto-install of libmpv on macOS is not supported yet. Install mpv via Homebrew: brew install mpv")
		return
	default:
		emit("error", 0, "Unsupported platform for libmpv auto-install")
		return
	}

	if p, ok := d.MpvLibOrBinary(); ok {
		emit("complete", 100, "Native player installed: "+p)
	} else {
		emit("error", 0, "Download finished but libmpv was not found — try installing mpv from your package manager")
	}
}

func (d *DepsService) downloadMpvWindows(emit func(string, int, string)) error {
	// Latest portable build from shinchiro winbuilds (mpv-2.dll + deps)
	emit("downloading", 10, "Resolving Windows mpv build…")
	apiURL := "https://api.github.com/repos/shinchiro/mpv-winbuild-cmake/releases/latest"
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Yaria-Deps")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("GitHub API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API HTTP %d", resp.StatusCode)
	}
	var rel struct {
		Assets []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return err
	}
	// Prefer .zip; .7z is fine too (pure Go via bodgit/sevenzip — no 7-Zip app needed)
	pick := func() (url, name string) {
		score := func(n string) int {
			n = strings.ToLower(n)
			if !strings.Contains(n, "x86_64") && !strings.Contains(n, "win64") && !strings.Contains(n, "amd64") {
				return -1
			}
			if strings.Contains(n, "debug") || strings.Contains(n, "dev") {
				return -1
			}
			s := 0
			if strings.HasSuffix(n, ".zip") {
				s += 60
			}
			if strings.HasSuffix(n, ".7z") {
				s += 50
			}
			if strings.Contains(n, "v3") {
				s -= 5
			}
			if strings.Contains(n, "mpv") {
				s += 5
			}
			return s
		}
		bestS := -1
		for _, a := range rel.Assets {
			sc := score(a.Name)
			if sc > bestS {
				bestS = sc
				url, name = a.BrowserDownloadURL, a.Name
			}
		}
		if bestS < 0 {
			return "", ""
		}
		return url, name
	}

	url, name := pick()
	if url == "" {
		return fmt.Errorf("no suitable Windows mpv asset in latest release")
	}

	emit("downloading", 15, "Downloading "+name+"…")
	tmp := filepath.Join(d.depsDir, "mpv_download.tmp")
	if err := d.downloadToFile(url, tmp, emit, 15, 80); err != nil {
		return err
	}
	emit("extracting", 85, "Extracting mpv…")
	defer os.Remove(tmp)

	low := strings.ToLower(name)
	switch {
	case strings.HasSuffix(low, ".zip"):
		return d.extractZipDlls(tmp, d.depsDir)
	case strings.HasSuffix(low, ".7z"):
		return d.extractSevenZipDlls(tmp, d.depsDir)
	default:
		return fmt.Errorf("unsupported archive type: %s", name)
	}
}

func isMpvPortableFile(base string) bool {
	low := strings.ToLower(base)
	ext := filepath.Ext(low)
	switch ext {
	case ".dll", ".exe", ".com":
		return true
	default:
		// portable packs sometimes ship conf next to exe — skip large extras
		return low == "mpv.conf" || low == "input.conf"
	}
}

// extractZipDlls unpacks portable mpv files (.exe/.dll) from a zip into destDir.
func (d *DepsService) extractZipDlls(archive, destDir string) error {
	r, err := zipOpen(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	foundExe, foundDll := false, false
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(f.Name)
		if !isMpvPortableFile(base) {
			continue
		}
		dest := filepath.Join(destDir, base)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(dest)
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
		_ = os.Chmod(dest, 0755)
		low := strings.ToLower(base)
		if low == "mpv.exe" || low == "mpv.com" {
			foundExe = true
		}
		if strings.HasSuffix(low, ".dll") {
			foundDll = true
		}
	}
	if !foundExe && !foundDll {
		return fmt.Errorf("no mpv.exe/DLLs found in zip")
	}
	return nil
}

// extractSevenZipDlls unpacks portable mpv files from a .7z archive (pure Go).
func (d *DepsService) extractSevenZipDlls(archive, destDir string) error {
	r, err := sevenzip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("open 7z: %w", err)
	}
	defer r.Close()

	foundExe, foundDll := false, false
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(f.Name)
		if !isMpvPortableFile(base) {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open %s: %w", base, err)
		}
		dest := filepath.Join(destDir, base)
		out, err := os.Create(dest)
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
		_ = os.Chmod(dest, 0755)
		low := strings.ToLower(base)
		if low == "mpv.exe" || low == "mpv.com" {
			foundExe = true
		}
		if strings.HasSuffix(low, ".dll") {
			foundDll = true
		}
	}
	if !foundExe && !foundDll {
		return fmt.Errorf("no mpv.exe/DLLs found in 7z archive")
	}
	return nil
}

func (d *DepsService) downloadMpvLinux(emit func(string, int, string)) error {
	// 1) pacman download (Arch) without root
	if _, err := exec.LookPath("pacman"); err == nil {
		emit("downloading", 20, "Downloading mpv package (pacman)…")
		cache := filepath.Join(d.depsDir, "pkgcache")
		os.MkdirAll(cache, 0755)
		cmd := exec.Command("pacman", "-Swdd", "--noconfirm", "--cachedir", cache, "mpv")
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err == nil {
			emit("extracting", 70, "Extracting libmpv from package…")
			if err := d.extractLibmpvFromPacmanCache(cache); err == nil {
				return nil
			} else {
				emit("downloading", 75, "Package extract failed: "+err.Error())
			}
		} else {
			emit("downloading", 30, "pacman download skipped: "+string(out))
		}
	}

	// 2) apt-get download (Debian/Ubuntu) without root
	if _, err := exec.LookPath("apt-get"); err == nil {
		emit("downloading", 35, "Downloading libmpv (apt)…")
		cmd := exec.Command("apt-get", "download", "libmpv2")
		cmd.Dir = d.depsDir
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err == nil {
			emit("extracting", 70, "Extracting libmpv from .deb…")
			if err := d.extractLibmpvFromDebs(d.depsDir); err == nil {
				return nil
			}
			emit("downloading", 75, "deb extract failed: "+err.Error())
		} else {
			// try libmpv1 older name
			cmd = exec.Command("apt-get", "download", "libmpv1")
			cmd.Dir = d.depsDir
			hideConsole(cmd)
			if out2, err2 := cmd.CombinedOutput(); err2 == nil {
				if err := d.extractLibmpvFromDebs(d.depsDir); err == nil {
					return nil
				}
			} else {
				emit("downloading", 40, "apt download skipped: "+string(out)+string(out2))
			}
		}
	}

	return fmt.Errorf("could not auto-download libmpv — install with: sudo pacman -S mpv  OR  sudo apt install libmpv2 mpv")
}

func (d *DepsService) extractLibmpvFromPacmanCache(cache string) error {
	entries, _ := os.ReadDir(cache)
	var pkg string
	for _, e := range entries {
		n := e.Name()
		if strings.HasPrefix(n, "mpv-") && (strings.HasSuffix(n, ".pkg.tar.zst") || strings.HasSuffix(n, ".pkg.tar.xz")) {
			pkg = filepath.Join(cache, n)
			break
		}
	}
	if pkg == "" {
		return fmt.Errorf("mpv package not found in cache")
	}
	tmp := filepath.Join(d.depsDir, "mpv_pkg_extract")
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0755)
	defer os.RemoveAll(tmp)

	var cmd *exec.Cmd
	if strings.HasSuffix(pkg, ".zst") {
		if _, err := exec.LookPath("bsdtar"); err == nil {
			cmd = exec.Command("bsdtar", "-xf", pkg, "-C", tmp)
		} else {
			cmd = exec.Command("tar", "--use-compress-program=zstd", "-xf", pkg, "-C", tmp)
		}
	} else {
		cmd = exec.Command("tar", "-xf", pkg, "-C", tmp)
	}
	hideConsole(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar: %v (%s)", err, string(out))
	}
	return d.copyLibmpvFromTree(tmp)
}

func (d *DepsService) extractLibmpvFromDebs(dir string) error {
	entries, _ := os.ReadDir(dir)
	var deb string
	for _, e := range entries {
		n := e.Name()
		if strings.HasPrefix(n, "libmpv") && strings.HasSuffix(n, ".deb") {
			deb = filepath.Join(dir, n)
			break
		}
	}
	if deb == "" {
		return fmt.Errorf("no libmpv deb found")
	}
	tmp := filepath.Join(d.depsDir, "mpv_deb_extract")
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0755)
	defer os.RemoveAll(tmp)

	if _, err := exec.LookPath("dpkg-deb"); err == nil {
		cmd := exec.Command("dpkg-deb", "-x", deb, tmp)
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("dpkg-deb: %v (%s)", err, string(out))
		}
	} else {
		// ar x + tar
		cmd := exec.Command("sh", "-c", fmt.Sprintf("cd %q && ar x %q && tar xf data.tar.* -C %q", tmp, deb, tmp))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ar/tar: %v (%s)", err, string(out))
		}
	}
	return d.copyLibmpvFromTree(tmp)
}

func (d *DepsService) copyLibmpvFromTree(root string) error {
	var found string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := info.Name()
		if strings.HasPrefix(base, "libmpv.so") {
			dest := filepath.Join(d.depsDir, base)
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			_ = os.WriteFile(dest, data, 0755)
			if found == "" || base == "libmpv.so.2" || base == "libmpv.so" {
				found = dest
			}
		}
		return nil
	})
	if found == "" {
		return fmt.Errorf("libmpv.so not found in package")
	}
	// Convenience symlink name
	_ = os.Symlink(filepath.Base(found), filepath.Join(d.depsDir, "libmpv.so"))
	return nil
}

func (d *DepsService) downloadToFile(url, dest string, emit func(string, int, string), pctLo, pctHi int) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	total := resp.ContentLength
	var got int64
	buf := make([]byte, 256*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			got += int64(n)
			if total > 0 && emit != nil {
				frac := float64(got) / float64(total)
				pct := pctLo + int(frac*float64(pctHi-pctLo))
				emit("downloading", pct, fmt.Sprintf("Downloading… %dMB / %dMB", got/(1024*1024), total/(1024*1024)))
			}
		}
		if readErr != nil {
			break
		}
	}
	return nil
}

func (d *DepsService) extractZipNamed(archive string, want map[string]string) error {
	r, err := zipOpen(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	found := 0
	for _, f := range r.File {
		base := filepath.Base(f.Name)
		dest, ok := want[base]
		if !ok {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(dest)
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
		found++
	}
	if found == 0 {
		return fmt.Errorf("no matching DLLs in zip")
	}
	return nil
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
