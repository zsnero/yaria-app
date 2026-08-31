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
	"yaria/pkg/appconfig"
	"yaria/pkg/yaria/config"
	"yaria/pkg/yaria/deps"
	"yaria/pkg/yaria/downloader"
)

// activeDownload tracks the state of a single download.
type activeDownload struct {
	ID              string
	URL             string
	Title           string
	Thumbnail       string
	Status          string // "queued", "metadata", "downloading", "paused", "complete", "error", "cancelled"
	Percent         float64
	Speed           string
	ETA             string
	Error           string // UI-facing; only set on real failure
	lastYtdlpErr    string // non-fatal stderr ERROR: lines kept for final failure message
	StartedAt       string
	startedUnix     int64 // monotonic order key (ms) — keeps UI list stable
	cancel          context.CancelFunc
	lastEmit        time.Time
	pipeWriter      *io.PipeWriter
	Resolution      string
	DownloadDir     string
	AudioOnly       bool
	AudioFormat     string
	ContainerFormat string
	FilePath        string
	Referer         string // page URL for hotlink CDNs (from browser extension)
	// Multi-file tracking (video+audio separate downloads)
	fileIndex   int     // 0 = first file, 1 = second file
	fileParts   int     // total parts (usually 2 for video+audio)
	prevPercent float64 // percent completed from previous files
}

// DownloadService provides video download methods to the frontend via Wails bindings.
type DownloadService struct {
	ctx             context.Context
	mu              sync.Mutex
	dl              *downloader.YTDLPDownloader
	cfg             *config.Config
	downloads       map[string]*activeDownload
	nextID          int
	depsReady       bool
	depsInitStarted bool
	store           *DownloadStore
	maxRunning      int
	running         int
	waitQueue       []string // IDs of queued-for-slot downloads
}

// NewDownloadService creates a DownloadService with default config.
func NewDownloadService() *DownloadService {
	store, _ := NewDownloadStore() // non-fatal if store fails
	nextID := 0
	if store != nil {
		// Continue IDs past existing history so we never overwrite dl_1, etc.
		for _, r := range store.GetAll() {
			var n int
			if _, err := fmt.Sscanf(r.ID, "dl_%d", &n); err == nil && n > nextID {
				nextID = n
			}
		}
	}
	return &DownloadService{
		downloads:  make(map[string]*activeDownload),
		cfg:        config.New(),
		store:      store,
		maxRunning: 3,
		nextID:     nextID,
	}
}

