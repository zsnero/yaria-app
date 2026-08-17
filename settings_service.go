package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dustin/go-humanize"
	"yaria/pkg/appconfig"
	"yaria/pkg/mantorex"
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

// refreshTMDBClients is set by the pro build to refresh TMDB/Search clients when the key changes.
// In free builds, this is a no-op.
var refreshTMDBClients = func(key string) {}

// GetTMDBKey returns the saved TMDB API key (masked for display).
// When using the built-in fallback key, the secret is never returned to the UI.
func (s *SettingsService) GetTMDBKey() map[string]interface{} {
	userKey := appconfig.UserTMDBApiKey()
	if userKey != "" {
		masked := userKey
		if len(userKey) > 8 {
			masked = userKey[:4] + strings.Repeat("*", len(userKey)-8) + userKey[len(userKey)-4:]
		}
		return map[string]interface{}{
			"key":           masked,
			"configured":    true,
			"using_default": false,
		}
	}
	if appconfig.UsingBuiltinTMDB() {
		return map[string]interface{}{
			"key":           "",
			"configured":    true,
			"using_default": true,
		}
	}
	return map[string]interface{}{"key": "", "configured": false, "using_default": false}
}

// SaveTMDBKey saves the TMDB API key to config.
// Pro services (SearchService, TMDBService) read the key from appconfig on
// each call, so no explicit refresh is needed here.
func (s *SettingsService) SaveTMDBKey(key string) map[string]interface{} {
	key = strings.TrimSpace(key)
	if err := appconfig.SetTMDBApiKey(key); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("failed to save: %v", err)}
	}
	// Notify pro services to refresh their TMDB clients with the new key
	refreshTMDBClients(key)
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

// SaveJackett saves Jackett/Torznab configuration.
func (s *SettingsService) SaveJackett(enabled bool, urlStr, apiKey string) map[string]interface{} {
	urlStr = strings.TrimSpace(urlStr)
	apiKey = strings.TrimSpace(apiKey)

	if err := appconfig.SetJackettEnabled(enabled); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if err := appconfig.SetJackettURL(urlStr); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	if err := appconfig.SetJackettAPIKey(apiKey); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "saved"}
}

// GetJackett returns the current Jackett/Torznab configuration.
func (s *SettingsService) GetJackett() map[string]interface{} {
	return map[string]interface{}{
		"enabled": appconfig.JackettEnabled(),
		"url":     appconfig.JackettURL(),
		"api_key": appconfig.JackettAPIKey(),
	}
}

// providerDescriptions maps provider names to short descriptions for the UI.
var providerDescriptions = map[string]string{
	"ThePirateBay":  "General torrents, most popular public tracker",
	"1337x":         "General torrents, popular indexed tracker",
	"YTS":           "Movies, high-quality movie torrents",
	"TorrentGalaxy": "General torrents, active community",
	"SolidTorrents": "General torrents, meta-search",
	"TorrentsCSV":   "General torrents, JSON API, fast",
	"Nyaa":          "Anime torrents",
	"EZTV":          "TV shows, classic TV tracker",
	"LimeTorrents":  "General torrents, large catalog",
	"Sukebei":       "Adult content (NSFW)",
	"Bitsearch":     "General torrents, decentralized search",
	"Knaben":        "General torrents, multi-source aggregator",
	"BTDigg":        "General torrents, DHT search engine",
	"TorrentProject":"General torrents, torrent search engine",
	"Jackett":       "500+ sites via local Jackett instance (requires setup)",
}

// GetProviders returns all providers with their enabled status for the UI.
func (s *SettingsService) GetProviders() []map[string]interface{} {
	enabled := appconfig.EnabledProviders()
	isDefault := len(enabled) == 0
	enabledSet := make(map[string]bool, len(enabled))
	for _, n := range enabled {
		enabledSet[n] = true
	}

	var out []map[string]interface{}
	for _, name := range mantorex.ProviderNames() {
		isEnabled := (isDefault && name != "Jackett") || enabledSet[name]
		out = append(out, map[string]interface{}{
			"name":        name,
			"enabled":     isEnabled,
			"description": providerDescriptions[name],
		})
	}
	return out
}

