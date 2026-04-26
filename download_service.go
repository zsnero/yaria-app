package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	ID        string
	URL       string
	Title     string
	Thumbnail string
	Status    string // "queued", "metadata", "downloading", "complete", "error", "cancelled"
	Percent   float64
	Speed     string
	ETA       string
	Error     string
	StartedAt string
	cancel    context.CancelFunc
}

// DownloadService provides video download methods to the frontend via Wails bindings.
type DownloadService struct {
	ctx       context.Context
	mu        sync.Mutex
	dl        *downloader.YTDLPDownloader
	cfg       *config.Config
	downloads map[string]*activeDownload
	nextID    int
	depsReady bool
}

// NewDownloadService creates a DownloadService with default config.
func NewDownloadService() *DownloadService {
	return &DownloadService{
		downloads: make(map[string]*activeDownload),
		cfg:       config.New(),
	}
}

// startup is called by Wails OnStartup.
func (d *DownloadService) startup(ctx context.Context) {
	d.ctx = ctx
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
		m := map[string]interface{}{
			"format_id":   f.ID,
			"resolution":  fmt.Sprintf("%dp", f.Height),
			"height":      f.Height,
			"ext":         f.Ext,
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
// Returns immediately with the download ID.
func (d *DownloadService) StartDownload(rawURL, resolution, downloadDir string, audioOnly bool, audioFormat string) map[string]interface{} {
	if d.dl == nil {
		return map[string]interface{}{"error": "downloader not initialized"}
	}
	url := cleanURL(rawURL)

	d.mu.Lock()
	d.nextID++
	id := fmt.Sprintf("dl_%d", d.nextID)

	dlCtx, cancel := context.WithCancel(d.ctx)
	thumb := d.getThumbnailURL(url)
	ad := &activeDownload{
		ID:        id,
		URL:       url,
		Title:     url,
		Thumbnail: thumb,
		Status:    "queued",
		StartedAt: time.Now().Format("2006-01-02 15:04"),
		cancel:    cancel,
	}
	d.downloads[id] = ad
	d.mu.Unlock()

	go func() {
		// Fetch metadata first
		d.updateDownload(id, "metadata", 0, "", "", "")
		_, title, err := d.dl.GetMetadata([]string{url})
		if err != nil {
			d.updateDownload(id, "error", 0, "", "", err.Error())
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
		tempDir := filepath.Join(dest, ".yaria_tmp_"+id)
		_ = os.MkdirAll(tempDir, 0755)

		// Set per-download config on the shared downloader.
		d.mu.Lock()
		origRes := d.cfg.Resolution
		origAudio := d.cfg.IsAudioOnly
		origAudioFmt := d.cfg.AudioFormat
		if resolution != "" {
			d.cfg.Resolution = resolutionToFormat(resolution)
		}
		d.cfg.IsAudioOnly = audioOnly
		if audioFormat != "" {
			d.cfg.AudioFormat = audioFormat
		}
		d.mu.Unlock()

		// Pipe stdout/stderr through a progress parser
		pr, pw := io.Pipe()
		d.cfg.Stdout = pw
		d.cfg.Stderr = pw
		go d.parseProgress(dlCtx, id, pr)

		d.updateDownload(id, "downloading", 0, "", "", "")

		success, dlErr := d.dl.Download([]string{url}, tempDir)
		pw.Close()

		// Restore original config
		d.mu.Lock()
		d.cfg.Resolution = origRes
		d.cfg.IsAudioOnly = origAudio
		d.cfg.AudioFormat = origAudioFmt
		d.cfg.Stdout = nil
		d.cfg.Stderr = nil
		d.mu.Unlock()

		// Check for cancellation
		select {
		case <-dlCtx.Done():
			_ = os.RemoveAll(tempDir)
			return
		default:
		}

		if dlErr != nil || !success {
			errMsg := "download failed"
			if dlErr != nil {
				errMsg = dlErr.Error()
			}
			d.updateDownload(id, "error", 0, "", "", errMsg)
			_ = os.RemoveAll(tempDir)
			return
		}

		// Move completed files from temp to destination
		entries, _ := os.ReadDir(tempDir)
		for _, e := range entries {
			src := filepath.Join(tempDir, e.Name())
			dst := filepath.Join(dest, e.Name())
			_ = os.Rename(src, dst)
		}
		_ = os.RemoveAll(tempDir)

		d.updateDownload(id, "complete", 100, "", "", "")
	}()

	return map[string]interface{}{
		"id":     id,
		"status": "queued",
	}
}

// CancelDownload cancels an active download.
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
	d.updateDownload(id, "cancelled", ad.Percent, "", "", "")
	return map[string]interface{}{"status": "cancelled"}
}

// GetDownloads lists all downloads (active and completed).
func (d *DownloadService) GetDownloads() []map[string]interface{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	result := make([]map[string]interface{}, 0, len(d.downloads))
	for _, ad := range d.downloads {
		result = append(result, map[string]interface{}{
			"id":         ad.ID,
			"url":        ad.URL,
			"title":      ad.Title,
			"thumbnail":  ad.Thumbnail,
			"status":     ad.Status,
			"percent":    ad.Percent,
			"speed":      ad.Speed,
			"eta":        ad.ETA,
			"error":      ad.Error,
			"started_at": ad.StartedAt,
		})
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
	if !ok {
		return map[string]interface{}{"error": "download not found"}
	}
	return map[string]interface{}{"status": "removed"}
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

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// resolutionToFormat converts a display resolution like "1080p" to a yt-dlp
// format selector. The Download() method appends "+bestaudio/best" to this,
// so we return only the video part. If the input looks like a format ID
// (numeric, e.g. "303"), it's returned as-is.
func resolutionToFormat(res string) string {
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
	if ok {
		title = ad.Title
	}
	d.mu.Unlock()

	wailsRuntime.EventsEmit(d.ctx, "download-progress", map[string]interface{}{
		"id":      id,
		"status":  status,
		"percent": percent,
		"speed":   speed,
		"eta":     eta,
		"error":   errMsg,
		"title":   title,
	})
}

// splitCRLF is a bufio.SplitFunc that handles \n, \r\n, and bare \r
// (yt-dlp uses \r for in-place progress updates).
func splitCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, bytes.TrimRight(data[:i], "\r"), nil
	}
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// parseProgress reads yt-dlp / aria2c output and extracts download progress.
// Regex patterns are adapted from YariaPlus daemon/manager.go.
func (d *DownloadService) parseProgress(ctx context.Context, id string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	scanner.Split(splitCRLF)

	ytdlpProgressRegex := regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%`)
	aria2cProgressRegex := regexp.MustCompile(`\((\d+)%\)`)
	speedRegex := regexp.MustCompile(`(?:DL:|at\s+)(\d+\.?\d*\w+/?s)`)
	etaRegex := regexp.MustCompile(`ETA[:\s]+(\S+)`)
	bytesProgressRegex := regexp.MustCompile(`([0-9.]+)\s*([kKmMgGtT]?i?B)/([0-9.]+)\s*([kKmMgGtT]?i?B)`)

	unitToMultiplier := func(unit string) float64 {
		switch strings.ToUpper(unit) {
		case "B":
			return 1
		case "KB":
			return 1e3
		case "KIB":
			return 1024
		case "MB":
			return 1e6
		case "MIB":
			return 1024 * 1024
		case "GB":
			return 1e9
		case "GIB":
			return 1024 * 1024 * 1024
		case "TB":
			return 1e12
		case "TIB":
			return 1024 * 1024 * 1024 * 1024
		}
		return 1
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var percent float64
		var speed, eta string
		matched := false

		if matches := ytdlpProgressRegex.FindStringSubmatch(line); len(matches) >= 2 {
			percent, _ = strconv.ParseFloat(matches[1], 64)
			matched = true
		} else if matches := aria2cProgressRegex.FindStringSubmatch(line); len(matches) >= 2 {
			percent, _ = strconv.ParseFloat(matches[1], 64)
			matched = true
		} else if bm := bytesProgressRegex.FindStringSubmatch(line); len(bm) == 5 {
			cur, _ := strconv.ParseFloat(bm[1], 64)
			tot, _ := strconv.ParseFloat(bm[3], 64)
			mu := unitToMultiplier(bm[2])
			mt := unitToMultiplier(bm[4])
			if mt > 0 && tot > 0 {
				percent = (cur * mu) / (tot * mt) * 100.0
				matched = true
			}
		}

		if matched {
			if sm := speedRegex.FindStringSubmatch(line); len(sm) >= 2 {
				speed = sm[1]
			}
			if em := etaRegex.FindStringSubmatch(line); len(em) >= 2 {
				eta = em[1]
			}
			d.updateDownload(id, "downloading", percent, speed, eta, "")
		}
	}
}