// startup is called by Wails OnStartup.
func (d *DownloadService) startup(ctx context.Context) {
	d.ctx = ctx
	// Restore preferred download folder (Bridge + Yaria downloader)
	if dir := strings.TrimSpace(appconfig.BrowserExtensionDownloadDir()); dir != "" {
		d.mu.Lock()
		d.cfg.DownloadLocation = dir
		d.mu.Unlock()
	}
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
// Safe to call multiple times (App + Home both invoke it) — only one run.
func (d *DownloadService) InitDeps() map[string]interface{} {
	d.mu.Lock()
	if d.depsReady && d.dl != nil {
		d.mu.Unlock()
		// Already ready — re-emit so late subscribers update UI
		if d.ctx != nil {
			wailsRuntime.EventsEmit(d.ctx, "deps-ready", map[string]interface{}{"status": "ready"})
		}
		return map[string]interface{}{"status": "ready"}
	}
	if d.depsInitStarted {
		d.mu.Unlock()
		return map[string]interface{}{"status": "initializing"}
	}
	d.depsInitStarted = true
	d.mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprintf("dependency setup panic: %v", r)
				if d.ctx != nil {
					wailsRuntime.EventsEmit(d.ctx, "deps-error", map[string]interface{}{"error": msg})
				}
				d.mu.Lock()
				d.depsInitStarted = false // allow retry
				d.mu.Unlock()
			}
		}()

		// Capture installer output and emit as UI events.
		// Never assign nil writers — fmt.Fprintf panics on nil io.Writer, and
		// concurrent callers used to race on cfg.Stdout/Stderr.
		pr, pw := io.Pipe()
		d.mu.Lock()
		d.cfg.Stdout = pw
		d.cfg.Stderr = pw
		d.mu.Unlock()

		go func() {
			scanner := bufio.NewScanner(pr)
			for scanner.Scan() {
				line := scanner.Text()
				if line != "" && d.ctx != nil {
					wailsRuntime.EventsEmit(d.ctx, "deps-progress", map[string]interface{}{
						"message": line,
					})
				}
			}
		}()

		dl, err := downloader.New(d.cfg)
		_ = pw.Close()

		// Restore non-nil discard writers (downloads attach their own pipes)
		d.mu.Lock()
		d.cfg.Stdout = io.Discard
		d.cfg.Stderr = io.Discard
		d.mu.Unlock()

		if err != nil {
			if d.ctx != nil {
				wailsRuntime.EventsEmit(d.ctx, "deps-error", map[string]interface{}{
					"error": err.Error(),
				})
			}
			d.mu.Lock()
			d.depsInitStarted = false // allow retry
			d.mu.Unlock()
			return
		}
		d.mu.Lock()
		d.dl = dl
		d.depsReady = true
		d.mu.Unlock()
		if d.ctx != nil {
			wailsRuntime.EventsEmit(d.ctx, "deps-ready", map[string]interface{}{
				"status": "ready",
			})
		}
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
// Never blocks the UI forever: yt-dlp is bounded and OEmbed is used as backup.
func (d *DownloadService) FetchMetadata(rawURL string) map[string]interface{} {
	url := cleanURL(rawURL)

	// Try yt-dlp first (if initialized), with a hard wall-clock budget
	if d.dl != nil {
		type metaResult struct {
			playlistInfo string
			title        string
			err          error
		}
		ch := make(chan metaResult, 1)
		go func() {
			pi, title, err := d.dl.GetMetadata([]string{url})
			ch <- metaResult{pi, title, err}
		}()

		select {
		case res := <-ch:
			if res.err == nil && res.title != "" {
				result := map[string]interface{}{"title": res.title}
				if res.playlistInfo != "" {
					parts := strings.Split(res.playlistInfo, "&")
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
		case <-time.After(50 * time.Second):
			// yt-dlp hung (cookie DB / network) — use OEmbed
		}
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
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
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

// FetchInfo gets metadata and formats in a single yt-dlp call.
// Preferred over FetchMetadata + ListFormats (avoids double network round-trip).
func (d *DownloadService) FetchInfo(rawURL string) map[string]interface{} {
	url := cleanURL(rawURL)
	if d.dl == nil {
		// Still try OEmbed metadata so UI is not empty
		if fb := d.fetchMetadataFallback(url); fb != nil {
			fb["video"] = []map[string]interface{}{}
			fb["audio"] = []map[string]interface{}{}
			return fb
		}
		return map[string]interface{}{"error": "downloader not initialized"}
	}

	type infoResult struct {
		info *downloader.VideoInfo
		err  error
	}
	ch := make(chan infoResult, 1)
	go func() {
		info, err := d.dl.GetVideoInfo(url)
		ch <- infoResult{info, err}
	}()

	select {
	case res := <-ch:
		if res.err == nil && res.info != nil && res.info.Title != "" {
			result := map[string]interface{}{
				"title": res.info.Title,
			}
			if res.info.Uploader != "" {
				result["uploader"] = res.info.Uploader
			}
			if res.info.Duration > 0 {
				result["duration"] = res.info.Duration
			}
			thumb := res.info.Thumbnail
			if thumb == "" {
				thumb = d.getThumbnailURL(url)
			}
			if thumb != "" {
				result["thumbnail"] = thumb
			}
			videoFmts, audioFmts := splitFormats(res.info.Formats)
			result["video"] = videoFmts
			result["audio"] = audioFmts
			return result
		}
	case <-time.After(50 * time.Second):
		// hung — fall through
	}

	// Fallback: OEmbed metadata only (no formats)
	if fb := d.fetchMetadataFallback(url); fb != nil {
		fb["video"] = []map[string]interface{}{}
		fb["audio"] = []map[string]interface{}{}
		return fb
	}
	thumb := d.getThumbnailURL(url)
	r := map[string]interface{}{
		"title": url,
		"video": []map[string]interface{}{},
		"audio": []map[string]interface{}{},
	}
	if thumb != "" {
		r["thumbnail"] = thumb
	}
	return r
}

// ListFormats lists available video formats/resolutions for a URL.
// Returns {video: [...], audio: [...]} separated by type.
// Prefer FetchInfo when metadata is also needed.
func (d *DownloadService) ListFormats(rawURL string) map[string]interface{} {
	if d.dl == nil {
		return map[string]interface{}{"error": "downloader not initialized"}
	}
	url := cleanURL(rawURL)

	// Prefer single JSON dump (same path as FetchInfo) for reliability
	if info, err := d.dl.GetVideoInfo(url); err == nil && info != nil {
		videoFmts, audioFmts := splitFormats(info.Formats)
		return map[string]interface{}{
			"video": videoFmts,
			"audio": audioFmts,
		}
	}

	formats, err := d.dl.GetFormats(url)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	videoFmts, audioFmts := splitFormats(formats)
	return map[string]interface{}{
		"video": videoFmts,
		"audio": audioFmts,
	}
}

func splitFormats(formats []downloader.Format) (videoFmts, audioFmts []map[string]interface{}) {
	for _, f := range formats {
		ext := f.Ext
		extLower := strings.ToLower(ext)
		if strings.Contains(extLower, ".") && !strings.HasPrefix(extLower, "mp4") && !strings.HasPrefix(extLower, "webm") {
			ext = ""
		}
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
	return videoFmts, audioFmts
}

// StartDownload begins a download in a goroutine and emits progress events.
// Returns immediately with the download ID. If the max concurrent limit is
// reached, the download is placed in a wait queue.
func (d *DownloadService) StartDownload(rawURL, resolution, downloadDir string, audioOnly bool, audioFormat string, containerFormat string) map[string]interface{} {
	return d.startDownload(rawURL, "", "", resolution, downloadDir, audioOnly, audioFormat, containerFormat)
}

// startDownload is the shared implementation; title/referer may be preset (extension).
func (d *DownloadService) startDownload(rawURL, title, referer, resolution, downloadDir string, audioOnly bool, audioFormat string, containerFormat string) map[string]interface{} {
	if d.dl == nil {
		return map[string]interface{}{"error": "downloader not initialized"}
	}
	url := cleanURL(rawURL)
	if containerFormat == "" {
		containerFormat = "mp4"
	}
	destCheck := expandTilde(downloadDir)
	if destCheck == "" {
		home, _ := os.UserHomeDir()
		destCheck = filepath.Join(home, "Downloads")
	}
	if err := ensureDiskSpace(destCheck, minFreeDiskBytes); err != nil {
		return map[string]interface{}{"error": err.Error(), "error_type": "disk_full"}
	}

	d.mu.Lock()
	d.nextID++
	id := fmt.Sprintf("dl_%d", d.nextID)

	thumb := d.getThumbnailURL(url)
	displayTitle := strings.TrimSpace(title)
	if displayTitle == "" {
		displayTitle = url
	}
	now := time.Now()
	ad := &activeDownload{
		ID:              id,
		URL:             url,
		Title:           displayTitle,
		Thumbnail:       thumb,
		Status:          "queued",
		StartedAt:       now.Format("2006-01-02 15:04:05"),
		startedUnix:     now.UnixMilli(),
		Resolution:      resolution,
		DownloadDir:     downloadDir,
		AudioOnly:       audioOnly,
		AudioFormat:     audioFormat,
		ContainerFormat: containerFormat,
		Referer:         strings.TrimSpace(referer),
	}
	d.downloads[id] = ad

	if d.running >= d.maxRunning {
		d.waitQueue = append(d.waitQueue, id)
		d.mu.Unlock()
		d.updateDownload(id, "queued", 0, "", "", "")
		return map[string]interface{}{"id": id, "status": "queued"}
	}
	d.running++
	// Stagger parallel starts slightly so the same CDN is less likely to reset all at once
	delay := time.Duration(d.running-1) * 400 * time.Millisecond
	d.mu.Unlock()

	go func() {
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-d.ctx.Done():
				return
			}
		}
		d.runDownload(id, url, resolution, downloadDir, audioOnly, audioFormat, containerFormat)
	}()

	return map[string]interface{}{
		"id":     id,
		"status": "queued",
	}
}

// runDownload executes the actual download in a goroutine. When finished,
// it starts the next queued download if any.
func (d *DownloadService) runDownload(id, url, resolution, downloadDir string, audioOnly bool, audioFormat string, containerFormat string) {
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

	// Fetch metadata first (skipped / soft-fail for direct file URLs from extension)
	d.updateDownload(id, "metadata", 0, "", "", "")
	directFile := downloader.IsDirectMediaURL(url)
	if directFile {
		// Keep extension-provided title when present
		d.mu.Lock()
		if ad.Title == "" || ad.Title == ad.URL {
			ad.Title = titleFromMediaURL(url)
		}
		d.mu.Unlock()
	} else {
		_, title, err := d.dl.GetMetadata([]string{url})
		if err != nil {
			errLow := strings.ToLower(err.Error())
			// Soft-fail: missing title alone must not kill the download
			soft := strings.Contains(errLow, "no title found") ||
				strings.Contains(errLow, "unable to extract title")
			if soft {
				d.mu.Lock()
				if ad.Title == "" || ad.Title == ad.URL {
					if t := titleFromMediaURL(url); t != "" {
						ad.Title = t
					} else {
						ad.Title = "Download"
					}
				}
				d.mu.Unlock()
			} else {
				_, userMsg := classifyError(err.Error())
				if strings.Contains(errLow, "json") || strings.Contains(strings.ToLower(userMsg), "json") {
					userMsg = "Site extractor failed. In the extension, pick a quality (not only Best/yt-dlp), or update yt-dlp."
				}
				d.updateDownload(id, "error", 0, "", "", userMsg)
				return
			}
		} else {
			d.mu.Lock()
			if title != "" {
				ad.Title = title
			} else if ad.Title == "" || ad.Title == ad.URL {
				ad.Title = titleFromMediaURL(url)
			}
			d.mu.Unlock()
		}
	}

	// Resolve destination directory (expand ~ since Go doesn't do shell expansion)
	dest := expandTilde(downloadDir)
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
	origContainer := d.cfg.ContainerFormat
	origReferer := d.cfg.Referer
	origAria := d.cfg.UseAria2c
	ref := ad.Referer
	if resolution != "" {
		d.cfg.Resolution = resolutionToFormat(resolution)
	}
	d.cfg.IsAudioOnly = audioOnly
	if audioFormat != "" {
		d.cfg.AudioFormat = audioFormat
	}
	d.cfg.ContainerFormat = containerFormat
	// Leading dot hides incomplete output; finalize unveils after success.
	// Folder mode: .Title/Title.ext → Title/Title.ext
	// Flat mode:    .Title.ext → Title.ext
	if appconfig.YariaDownloadInFolder() {
		d.cfg.OutputTemplate = ".%(title)s/%(title)s.%(ext)s"
	} else {
		d.cfg.OutputTemplate = ".%(title)s.%(ext)s"
	}
	d.cfg.Referer = ref
	// aria2 stays enabled when configured; downloader tries aria2 first on
	// direct files and retries without it if the CDN resets the connection.
	d.mu.Unlock()

	// Pipe stdout/stderr through a progress parser
	pr, pw := io.Pipe()
	d.mu.Lock()
	ad.pipeWriter = pw
	d.mu.Unlock()
	d.cfg.Stdout = pw
	d.cfg.Stderr = pw // never leave these nil — fmt.Fprintf panics on nil Writer
	go d.parseProgress(dlCtx, id, pr)

	d.updateDownload(id, "downloading", 0, "", "", "")

	// Download directly into dest -- yt-dlp creates %(title)s/ subfolder
	// User can open the folder during download and see aria2 pieces
	success, dlErr := d.dl.Download([]string{url}, dest)
	pw.Close()

	// Restore original config (writers must stay non-nil)
	d.mu.Lock()
	d.cfg.Resolution = origRes
	d.cfg.IsAudioOnly = origAudio
	d.cfg.AudioFormat = origAudioFmt
	d.cfg.OutputTemplate = origTemplate
	d.cfg.ContainerFormat = origContainer
	d.cfg.Referer = origReferer
	d.cfg.UseAria2c = origAria
	d.cfg.Stdout = io.Discard
	d.cfg.Stderr = io.Discard
	d.mu.Unlock()

	// Check for cancellation / pause — don't overwrite those statuses
	select {
	case <-dlCtx.Done():
		d.mu.Lock()
		st := ""
		if a, ok := d.downloads[id]; ok {
			st = a.Status
		}
		d.mu.Unlock()
		if st != "paused" && st != "cancelled" {
			d.updateDownload(id, "cancelled", 0, "", "", "")
		}
		return
	default:
	}

	if dlErr != nil || !success {
		d.mu.Lock()
		st := ""
		if a, ok := d.downloads[id]; ok {
			st = a.Status
			if st == "paused" || st == "cancelled" {
				d.mu.Unlock()
				return
			}
		}
		errMsg := "download failed"
		if dlErr != nil {
			errMsg = dlErr.Error()
		}
		// Prefer a more specific ERROR line captured from yt-dlp stderr (retries etc.)
		if ad != nil {
			captured := ad.lastYtdlpErr
			if captured == "" {
				captured = ad.Error
			}
			if captured != "" && (errMsg == "download failed" || strings.Contains(errMsg, "all download attempts")) {
				errMsg = captured
			}
		}
		d.mu.Unlock()
		errType, userMsg := classifyError(errMsg)
		d.updateDownload(id, "error", 0, "", "", userMsg)
		_ = errType
		return
	}

	// Finalize: unveil hidden .Title/ folders, reject leftover .part fragments
	d.mu.Lock()
	title := ""
	if ad != nil {
		title = ad.Title
	}
	d.mu.Unlock()

	started := time.Time{}
	d.mu.Lock()
	if ad != nil {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", ad.StartedAt, time.Local); err == nil {
			started = t.Add(-2 * time.Minute)
		} else if t, err := time.ParseInLocation("2006-01-02 15:04", ad.StartedAt, time.Local); err == nil {
			started = t.Add(-2 * time.Minute)
		}
	}
	d.mu.Unlock()

	videoPath, finErr := finalizeDownloadOutput(dest, title, started)
	if finErr != nil {
		d.mu.Lock()
		st := ""
		if a, ok := d.downloads[id]; ok {
			st = a.Status
		}
		d.mu.Unlock()
		if st != "paused" && st != "cancelled" {
			d.updateDownload(id, "error", 0, "", "", finErr.Error())
		}
		return
	}

	d.mu.Lock()
	if ad != nil {
		ad.FilePath = videoPath
	}
	d.mu.Unlock()

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
			go d.runDownload(nextID, ad.URL, ad.Resolution, ad.DownloadDir, ad.AudioOnly, ad.AudioFormat, ad.ContainerFormat)
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

// PauseDownload stops an in-progress download but keeps the entry for resume.
// Partial files on disk are left in place (yt-dlp can continue later).
func (d *DownloadService) PauseDownload(id string) map[string]interface{} {
	d.mu.Lock()
	ad, ok := d.downloads[id]
	if !ok {
		d.mu.Unlock()
		return map[string]interface{}{"error": "download not found"}
	}
	st := ad.Status
	if st != "downloading" && st != "metadata" && st != "queued" && st != "processing" {
		d.mu.Unlock()
		return map[string]interface{}{"error": "download is not active"}
	}
	// Remove from wait queue if queued
	if st == "queued" {
		nq := d.waitQueue[:0]
		for _, qid := range d.waitQueue {
			if qid != id {
				nq = append(nq, qid)
			}
		}
		d.waitQueue = nq
	}
	cancel := ad.cancel
	pw := ad.pipeWriter
	pct := ad.Percent
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if pw != nil {
		pw.Close()
	}
	d.updateDownload(id, "paused", pct, "", "", "")
	return map[string]interface{}{"status": "paused"}
}

// ResumeDownload continues a paused (or failed) download with the same settings.
func (d *DownloadService) ResumeDownload(id string) map[string]interface{} {
	return d.restartDownload(id, false)
}

// RetryDownload restarts an errored or cancelled download.
func (d *DownloadService) RetryDownload(id string) map[string]interface{} {
	return d.restartDownload(id, true)
}

func (d *DownloadService) restartDownload(id string, fromError bool) map[string]interface{} {
	if d.dl == nil {
		return map[string]interface{}{"error": "downloader not initialized"}
	}

	d.mu.Lock()
	ad, ok := d.downloads[id]
	var url, title, referer, resolution, downloadDir, audioFormat, containerFormat string
	var audioOnly bool
	var thumb string
	if ok {
		st := ad.Status
		if fromError {
			if st != "error" && st != "cancelled" && st != "paused" {
				d.mu.Unlock()
				return map[string]interface{}{"error": "only failed, cancelled, or paused downloads can be retried"}
			}
		} else if st != "paused" && st != "error" && st != "cancelled" {
			d.mu.Unlock()
			return map[string]interface{}{"error": "download is not paused"}
		}
		url = ad.URL
		title = ad.Title
		referer = ad.Referer
		resolution = ad.Resolution
		downloadDir = ad.DownloadDir
		audioOnly = ad.AudioOnly
		audioFormat = ad.AudioFormat
		containerFormat = ad.ContainerFormat
		thumb = ad.Thumbnail
		// Reset runtime fields
		ad.Status = "queued"
		ad.Error = ""
		ad.lastYtdlpErr = ""
		ad.Percent = 0
		ad.Speed = ""
		ad.ETA = ""
		ad.cancel = nil
		ad.pipeWriter = nil
	}
	d.mu.Unlock()

	if !ok {
		// History-only record from previous session
		if d.store == nil {
			return map[string]interface{}{"error": "download not found"}
		}
		r, err := d.store.Get(id)
		if err != nil || r == nil {
			return map[string]interface{}{"error": "download not found"}
		}
		if r.Status != "error" && r.Status != "cancelled" && r.Status != "paused" {
			return map[string]interface{}{"error": "only failed or cancelled downloads can be retried"}
		}
		url = r.URL
		title = r.Title
		thumb = r.Thumbnail
		downloadDir = ""
		if r.FilePath != "" {
			// parent of title folder
			downloadDir = filepath.Dir(filepath.Dir(r.FilePath))
		}
		if downloadDir == "" {
			home, _ := os.UserHomeDir()
			downloadDir = filepath.Join(home, "Downloads")
		}
		containerFormat = "mp4"
		// Re-create active entry with same id
		d.mu.Lock()
		now := time.Now()
		ad = &activeDownload{
			ID:              id,
			URL:             url,
			Title:           title,
			Thumbnail:       thumb,
			Status:          "queued",
			StartedAt:       now.Format("2006-01-02 15:04:05"),
			startedUnix:     now.UnixMilli(),
			DownloadDir:     downloadDir,
			ContainerFormat: containerFormat,
		}
		d.downloads[id] = ad
		d.mu.Unlock()
	}

	d.mu.Lock()
	if d.running >= d.maxRunning {
		// avoid duplicate queue entries
		found := false
		for _, qid := range d.waitQueue {
			if qid == id {
				found = true
				break
			}
		}
		if !found {
			d.waitQueue = append(d.waitQueue, id)
		}
		d.mu.Unlock()
		d.updateDownload(id, "queued", 0, "", "", "")
		return map[string]interface{}{"id": id, "status": "queued"}
	}
	d.running++
	d.mu.Unlock()

	_ = title
	_ = referer
	go d.runDownload(id, url, resolution, downloadDir, audioOnly, audioFormat, containerFormat)
	return map[string]interface{}{"id": id, "status": "queued"}
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

	// Stable newest-first order (map iteration is random in Go).
	// Tie-break with id so equal timestamps never shuffle the UI.
	sort.SliceStable(active, func(i, j int) bool {
		if active[i].startedUnix != active[j].startedUnix {
			return active[i].startedUnix > active[j].startedUnix
		}
		if active[i].StartedAt != active[j].StartedAt {
			return active[i].StartedAt > active[j].StartedAt
		}
		return active[i].ID > active[j].ID
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
			"started_unix": ad.startedUnix,
			"file_path":    ad.FilePath,
			"download_dir": ad.DownloadDir,
		})
	}

	// Add stored history (completed/errored downloads from previous sessions)
	type histRow struct {
		m map[string]interface{}
		t string
		id string
	}
	var history []histRow
	if d.store != nil {
		for _, r := range d.store.GetAll() {
			if !activeIDs[r.ID] {
				// Get directory from file path
				dlDir := ""
				if r.FilePath != "" {
					dlDir = filepath.Dir(filepath.Dir(r.FilePath)) // parent of title folder
				}
				history = append(history, histRow{
					t:  r.StartedAt,
					id: r.ID,
					m: map[string]interface{}{
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
					},
				})
			}
		}
	}
	sort.SliceStable(history, func(i, j int) bool {
		if history[i].t != history[j].t {
			return history[i].t > history[j].t
		}
		return history[i].id > history[j].id
	})
	for _, h := range history {
		result = append(result, h.m)
	}
	return result
}

// RemoveDownload removes a download from the list (only if complete/error/cancelled/paused).
func (d *DownloadService) RemoveDownload(id string) map[string]interface{} {
	d.mu.Lock()
	ad, ok := d.downloads[id]
	if ok && (ad.Status == "complete" || ad.Status == "error" || ad.Status == "cancelled" || ad.Status == "paused") {
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
		if isSafeToDeleteDir(dir) {
			os.RemoveAll(dir)
		}
	}

	// Remove from list and store
	return d.RemoveDownload(id)
}

// PlayDownloadedFile resolves the downloaded file path for in-app play.
// Never scans a shared download root for "largest file" — that picks the wrong video.
func (d *DownloadService) PlayDownloadedFile(id string) map[string]interface{} {
	var filePath, title, downloadDir string
	d.mu.Lock()
	if ad, ok := d.downloads[id]; ok {
		filePath = ad.FilePath
		title = ad.Title
		downloadDir = ad.DownloadDir
	}
	d.mu.Unlock()

	if d.store != nil {
		if r, err := d.store.Get(id); err == nil {
			if filePath == "" {
				filePath = r.FilePath
			}
			if title == "" {
				title = r.Title
			}
		}
	}

	// Stored path valid? (never play .part / aria2 fragments or hidden leftovers)
	if filePath != "" {
		_ = unveilPathParents(filePath)
		// Re-resolve after unveil
		if strings.Contains(filepath.Base(filepath.Dir(filePath)), ".") {
			if f := findDownloadVideo(filepath.Dir(filepath.Dir(filePath)), title, time.Time{}); f != "" {
				filePath = f
			}
		}
		if st, err := os.Stat(filePath); err == nil && !st.IsDir() && !isIncompleteMediaPath(filePath) {
			// Prefer path not under a still-hidden directory
			if !pathHasHiddenComponent(filePath) {
				return map[string]interface{}{
					"status": "ok",
					"file":   filePath,
					"path":   filePath,
					"title":  filepath.Base(filePath),
				}
			}
		}
		// Stale / incomplete / hidden — search only its parent title folder
		parent := filepath.Dir(filePath)
		_ = unveilPathParents(filePath)
		if found := findDownloadVideo(parent, title, time.Time{}); found != "" && !isIncompleteMediaPath(found) {
			return map[string]interface{}{
				"status": "ok",
				"file":   found,
				"path":   found,
				"title":  filepath.Base(found),
			}
		}
	}

	// Resolve via title under download dir (title subfolder only)
	dirs := []string{}
	if downloadDir != "" {
		dirs = append(dirs, expandTilde(downloadDir))
	}
	d.mu.Lock()
	if d.cfg != nil && d.cfg.DownloadLocation != "" {
		dirs = append(dirs, expandTilde(d.cfg.DownloadLocation))
	}
	d.mu.Unlock()
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Downloads", "Mantorex"))
	}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		unveilHiddenDownloadDirs(dir)
		if found := findDownloadVideo(dir, title, time.Time{}); found != "" && !isIncompleteMediaPath(found) && !pathHasHiddenComponent(found) {
			// Persist corrected path
			if d.store != nil {
				if r, err := d.store.Get(id); err == nil {
					r.FilePath = found
					if info, err := os.Stat(found); err == nil {
						r.FileSize = info.Size()
					}
					_ = d.store.Save(*r)
				}
			}
			d.mu.Lock()
			if ad, ok := d.downloads[id]; ok {
				ad.FilePath = found
			}
			d.mu.Unlock()
			return map[string]interface{}{
				"status": "ok",
				"file":   found,
				"path":   found,
				"title":  filepath.Base(found),
			}
		}
	}

	return map[string]interface{}{"error": "file path not found"}
}

