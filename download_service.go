package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"yaria/pkg/yaria/config"
	"yaria/pkg/yaria/deps"
	"yaria/pkg/yaria/downloader"
)

// activeDownload tracks the state of a single download.
type activeDownload struct {
	ID          string
	URL         string
	Title       string
	Thumbnail   string
	Status      string // "queued", "metadata", "downloading", "complete", "error", "cancelled"
	Percent     float64
	Speed       string
	ETA         string
	Error       string
	StartedAt   string
	cancel      context.CancelFunc
	lastEmit    time.Time
	pipeWriter  *io.PipeWriter
	Resolution  string
	DownloadDir string
	AudioOnly   bool
	AudioFormat string
	FilePath    string
	// Multi-file tracking (video+audio separate downloads)
	fileIndex   int     // 0 = first file, 1 = second file
	fileParts   int     // total parts (usually 2 for video+audio)
	prevPercent float64 // percent completed from previous files
}

// DownloadService provides video download methods to the frontend via Wails bindings.
type DownloadService struct {
	ctx        context.Context
	mu         sync.Mutex
	dl         *downloader.YTDLPDownloader
	cfg        *config.Config
	downloads  map[string]*activeDownload
	nextID     int
	depsReady  bool
	store      *DownloadStore
	maxRunning int
	running    int
	waitQueue  []string // IDs of queued-for-slot downloads
}

// NewDownloadService creates a DownloadService with default config.
func NewDownloadService() *DownloadService {
	store, _ := NewDownloadStore() // non-fatal if store fails
	return &DownloadService{
		downloads:  make(map[string]*activeDownload),
		cfg:        config.New(),
		store:      store,
		maxRunning: 3,
	}
}

// startup is called by Wails OnStartup.
func (d *DownloadService) startup(ctx context.Context) {
	d.ctx = ctx
}

// shutdown is called by Wails OnShutdown to close the download store.
func (d *DownloadService) shutdown(ctx context.Context) {
	if d.store != nil {
		d.store.Close()
	}
}

