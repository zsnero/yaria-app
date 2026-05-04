package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// PlayerService provides file/folder open utilities.
// External player launches (mpv/vlc) have been removed -- all playback
// happens in-app via the <video> element with FFmpeg transcode support.
type PlayerService struct {
	ctx context.Context
	mu  sync.Mutex
}

func (p *PlayerService) startup(ctx context.Context) {
	p.ctx = ctx
}

// OpenFolder opens the containing folder of a file/directory in the system file manager.
func (p *PlayerService) OpenFolder(filePath string) map[string]interface{} {
	if filePath == "" {
		return map[string]interface{}{"error": "no path"}
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return map[string]interface{}{"error": "path not found: " + err.Error()}
	}

	dir := filePath
	if !info.IsDir() {
		dir = filepath.Dir(filePath)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", dir)
	default:
		return map[string]interface{}{"error": "unsupported platform"}
	}

	if err := cmd.Start(); err != nil {
		return map[string]interface{}{"error": "failed to open folder: " + err.Error()}
	}
	go cmd.Wait()

	return map[string]interface{}{"status": "opened", "path": dir}
}

// PlayFile finds the largest video in a directory and returns its path
// for the frontend to navigate to the in-app player.
func (p *PlayerService) PlayFile(filePath string) map[string]interface{} {
	if filePath == "" {
		return map[string]interface{}{"error": "no file path"}
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return map[string]interface{}{"error": "file not found: " + err.Error()}
	}

	actualFile := filePath
	if info.IsDir() {
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

	return map[string]interface{}{"status": "ok", "file": actualFile, "title": filepath.Base(actualFile)}
}
