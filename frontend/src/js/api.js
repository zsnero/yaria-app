// Wails bridge API for Yaria desktop app
const API = {
  // Settings
  async saveTMDBKey(key) {
    return window.go.main.SettingsService.SaveTMDBKey(key);
  },
  async getCacheStats() {
    return window.go.main.SettingsService.GetCacheStats();
  },
  async clearCache(type) {
    return window.go.main.SettingsService.ClearCache(type);
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