// InitDeps initializes the downloader (auto-install yt-dlp, aria2c, etc.).
// Heavy work runs in a goroutine; returns immediately.
// Emits "deps-progress" events with each line of output from the installer,
// then "deps-error" on failure or "deps-ready" on success.
func (d *DownloadService) InitDeps() map[string]interface{} {
	go func() {
		// Capture installer output and emit as UI events
		pr, pw := io.Pipe()
		d.cfg.Stdout = pw
		d.cfg.Stderr = pw

		// Read output lines and emit to frontend
		go func() {
			scanner := bufio.NewScanner(pr)
			for scanner.Scan() {
				line := scanner.Text()
				if line != "" {
					wailsRuntime.EventsEmit(d.ctx, "deps-progress", map[string]interface{}{
						"message": line,
					})
				}
			}
		}()

		dl, err := downloader.New(d.cfg)
		pw.Close()

		// Reset stdout/stderr so downloads use their own pipe
		d.cfg.Stdout = nil
		d.cfg.Stderr = nil

		if err != nil {
			wailsRuntime.EventsEmit(d.ctx, "deps-error", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		d.mu.Lock()
		d.dl = dl
		d.depsReady = true
		d.mu.Unlock()
		wailsRuntime.EventsEmit(d.ctx, "deps-ready", map[string]interface{}{
			"status": "ready",
		})
	}()
	return map[string]interface{}{"status": "initializing"}
}

// CheckDeps returns the status of all dependencies (yt-dlp, aria2c, etc.).
func (d *DownloadService) CheckDeps() []map[string]interface{} {
	depsList := deps.CheckAll()
	result := make([]map[string]interface{}, len(depsList))
	for i, dep := range depsList {
		result[i] = map[string]interface{}{
			"name":    dep.Name,
			"status":  dep.Status,
			"path":    dep.Path,
			"message": dep.Message,
		}
	}
	return result
}

// cleanURL removes shell escape characters from pasted URLs.
func cleanURL(url string) string {
	url = strings.TrimSpace(url)
	url = strings.ReplaceAll(url, "\\?", "?")
	url = strings.ReplaceAll(url, "\\=", "=")
	url = strings.ReplaceAll(url, "\\&", "&")
	url = strings.ReplaceAll(url, "\\#", "#")
	return url
}

// FetchMetadata gets video title, thumbnail, and playlist info for a URL.
// Falls back to OEmbed/noembed API if yt-dlp fails (e.g. bot detection).
func (d *DownloadService) FetchMetadata(rawURL string) map[string]interface{} {
	url := cleanURL(rawURL)

	// Try yt-dlp first (if initialized)
	if d.dl != nil {
		playlistInfo, title, err := d.dl.GetMetadata([]string{url})
		if err == nil && title != "" {
			result := map[string]interface{}{"title": title}
			if playlistInfo != "" {
				parts := strings.Split(playlistInfo, "&")
				if len(parts) >= 3 {
					result["playlist_id"] = parts[0]
					result["playlist_title"] = parts[1]
					result["playlist_count"] = parts[2]
				}
			}
			thumb := d.getThumbnailURL(url)
			if thumb != "" {
				result["thumbnail"] = thumb
			}
			return result
		}
		// yt-dlp failed -- fall through to fallback
	}

	// Fallback: use noembed.com (free OEmbed proxy, no auth needed)
	result := d.fetchMetadataFallback(url)
	if result != nil {
		return result
	}

	// Last resort: return URL-derived info
	thumb := d.getThumbnailURL(url)
	r := map[string]interface{}{
		"title": url,
	}
	if thumb != "" {
		r["thumbnail"] = thumb
	}
	return r
}

// fetchMetadataFallback uses noembed.com to get video title/thumbnail
// without needing cookies or authentication.
func (d *DownloadService) fetchMetadataFallback(videoURL string) map[string]interface{} {
	apiURL := "https://noembed.com/embed?url=" + videoURL
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}

	title, _ := data["title"].(string)
	thumb, _ := data["thumbnail_url"].(string)
	author, _ := data["author_name"].(string)

	if title == "" {
		return nil
	}

	result := map[string]interface{}{
		"title": title,
	}
	if thumb != "" {
		result["thumbnail"] = thumb
	}
	if author != "" {
		result["uploader"] = author
	}
	return result
}

// getThumbnailURL returns a thumbnail URL for a video.
// For YouTube, constructs it directly (no yt-dlp call needed).
// For other sites, returns empty (not critical).
func (d *DownloadService) getThumbnailURL(url string) string {
	// YouTube: extract video ID and construct thumbnail URL directly
	if strings.Contains(url, "youtube.com/watch") || strings.Contains(url, "youtu.be/") {
		videoID := ""
		if strings.Contains(url, "v=") {
			parts := strings.SplitN(url, "v=", 2)
			if len(parts) == 2 {
				videoID = strings.SplitN(parts[1], "&", 2)[0]
			}
		} else if strings.Contains(url, "youtu.be/") {
			parts := strings.SplitN(url, "youtu.be/", 2)
			if len(parts) == 2 {
				videoID = strings.SplitN(parts[1], "?", 2)[0]
			}
		}
		if videoID != "" {
			return "https://i.ytimg.com/vi/" + videoID + "/hqdefault.jpg"
		}
	}
	return ""
}

// ListFormats lists available video formats/resolutions for a URL.
// Returns {video: [...], audio: [...]} separated by type.
func (d *DownloadService) ListFormats(rawURL string) map[string]interface{} {
	if d.dl == nil {
		return map[string]interface{}{"error": "downloader not initialized"}
	}
	url := cleanURL(rawURL)
	formats, err := d.dl.GetFormats(url)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	var videoFmts, audioFmts []map[string]interface{}
	for _, f := range formats {
		// Clean up extension -- sometimes yt-dlp puts codec IDs like "MP4A.40.2"
		ext := f.Ext
		extLower := strings.ToLower(ext)
		if strings.Contains(extLower, ".") && !strings.HasPrefix(extLower, "mp4") && !strings.HasPrefix(extLower, "webm") {
			ext = "" // codec identifier, not a file extension
		}
		// Simplify known extensions
		if strings.HasPrefix(extLower, "mp4") {
			ext = "MP4"
		} else if strings.HasPrefix(extLower, "webm") {
			ext = "WebM"
		}

		m := map[string]interface{}{
			"format_id":   f.ID,
			"resolution":  fmt.Sprintf("%dp", f.Height),
			"height":      f.Height,
			"ext":         ext,
			"is_audio":    f.IsAudio,
			"protocol":    f.Protocol,
			"filesize":    f.FileSize,
			"format_note": fmt.Sprintf("%dp", f.Height),
		}
		if f.IsAudio {
			m["resolution"] = "Audio"
			m["format_note"] = "Audio (" + f.Ext + ")"
			audioFmts = append(audioFmts, m)
		} else {
			videoFmts = append(videoFmts, m)
		}
	}

	return map[string]interface{}{
		"video": videoFmts,
		"audio": audioFmts,
	}
}

// StartDownload begins a download in a goroutine and emits progress events.
// Returns immediately with the download ID. If the max concurrent limit is
// reached, the download is placed in a wait queue.
func (d *DownloadService) StartDownload(rawURL, resolution, downloadDir string, audioOnly bool, audioFormat string) map[string]interface{} {
	if d.dl == nil {
		return map[string]interface{}{"error": "downloader not initialized"}
	}
	url := cleanURL(rawURL)

	d.mu.Lock()
	d.nextID++
	id := fmt.Sprintf("dl_%d", d.nextID)

	thumb := d.getThumbnailURL(url)
	ad := &activeDownload{
		ID:          id,
		URL:         url,
		Title:       url,
		Thumbnail:   thumb,
		Status:      "queued",
		StartedAt:   time.Now().Format("2006-01-02 15:04"),
		Resolution:  resolution,
		DownloadDir: downloadDir,
		AudioOnly:   audioOnly,
		AudioFormat: audioFormat,
	}
	d.downloads[id] = ad

	if d.running >= d.maxRunning {
		d.waitQueue = append(d.waitQueue, id)
		d.mu.Unlock()
		d.updateDownload(id, "queued", 0, "", "", "")
		return map[string]interface{}{"id": id, "status": "queued"}
	}
	d.running++
	d.mu.Unlock()

	go d.runDownload(id, url, resolution, downloadDir, audioOnly, audioFormat)

	return map[string]interface{}{
		"id":     id,
		"status": "queued",
	}
}

// runDownload executes the actual download in a goroutine. When finished,
// it starts the next queued download if any.
func (d *DownloadService) runDownload(id, url, resolution, downloadDir string, audioOnly bool, audioFormat string) {
	defer d.startNextQueued()

	dlCtx, cancel := context.WithCancel(d.ctx)
	defer cancel()
	d.mu.Lock()
	ad, ok := d.downloads[id]
	if ok {
		ad.cancel = cancel
	}
	d.mu.Unlock()
	if !ok {
		return
	}

	// Fetch metadata first
	d.updateDownload(id, "metadata", 0, "", "", "")
	_, title, err := d.dl.GetMetadata([]string{url})
	if err != nil {
		_, userMsg := classifyError(err.Error())
		d.updateDownload(id, "error", 0, "", "", userMsg)
		return
	}

	d.mu.Lock()
	ad.Title = title
	d.mu.Unlock()

	// Resolve destination directory
	dest := downloadDir
	if dest == "" {
		home, _ := os.UserHomeDir()
		dest = filepath.Join(home, "Downloads")
	}
	_ = os.MkdirAll(dest, 0755)

	// Set per-download config on the shared downloader.
	// Use the same output template as the CLI daemon:
	// %(title)s/%(title)s.%(ext)s -- creates a folder per video
	d.mu.Lock()
	origRes := d.cfg.Resolution
	origAudio := d.cfg.IsAudioOnly
	origAudioFmt := d.cfg.AudioFormat
	origTemplate := d.cfg.OutputTemplate
	if resolution != "" {
		d.cfg.Resolution = resolutionToFormat(resolution)
	}
	d.cfg.IsAudioOnly = audioOnly
	if audioFormat != "" {
		d.cfg.AudioFormat = audioFormat
	}
	d.cfg.OutputTemplate = ".%(title)s/%(title)s.%(ext)s"
	d.mu.Unlock()

	// Pipe stdout/stderr through a progress parser
	pr, pw := io.Pipe()
	d.mu.Lock()
	ad.pipeWriter = pw
	d.mu.Unlock()
	d.cfg.Stdout = pw
	d.cfg.Stderr = pw
	go d.parseProgress(dlCtx, id, pr)

	d.updateDownload(id, "downloading", 0, "", "", "")

	// Download directly into dest -- yt-dlp creates %(title)s/ subfolder
	// User can open the folder during download and see aria2 pieces
	success, dlErr := d.dl.Download([]string{url}, dest)
	pw.Close()

	// Restore original config
	d.mu.Lock()
	d.cfg.Resolution = origRes
	d.cfg.IsAudioOnly = origAudio
	d.cfg.AudioFormat = origAudioFmt
	d.cfg.OutputTemplate = origTemplate
	d.cfg.Stdout = nil
	d.cfg.Stderr = nil
	d.mu.Unlock()

	// Check for cancellation
	select {
	case <-dlCtx.Done():
		return
	default:
	}

	if dlErr != nil || !success {
		errMsg := "download failed"
		if dlErr != nil {
			errMsg = dlErr.Error()
		}
		errType, userMsg := classifyError(errMsg)
		d.updateDownload(id, "error", 0, "", "", userMsg)
		_ = errType
		return
	}

	// Rename hidden folder to visible: .Title/ -> Title/
	if title != "" {
		hiddenDir := filepath.Join(dest, "."+title)
		visibleDir := filepath.Join(dest, title)
		if _, err := os.Stat(hiddenDir); err == nil {
			os.Rename(hiddenDir, visibleDir)
		}

		// Find the downloaded file
		d.mu.Lock()
		if ad != nil {
			filepath.Walk(visibleDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				if !strings.HasSuffix(info.Name(), ".part") {
					ad.FilePath = path
					return filepath.SkipAll
				}
				return nil
			})
		}
		d.mu.Unlock()
	}

	d.updateDownload(id, "complete", 100, "", "", "")

	// Notify frontend that a download completed, suggesting library addition
	go d.autoAddToLibrary(ad)
}

