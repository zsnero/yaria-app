package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"
)

// PlayerService detects and launches external media players (mpv, vlc).
// Used as a fallback when WebKitGTK can't decode a codec.
type PlayerService struct {
	ctx context.Context
	mu  sync.Mutex
}

func (p *PlayerService) startup(ctx context.Context) {
	p.ctx = ctx
}

type playerInfo struct {
	Name string `json:"name"`
}

var playerDefs = []struct {
	name    string
	binary  string
	flags   []string
}{
	{"mpv", "mpv", []string{"--cache=yes", "--cache-secs=120", "--demuxer-max-bytes=500MiB", "--force-seekable=yes"}},
	{"vlc", "vlc", []string{"--network-caching=3000", "--file-caching=3000"}},
}

// DetectPlayers returns installed media players.
func (p *PlayerService) DetectPlayers() []playerInfo {
	var result []playerInfo
	for _, pd := range playerDefs {
		if _, err := exec.LookPath(pd.binary); err == nil {
			result = append(result, playerInfo{Name: pd.name})
		}
	}
	return result
}

// LaunchPlayer opens a stream URL in the specified player.
func (p *PlayerService) LaunchPlayer(streamURL string, playerName string, title string) map[string]interface{} {
	if streamURL == "" {
		return map[string]interface{}{"error": "no stream URL"}
	}

	// Find player
	var chosen *struct {
		name   string
		binary string
		flags  []string
	}
	for i := range playerDefs {
		if playerName == "" || playerDefs[i].name == playerName {
			if _, err := exec.LookPath(playerDefs[i].binary); err == nil {
				chosen = &playerDefs[i]
				break
			}
		}
	}
	if chosen == nil {
		return map[string]interface{}{
			"error":   "no media player found",
			"message": installHint(),
		}
	}

	args := append([]string{}, chosen.flags...)
	if chosen.name == "mpv" && title != "" {
		args = append(args, fmt.Sprintf("--force-media-title=%s", title))
	}
	args = append(args, streamURL)

	cmd := exec.Command(chosen.binary, args...)
	setProcAttrDetached(cmd)

	if err := cmd.Start(); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("failed to launch %s: %v", chosen.name, err)}
	}
	go cmd.Wait()

	return map[string]interface{}{"status": "launched", "player": chosen.name}
}

func installHint() string {
	switch runtime.GOOS {
	case "linux":
		return "Install mpv: sudo pacman -S mpv (Arch) / sudo apt install mpv (Debian)"
	case "darwin":
		return "Install mpv: brew install mpv"
	default:
		return "Install mpv from https://mpv.io"
	}
}