var downloadVideoExts = map[string]bool{
	".mp4": true, ".mkv": true, ".webm": true, ".avi": true, ".mov": true, ".m4v": true, ".ts": true,
}

// findDownloadVideo locates the video for one download inside dest.
// Prefer a title-matching subfolder; never return an unrelated sibling download.
func findDownloadVideo(dest, title string, notBefore time.Time) string {
	if dest == "" {
		return ""
	}
	dest = expandTilde(dest)
	if st, err := os.Stat(dest); err != nil {
		return ""
	} else if !st.IsDir() {
		// dest is already a file
		ext := strings.ToLower(filepath.Ext(dest))
		if downloadVideoExts[ext] {
			return dest
		}
		return ""
	}

	normTitle := normalizeTitleKey(title)

	// 1) Exact / fuzzy match title subfolder OR flat file Title.ext in dest
	if normTitle != "" {
		entries, _ := os.ReadDir(dest)
		var matchedDirs []string
		var bestFlat string
		var bestFlatTime time.Time
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			if e.IsDir() {
				if titleKeysMatch(normTitle, normalizeTitleKey(name)) {
					matchedDirs = append(matchedDirs, filepath.Join(dest, name))
				}
				continue
			}
			// Flat layout: Title.ext
			ext := strings.ToLower(filepath.Ext(name))
			if !downloadVideoExts[ext] || isIncompleteMediaPath(name) {
				continue
			}
			base := strings.TrimSuffix(name, filepath.Ext(name))
			if !titleKeysMatch(normTitle, normalizeTitleKey(base)) {
				continue
			}
			p := filepath.Join(dest, name)
			info, err := e.Info()
			if err != nil {
				continue
			}
			if !notBefore.IsZero() && info.ModTime().Before(notBefore.Add(-2*time.Second)) {
				continue
			}
			if bestFlat == "" || info.ModTime().After(bestFlatTime) {
				bestFlat, bestFlatTime = p, info.ModTime()
			}
		}
		// Newest video among matched title folders
		var best string
		var bestTime time.Time
		for _, dir := range matchedDirs {
			if f, t := newestVideoIn(dir, notBefore); f != "" && (best == "" || t.After(bestTime)) {
				best, bestTime = f, t
			}
		}
		if best != "" && (bestFlat == "" || bestTime.After(bestFlatTime) || bestTime.Equal(bestFlatTime)) {
			return best
		}
		if bestFlat != "" {
			return bestFlat
		}
		if best != "" {
			return best
		}
	}

	// 2) If dest itself looks like a title folder (contains videos only for one item)
	if f, _ := newestVideoIn(dest, notBefore); f != "" {
		// Only accept if file is directly in dest or one level down matching title
		rel, err := filepath.Rel(dest, f)
		if err == nil {
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) == 1 {
				// loose file in dest — only OK when notBefore filters to this download
				if !notBefore.IsZero() {
					return f
				}
			} else if len(parts) >= 2 && normTitle != "" && titleKeysMatch(normTitle, normalizeTitleKey(parts[0])) {
				return f
			} else if len(parts) >= 2 && !notBefore.IsZero() {
				// recent file in some subfolder after download started
				return f
			}
		}
	}

	// 3) Newest video under dest modified after notBefore (this session only)
	if !notBefore.IsZero() {
		if f, _ := newestVideoIn(dest, notBefore); f != "" {
			return f
		}
	}

	return ""
}