// startNextQueued decrements the running counter and starts the next
// queued download if there is one.
func (d *DownloadService) startNextQueued() {
	d.mu.Lock()
	d.running--
	for len(d.waitQueue) > 0 {
		nextID := d.waitQueue[0]
		d.waitQueue = d.waitQueue[1:]
		ad, ok := d.downloads[nextID]
		if ok && ad.Status == "queued" {
			d.running++
			go d.runDownload(nextID, ad.URL, ad.Resolution, ad.DownloadDir, ad.AudioOnly, ad.AudioFormat)
			d.mu.Unlock()
			return
		}
	}
	d.mu.Unlock()
}

// SetMaxConcurrent sets the max number of concurrent downloads (1-10).
func (d *DownloadService) SetMaxConcurrent(n int) map[string]interface{} {
	if n < 1 {
		n = 1
	}
	if n > 10 {
		n = 10
	}
	d.mu.Lock()
	d.maxRunning = n
	d.mu.Unlock()
	return map[string]interface{}{"max_running": n}
}

// GetMaxConcurrent returns the current max concurrent download limit.
func (d *DownloadService) GetMaxConcurrent() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.maxRunning
}

// CancelDownload cancels an active download.
// Closes the pipe writer to force yt-dlp/ffmpeg to exit on broken pipe.
func (d *DownloadService) CancelDownload(id string) map[string]interface{} {
	d.mu.Lock()
	ad, ok := d.downloads[id]
	d.mu.Unlock()
	if !ok {
		return map[string]interface{}{"error": "download not found"}
	}
	if ad.cancel != nil {
		ad.cancel()
	}
	if ad.pipeWriter != nil {
		ad.pipeWriter.Close()
	}
	d.updateDownload(id, "cancelled", ad.Percent, "", "", "")
	return map[string]interface{}{"status": "cancelled"}
}

