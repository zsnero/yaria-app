package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
// If resumeSeconds > 0, the player starts at that position.
func (p *PlayerService) LaunchPlayer(streamURL string, playerName string, title string, resumeSeconds int) map[string]interface{} {
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
	if resumeSeconds > 0 {
		if chosen.name == "mpv" {
			args = append(args, fmt.Sprintf("--start=%d", resumeSeconds))
		} else if chosen.name == "vlc" {
			args = append(args, fmt.Sprintf("--start-time=%d", resumeSeconds))
		}
	}
	// Auto-discover subtitle files
	if strings.HasPrefix(streamURL, "/") && chosen.name == "mpv" {
		for _, sub := range findSubtitleFiles(streamURL) {
			args = append(args, "--sub-file="+sub)
		}
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

func findSubtitleFiles(videoPath string) []string {
	if videoPath == "" {
		return nil
	}
	dir := filepath.Dir(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))

	subExts := []string{".srt", ".ass", ".ssa", ".sub", ".vtt", ".idx"}
	var subs []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		for _, subExt := range subExts {
			if ext == subExt {
				// Check if subtitle matches the video name
				subBase := strings.TrimSuffix(name, filepath.Ext(name))
				// Match if subtitle starts with the video name (handles .en.srt, .forced.srt etc)
				if strings.HasPrefix(strings.ToLower(subBase), strings.ToLower(base)) {
					subs = append(subs, filepath.Join(dir, name))
				}
			}
		}
	}
	return subs
}

// PlayFile launches a local video file in the best available player.
func (p *PlayerService) PlayFile(filePath string) map[string]interface{} {
	if filePath == "" {
		return map[string]interface{}{"error": "no file path"}
	}

	// If it's a directory, find the largest video file in it
	info, err := os.Stat(filePath)
	if err != nil {
		return map[string]interface{}{"error": "file not found: " + err.Error()}
	}

	actualFile := filePath
	if info.IsDir() {
		// Find largest video file in directory
		var bestFile string
		var bestSize int64
		videoExts := map[string]bool{".mp4": true, ".mkv": true, ".avi": true, ".webm": true, ".mov": true, ".m4v": true}
		filepath.Walk(filePath, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if videoExts[ext] && fi.Size() > bestSize {
				bestSize = fi.Size()
				bestFile = path
			}
			return nil
		})
		if bestFile == "" {
			return map[string]interface{}{"error": "no video files found in directory"}
		}
		actualFile = bestFile
	}

	return p.LaunchPlayer(actualFile, "", filepath.Base(actualFile), 0)
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
