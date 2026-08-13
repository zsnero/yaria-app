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
	"time"

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

// ListStorageDevices returns attached storage (external HDDs, USB drives,
// mounted partitions) so the in-app file picker can reach them.
func (d *DepsService) ListStorageDevices() []map[string]interface{} {
	return mountedStorageDevices()
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

// MpvLibPath returns a path to libmpv shared library only (for dlopen embed).
func (d *DepsService) MpvLibPath() (string, bool) {
	for _, name := range []string{"libmpv.so.2", "libmpv.so", "libmpv.so.1"} {
		p := filepath.Join(d.depsDir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	for _, p := range []string{
		"/usr/lib/libmpv.so.2", "/usr/lib/libmpv.so",
		"/usr/lib64/libmpv.so.2", "/usr/lib64/libmpv.so",
		"/usr/lib/x86_64-linux-gnu/libmpv.so.2",
		"/usr/lib/aarch64-linux-gnu/libmpv.so.2",
	} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, true
		}
	}
	return "", false
}

// MpvLibOrBinary returns a path to libmpv or mpv binary if found (system or deps dir).
func (d *DepsService) MpvLibOrBinary() (string, bool) {
	if p, ok := d.MpvLibPath(); ok {
		return p, true
	}
	// Bundled / previously downloaded — prefer executable for Windows subprocess embed
	for _, name := range []string{
		"mpv.exe", "mpv.com", binaryName("mpv"), "mpv",
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
			// Optional dependency — don't hard-fail the app setup banner
			emit("complete", 100, "Native player skipped (optional): "+err.Error())
			return
		}
	case "linux":
		if err := d.downloadMpvLinux(emit); err != nil {
			emit("complete", 100, "Native player skipped (optional): "+err.Error())
			return
		}
	case "darwin":
		emit("complete", 100, "Native player skipped (optional): install mpv via Homebrew — brew install mpv")
		return
	default:
		emit("complete", 100, "Native player skipped (optional): unsupported platform")
		return
	}

	if p, ok := d.MpvLibOrBinary(); ok {
		emit("complete", 100, "Native player installed: "+p)
	} else {
		emit("complete", 100, "Native player skipped (optional) — WebView player still works")
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
	var lastErr error
	family := linuxDistroFamily()

	tryOK := func() bool {
		_, ok := d.MpvLibPath()
		return ok
	}

	// --- Arch family ---
	if family == "arch" || hasCmd("pacman") {
		if hasCmd("pacman") {
			emit("downloading", 10, "Downloading mpv via pacman (no root)…")
			if err := d.downloadMpvViaUserPacman(emit); err == nil && tryOK() {
				return nil
			} else if err != nil {
				lastErr = err
				emit("downloading", 15, "pacman: "+trimOut(err.Error()))
			}
		}
		// Arch package mirrors (only on Arch-like — glibc/soname match)
		if family == "arch" || hasCmd("pacman") {
			emit("downloading", 20, "Downloading libmpv from Arch mirrors…")
			if err := d.downloadMpvArchHTTP(emit); err == nil && tryOK() {
				return nil
			} else if err != nil {
				lastErr = err
				emit("downloading", 25, "arch mirror: "+trimOut(err.Error()))
			}
		}
	}

	// --- Debian / Ubuntu / Mint / Pop! ---
	if family == "debian" || hasCmd("apt-get") {
		emit("downloading", 30, "Downloading libmpv (apt, no root)…")
		if err := d.downloadMpvApt(emit); err == nil && tryOK() {
			return nil
		} else if err != nil {
			lastErr = err
			emit("downloading", 40, "apt: "+trimOut(err.Error()))
		}
	}

	// --- Fedora / RHEL / CentOS / Nobara ---
	if family == "fedora" || hasCmd("dnf") || hasCmd("yum") || hasCmd("microdnf") {
		emit("downloading", 45, "Downloading libmpv (dnf/yum, no root)…")
		if err := d.downloadMpvDnf(emit); err == nil && tryOK() {
			return nil
		} else if err != nil {
			lastErr = err
			emit("downloading", 55, "dnf: "+trimOut(err.Error()))
		}
	}

	// --- openSUSE ---
	if family == "suse" || hasCmd("zypper") {
		emit("downloading", 60, "Downloading libmpv (zypper)…")
		if err := d.downloadMpvZypper(emit); err == nil && tryOK() {
			return nil
		} else if err != nil {
			lastErr = err
			emit("downloading", 70, "zypper: "+trimOut(err.Error()))
		}
	}

	// Generic last resort: if we're not sure of family, try apt then dnf then arch HTTP
	if family == "unknown" {
		if hasCmd("apt-get") {
			_ = d.downloadMpvApt(emit)
			if tryOK() {
				return nil
			}
		}
		if hasCmd("dnf") {
			_ = d.downloadMpvDnf(emit)
			if tryOK() {
				return nil
			}
		}
		if err := d.downloadMpvArchHTTP(emit); err == nil && tryOK() {
			return nil
		} else if err != nil {
			lastErr = err
		}
	}

	hint := linuxMpvInstallHint()
	if lastErr != nil {
		return fmt.Errorf("could not auto-download libmpv (%v). WebView still works. Optional: %s", lastErr, hint)
	}
	return fmt.Errorf("could not auto-download libmpv. WebView still works. Optional: %s", hint)
}

func hasCmd(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// linuxDistroFamily returns arch|debian|fedora|suse|unknown from /etc/os-release.
func linuxDistroFamily() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown"
	}
	low := strings.ToLower(string(data))
	id, like := "", ""
	for _, line := range strings.Split(low, "\n") {
		if strings.HasPrefix(line, "id=") {
			id = strings.Trim(strings.TrimPrefix(line, "id="), `"'`)
		}
		if strings.HasPrefix(line, "id_like=") {
			like = strings.Trim(strings.TrimPrefix(line, "id_like="), `"'`)
		}
	}
	blob := id + " " + like
	switch {
	case strings.Contains(blob, "arch") || strings.Contains(blob, "manjaro") ||
		strings.Contains(blob, "cachy") || strings.Contains(blob, "endeavouros") ||
		strings.Contains(blob, "garuda") || strings.Contains(blob, "artix"):
		return "arch"
	case strings.Contains(blob, "debian") || strings.Contains(blob, "ubuntu") ||
		strings.Contains(blob, "mint") || strings.Contains(blob, "pop") ||
		strings.Contains(blob, "elementary") || strings.Contains(blob, "zorin") ||
		strings.Contains(blob, "raspbian") || strings.Contains(blob, "kali"):
		return "debian"
	case strings.Contains(blob, "fedora") || strings.Contains(blob, "rhel") ||
		strings.Contains(blob, "centos") || strings.Contains(blob, "rocky") ||
		strings.Contains(blob, "alma") || strings.Contains(blob, "nobara") ||
		strings.Contains(blob, "mageia") || strings.Contains(blob, "openmandriva"):
		return "fedora"
	case strings.Contains(blob, "suse") || strings.Contains(blob, "opensuse") ||
		strings.Contains(blob, "sles"):
		return "suse"
	default:
		return "unknown"
	}
}

