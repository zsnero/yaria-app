package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"yaria/pkg/appconfig"
)

// ExtensionBridge exposes a localhost HTTP API for the Yaria browser extension.
type ExtensionBridge struct {
	ctx       context.Context
	mu        sync.Mutex
	server    *http.Server
	running   bool
	port      int
	downloads *DownloadService
}

func NewExtensionBridge() *ExtensionBridge {
	return &ExtensionBridge{}
}

// LinkDownloadService connects the bridge to the downloader.
func (b *ExtensionBridge) LinkDownloadService(d *DownloadService) {
	b.downloads = d
}

func (b *ExtensionBridge) startup(ctx context.Context) {
	b.ctx = ctx
	b.ensureToken()
	if appconfig.BrowserExtensionEnabled() {
		_ = b.Start()
	}
}

func (b *ExtensionBridge) shutdown() {
	b.Stop()
}

func (b *ExtensionBridge) ensureToken() {
	if appconfig.BrowserExtensionToken() != "" {
		return
	}
	tok, err := randomToken(24)
	if err != nil {
		return
	}
	_ = appconfig.SetBrowserExtensionToken(tok)
}

func randomToken(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// GetStatus returns bridge status for the settings UI.
func (b *ExtensionBridge) GetStatus() map[string]interface{} {
	b.mu.Lock()
	running := b.running
	port := b.port
	if port == 0 {
		port = appconfig.BrowserExtensionPort()
	}
	b.mu.Unlock()

	return map[string]interface{}{
		"enabled": appconfig.BrowserExtensionEnabled(),
		"running": running,
		"port":    port,
		"token":   appconfig.BrowserExtensionToken(),
		"host":    "127.0.0.1",
	}
}

// SetEnabled turns the bridge on/off and persists the preference.
func (b *ExtensionBridge) SetEnabled(enabled bool) map[string]interface{} {
	if err := appconfig.SetBrowserExtensionEnabled(enabled); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if enabled {
		b.ensureToken()
		if res := b.Start(); res["error"] != nil {
			return res
		}
	} else {
		b.Stop()
	}
	return b.GetStatus()
}

// SetPort updates the listen port (restarts if running).
func (b *ExtensionBridge) SetPort(port int) map[string]interface{} {
	if port <= 0 || port > 65535 {
		return map[string]interface{}{"error": "port must be 1–65535"}
	}
	wasRunning := false
	b.mu.Lock()
	wasRunning = b.running
	b.mu.Unlock()

	if wasRunning {
		b.Stop()
	}
	if err := appconfig.SetBrowserExtensionPort(port); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if wasRunning || appconfig.BrowserExtensionEnabled() {
		if res := b.Start(); res["error"] != nil {
			return res
		}
	}
	return b.GetStatus()
}

// RegenerateToken creates a new pairing token.
func (b *ExtensionBridge) RegenerateToken() map[string]interface{} {
	tok, err := randomToken(24)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if err := appconfig.SetBrowserExtensionToken(tok); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return b.GetStatus()
}

// Start starts the localhost bridge server.
func (b *ExtensionBridge) Start() map[string]interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return map[string]interface{}{"status": "already_running", "port": b.port}
	}
	if b.downloads == nil {
		return map[string]interface{}{"error": "download service not available"}
	}

	port := appconfig.BrowserExtensionPort()
	mux := http.NewServeMux()
	mux.HandleFunc("/extension/ping", b.handlePing)
	mux.HandleFunc("/extension/download", b.handleDownload)
	mux.HandleFunc("/extension/focus", b.handleFocus)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("listen %s: %v", addr, err)}
	}

	srv := &http.Server{
		Handler:           b.withMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
	}
	b.server = srv
	b.port = port
	b.running = true

	go func() {
		_ = srv.Serve(ln)
	}()

	return map[string]interface{}{"status": "started", "port": port}
}