// GetDownloads lists all downloads (active in-memory + persisted history).
// Results are sorted by started_at descending (newest first) for stable UI ordering.
func (d *DownloadService) GetDownloads() []map[string]interface{} {
	d.mu.Lock()
	// Active downloads from memory
	active := make([]*activeDownload, 0, len(d.downloads))
	activeIDs := make(map[string]bool)
	for _, ad := range d.downloads {
		activeIDs[ad.ID] = true
		active = append(active, ad)
	}
	d.mu.Unlock()

	// Sort active downloads by StartedAt descending for stable ordering.
	// Map iteration is random in Go, so without sorting the UI jumps around.
	sort.Slice(active, func(i, j int) bool {
		return active[i].StartedAt > active[j].StartedAt
	})

	result := make([]map[string]interface{}, 0, len(active))
	for _, ad := range active {
		result = append(result, map[string]interface{}{
			"id":           ad.ID,
			"url":          ad.URL,
			"title":        ad.Title,
			"thumbnail":    ad.Thumbnail,
			"status":       ad.Status,
			"percent":      ad.Percent,
			"speed":        ad.Speed,
			"eta":          ad.ETA,
			"error":        ad.Error,
			"started_at":   ad.StartedAt,
			"file_path":    ad.FilePath,
			"download_dir": ad.DownloadDir,
		})
	}

	// Add stored history (completed/errored downloads from previous sessions)
	if d.store != nil {
		for _, r := range d.store.GetAll() {
			if !activeIDs[r.ID] {
				// Get directory from file path
				dlDir := ""
				if r.FilePath != "" {
					dlDir = filepath.Dir(filepath.Dir(r.FilePath)) // parent of title folder
				}
				result = append(result, map[string]interface{}{
					"id":           r.ID,
					"url":          r.URL,
					"title":        r.Title,
					"thumbnail":    r.Thumbnail,
					"status":       r.Status,
					"percent":      r.Percent,
					"error":        r.Error,
					"started_at":   r.StartedAt,
					"file_path":    r.FilePath,
					"file_size":    r.FileSize,
					"download_dir": dlDir,
				})
			}
		}
	}
	return result
}