// finalizeDownloadOutput unveils hidden download folders, ensures no .part
// fragments remain, and returns the playable media path.
func finalizeDownloadOutput(dest, title string, notBefore time.Time) (string, error) {
	dest = expandTilde(dest)
	if dest == "" {
		return "", fmt.Errorf("download folder missing")
	}

	// Unveil .Title → Title dirs and .Title.ext → Title.ext flat files
	unveilHiddenDownloadDirs(dest)

	// Flat incomplete fragments for this title in download root (e.g. .Title.mp4.part)
	if titleHasIncompleteInRoot(dest, title, notBefore) {
		time.Sleep(1500 * time.Millisecond)
		unveilHiddenDownloadDirs(dest)
		if titleHasIncompleteInRoot(dest, title, notBefore) {
			return "", fmt.Errorf("download incomplete — still fragmented (.part). Use Retry")
		}
	}

	// Prefer folders matching this title (visible first, then still-hidden)
	candidates := downloadDirsForTitle(dest, title, notBefore)
	if len(candidates) == 0 {
		// Flat or loose file under dest
		if f := findDownloadVideo(dest, title, notBefore); f != "" {
			if isIncompleteMediaPath(f) || pathHasHiddenComponent(f) {
				return "", fmt.Errorf("download incomplete — fragment files remain (.part). Use Retry")
			}
			_ = unveilPathParents(f)
			if f2 := findDownloadVideo(dest, title, notBefore); f2 != "" && !isIncompleteMediaPath(f2) {
				f = f2
			}
			return f, nil
		}
		return "", fmt.Errorf("download finished but no video file was found. Use Retry")
	}

	// Check each candidate folder for incomplete aria2/yt-dlp parts
	var bestVideo string
	var bestTime time.Time
	for _, dir := range candidates {
		if hasIncompleteFragments(dir) {
			// One more moment — ffmpeg merge sometimes lags process exit
			time.Sleep(1500 * time.Millisecond)
			unveilHiddenDownloadDirs(dest)
			if hasIncompleteFragments(dir) {
				// Still fragmented → not actually complete
				return "", fmt.Errorf("download incomplete — still fragmented (.part). Use Retry")
			}
		}
		if f, t := newestVideoIn(dir, notBefore); f != "" && (bestVideo == "" || t.After(bestTime)) {
			bestVideo, bestTime = f, t
		}
	}
	if bestVideo == "" {
		// Try whole dest once more after unveil
		if f := findDownloadVideo(dest, title, notBefore); f != "" && !isIncompleteMediaPath(f) {
			return f, nil
		}
		return "", fmt.Errorf("download incomplete — no finished video file. Use Retry")
	}
	if isIncompleteMediaPath(bestVideo) {
		return "", fmt.Errorf("download incomplete — fragment files remain (.part). Use Retry")
	}
	return bestVideo, nil
}