func linuxMpvInstallHint() string {
	switch linuxDistroFamily() {
	case "debian":
		return "sudo apt install libmpv2 mpv"
	case "fedora":
		return "sudo dnf install mpv-libs mpv"
	case "suse":
		return "sudo zypper install libmpv2 mpv"
	case "arch":
		return "sudo pacman -S mpv"
	default:
		return "install mpv / libmpv via your package manager"
	}
}

func (d *DepsService) downloadMpvApt(emit func(string, int, string)) error {
	if !hasCmd("apt-get") {
		return fmt.Errorf("apt-get not found")
	}
	pkgs := []string{"libmpv2", "libmpv1", "libmpv2t64"}
	// Best-effort codec deps (names vary by Ubuntu release)
	extra := []string{
		"libavcodec61", "libavcodec60", "libavcodec59", "libavcodec58",
		"libavformat61", "libavformat60", "libavformat59", "libavformat58",
		"libavutil59", "libavutil58", "libavutil57", "libavutil56",
		"libswscale8", "libswscale7", "libswscale6", "libswscale5",
		"libswresample5", "libswresample4", "libswresample3",
		"libplacebo349", "libplacebo338", "libplacebo292", "libplacebo264",
		"libass9", "libass9t64",
	}
	got := false
	for _, pkg := range pkgs {
		cmd := exec.Command("apt-get", "download", pkg)
		cmd.Dir = d.depsDir
		hideConsole(cmd)
		if _, err := cmd.CombinedOutput(); err == nil {
			got = true
			break
		}
	}
	if !got {
		return fmt.Errorf("apt-get download libmpv failed (run apt update?)")
	}
	for _, pkg := range extra {
		cmd := exec.Command("apt-get", "download", pkg)
		cmd.Dir = d.depsDir
		hideConsole(cmd)
		_, _ = cmd.CombinedOutput()
	}
	emit("extracting", 65, "Extracting libraries from .deb packages…")
	if err := d.extractAllLibsFromDebs(d.depsDir); err != nil {
		return err
	}
	if _, ok := d.MpvLibPath(); !ok {
		return fmt.Errorf("libmpv.so not found in debs")
	}
	return nil
}

