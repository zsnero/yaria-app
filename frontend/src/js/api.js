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
  async deleteDownloadFiles(id) {
    return window.go.main.DownloadService.DeleteDownloadFiles(id);
  },
  async playDownloadedFile(id) {
    return window.go.main.DownloadService.PlayDownloadedFile(id);
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


  // File utilities
  async playFile(filePath) {
    return window.go.main.PlayerService.PlayFile(filePath);
  },
  async openFolder(filePath) {
    return window.go.main.PlayerService.OpenFolder(filePath);
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

  // Local Media Library
  async getMediaFolders() {
    return window.go.main.MediaService.GetMediaFolders();
  },
  async addMediaFolder(path, mediaType) {
    return window.go.main.MediaService.AddMediaFolder(path, mediaType);
  },
  async removeMediaFolder(path, mediaType) {
    return window.go.main.MediaService.RemoveMediaFolder(path, mediaType);
  },
  async browseMediaFolder() {
    return window.go.main.MediaService.BrowseMediaFolder();
  },
  async scanMedia() {
    return window.go.main.MediaService.ScanAll();
  },
  async scanMediaStatus() {
    return window.go.main.MediaService.ScanStatus();
  },
  async getAllMedia(sortBy) {
    return window.go.main.MediaService.GetAllMedia(sortBy || 'added');
  },
  async getLocalMovies(sortBy) {
    return window.go.main.MediaService.GetMovies(sortBy || 'added');
  },
  async getLocalTVShows() {
    return window.go.main.MediaService.GetTVShows();
  },
  async getRecentlyAddedMedia(days) {
    return window.go.main.MediaService.GetRecentlyAdded(days || 30);
  },
  async searchLocalMedia(query) {
    return window.go.main.MediaService.SearchMedia(query);
  },
  async getMediaDetails(id) {
    return window.go.main.MediaService.GetMediaDetails(id);
  },
  async getMediaThumbnail(id) {
    return window.go.main.MediaService.GetThumbnailData(id);
  },
  async playLocalMedia(id) {
    return window.go.main.MediaService.PlayMedia(id);
  },
  async playLocalMediaExternal(id) {
    return window.go.main.MediaService.PlayMediaExternal(id);
  },
  async getLocalStreamURL() {
    return window.go.main.MediaService.GetLocalStreamURL();
  },
  async matchMediaTMDB(mediaId, tmdbId, mediaType) {
    return window.go.main.MediaService.MatchTMDB(mediaId, tmdbId, mediaType || '');
  },
  async getUnmatchedMedia() {
    return window.go.main.MediaService.GetUnmatched();
  },
  // Remote Sources
  async getRemoteSources() {
    return window.go.main.RemoteService.GetSources();
  },
  async addRemoteSource(sourceJSON) {
    return window.go.main.RemoteService.AddSource(sourceJSON);
  },
  async removeRemoteSource(id) {
    return window.go.main.RemoteService.RemoveSource(id);
  },
  async connectRemote(id, password) {
    return window.go.main.RemoteService.Connect(id, password || '');
  },
  async disconnectRemote(id) {
    return window.go.main.RemoteService.Disconnect(id);
  },
  async isRemoteConnected(id) {
    return window.go.main.RemoteService.IsConnected(id);
  },
  async connectedSources() {
    return window.go.main.RemoteService.ConnectedSources();
  },
  async browseRemote(sourceId, path) {
    return window.go.main.RemoteService.BrowseRemote(sourceId, path || '/');
  },
  async getRemoteStreamURL(sourceId, filePath) {
    return window.go.main.RemoteService.GetRemoteStreamURL(sourceId, filePath);
  },
  async discoverDevices() {
    return window.go.main.RemoteService.DiscoverDevices();
  },
  async quickConnect(host, port, type) {
    return window.go.main.RemoteService.QuickConnect(host, port, type);
  },
  async selectKeyFile() {
    return window.go.main.RemoteService.SelectKeyFile();
  },
  // Media Server
  async startMediaServer() {
    return window.go.main.MediaServer.Start();
  },
  async stopMediaServer() {
    return window.go.main.MediaServer.Stop();
  },
  async mediaServerStatus() {
    return window.go.main.MediaServer.Status();
  },
  async setMediaServerPort(port) {
    return window.go.main.MediaServer.SetPort(port);
  },
  async setMediaServerPin(pin) {
    return window.go.main.MediaServer.SetPin(pin);
  },
  // Watch state (Feature 4)
  async markMediaWatched(id) {
    return window.go.main.MediaService.MarkWatched(id);
  },
  async markMediaUnwatched(id) {
    return window.go.main.MediaService.MarkUnwatched(id);
  },
  async updateResumePosition(id, posSeconds) {
    return window.go.main.MediaService.UpdateResumePosition(id, posSeconds);
  },
  async getContinueWatchingMedia() {
    return window.go.main.MediaService.GetContinueWatching();
  },
  async getWatchHistory() {
    return window.go.main.MediaService.GetWatchHistory();
  },
  // Music (Feature 7)
  async scanMusic() {
    return window.go.main.MediaService.ScanMusic();
  },
  async getMusicLibrary() {
    return window.go.main.MediaService.GetMusicLibrary();
  },
  // HW acceleration (Feature 11)
  async detectHWAccel() {
    return window.go.main.MediaService.DetectHWAccel();
  },
  // NFO export (Feature 12)
  async writeNFO(id) {
    return window.go.main.MediaService.WriteNFO(id);
  },
  // DLNA (Feature 6)
  async startDLNA() {
    return window.go.main.DLNAService.Start();
  },
  async stopDLNA() {
    return window.go.main.DLNAService.Stop();
  },
  async dlnaStatus() {
    return window.go.main.DLNAService.Status();
  },
  // Profiles (Feature 8)
  async getProfiles() {
    return window.go.main.MediaService.GetProfiles();
  },
  async addProfile(name, pin, avatar) {
    return window.go.main.MediaService.AddProfile(name, pin || '', avatar || '');
  },
  async removeProfile(id) {
    return window.go.main.MediaService.RemoveProfile(id);
  },
  // Lyrics (Feature 14)
  async getLyrics(id) {
    return window.go.main.MediaService.GetLyrics(id);
  },
  // Updater
  async getAppVersion() {
    return window.go.main.UpdaterService.GetVersion();
  },
  async checkUpdate() {
    return window.go.main.UpdaterService.CheckUpdate();
  },
  async startUpdate() {
    return window.go.main.UpdaterService.Update();
  },
  async updateStatus() {
    return window.go.main.UpdaterService.UpdateStatus();
  },
  async getLocalPageData() {
    return window.go.main.MediaService.GetLocalPageData();
  },
  async getMediaCount() {
    return window.go.main.MediaService.GetMediaCount();
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

// Filter out torrents with codecs that WebKitGTK cannot reliably decode.
// Only applies on Linux (WebKitGTK) when the filter is enabled (default: on).
// Users can disable this in Settings > General to show all formats.
// Windows (WebView2/Chromium) and macOS (native WebKit) are unaffected.
const _isLinux = navigator.platform.indexOf('Linux') !== -1;
const _unsupportedRE = /\b(hevc|[hx]\.?265|10[.\-\s]?bit)\b/i;
function filterPlayableTorrents(torrents) {
  if (!_isLinux || !torrents || !Array.isArray(torrents)) return torrents;
  if (localStorage.getItem('yaria_show_all_formats') === '1') return torrents;
  return torrents.filter(t => !_unsupportedRE.test(t.title || ''));
}


// Custom confirm dialog (replaces browser confirm())
function appConfirm(message, onConfirm, onCancel) {
  const overlay = document.createElement('div');
  overlay.className = 'app-confirm-overlay';
  overlay.innerHTML = `
    <div class="app-confirm-modal">
      <p class="app-confirm-msg">${esc(message)}</p>
      <div class="app-confirm-actions">
        <button class="btn btn-ghost" id="app-confirm-cancel">Cancel</button>
        <button class="btn btn-primary" id="app-confirm-ok">Confirm</button>
      </div>
    </div>
  `;
  document.body.appendChild(overlay);
  overlay.querySelector('#app-confirm-ok').addEventListener('click', () => { overlay.remove(); if (onConfirm) onConfirm(); });
  overlay.querySelector('#app-confirm-cancel').addEventListener('click', () => { overlay.remove(); if (onCancel) onCancel(); });
  overlay.addEventListener('click', (e) => { if (e.target === overlay) { overlay.remove(); if (onCancel) onCancel(); } });
  overlay.querySelector('#app-confirm-ok').focus();
}