func unveilHiddenDownloadDirs(dest string) {
	entries, err := os.ReadDir(dest)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if name == "." || name == ".." || !strings.HasPrefix(name, ".") {
			continue
		}
		// Skip system/dotdirs that aren't download outputs
		if name == ".cache" || name == ".config" || name == ".local" || name == ".yaria" {
			continue
		}
		hidden := filepath.Join(dest, name)
		visible := filepath.Join(dest, name[1:])
		if e.IsDir() {
			_ = unveilDir(hidden, visible)
			continue
		}
		// Flat mode: .Title.ext → Title.ext (skip incomplete parts)
		if isIncompleteMediaPath(hidden) {
			continue
		}
		if _, err := os.Stat(visible); err == nil {
			// Prefer larger file if both exist
			si, _ := os.Stat(hidden)
			di, _ := os.Stat(visible)
			if si != nil && di != nil && si.Size() > di.Size() {
				_ = os.Remove(visible)
				_ = os.Rename(hidden, visible)
			} else {
				_ = os.Remove(hidden)
			}
			continue
		}
		_ = os.Rename(hidden, visible)
	}
}

func unveilDir(hidden, visible string) error {
	if _, err := os.Stat(hidden); err != nil {
		return err
	}
	if _, err := os.Stat(visible); os.IsNotExist(err) {
		return os.Rename(hidden, visible)
	}
	// Visible already exists — move contents then remove hidden
	entries, err := os.ReadDir(hidden)
	if err != nil {
		return err
	}
	for _, e := range entries {
		src := filepath.Join(hidden, e.Name())
		dst := filepath.Join(visible, e.Name())
		if _, err := os.Stat(dst); err == nil {
			// prefer larger / newer file
			si, _ := os.Stat(src)
			di, _ := os.Stat(dst)
			if si != nil && di != nil && si.Size() > di.Size() {
				_ = os.Remove(dst)
				_ = os.Rename(src, dst)
			} else {
				_ = os.RemoveAll(src)
			}
			continue
		}
		_ = os.Rename(src, dst)
	}
	// Remove leftover empty hidden dir (ignore if not empty)
	_ = os.Remove(hidden)
	return nil
}

