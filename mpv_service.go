package main

import (
	"context"
	"sync"

	"yaria/pkg/appconfig"
)

// MpvService exposes the optional native libmpv player to the frontend.
// Real implementation is platform-specific (see mpv_player_*.go).
type MpvService struct {
	ctx context.Context
	mu  sync.Mutex
}

func NewMpvService() *MpvService {
	return &MpvService{}
}

// MpvTrack describes one track (video/audio/sub) reported by the native player.
type MpvTrack struct {
	ID       int    `json:"id"`
	Type     string `json:"type"`
	Lang     string `json:"lang,omitempty"`
	Title    string `json:"title,omitempty"`
	Selected bool   `json:"selected"`
	Default  bool   `json:"default,omitempty"`
	External bool   `json:"external,omitempty"`
	Codec    string `json:"codec,omitempty"`
}

func (m *MpvService) startup(ctx context.Context) {
	m.ctx = ctx
}

func (m *MpvService) shutdown() {
	m.Stop()
}

// Available reports whether the native player can run on this system.
func (m *MpvService) Available() map[string]interface{} {
	ok, reason := mpvPlatformAvailable()
	return map[string]interface{}{
		"available": ok,
		"reason":    reason,
		"backend":   appconfig.GetUISettings().PlayerBackend,
	}
}

// GetBackend returns the configured player backend ("webview" | "libmpv").
func (m *MpvService) GetBackend() string {
	b := appconfig.GetUISettings().PlayerBackend
	if b != "libmpv" {
		return "webview"
	}
	return b
}

// Start creates the embed surface and libmpv instance (no file yet).
func (m *MpvService) Start() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformStart(m.ctx); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

// LoadFile loads a local path or HTTP(S) URL into the player.
func (m *MpvService) LoadFile(pathOrURL string) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformLoad(pathOrURL); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

// SetBounds positions the native child window (CSS px relative to client area).
func (m *MpvService) SetBounds(x, y, w, h float64) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformSetBounds(x, y, w, h); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

// SetVisible shows or hides the native surface.
func (m *MpvService) SetVisible(visible bool) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	mpvPlatformSetVisible(visible)
	return map[string]interface{}{"status": "ok"}
}

func (m *MpvService) Play() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformSetPause(false); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

func (m *MpvService) Pause() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformSetPause(true); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

func (m *MpvService) TogglePause() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformTogglePause(); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

func (m *MpvService) Seek(seconds float64) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformSeek(seconds); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

func (m *MpvService) SetVolume(vol float64) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformSetVolume(vol); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

// SetSubtitle adds an external subtitle file to the currently playing media.
func (m *MpvService) SetSubtitle(path string) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformSetSubtitle(path); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

func (m *MpvService) GetTime() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return mpvPlatformGetTime()
}

func (m *MpvService) GetDuration() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return mpvPlatformGetDuration()
}

func (m *MpvService) IsPaused() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return mpvPlatformIsPaused()
}

// Stop destroys the player and child window.
func (m *MpvService) Stop() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	mpvPlatformStop()
	return map[string]interface{}{"status": "ok"}
}

// GetTracks returns the current file's video/audio/subtitle tracks.
func (m *MpvService) GetTracks() []MpvTrack {
	m.mu.Lock()
	defer m.mu.Unlock()
	return mpvPlatformGetTracks()
}

// SetAudioTrack selects an audio track by id (0 = mpv auto).
func (m *MpvService) SetAudioTrack(id int) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformSetAudioTrack(id); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

// SetSubtitleTrack selects a subtitle track by id (0 = auto, -1 = off).
func (m *MpvService) SetSubtitleTrack(id int) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformSetSubtitleTrack(id); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

// SetSubtitleEnabled toggles subtitle visibility without changing the track.
func (m *MpvService) SetSubtitleEnabled(on bool) map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := mpvPlatformSetSubtitleEnabled(on); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok"}
}

// mpvAspectOrder is the cycle order for the frontend aspect-ratio button.
// "original" is first because it's the default aspect-ratio mode.
var mpvAspectOrder = []string{"original", "fill", "stretch", "16:9", "4:3", "2.35"}

// mpvAspectLabel returns the short display label for an aspect-ratio mode.
func mpvAspectLabel(mode string) string {
	switch mode {
	case "stretch":
		return "Stretch"
	case "fill":
		return "Fill"
	case "original":
		return "Original"
	default:
		return mode // "16:9", "4:3", "2.35"
	}
}

// CycleAspect advances to the next aspect-ratio mode and applies it to the
// running player.
func (m *MpvService) CycleAspect() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	cur := mpvPlatformGetAspectMode()
	next := mpvAspectOrder[0]
	for i, mo := range mpvAspectOrder {
		if mo == cur {
			next = mpvAspectOrder[(i+1)%len(mpvAspectOrder)]
			break
		}
	}
	if err := mpvPlatformSetAspectMode(next); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "ok", "mode": next, "label": mpvAspectLabel(next)}
}

// GetAspectMode returns the current aspect-ratio mode and its label.
func (m *MpvService) GetAspectMode() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	mode := mpvPlatformGetAspectMode()
	return map[string]interface{}{"mode": mode, "label": mpvAspectLabel(mode)}
}