func (d *DepsService) downloadMpvDnf(emit func(string, int, string)) error {
	cache := filepath.Join(d.depsDir, "rpmcache")
	os.MkdirAll(cache, 0755)
	pkgs := []string{"mpv-libs", "mpv", "libmpv", "mpv-libs-unstable"}

	var dlErr error
	if hasCmd("dnf") {
		// dnf download does not need root
		args := append([]string{"download", "--destdir=" + cache, "-y"}, pkgs...)
		cmd := exec.Command("dnf", args...)
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			// Try minimal set
			cmd = exec.Command("dnf", "download", "--destdir="+cache, "-y", "mpv-libs")
			hideConsole(cmd)
			if out2, err2 := cmd.CombinedOutput(); err2 != nil {
				dlErr = fmt.Errorf("%s | %s", trimOut(string(out)), trimOut(string(out2)))
			}
		}
	} else if hasCmd("yumdownloader") {
		cmd := exec.Command("yumdownloader", "--destdir="+cache, "mpv-libs", "mpv")
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			dlErr = fmt.Errorf("%s", trimOut(string(out)))
		}
	} else if hasCmd("yum") {
		cmd := exec.Command("yum", "download", "--destdir="+cache, "mpv-libs")
		hideConsole(cmd)
		if out, err := cmd.CombinedOutput(); err != nil {
			dlErr = fmt.Errorf("%s", trimOut(string(out)))
		}
	} else {
		return fmt.Errorf("dnf/yum not found")
	}

	emit("extracting", 70, "Extracting libraries from RPM packages…")
	if err := d.extractAllLibsFromRpms(cache); err != nil {
		if dlErr != nil {
			return fmt.Errorf("%v; extract: %v", dlErr, err)
		}
		return err
	}
	if _, ok := d.MpvLibPath(); !ok {
		if dlErr != nil {
			return dlErr
		}
		return fmt.Errorf("libmpv.so not found in RPMs")
	}
	return nil
}

func (d *DepsService) downloadMpvZypper(emit func(string, int, string)) error {
	if !hasCmd("zypper") {
		return fmt.Errorf("zypper not found")
	}
	cache := filepath.Join(d.depsDir, "rpmcache")
	os.MkdirAll(cache, 0755)
	// zypper install --download-only needs root on some versions; try download command
	for _, pkg := range []string{"libmpv2", "libmpv1", "mpv"} {
		cmd := exec.Command("zypper", "--non-interactive", "--pkg-cache-dir", cache, "download", pkg)
		hideConsole(cmd)
		_, _ = cmd.CombinedOutput()
	}
	// Packages may land in cache/packages/*
	emit("extracting", 75, "Extracting libraries from zypper packages…")
	if err := d.extractAllLibsFromRpms(cache); err != nil {
		return err
	}
	if _, ok := d.MpvLibPath(); !ok {
		return fmt.Errorf("libmpv.so not found in zypper packages")
	}
	return nil
}

// extractAllLibsFromRpms walks dir for .rpm files and extracts shared libs.
func (d *DepsService) extractAllLibsFromRpms(dir string) error {
	var rpms []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".rpm") {
			rpms = append(rpms, path)
		}
		return nil
	})
	if len(rpms) == 0 {
		return fmt.Errorf("no RPM files in %s", dir)
	}
	tmp := filepath.Join(d.depsDir, "mpv_rpm_extract")
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0755)
	defer os.RemoveAll(tmp)

	extracted := 0
	for _, rpm := range rpms {
		// bsdtar handles rpm; fallback rpm2cpio | cpio
		var cmd *exec.Cmd
		if hasCmd("bsdtar") {
			cmd = exec.Command("bsdtar", "-xf", rpm, "-C", tmp)
		} else if hasCmd("rpm2cpio") && hasCmd("cpio") {
			// shell pipeline
			cmd = exec.Command("sh", "-c", fmt.Sprintf("rpm2cpio %q | cpio -idm -D %q", rpm, tmp))
		} else {
			continue
		}
		hideConsole(cmd)
		if _, err := cmd.CombinedOutput(); err != nil {
			continue
		}
		extracted++
	}
	if extracted == 0 {
		return fmt.Errorf("could not extract any RPMs (need bsdtar or rpm2cpio)")
	}
	return d.copySharedLibsFromTree(tmp)
}