// RemoveDownload removes a download from the list (only if complete/error/cancelled).
func (d *DownloadService) RemoveDownload(id string) map[string]interface{} {
	d.mu.Lock()
	ad, ok := d.downloads[id]
	if ok && (ad.Status == "complete" || ad.Status == "error" || ad.Status == "cancelled") {
		delete(d.downloads, id)
	}
	d.mu.Unlock()

	// Also remove from persistent store
	if d.store != nil {
		d.store.Delete(id)
	}

	if !ok {
		// Might be a store-only record (from a previous session)
		return map[string]interface{}{"status": "removed"}
	}
	return map[string]interface{}{"status": "removed"}
}

// DeleteDownloadFiles removes a download from history AND deletes the downloaded files.
func (d *DownloadService) DeleteDownloadFiles(id string) map[string]interface{} {
	// Get file path before removing
	var filePath string
	d.mu.Lock()
	if ad, ok := d.downloads[id]; ok {
		filePath = ad.FilePath
	}
	d.mu.Unlock()

	// Check store if not in active downloads
	if filePath == "" && d.store != nil {
		if r, err := d.store.Get(id); err == nil {
			filePath = r.FilePath
		}
	}

	// Delete the file/folder
	if filePath != "" {
		// filePath points to a file inside a title folder -- delete the parent folder
		dir := filepath.Dir(filePath)
		if dir != "" && dir != "." && dir != "/" {
			os.RemoveAll(dir)
		}
	}

	// Remove from list and store
	return d.RemoveDownload(id)
}

// PlayDownloadedFile opens the downloaded file with the system player.
func (d *DownloadService) PlayDownloadedFile(id string) map[string]interface{} {
	var filePath string
	d.mu.Lock()
	if ad, ok := d.downloads[id]; ok {
		filePath = ad.FilePath
	}
	d.mu.Unlock()

	if filePath == "" && d.store != nil {
		if r, err := d.store.Get(id); err == nil {
			filePath = r.FilePath
		}
	}

	if filePath == "" {
		return map[string]interface{}{"error": "file path not found"}
	}
	if _, err := os.Stat(filePath); err != nil {
		return map[string]interface{}{"error": "file not found on disk"}
	}

	// Find a media player
	for _, name := range []string{"mpv", "vlc", "celluloid", "totem", "xdg-open"} {
		if p, err := exec.LookPath(name); err == nil {
			go exec.Command(p, filePath).Run()
			return map[string]interface{}{"status": "playing", "player": name}
		}
	}
	return map[string]interface{}{"error": "no media player found"}
}

