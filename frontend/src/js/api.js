// Wails bridge API for Yaria desktop app
const API = {
  // Settings
  async getTMDBKey() {
    return window.go.main.SettingsService.GetTMDBKey();
  },
  async saveTMDBKey(key) {
    return window.go.main.SettingsService.SaveTMDBKey(key);
  },
  async getCacheStats() {
    return window.go.main.SettingsService.GetCacheStats();
  },
  async clearCache(type) {
    return window.go.main.SettingsService.ClearCache(type);
  },
  async saveProxy(type, addr) {
    return window.go.main.SettingsService.SaveProxy(type || 'none', addr || '');
  },
  async getProxy() {
    return window.go.main.SettingsService.GetProxy();
  },
  async saveSpeedLimit(limitBytes) {
    return window.go.main.SettingsService.SaveSpeedLimit(limitBytes || 0);
  },
  async getSpeedLimit() {
    return window.go.main.SettingsService.GetSpeedLimit();
  },

  // Yaria Download
  async initDeps() {
    return window.go.main.DownloadService.InitDeps();
  },
  async checkDeps() {
    return window.go.main.DownloadService.CheckDeps();
  },
  async fetchMetadata(url) {
    return window.go.main.DownloadService.FetchMetadata(url);
  },
  async listFormats(url) {
    return window.go.main.DownloadService.ListFormats(url);
  },
  async startDownload(url, resolution, downloadDir, audioOnly, audioFormat) {
    return window.go.main.DownloadService.StartDownload(url, resolution, downloadDir, audioOnly, audioFormat || 'mp3');
  },
  async cancelDownload(id) {
    return window.go.main.DownloadService.CancelDownload(id);
  },
  async getDownloads() {
    return window.go.main.DownloadService.GetDownloads();
  },
  async removeDownload(id) {
    return window.go.main.DownloadService.RemoveDownload(id);
  },
  async selectDownloadDir() {
    return window.go.main.DownloadService.SelectDownloadDir();
  },
  async getDownloadDir() {
    return window.go.main.DownloadService.GetDownloadDir();
  },
  async checkExistingDownload(url, downloadDir) {
    return window.go.main.DownloadService.CheckExistingDownload(url, downloadDir || '');
  },
  async detectPlaylist(url) {
    return window.go.main.DownloadService.DetectPlaylist(url);
  },
  async setMaxConcurrent(n) {
    return window.go.main.DownloadService.SetMaxConcurrent(n);
  },
  async getMaxConcurrent() {
    return window.go.main.DownloadService.GetMaxConcurrent();
  },

  // License
  async checkLicense() {
    return window.go.main.LicenseService.CheckLicense();
  },
  async activateKey(key) {
    return window.go.main.LicenseService.ActivateKey(key);
  },
  async deactivate() {
    return window.go.main.LicenseService.Deactivate();
  },
  async getDeviceInfo() {
    return window.go.main.LicenseService.GetDeviceInfo();
  },
  async isPro() {
    return window.go.main.LicenseService.IsPro();
  },

  // File picker
  async listDirectories(path) {
    return window.go.main.DepsService.ListDirectories(path);
  },

  // App Dependencies
  async checkAppDeps() {
    return window.go.main.DepsService.CheckDeps();
  },
  async installFFmpeg() {
    return window.go.main.DepsService.InstallFFmpeg();
  },
  async ffmpegPath() {
    return window.go.main.DepsService.FFmpegPath();
  },
  async getStreamDetails(filePath) {
    return window.go.main.DepsService.GetStreamDetails(filePath);
  },

  // Codecs
  async checkCodecs() {
    return window.go.main.CodecService.CheckCodecs();
  },
  async installCodecs() {
    return window.go.main.CodecService.InstallCodecs();
  },

  // Player (mpv/vlc fallback)
  async detectPlayers() {
    return window.go.main.PlayerService.DetectPlayers();
  },
  async playFile(filePath) {
    return window.go.main.PlayerService.PlayFile(filePath);
  },
  async launchPlayer(streamURL, playerName, title, resumeSeconds) {
    return window.go.main.PlayerService.LaunchPlayer(streamURL, playerName || '', title || '', resumeSeconds || 0);
  },

  // --- Pro methods (Mantorex) ---
  // These call stub services in free builds (return errors).
  // Real implementations are compiled in pro builds.
  async metaSearch(q) {
    return window.go.main.SearchService.MetaSearch(q);
  },
  async metaTrending(mediaType) {
    return window.go.main.SearchService.MetaTrending(mediaType || 'all');
  },
  async searchTorrents(q, category, sortBy, title) {
    return window.go.main.SearchService.SearchTorrents(q, category || 'All', sortBy || '', title || '');
  },
  async startStream(magnet) {
    return window.go.main.StreamService.StartStream(magnet);
  },
  async getStatus() {
    return window.go.main.StreamService.GetStatus();
  },
  async stopStream() {
    return window.go.main.StreamService.StopStream();
  },
  async pauseStream() {
    return window.go.main.StreamService.PauseStream();
  },
  async resumeStream() {
    return window.go.main.StreamService.ResumeStream();
  },
  async listFiles() {
    return window.go.main.StreamService.ListFiles();
  },
  async selectFile(index) {
    return window.go.main.StreamService.SelectFile(index);
  },
  // Torrent Downloads
  async addTorrentDownload(magnet, title, dir) {
    return window.go.main.TorrentDownloadService.AddDownload(magnet, title || '', dir || '');
  },
  async listTorrentDownloads() {
    return window.go.main.TorrentDownloadService.ListDownloads();
  },
  async removeTorrentDownload(id) {
    return window.go.main.TorrentDownloadService.RemoveDownload(id);
  },
  async deleteTorrentDownload(id) {
    return window.go.main.TorrentDownloadService.DeleteDownload(id);
  },
  async cancelTorrentDownload(id) {
    return window.go.main.TorrentDownloadService.CancelDownload(id);
  },
  async pauseTorrentDownload(id) {
    return window.go.main.TorrentDownloadService.PauseDownload(id);
  },
  async resumeTorrentDownload(id) {
    return window.go.main.TorrentDownloadService.ResumeDownload(id);
  },
  async getTorrentDownloadDir() {
    return window.go.main.TorrentDownloadService.GetDownloadDir();
  },
  async selectTorrentDownloadDir() {
    return window.go.main.TorrentDownloadService.SelectDownloadDir();
  },

  async getLibrary() {
    return window.go.main.LibraryService.GetAll();
  },
  async addToLibrary(item) {
    return window.go.main.LibraryService.Add(item);
  },
  async removeFromLibrary(id) {
    return window.go.main.LibraryService.Remove(id);
  },
  async updateProgress(id, progress) {
    return window.go.main.LibraryService.UpdateProgress(id, progress);
  },
  async findInLibrary(tmdbId, mediaType) {
    if (tmdbId) return window.go.main.LibraryService.FindByTMDBID(tmdbId, mediaType || '');
    return null;
  },
  async findInLibraryByTitle(title, mediaType, year) {
    return window.go.main.LibraryService.FindByTitle(title || '', mediaType || '', year || '');
  },
  async updateEpisodeProgress(id, season, episode, title, timeSeconds, durationSeconds) {
    return window.go.main.LibraryService.UpdateEpisodeProgress(id, season, episode, title || '', timeSeconds || 0, durationSeconds || 0);
  },
  async markEpisodeWatched(id, season, episode) {
    return window.go.main.LibraryService.MarkEpisodeWatched(id, season, episode);
  },
  async setLastKnownEpisode(id, season, episode, airDate) {
    return window.go.main.LibraryService.SetLastKnownEpisode(id, season, episode, airDate || '');
  },
  async getRecentlyAdded(days) {
    return window.go.main.LibraryService.GetRecentlyAdded(days || 7);
  },
  async getInProgress() {
    return window.go.main.LibraryService.GetInProgress();
  },
  async exportLibrary() {
    return window.go.main.LibraryService.ExportLibrary();
  },
  async importLibrary(jsonData) {
    return window.go.main.LibraryService.ImportLibrary(jsonData);
  },
  async tmdbSearch(q, page) {
    return window.go.main.TMDBService.SearchMulti(q, page || 1);
  },
  async tmdbTrending(type, page) {
    return window.go.main.TMDBService.GetTrending(type || 'all', page || 1);
  },
  async tmdbMovieDetails(id) {
    return window.go.main.TMDBService.GetMovieDetails(id);
  },
  async tmdbTVDetails(id) {
    return window.go.main.TMDBService.GetTVDetails(id);
  },
  async tmdbSeasonDetails(tvId, season) {
    return window.go.main.TMDBService.GetSeasonDetails(tvId, season);
  },
};

async function getStreamURL() {
  return window.go.main.StreamService.GetStreamURL();
}
async function getTranscodeURL() {
  return window.go.main.StreamService.GetTranscodeURL();
}

function esc(s) {
  if (!s) return '';
  const d = document.createElement('div');
  d.textContent = String(s);
  return d.innerHTML;
}

const NO_POSTER = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='150' height='225'%3E%3Crect fill='%231a1a2e' width='150' height='225' rx='8'/%3E%3Ctext fill='%23555570' x='75' y='112' text-anchor='middle' font-size='12' font-family='sans-serif'%3ENo Poster%3C/text%3E%3C/svg%3E";