// SaveProviders saves the list of enabled provider names.
func (s *SettingsService) SaveProviders(names []string) map[string]interface{} {
	if err := appconfig.SetEnabledProviders(names); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	return map[string]interface{}{"status": "saved"}
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

// GetUISettings returns desktop UI preferences from disk (survives rebuilds).
func (s *SettingsService) GetUISettings() map[string]interface{} {
	ui := appconfig.GetUISettings()
	ps := appconfig.GetPlayerSettings()
	return map[string]interface{}{
		"font":                    ui.Font,
		"font_size":               ui.FontSize,
		"scale":                   ui.Scale,
		"animations":              ui.Animations,
		"blur":                    ui.Blur,
		"blur_set":                appconfig.BlurIsSet(),
		"player_backend":          ui.PlayerBackend,
		"startup_tab":             ui.StartupTab,
		"mantorex_legal_accepted": ui.MantorexLegalAccepted,
		// false until the user has saved UI prefs at least once
		"configured": appconfig.UIConfigured(),
		// Native player tuning (Settings → Player)
		"player_hwdec":            ps.Hwdec,
		"player_cache":            ps.Cache,
		"player_hq_scale":         ps.HqScale,
		"player_deinterlace":      ps.Deinterlace,
		"player_load_user_config": ps.LoadUserConfig,
	}
}

// SaveUISettings persists desktop UI preferences to ~/.config/yaria/app.toml.
func (s *SettingsService) SaveUISettings(settings map[string]interface{}) map[string]interface{} {
	ui := appconfig.GetUISettings()
	if v, ok := settings["font"].(string); ok && v != "" {
		ui.Font = v
	}
	if v, ok := settings["font_size"].(string); ok && v != "" {
		ui.FontSize = v
	}
	// Accept number or string for scale/font_size from JS
	if v, ok := settings["font_size"].(float64); ok {
		ui.FontSize = fmt.Sprintf("%d", int(v))
	}
	if v, ok := settings["scale"].(string); ok && v != "" {
		ui.Scale = v
	}
	if v, ok := settings["scale"].(float64); ok {
		ui.Scale = fmt.Sprintf("%d", int(v))
	}
	if v, ok := settings["animations"].(bool); ok {
		ui.Animations = v
	}
	if v, ok := settings["blur"].(bool); ok {
		ui.Blur = v
	}
	if v, ok := settings["player_backend"].(string); ok && v != "" {
		if v == "libmpv" {
			ui.PlayerBackend = "libmpv"
		} else {
			ui.PlayerBackend = "webview"
		}
	}
	if v, ok := settings["startup_tab"].(string); ok && v != "" {
		if v == "mantorex" {
			ui.StartupTab = "mantorex"
		} else {
			ui.StartupTab = "yaria"
		}
	}
	if v, ok := settings["mantorex_legal_accepted"].(bool); ok {
		ui.MantorexLegalAccepted = v
	}
	if err := appconfig.SetUISettings(ui); err != nil {
		return map[string]interface{}{"error": err.Error()}
	}

	// Native player options (optional keys — merge with existing)
	ps := appconfig.GetPlayerSettings()
	changed := false
	if v, ok := settings["player_hwdec"].(string); ok && v != "" {
		ps.Hwdec = v
		changed = true
	}
	if v, ok := settings["player_cache"].(string); ok && v != "" {
		ps.Cache = v
		changed = true
	}
	if v, ok := settings["player_hq_scale"].(bool); ok {
		ps.HqScale = v
		changed = true
	}
	if v, ok := settings["player_deinterlace"].(bool); ok {
		ps.Deinterlace = v
		changed = true
	}
	if v, ok := settings["player_load_user_config"].(bool); ok {
		ps.LoadUserConfig = v
		changed = true
	}
	if changed {
		if err := appconfig.SetPlayerSettings(ps); err != nil {
			return map[string]interface{}{"error": err.Error()}
		}
	}
	return map[string]interface{}{"status": "saved"}
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