// SetDownloadDir sets the default download directory.
func (d *DownloadService) SetDownloadDir(dir string) map[string]interface{} {
	d.mu.Lock()
	d.cfg.DownloadLocation = dir
	d.mu.Unlock()
	return map[string]interface{}{"status": "ok", "dir": dir}
}

// GetDownloadDir returns the current download directory.
func (d *DownloadService) GetDownloadDir() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cfg.DownloadLocation != "" {
		return d.cfg.DownloadLocation
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads")
}

// SelectDownloadDir opens a native directory picker and returns the chosen path.
func (d *DownloadService) SelectDownloadDir() string {
	dir, err := wailsRuntime.OpenDirectoryDialog(d.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Download Directory",
	})
	if err != nil || dir == "" {
		return ""
	}
	d.mu.Lock()
	d.cfg.DownloadLocation = dir
	d.mu.Unlock()
	return dir
}

// CheckExistingDownload checks if a download for the same URL is already active.
func (d *DownloadService) CheckExistingDownload(rawURL, downloadDir string) map[string]interface{} {
	url := cleanURL(rawURL)

	d.mu.Lock()
	for _, ad := range d.downloads {
		if ad.URL == url && (ad.Status == "downloading" || ad.Status == "metadata" || ad.Status == "queued") {
			d.mu.Unlock()
			return map[string]interface{}{
				"exists":  true,
				"status":  ad.Status,
				"id":      ad.ID,
				"message": "This URL is already being downloaded",
			}
		}
	}
	d.mu.Unlock()

	return map[string]interface{}{"exists": false}
}

// sendNotification sends a native desktop notification.
func sendNotification(title, message string) {
	switch goruntime.GOOS {
	case "linux":
		exec.Command("notify-send", "-a", "Yaria", "-i", "video-x-generic", title, message).Start()
	case "darwin":
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
		exec.Command("osascript", "-e", script).Start()
	case "windows":
		ps := fmt.Sprintf(`[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms'); $n = New-Object System.Windows.Forms.NotifyIcon; $n.Icon = [System.Drawing.SystemIcons]::Information; $n.Visible = $true; $n.ShowBalloonTip(5000, '%s', '%s', 'Info')`, title, message)
		exec.Command("powershell", "-Command", ps).Start()
	}
}

