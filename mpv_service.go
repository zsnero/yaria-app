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
