package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"yaria/pkg/appconfig"
)

// isMetadataFile checks if a filename is a database/metadata file.
func isMetadataFile(name string) bool {
	metaExts := []string{".db", ".db-wal", ".db-shm", ".db.lock"}
	for _, ext := range metaExts {
		if strings.HasSuffix(name, ext) {
			return true
		}
	}
	return false
}

// SettingsService provides configuration management methods to the frontend.
type SettingsService struct{}

// SaveTMDBKey saves the TMDB API key to config.
// Pro services (SearchService, TMDBService) read the key from appconfig on
// each call, so no explicit refresh is needed here.
func (s *SettingsService) SaveTMDBKey(key string) map[string]interface{} {
	key = strings.TrimSpace(key)
	if err := appconfig.SetTMDBApiKey(key); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("failed to save: %v", err)}
	}
	return map[string]interface{}{"status": "saved"}
}

// GetCacheStats returns information about cached/downloaded data in the data directory.
func (s *SettingsService) GetCacheStats() map[string]interface{} {
	dataDir := getDataDir()

	var partialFiles, metaFiles, torrentDirs int
	var partialSize, metaSize, dirSize int64

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return map[string]interface{}{
			"data_dir":      dataDir,
			"partial_files": 0, "partial_size": "0 B",
			"meta_files": 0, "meta_size": "0 B",
			"torrent_dirs": 0, "dir_size": "0 B",
			"total_size": "0 B",
		}
	}

	for _, e := range entries {
		name := e.Name()
		info, err := e.Info()
		if err != nil {
			continue
		}

		if e.IsDir() {
			if name == "subtitles" {
				continue
			}
			torrentDirs++
			dirSize += calcDirSize(filepath.Join(dataDir, name))
		} else if strings.HasSuffix(name, ".part") {
			partialFiles++
			partialSize += info.Size()
		} else if isMetadataFile(name) {
			metaFiles++
			metaSize += info.Size()
		}
	}

	return map[string]interface{}{
		"data_dir":      dataDir,
		"partial_files": partialFiles,
		"partial_size":  humanize.Bytes(uint64(partialSize)),
		"meta_files":    metaFiles,
		"meta_size":     humanize.Bytes(uint64(metaSize)),
		"data_dirs":     torrentDirs,
		"dir_size":      humanize.Bytes(uint64(dirSize)),
		"total_size":    humanize.Bytes(uint64(partialSize + metaSize + dirSize)),
	}
}

// ClearCache removes cached data. cacheType can be: "partial", "meta", "dirs", or "all".
func (s *SettingsService) ClearCache(cacheType string) map[string]interface{} {
	dataDir := getDataDir()

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return map[string]interface{}{"error": "failed to read data directory"}
	}

	removed := 0
	var freedBytes int64

	for _, e := range entries {
		name := e.Name()
		fullPath := filepath.Join(dataDir, name)

		if e.IsDir() && name == "subtitles" {
			continue
		}

		shouldRemove := false
		switch cacheType {
		case "partial":
			shouldRemove = strings.HasSuffix(name, ".part")
		case "meta":
			shouldRemove = isMetadataFile(name)
		case "dirs":
			shouldRemove = e.IsDir()
		case "all":
			shouldRemove = true
		}

		if shouldRemove {
			info, _ := e.Info()
			if e.IsDir() {
				freedBytes += calcDirSize(fullPath)
				os.RemoveAll(fullPath)
			} else {
				if info != nil {
					freedBytes += info.Size()
				}
				os.Remove(fullPath)
			}
			removed++
		}
	}

	return map[string]interface{}{
		"removed": removed,
		"freed":   humanize.Bytes(uint64(freedBytes)),
	}
}

// SaveProxy saves proxy configuration.
func (s *SettingsService) SaveProxy(proxyType, proxyAddr string) map[string]interface{} {
	proxyType = strings.TrimSpace(proxyType)
	proxyAddr = strings.TrimSpace(proxyAddr)

	if err := appconfig.SetProxyType(proxyType); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if proxyAddr != "" {
		if err := appconfig.SetProxyAddr(proxyAddr); err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
	}
	return map[string]interface{}{"status": "saved"}
}

// GetProxy returns the current proxy configuration.
func (s *SettingsService) GetProxy() map[string]interface{} {
	return map[string]interface{}{
		"type": appconfig.ProxyType(),
		"addr": appconfig.ProxyAddr(),
	}
}

// SaveSpeedLimit saves the download speed limit in bytes/sec. 0 = unlimited.
func (s *SettingsService) SaveSpeedLimit(limitBytes int64) map[string]interface{} {
	if err := appconfig.SetSpeedLimit(limitBytes); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "saved", "limit": limitBytes}
}

// GetSpeedLimit returns the current speed limit in bytes/sec.
func (s *SettingsService) GetSpeedLimit() int64 {
	return appconfig.SpeedLimit()
}

// calcDirSize recursively calculates the total size of a directory.
func calcDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}