// DetectPlaylist performs a quick URL-based check to determine if the URL
// points to a playlist. This is a fast heuristic (no yt-dlp call).
func (d *DownloadService) DetectPlaylist(rawURL string) map[string]interface{} {
	url := cleanURL(rawURL)

	isPlaylistURL := strings.Contains(url, "playlist?list=") ||
		strings.Contains(url, "&list=") ||
		strings.Contains(url, "/playlist/") ||
		strings.Contains(url, "/sets/")

	return map[string]interface{}{
		"is_playlist": isPlaylistURL,
		"url":         url,
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// classifyError categorizes a download error message into a type and
// user-friendly message to help the user understand what went wrong.
func classifyError(errMsg string) (string, string) {
	msg := strings.ToLower(errMsg)
	switch {
	case strings.Contains(msg, "sign in") || strings.Contains(msg, "not a bot") || strings.Contains(msg, "cookies"):
		return "auth", "Authentication required. Try logging into the site in your browser first."
	case strings.Contains(msg, "429") || strings.Contains(msg, "too many requests"):
		return "rate_limit", "Rate limited. Wait a few minutes and try again."
	case strings.Contains(msg, "geo") || strings.Contains(msg, "not available in your country") || strings.Contains(msg, "blocked"):
		return "geo_blocked", "This video is not available in your region. Try using a VPN."
	case strings.Contains(msg, "private") || strings.Contains(msg, "unavailable") || strings.Contains(msg, "removed") || strings.Contains(msg, "deleted"):
		return "unavailable", "This video is unavailable (private, deleted, or removed)."
	case strings.Contains(msg, "unsupported url"):
		return "unsupported", "This URL is not supported."
	case strings.Contains(msg, "network") || strings.Contains(msg, "connection") || strings.Contains(msg, "timeout"):
		return "network", "Network error. Check your internet connection."
	case strings.Contains(msg, "age") || strings.Contains(msg, "restricted"):
		return "age_restricted", "Age-restricted content. Browser cookies are required."
	default:
		return "unknown", errMsg
	}
}

// autoAddToLibrary emits a frontend event suggesting the user add a
// completed download to the library. This avoids cross-service coupling.
func (d *DownloadService) autoAddToLibrary(ad *activeDownload) {
	wailsRuntime.EventsEmit(d.ctx, "download-complete-library", map[string]interface{}{
		"title":     ad.Title,
		"thumbnail": ad.Thumbnail,
		"url":       ad.URL,
	})
}

// resolutionToFormat converts a display resolution like "1080p" to a yt-dlp
// format selector. The Download() method appends "+bestaudio/best" to this,
// so we return only the video part. If the input looks like a format ID
// (numeric, e.g. "303"), it's returned as-is.
func resolutionToFormat(res string) string {
	// "best" = let yt-dlp pick automatically (bestvideo+bestaudio/best)
	if res == "best" {
		return ""
	}
	// Already a numeric format ID (e.g. "303", "137")
	if _, err := strconv.Atoi(res); err == nil {
		return res
	}
	// Convert "1080p" -> "bestvideo[height<=1080]"
	height := strings.TrimSuffix(strings.ToLower(res), "p")
	if _, err := strconv.Atoi(height); err == nil {
		return "bestvideo[height<=" + height + "]"
	}
	// Unknown format, return as-is
	return res
}

// updateDownload updates download state and emits a "download-progress" event.
// Progress events are rate-limited to at most once per second, except for
// important status changes (complete, error, cancelled, metadata, queued)
// which are always emitted immediately.
func (d *DownloadService) updateDownload(id, status string, percent float64, speed, eta, errMsg string) {
	d.mu.Lock()
	ad, ok := d.downloads[id]
	if ok {
		ad.Status = status
		ad.Percent = percent
		ad.Speed = speed
		ad.ETA = eta
		ad.Error = errMsg
	}
	title := ""
	thumbnail := ""
	url := ""
	startedAt := ""
	if ok {
		title = ad.Title
		thumbnail = ad.Thumbnail
		url = ad.URL
		startedAt = ad.StartedAt
	}

	// Rate limit: emit at most once per second, unless status is important
	shouldEmit := false
	if ok {
		isImportant := status == "complete" || status == "error" || status == "cancelled" || status == "metadata" || status == "queued" || status == "processing"
		if isImportant || time.Since(ad.lastEmit) >= time.Second {
			shouldEmit = true
			ad.lastEmit = time.Now()
		}
	}
	d.mu.Unlock()

	// Persist terminal states to disk
	if status == "complete" || status == "error" || status == "cancelled" {
		if d.store != nil {
			filePath := ""
			d.mu.Lock()
			if ad, ok := d.downloads[id]; ok {
				filePath = ad.FilePath
			}
			d.mu.Unlock()
			// Get file size from disk
			var fileSize int64
			if filePath != "" {
				if info, err := os.Stat(filePath); err == nil {
					fileSize = info.Size()
				}
			}
			d.store.Save(DownloadRecord{
				ID:        id,
				URL:       url,
				Title:     title,
				Thumbnail: thumbnail,
				Status:    status,
				Percent:   percent,
				Error:     errMsg,
				StartedAt: startedAt,
				FilePath:  filePath,
				FileSize:  fileSize,
			})
		}
	}

	if shouldEmit {
		wailsRuntime.EventsEmit(d.ctx, "download-progress", map[string]interface{}{
			"id":      id,
			"status":  status,
			"percent": percent,
			"speed":   speed,
			"eta":     eta,
			"error":   errMsg,
			"title":   title,
		})

		// Desktop notification on terminal states
		if status == "complete" {
			go sendNotification("Download Complete", title)
		} else if status == "error" && errMsg != "" {
			go sendNotification("Download Failed", title+": "+errMsg)
		}
	}
}

// parseProgress reads yt-dlp / aria2c output byte-by-byte and extracts progress.
// Handles multi-file downloads (video+audio) by combining progress.
func (d *DownloadService) parseProgress(ctx context.Context, id string, reader io.Reader) {
	ytdlpProgressRegex := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%`)
	aria2cFullRegex := regexp.MustCompile(`\[#[0-9a-f]+\s+.*?\((\d+)%\).*?DL:([0-9.]+\S+)(?:.*?ETA:(\S+))?`)
	aria2cProgressRegex := regexp.MustCompile(`\((\d+)%\)`)
	speedRegex := regexp.MustCompile(`(?:DL:|at\s+)(\d+\.?\d*\S+/s)`)
	etaRegex := regexp.MustCompile(`ETA[:\s]+(\S+)`)

	// Track multi-file progress (video+audio are downloaded separately)
	fileIndex := 0
	lastRawPercent := 0.0

	computeOverall := func(rawPercent float64) float64 {
		// Detect file transition: progress drops significantly
		if rawPercent < lastRawPercent-20 && lastRawPercent > 80 {
			fileIndex++
		}
		lastRawPercent = rawPercent

		// For 2-part downloads (video+audio), first file = 0-80%, second = 80-100%
		// Video is typically 80% of total size, audio is 20%
		switch fileIndex {
		case 0:
			return rawPercent * 0.8
		case 1:
			return 80 + rawPercent*0.2
		default:
			return rawPercent
		}
	}

	processLine := func(line string) {
		if line == "" {
			return
		}

		var rawPercent float64
		var speed, eta string
		matched := false

		// Try aria2c full format first: [#hex SIZE/TOTAL(PCT%) CN:N DL:SPEED ETA:TIME]
		if am := aria2cFullRegex.FindStringSubmatch(line); len(am) >= 2 {
			rawPercent, _ = strconv.ParseFloat(am[1], 64)
			if len(am) >= 3 && am[2] != "" {
				speed = am[2] + "/s"
			}
			if len(am) >= 4 && am[3] != "" {
				eta = am[3]
			}
			matched = true
		} else if matches := ytdlpProgressRegex.FindStringSubmatch(line); len(matches) >= 2 {
			rawPercent, _ = strconv.ParseFloat(matches[1], 64)
			matched = true
		} else if matches := aria2cProgressRegex.FindStringSubmatch(line); len(matches) >= 2 {
			rawPercent, _ = strconv.ParseFloat(matches[1], 64)
			matched = true
		}

		if matched {
			if speed == "" {
				if sm := speedRegex.FindStringSubmatch(line); len(sm) >= 2 {
					speed = sm[1]
				}
				// aria2c DL: without /s
				if speed == "" {
					dlRegex := regexp.MustCompile(`DL:([0-9.]+\S+)`)
					if dm := dlRegex.FindStringSubmatch(line); len(dm) >= 2 {
						speed = dm[1] + "/s"
					}
				}
			}
			if eta == "" {
				if em := etaRegex.FindStringSubmatch(line); len(em) >= 2 {
					eta = em[1]
				}
			}
			overall := computeOverall(rawPercent)
			d.updateDownload(id, "downloading", overall, speed, eta, "")
			return
		}

		// Detect post-processing phases
		if strings.Contains(line, "[Merger]") || strings.Contains(line, "[ExtractAudio]") ||
			strings.Contains(line, "[FixupM3u8]") || strings.Contains(line, "[ffmpeg]") ||
			strings.Contains(line, "[MoveFiles]") {
			d.updateDownload(id, "processing", 99, "", "", "")
			return
		}
	}

	// Read byte-by-byte to avoid pipe buffering issues on Linux
	var lineBuf []byte
	oneByte := make([]byte, 1)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := reader.Read(oneByte)
		if n == 1 {
			if oneByte[0] == '\r' || oneByte[0] == '\n' {
				if len(lineBuf) > 0 {
					processLine(string(lineBuf))
					lineBuf = lineBuf[:0]
				}
			} else {
				lineBuf = append(lineBuf, oneByte[0])
			}
	}
	if err != nil {
			if len(lineBuf) > 0 {
				processLine(string(lineBuf))
			}
			break
		}
	}
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
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
	if err != nil {
		os.Remove(dst)
		return err
	}
	return out.Close()
}