// downloadMpvViaUserPacman syncs package DBs into a user-owned path and downloads
// mpv without root — works on Arch/CachyOS live sessions.
func (d *DepsService) downloadMpvViaUserPacman(emit func(string, int, string)) error {
	home, _ := os.UserHomeDir()
	dbPath := filepath.Join(home, ".cache", "yaria", "pacman-db")
	cache := filepath.Join(d.depsDir, "pkgcache")
	os.MkdirAll(dbPath, 0755)
	os.MkdirAll(cache, 0755)

	// Refresh sync DBs into user dbpath
	sync := exec.Command("pacman", "-Sy", "--noconfirm", "--dbpath", dbPath, "--cachedir", cache)
	hideConsole(sync)
	if out, err := sync.CombinedOutput(); err != nil {
		// Still try download; DBs might already exist
		_ = out
	}

	// Download mpv + deps (no install)
	cmd := exec.Command("pacman", "-Sw", "--noconfirm", "--dbpath", dbPath, "--cachedir", cache, "mpv")
	hideConsole(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Dependency-less fallback
		cmd2 := exec.Command("pacman", "-Swdd", "--noconfirm", "--dbpath", dbPath, "--cachedir", cache, "mpv")
		hideConsole(cmd2)
		if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
			return fmt.Errorf("%s | %s", trimOut(string(out)), trimOut(string(out2)))
		}
	}

	emit("extracting", 55, "Extracting libmpv from packages…")
	return d.extractAllLibsFromPacmanCache(cache)
}

// downloadMpvArchHTTP fetches the mpv package (and ffmpeg if needed) from Arch mirrors.
func (d *DepsService) downloadMpvArchHTTP(emit func(string, int, string)) error {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	} else if arch == "arm64" {
		arch = "aarch64"
	}
	// Packages that commonly provide libmpv + runtime codecs
	pkgs := []string{"mpv", "ffmpeg"}
	cache := filepath.Join(d.depsDir, "pkgcache")
	os.MkdirAll(cache, 0755)

	got := 0
	for i, pkg := range pkgs {
		pct := 30 + i*15
		emit("downloading", pct, "Fetching "+pkg+" package…")
		url, filename, err := archPackageDownloadURL(pkg, arch)
		if err != nil {
			if pkg == "mpv" {
				return err
			}
			continue
		}
		dest := filepath.Join(cache, filename)
		if err := d.downloadToFile(url, dest, emit, pct, pct+12); err != nil {
			if pkg == "mpv" {
				return err
			}
			continue
		}
		got++
	}
	if got == 0 {
		return fmt.Errorf("no packages downloaded")
	}
	emit("extracting", 75, "Extracting libraries…")
	return d.extractAllLibsFromPacmanCache(cache)
}