// Stop stops the bridge server.
func (b *ExtensionBridge) Stop() map[string]interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.running || b.server == nil {
		b.running = false
		return map[string]interface{}{"status": "stopped"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = b.server.Shutdown(ctx)
	b.server = nil
	b.running = false
	return map[string]interface{}{"status": "stopped"}
}

func (b *ExtensionBridge) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extension pages are moz-extension:// / chrome-extension:// origins.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !appconfig.BrowserExtensionEnabled() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
				"ok":    false,
				"error": "browser integration is disabled in Yaria Settings",
			})
			return
		}
		if !b.authorize(r) {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"ok":    false,
				"error": "invalid or missing token",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (b *ExtensionBridge) authorize(r *http.Request) bool {
	token := appconfig.BrowserExtensionToken()
	if token == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		got := strings.TrimSpace(auth[7:])
		return got == token
	}
	// Also accept X-Yaria-Token for simpler clients
	if r.Header.Get("X-Yaria-Token") == token {
		return true
	}
	return false
}

func (b *ExtensionBridge) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":          true,
		"version":     AppVersion,
		"integration": true,
	})
}

type extensionDownloadRequest struct {
	URL         string            `json:"url"`
	PageURL     string            `json:"page_url"`
	Title       string            `json:"title"`
	Quality     string            `json:"quality"`
	Kind        string            `json:"kind"`
	IsAudioOnly bool              `json:"is_audio_only"`
	Referrer    string            `json:"referrer"`
	Headers     map[string]string `json:"headers"`
	Source      string            `json:"source"`
	Label       string            `json:"label"`
}

func (b *ExtensionBridge) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "bad body"})
		return
	}
	var req extensionDownloadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "invalid json"})
		return
	}

	url := strings.TrimSpace(req.URL)
	page := strings.TrimSpace(req.PageURL)
	if url == "" {
		url = page
	}
	if url == "" || (!strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")) {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "url required (http/https)"})
		return
	}

	// page kind → watch URL for yt-dlp extractors
	// video/hls/dash with a real media URL → keep direct link (works when site extractors break)
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "page" && page != "" && (strings.HasPrefix(page, "http://") || strings.HasPrefix(page, "https://")) {
		url = page
	}

	// For direct files, don't pass resolution format selectors (single progressive file)
	resolution := normalizeExtensionQuality(req.Quality)
	if kind == "video" || kind == "hls" || kind == "dash" || kind == "audio" {
		// Direct URL path — resolution already baked into the chosen link
		if kind != "page" {
			resolution = "best"
		}
	}

	audioOnly := req.IsAudioOnly || kind == "audio"
	audioFormat := ""
	if audioOnly {
		audioFormat = "mp3"
	}
	dir := ""
	if b.downloads != nil {
		dir = b.downloads.GetDownloadDir()
	}

	referer := strings.TrimSpace(req.Referrer)
	if referer == "" {
		referer = page
	}
	// Direct file downloads need the watch-page referer, not the CDN URL
	if referer == "" || referer == url {
		referer = page
	}

	result := b.downloads.startDownload(url, strings.TrimSpace(req.Title), referer, resolution, dir, audioOnly, audioFormat, "mp4")
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": errMsg})
		return
	}

	// Bring app forward and open Downloads
	b.focusDownloads()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"id":     result["id"],
		"status": result["status"],
	})
}

func (b *ExtensionBridge) handleFocus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}
	target := "downloads"
	var body struct {
		Target string `json:"target"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body)
	if strings.TrimSpace(body.Target) != "" {
		target = strings.TrimSpace(body.Target)
	}
	b.focusTarget(target)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "target": target})
}

func (b *ExtensionBridge) focusDownloads() {
	b.focusTarget("downloads")
}

func (b *ExtensionBridge) focusTarget(target string) {
	if b.ctx == nil {
		return
	}
	wailsRuntime.WindowShow(b.ctx)
	wailsRuntime.WindowUnminimise(b.ctx)
	wailsRuntime.EventsEmit(b.ctx, "extension-focus", map[string]interface{}{
		"target": target,
	})
}

func normalizeExtensionQuality(q string) string {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" || q == "best" {
		return "best"
	}
	switch q {
	case "4k", "uhd":
		return "2160p"
	case "2k":
		return "1440p"
	case "fhd":
		return "1080p"
	case "hd":
		return "720p"
	case "sd":
		return "480p"
	}
	if strings.HasSuffix(q, "p") {
		return q
	}
	var n int
	if _, err := fmt.Sscanf(q, "%d", &n); err == nil && n > 0 {
		return fmt.Sprintf("%dp", n)
	}
	return q
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