func unveilPathParents(filePath string) error {
	dir := filepath.Dir(filePath)
	base := filepath.Base(dir)
	if strings.HasPrefix(base, ".") && base != "." && base != ".." {
		parent := filepath.Dir(dir)
		return unveilDir(dir, filepath.Join(parent, base[1:]))
	}
	return nil
}

func downloadDirsForTitle(dest, title string, notBefore time.Time) []string {
	normTitle := normalizeTitleKey(title)
	var dirs []string
	entries, _ := os.ReadDir(dest)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Match both visible and still-hidden names
		key := name
		if strings.HasPrefix(key, ".") {
			key = key[1:]
		}
		if normTitle != "" && !titleKeysMatch(normTitle, normalizeTitleKey(key)) {
			// Also accept any recent dir if title match fails (sanitized names differ)
			if notBefore.IsZero() {
				continue
			}
			info, err := e.Info()
			if err != nil || info.ModTime().Before(notBefore) {
				continue
			}
		}
		dirs = append(dirs, filepath.Join(dest, name))
	}
	// Unveil matched hidden dirs now
	var out []string
	for _, d := range dirs {
		base := filepath.Base(d)
		if strings.HasPrefix(base, ".") {
			vis := filepath.Join(filepath.Dir(d), base[1:])
			_ = unveilDir(d, vis)
			if st, err := os.Stat(vis); err == nil && st.IsDir() {
				out = append(out, vis)
				continue
			}
		}
		out = append(out, d)
	}
	return out
}

func hasIncompleteFragments(dir string) bool {
	found := false
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if isIncompleteMediaPath(path) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

// titleHasIncompleteInRoot detects flat-mode .part files for this title under dest.
func titleHasIncompleteInRoot(dest, title string, notBefore time.Time) bool {
	entries, err := os.ReadDir(dest)
	if err != nil {
		return false
	}
	normTitle := normalizeTitleKey(title)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isIncompleteMediaPath(name) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !notBefore.IsZero() && info.ModTime().Before(notBefore.Add(-2*time.Second)) {
			continue
		}
		// Strip leading '.' and incomplete suffixes for title match
		check := name
		if strings.HasPrefix(check, ".") {
			check = check[1:]
		}
		lower := strings.ToLower(check)
		for _, suf := range []string{".part.aria2", ".aria2", ".ytdl", ".part"} {
			if strings.HasSuffix(lower, suf) {
				check = check[:len(check)-len(suf)]
				lower = strings.ToLower(check)
			}
		}
		// also strip media ext: Title.mp4 left after .part removed
		base := strings.TrimSuffix(check, filepath.Ext(check))
		if normTitle == "" || titleKeysMatch(normTitle, normalizeTitleKey(base)) ||
			titleKeysMatch(normTitle, normalizeTitleKey(check)) {
			return true
		}
	}
	return false
}

func isIncompleteMediaPath(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	if strings.HasSuffix(name, ".part") || strings.HasSuffix(name, ".part.aria2") ||
		strings.HasSuffix(name, ".aria2") || strings.HasSuffix(name, ".ytdl") ||
		strings.Contains(name, ".part.") {
		return true
	}
	// temp yt-dlp formats like file.f137.mp4.part already covered by .part
	return false
}

func pathHasHiddenComponent(path string) bool {
	// true if any directory component starts with '.' (except . and ..)
	for _, p := range strings.Split(filepath.Clean(path), string(filepath.Separator)) {
		if p == "" || p == "." || p == ".." {
			continue
		}
		if strings.HasPrefix(p, ".") {
			return true
		}
	}
	return false
}

func newestVideoIn(dir string, notBefore time.Time) (string, time.Time) {
	var best string
	var bestTime time.Time
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Never descend into unrelated hidden dirs except our own (already unveiled)
		if info.IsDir() {
			base := info.Name()
			if strings.HasPrefix(base, ".") && base != "." && base != ".." {
				return filepath.SkipDir
			}
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, ".") || isIncompleteMediaPath(path) {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		if !downloadVideoExts[ext] {
			return nil
		}
		mt := info.ModTime()
		if !notBefore.IsZero() && mt.Before(notBefore) {
			return nil
		}
		if best == "" || mt.After(bestTime) || (mt.Equal(bestTime) && info.Size() > 0 && path > best) {
			// prefer newer; on tie prefer larger via secondary walk compare
			if best == "" || mt.After(bestTime) {
				best, bestTime = path, mt
			} else if mt.Equal(bestTime) {
				if prev, err := os.Stat(best); err == nil && info.Size() > prev.Size() {
					best, bestTime = path, mt
				}
			}
		}
		return nil
	})
	return best, bestTime
}

func normalizeTitleKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	// Keep letters/digits only so punctuation differences (： vs :, ｜ vs |) match
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func titleKeysMatch(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	// Prefix / contains for truncated sanitization
	if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		return true
	}
	if len(a) >= 12 && len(b) >= 12 {
		// Compare first 24 chars of normalized form
		n := 24
		if len(a) < n {
			n = len(a)
		}
		if len(b) < n {
			n = len(b)
		}
		return a[:n] == b[:n]
	}
	return false
}

// GetDownloadInFolder returns whether downloads are stored in a title subfolder.
func (d *DownloadService) GetDownloadInFolder() bool {
	return appconfig.YariaDownloadInFolder()
}

// SetDownloadInFolder toggles title-subfolder vs flat file layout (persisted).
func (d *DownloadService) SetDownloadInFolder(inFolder bool) map[string]interface{} {
	if err := appconfig.SetYariaDownloadInFolder(inFolder); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok", "in_folder": inFolder}
}

// SetDownloadDir sets the default download directory (persisted for Bridge + app).
func (d *DownloadService) SetDownloadDir(dir string) map[string]interface{} {
	dir = strings.TrimSpace(dir)
	d.mu.Lock()
	d.cfg.DownloadLocation = dir
	d.mu.Unlock()
	_ = appconfig.SetBrowserExtensionDownloadDir(dir)
	out := dir
	if out != "" {
		out = expandTilde(out)
	}
	return map[string]interface{}{"status": "ok", "dir": out}
}

// GetDownloadDir returns the current download directory (with ~ expanded).
// Used by Yaria Bridge extension downloads and the main downloader UI.
func (d *DownloadService) GetDownloadDir() string {
	d.mu.Lock()
	loc := d.cfg.DownloadLocation
	d.mu.Unlock()
	if loc == "" {
		loc = appconfig.BrowserExtensionDownloadDir()
	}
	if loc == "" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Downloads")
	}
	return expandTilde(loc)
}

// SelectDownloadDir opens a native directory picker and returns the chosen path.
// Works on Linux and Windows (Wails OpenDirectoryDialog).
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
	_ = appconfig.SetBrowserExtensionDownloadDir(dir)
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
		cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", ps)
		hideConsole(cmd)
		cmd.Start()
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

// titleFromMediaURL builds a readable title from a direct file URL path.
func titleFromMediaURL(raw string) string {
	u := strings.TrimSpace(raw)
	path := u
	if i := strings.Index(path, "://"); i >= 0 {
		path = path[i+3:]
		if j := strings.Index(path, "/"); j >= 0 {
			path = path[j:]
		}
	}
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if base == "" || base == "/" || base == "." {
		return "Download"
	}
	return base
}