// archPackageDownloadURL resolves an Arch package to a mirror URL via packages API.
func archPackageDownloadURL(pkg, arch string) (url, filename string, err error) {
	// Try extra then community/extra-testing style repos
	repos := []string{"extra", "core", "multilib"}
	client := &http.Client{Timeout: 30 * time.Second}
	for _, repo := range repos {
		api := fmt.Sprintf("https://archlinux.org/packages/%s/%s/%s/json/", repo, arch, pkg)
		req, _ := http.NewRequest("GET", api, nil)
		req.Header.Set("User-Agent", "Yaria-Deps")
		resp, e := client.Do(req)
		if e != nil || resp.StatusCode != 200 {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		var meta struct {
			Filename string `json:"filename"`
			Repo     string `json:"repo"`
			Arch     string `json:"arch"`
			PkgVer   string `json:"pkgver"`
			PkgRel   string `json:"pkgrel"`
			Epoch    int    `json:"epoch"`
		}
		decErr := json.NewDecoder(resp.Body).Decode(&meta)
		resp.Body.Close()
		if decErr != nil || meta.Filename == "" {
			continue
		}
		// Stable CDN
		url = fmt.Sprintf("https://geo.mirror.pkgbuild.com/%s/os/%s/%s", repo, arch, meta.Filename)
		return url, meta.Filename, nil
	}
	// Fallback: archlinux.org redirect endpoint
	redir := fmt.Sprintf("https://archlinux.org/packages/extra/%s/%s/download/", arch, pkg)
	return redir, pkg + ".pkg.tar.zst", nil
}

func trimOut(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}

// extractAllLibsFromPacmanCache unpacks every package in cache and copies shared libs + mpv binary.
func (d *DepsService) extractAllLibsFromPacmanCache(cache string) error {
	entries, err := os.ReadDir(cache)
	if err != nil {
		return err
	}
	tmp := filepath.Join(d.depsDir, "mpv_pkg_extract")
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0755)
	defer os.RemoveAll(tmp)

	extracted := 0
	for _, e := range entries {
		n := e.Name()
		if !(strings.HasSuffix(n, ".pkg.tar.zst") || strings.HasSuffix(n, ".pkg.tar.xz") || strings.HasSuffix(n, ".pkg.tar.gz")) {
			continue
		}
		pkg := filepath.Join(cache, n)
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
		if _, err := cmd.CombinedOutput(); err != nil {
			continue
		}
		extracted++
	}
	if extracted == 0 {
		return fmt.Errorf("no packages extracted from cache")
	}
	return d.copySharedLibsFromTree(tmp)
}

func (d *DepsService) extractAllLibsFromDebs(dir string) error {
	entries, _ := os.ReadDir(dir)
	tmp := filepath.Join(d.depsDir, "mpv_deb_extract")
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0755)
	defer os.RemoveAll(tmp)

	count := 0
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, ".deb") {
			continue
		}
		deb := filepath.Join(dir, n)
		if _, err := exec.LookPath("dpkg-deb"); err == nil {
			cmd := exec.Command("dpkg-deb", "-x", deb, tmp)
			hideConsole(cmd)
			if _, err := cmd.CombinedOutput(); err != nil {
				continue
			}
		} else {
			cmd := exec.Command("sh", "-c", fmt.Sprintf("cd %q && ar x %q && tar xf data.tar.* -C %q", tmp, deb, tmp))
			if _, err := cmd.CombinedOutput(); err != nil {
				continue
			}
		}
		count++
	}
	if count == 0 {
		return fmt.Errorf("no debs extracted")
	}
	return d.copySharedLibsFromTree(tmp)
}

// copySharedLibsFromTree copies libmpv, related .so*, and mpv binary into depsDir.
func (d *DepsService) copySharedLibsFromTree(root string) error {
	var foundLib string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := info.Name()
		// Shared libraries
		if strings.Contains(base, ".so") {
			// Keep only library-looking names
			if !strings.HasPrefix(base, "lib") && !strings.Contains(base, ".so.") {
				return nil
			}
			dest := filepath.Join(d.depsDir, base)
			if err := copyFileMode(path, dest, 0755); err != nil {
				return nil
			}
			if strings.HasPrefix(base, "libmpv.so") {
				if foundLib == "" || base == "libmpv.so.2" || base == "libmpv.so" {
					foundLib = dest
				}
			}
		}
		// mpv binary (optional; Windows-style IPC fallback / CLI)
		if base == "mpv" {
			dest := filepath.Join(d.depsDir, "mpv")
			_ = copyFileMode(path, dest, 0755)
		}
		return nil
	})
	if foundLib == "" {
		return fmt.Errorf("libmpv.so not found in packages")
	}
	link := filepath.Join(d.depsDir, "libmpv.so")
	_ = os.Remove(link)
	_ = os.Symlink(filepath.Base(foundLib), link)
	// Ensure soname libmpv.so.2 exists for dlopen
	if !strings.HasSuffix(foundLib, "libmpv.so.2") {
		link2 := filepath.Join(d.depsDir, "libmpv.so.2")
		if _, err := os.Stat(link2); err != nil {
			_ = os.Symlink(filepath.Base(foundLib), link2)
		}
	}
	return nil
}

func copyFileMode(src, dest string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
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