// classifyError categorizes a download error message into a type and
// user-friendly message to help the user understand what went wrong.
func classifyError(errMsg string) (string, string) {
	msg := strings.ToLower(errMsg)
	switch {
	case isDiskFullMessage(errMsg):
		return "disk_full", diskFullUserMsg
	case strings.Contains(msg, "could not copy") && strings.Contains(msg, "cookie"),
		strings.Contains(msg, "could not read browser cookies"),
		strings.Contains(msg, "cookie database"):
		return "cookies_locked", "Could not read browser cookies. Close Chrome/Edge completely and try again (public videos work without cookies)."
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
	case strings.Contains(msg, "connection reset") || strings.Contains(msg, "connection aborted") ||
		strings.Contains(msg, "ssl") || strings.Contains(msg, "tls"):
		return "network", "Connection blocked by the site/CDN. Try another quality, or open the video in your browser first (cookies), then retry."
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
	emitError := ""
	if ok {
		ad.Status = status
		ad.Percent = percent
		ad.Speed = speed
		ad.ETA = eta
		switch status {
		case "error":
			ad.Error = errMsg
		case "complete", "cancelled":
			ad.Error = ""
		case "downloading", "processing", "metadata", "queued":
			// Never surface mid-download yt-dlp retry noise on the UI
			if errMsg != "" {
				ad.lastYtdlpErr = errMsg
			}
			ad.Error = ""
		default:
			if errMsg != "" {
				ad.Error = errMsg
			}
		}
		emitError = ad.Error
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
		isImportant := status == "complete" || status == "error" || status == "cancelled" || status == "paused" || status == "metadata" || status == "queued" || status == "processing"
		if isImportant || time.Since(ad.lastEmit) >= time.Second {
			shouldEmit = true
			ad.lastEmit = time.Now()
		}
	}
	d.mu.Unlock()

	// Persist terminal / paused states to disk
	if status == "complete" || status == "error" || status == "cancelled" || status == "paused" {
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
			"error":   emitError,
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
// Handles HLS fragments, progressive files, and video+audio dual downloads.
//
// Important: yt-dlp prints "100% of … in 00:00:01" after EVERY HLS fragment.
// Treating those as job progress made the UI jump to 100% while still downloading.
func (d *DownloadService) parseProgress(ctx context.Context, id string, reader io.Reader) {
	// Live: "[download]  45.2% of ~120.00MiB at  2.5MiB/s ETA 00:30"
	ytdlpLiveRegex := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+~?\s*([\d.]+(?:[KMG]i?B)?)\s+at\s+(\S+)(?:\s+ETA\s+(\S+))?`)
	// Done summary (per fragment or finished stream): "... 100% of 1.50MiB in 00:00:01"
	ytdlpDoneRegex := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+~?\s*([\d.]+)([KMG]i?B)\s+in\s+`)
	fragRegex := regexp.MustCompile(`(?:frag|fragment)\s+(\d+)\s*/\s*(\d+)`)
	ytdlpBareRegex := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%`)
	aria2cFullRegex := regexp.MustCompile(`\[#[0-9a-fA-F]+\s+.*?\((\d+)%\).*?DL:([0-9.]+\S+)(?:.*?ETA:(\S+))?`)
	aria2cProgressRegex := regexp.MustCompile(`\((\d+)%\)`)
	speedRegex := regexp.MustCompile(`(?:DL:|at\s+)(\d+\.?\d*\S+/s)`)
	etaRegex := regexp.MustCompile(`ETA[:\s]+(\S+)`)
	aria2DLRegex := regexp.MustCompile(`DL:([0-9.]+\S+)`)

	streamIndex := 0 // 0 = first stream (video), 1 = audio after a real large finish
	highWaterMark := 0.0
	sawLiveProgress := false

	clamp := func(p float64) float64 {
		if p < 0 {
			return 0
		}
		// Keep headroom until process exit marks complete — avoid sticky fake 100%
		if p > 99.4 {
			return 99.4
		}
		return p
	}

	raise := func(overall float64, speed, eta string) {
		overall = clamp(overall)
		if overall > highWaterMark {
			highWaterMark = overall
		}
		d.updateDownload(id, "downloading", highWaterMark, speed, eta, "")
	}

	sizeToBytes := func(numStr, unit string) float64 {
		n, _ := strconv.ParseFloat(numStr, 64)
		u := strings.ToUpper(unit)
		switch {
		case strings.HasPrefix(u, "GI"):
			return n * 1024 * 1024 * 1024
		case strings.HasPrefix(u, "G"):
			return n * 1000 * 1000 * 1000
		case strings.HasPrefix(u, "MI"):
			return n * 1024 * 1024
		case strings.HasPrefix(u, "M"):
			return n * 1000 * 1000
		case strings.HasPrefix(u, "KI"):
			return n * 1024
		case strings.HasPrefix(u, "K"):
			return n * 1000
		default:
			return n
		}
	}

	// Single stream: use raw %. After a second stream starts, reserve headroom
	// so video@100% isn't shown as fully done while audio still runs.
	mapStream := func(raw float64) float64 {
		if streamIndex <= 0 {
			return raw
		}
		return 85 + raw*0.14
	}

	processLine := func(line string) {
		if line == "" {
			return
		}

		if strings.Contains(line, "[Merger]") || strings.Contains(line, "[ExtractAudio]") ||
			strings.Contains(line, "[FixupM3u8]") || strings.Contains(line, "[Fixup]") ||
			strings.Contains(line, "[ffmpeg]") || strings.Contains(line, "[MoveFiles]") {
			if highWaterMark < 99 {
				highWaterMark = 99
			}
			d.updateDownload(id, "processing", highWaterMark, "", "", "")
			return
		}

		// yt-dlp often prints ERROR: on fragment/format retries while still succeeding.
		// Keep for final failure text only — do not flip the UI to an error state.
		if strings.HasPrefix(strings.TrimSpace(line), "ERROR:") {
			msg := strings.TrimSpace(line)
			// Ignore empty "ERROR:" noise
			if len(msg) > len("ERROR:") {
				d.mu.Lock()
				if ad, ok := d.downloads[id]; ok {
					ad.lastYtdlpErr = msg
					// Ensure UI does not keep showing a stale error
					ad.Error = ""
				}
				d.mu.Unlock()
			}
			return
		}

		// Next Destination after first stream mostly done ≈ audio track
		if strings.Contains(line, "[download] Destination:") && sawLiveProgress && highWaterMark >= 80 {
			if streamIndex < 1 {
				streamIndex = 1
				// Cap first-stream mark so UI doesn't sit at ~100% during audio
				if highWaterMark > 88 {
					highWaterMark = 88
				}
			}
		}

		var speed, eta string
		if sm := speedRegex.FindStringSubmatch(line); len(sm) >= 2 {
			speed = sm[1]
		}
		if em := etaRegex.FindStringSubmatch(line); len(em) >= 2 {
			eta = em[1]
		}

		// 1) frag N/M — best signal for HLS/DASH
		if fm := fragRegex.FindStringSubmatch(line); len(fm) >= 3 {
			cur, _ := strconv.ParseFloat(fm[1], 64)
			total, _ := strconv.ParseFloat(fm[2], 64)
			if total > 0 && cur >= 0 {
				raw := (cur / total) * 100
				if lm := ytdlpLiveRegex.FindStringSubmatch(line); len(lm) >= 2 {
					if p, err := strconv.ParseFloat(lm[1], 64); err == nil && p >= 0 && p <= 100 {
						// Smooth within the current fragment slot
						raw = ((cur - 1) / total * 100) + (p/100.0)*(100.0/total)
						if raw < 0 {
							raw = (cur / total) * 100
						}
					}
					if len(lm) >= 4 && lm[3] != "" {
						speed = lm[3]
						if !strings.HasSuffix(strings.ToLower(speed), "/s") {
							speed += "/s"
						}
					}
					if len(lm) >= 5 && lm[4] != "" {
						eta = lm[4]
					}
				}
				sawLiveProgress = true
				raise(mapStream(raw), speed, eta)
				return
			}
		}

		// 2) Live yt-dlp line (has "at …", not finished "in …")
		if lm := ytdlpLiveRegex.FindStringSubmatch(line); len(lm) >= 2 {
			raw, _ := strconv.ParseFloat(lm[1], 64)
			if len(lm) >= 4 && lm[3] != "" {
				speed = lm[3]
				if !strings.HasSuffix(strings.ToLower(speed), "/s") {
					speed += "/s"
				}
			}
			if len(lm) >= 5 && lm[4] != "" {
				eta = lm[4]
			}
			sawLiveProgress = true
			raise(mapStream(raw), speed, eta)
			return
		}

		// 3) Completed-item lines: ignore tiny fragment finishes
		if dm := ytdlpDoneRegex.FindStringSubmatch(line); len(dm) >= 4 {
			raw, _ := strconv.ParseFloat(dm[1], 64)
			bytes := sizeToBytes(dm[2], dm[3])
			if bytes < 3*1024*1024 {
				if !sawLiveProgress && speed != "" {
					d.updateDownload(id, "downloading", highWaterMark, speed, eta, "")
				}
				return
			}
			if raw >= 99.0 {
				raise(mapStream(100), speed, eta)
				if streamIndex < 1 {
					streamIndex = 1
				}
			}
			return
		}

		// 4) aria2c full status line
		if am := aria2cFullRegex.FindStringSubmatch(line); len(am) >= 2 {
			raw, _ := strconv.ParseFloat(am[1], 64)
			if len(am) >= 3 && am[2] != "" {
				speed = am[2]
				if !strings.HasSuffix(speed, "/s") {
					speed += "/s"
				}
			}
			if len(am) >= 4 && am[3] != "" {
				eta = am[3]
			}
			sawLiveProgress = true
			raise(mapStream(raw), speed, eta)
			return
		}

		// 5) aria2 bare % only on aria-looking lines
		if strings.Contains(line, "DL:") || strings.Contains(line, "CN:") {
			if matches := aria2cProgressRegex.FindStringSubmatch(line); len(matches) >= 2 {
				raw, _ := strconv.ParseFloat(matches[1], 64)
				if speed == "" {
					if dm := aria2DLRegex.FindStringSubmatch(line); len(dm) >= 2 {
						speed = dm[1] + "/s"
					}
				}
				sawLiveProgress = true
				raise(mapStream(raw), speed, eta)
				return
			}
		}

		// 6) Bare yt-dlp % only after real live progress (skip fake 100% dumps)
		if sawLiveProgress {
			if matches := ytdlpBareRegex.FindStringSubmatch(line); len(matches) >= 2 {
				if strings.Contains(line, " in ") && strings.Contains(line, " of ") {
					return
				}
				raw, _ := strconv.ParseFloat(matches[1], 64)
				if raw >= 99.9 && !strings.Contains(line, "at ") && !strings.Contains(strings.ToUpper(line), "ETA") {
					return
				}
				raise(mapStream(raw), speed, eta)
			}
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
