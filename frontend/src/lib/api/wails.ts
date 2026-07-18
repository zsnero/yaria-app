// Typed Wails API bindings
// This wraps all window.go.main.* calls with TypeScript types

import type {
  TorrentResult,
  MetaResult,
  LibraryItem,
  LocalMedia,
  TVShowGroup,
  RemoteSource,
  RemoteFile,
  StreamStatus,
  DepInfo,
  SubtitleTrack,
  PrepareStreamResult,
  WatchProgress,
} from '../types';

// Helper to safely call Wails bindings
async function call<T>(fn: () => Promise<T>): Promise<T> {
  try {
    return await fn();
  } catch (e) {
    console.error('Wails call failed:', e);
    throw e;
  }
}

// Wait for Wails runtime to be ready
function wails(): any {
  return (window as any).go?.main;
}

function runtime(): any {
  return (window as any).runtime;
}

// === App ===
export const app = {
  getAppVersion: (): Promise<string> =>
    call(() => wails().UpdaterService.GetVersion()),
};

// === License ===
export const license = {
  check: (): Promise<any> =>
    call(() => wails().LicenseService.CheckLicense()),
  activate: (key: string): Promise<any> =>
    call(() => wails().LicenseService.ActivateKey(key)),
  startTrial: (): Promise<any> =>
    call(() => wails().LicenseService.StartTrial()),
  deactivate: (): Promise<any> =>
    call(() => wails().LicenseService.Deactivate()),
  isPro: (): Promise<boolean> =>
    call(() => wails().LicenseService.IsPro()),
  getDeviceInfo: (): Promise<any> =>
    call(() => wails().LicenseService.GetDeviceInfo()),
};

// === Settings ===
export const settings = {
  getTMDBKey: (): Promise<{ configured: boolean; key: string }> =>
    call(() => wails().SettingsService.GetTMDBKey()),
  saveTMDBKey: (key: string): Promise<any> =>
    call(() => wails().SettingsService.SaveTMDBKey(key)),
  getProxy: (): Promise<{ type: string; addr: string }> =>
    call(() => wails().SettingsService.GetProxy()),
  saveProxy: (type: string, addr: string): Promise<any> =>
    call(() => wails().SettingsService.SaveProxy(type, addr)),
  getCacheStats: (): Promise<any> =>
    call(() => wails().SettingsService.GetCacheStats()),
  clearCache: (type: string): Promise<any> =>
    call(() => wails().SettingsService.ClearCache(type)),
  getUISettings: (): Promise<{
    font: string;
    font_size: string;
    scale: string;
    animations: boolean;
    blur: boolean;
    blur_set?: boolean;
    configured?: boolean;
  }> => call(() => wails().SettingsService.GetUISettings()),
  saveUISettings: (settings: {
    font?: string;
    font_size?: string;
    scale?: string;
    animations?: boolean;
    blur?: boolean;
  }): Promise<any> => call(() => wails().SettingsService.SaveUISettings(settings)),
};

// === Downloads (Yaria) ===
export const downloads = {
  getMetadata: (url: string): Promise<any> =>
    call(() => wails().DownloadService.FetchMetadata(url)),
  start: (url: string, resolution: string, dir: string, audioOnly: boolean, audioFormat: string, containerFormat?: string): Promise<any> =>
    call(() => wails().DownloadService.StartDownload(url, resolution, dir, audioOnly, audioFormat, containerFormat || 'mp4')),
  cancel: (id: string): Promise<any> =>
    call(() => wails().DownloadService.CancelDownload(id)),
  remove: (id: string): Promise<any> =>
    call(() => wails().DownloadService.RemoveDownload(id)),
  list: (): Promise<any[]> =>
    call(() => wails().DownloadService.GetDownloads()),
  initDeps: (): Promise<any> =>
    call(() => wails().DownloadService.InitDeps()),
  getSpeedLimit: (): Promise<number> =>
    call(() => wails().SettingsService.GetSpeedLimit()),
  setSpeedLimit: (limit: number): Promise<any> =>
    call(() => wails().SettingsService.SaveSpeedLimit(limit)),
  getMaxConcurrent: (): Promise<number> =>
    call(() => wails().DownloadService.GetMaxConcurrent()),
  setMaxConcurrent: (max: number): Promise<any> =>
    call(() => wails().DownloadService.SetMaxConcurrent(max)),
  listFormats: (url: string): Promise<any> =>
    call(() => wails().DownloadService.ListFormats(url)),
  getDownloadDir: (): Promise<string> =>
    call(() => wails().DownloadService.GetDownloadDir()),
  checkExistingDownload: (url: string, dir: string): Promise<any> =>
    call(() => wails().DownloadService.CheckExistingDownload(url, dir)),
  deleteDownloadFiles: (id: string): Promise<any> =>
    call(() => wails().DownloadService.DeleteDownloadFiles(id)),
  playDownloadedFile: (id: string): Promise<any> =>
    call(() => wails().DownloadService.PlayDownloadedFile(id)),
  checkDeps: (): Promise<any> =>
    call(() => wails().DownloadService.CheckDeps()),
};

// === Dependencies ===
export const deps = {
  check: (): Promise<{ all_ready: boolean; deps: DepInfo[] }> =>
    call(() => wails().DepsService.CheckDeps()),
  installFFmpeg: (): Promise<any> =>
    call(() => wails().DepsService.InstallFFmpeg()),
  ffmpegPath: (): Promise<string> =>
    call(() => wails().DepsService.FFmpegPath()),
  listDirectories: (path: string): Promise<{ name: string; path: string }[]> =>
    call(() => wails().DepsService.ListDirectories(path)),
  listEntries: (path: string, fileExt = ''): Promise<{ name: string; path: string; is_dir?: boolean }[]> =>
    call(() => wails().DepsService.ListEntries(path, fileExt)),
  readTextFile: (path: string): Promise<{ data?: string; path?: string; error?: string }> =>
    call(() => wails().DepsService.ReadTextFile(path)),
  writeTextFile: (path: string, content: string): Promise<{ status?: string; path?: string; error?: string }> =>
    call(() => wails().DepsService.WriteTextFile(path, content)),
};

// === Torrent Search ===
export const search = {
  torrents: (q: string, category: string, sortBy: string, filterTitle?: string): Promise<{ results: TorrentResult[]; count: number }> =>
    call(() => wails().SearchService.SearchTorrents(q, category || 'All', sortBy || '', filterTitle || '')),
  meta: (q: string): Promise<{ results: MetaResult[] }> =>
    call(() => wails().SearchService.MetaSearch(q)),
};

// === TMDB ===
export const tmdb = {
  search: (q: string, page: number): Promise<any> =>
    call(() => wails().TMDBService.SearchMulti(q, page)),
  trending: (mediaType: string, page?: number): Promise<{ results: MetaResult[] }> =>
    call(() => wails().SearchService.MetaTrending(mediaType || 'all')),
  movieDetails: (id: number): Promise<any> =>
    call(() => wails().TMDBService.GetMovieDetails(id)),
  tvDetails: (id: number): Promise<any> =>
    call(() => wails().TMDBService.GetTVDetails(id)),
  seasonDetails: (tvId: number, season: number): Promise<any> =>
    call(() => wails().TMDBService.GetSeasonDetails(tvId, season)),
};

// === Streaming ===
export const stream = {
  start: (magnet: string): Promise<{ status: string; stream_id: number }> =>
    call(() => wails().StreamService.StartStream(magnet)),
  stop: (): Promise<any> =>
    call(() => wails().StreamService.StopStream()),
  pause: (): Promise<any> =>
    call(() => wails().StreamService.PauseStream()),
  resume: (): Promise<any> =>
    call(() => wails().StreamService.ResumeStream()),
  getStatus: (): Promise<StreamStatus> =>
    call(() => wails().StreamService.GetStatus()),
  getStreamURL: (): Promise<string> =>
    call(() => wails().StreamService.GetStreamURL()),
  getHLSURL: (): Promise<string> =>
    call(() => wails().StreamService.GetHLSURL()),
  getVODURL: (): Promise<string> =>
    call(() => wails().StreamService.GetVODURL()),
  isHLSActive: (): Promise<boolean> =>
    call(() => wails().StreamService.IsHLSActive()),
  isVODActive: (): Promise<boolean> =>
    call(() => wails().StreamService.IsVODActive()),
  prepareStream: (): Promise<PrepareStreamResult> =>
    call(() => wails().StreamService.PrepareStream()),
  prepareFileVOD: (filePath: string): Promise<PrepareStreamResult> =>
    call(() => wails().StreamService.PrepareFileVOD(filePath)),
  prepareURLVOD: (streamURL: string): Promise<PrepareStreamResult> =>
    call(() => wails().StreamService.PrepareURLVOD(streamURL)),
  getDuration: (): Promise<number> =>
    call(() => wails().StreamService.GetStreamDuration()),
  listFiles: (): Promise<any[]> =>
    call(() => wails().StreamService.ListFiles()),
  selectFile: (index: number): Promise<any> =>
    call(() => wails().StreamService.SelectFile(index)),
  listSubtitles: (): Promise<SubtitleTrack[]> =>
    call(() => wails().StreamService.ListSubtitleFiles()),
  getTranscodeURL: (): Promise<string> =>
    call(() => wails().StreamService.GetTranscodeURL()),
};

// === Library ===
export const library = {
  getAll: (): Promise<{ items: LibraryItem[] }> =>
    call(() => wails().LibraryService.GetAll()),
  add: (item: Partial<LibraryItem>): Promise<{ item: LibraryItem; duplicate: boolean }> =>
    call(() => wails().LibraryService.Add(item)),
  remove: (id: string): Promise<any> =>
    call(() => wails().LibraryService.Remove(id)),
  updateProgress: (id: string, progress: Partial<WatchProgress>): Promise<any> =>
    call(() => wails().LibraryService.UpdateProgress(id, progress)),
  updateMeta: (id: string, meta: Partial<LibraryItem>): Promise<any> =>
    call(() => wails().LibraryService.UpdateMeta(id, meta)),
  findByTMDBID: (tmdbId: number, mediaType: string): Promise<LibraryItem | null> =>
    call(() => wails().LibraryService.FindByTMDBID(tmdbId, mediaType)),
  findByTitle: (title: string, mediaType: string, year: string): Promise<LibraryItem | null> =>
    call(() => wails().LibraryService.FindByTitle(title, mediaType, year)),
  exportLibrary: (): Promise<{ data?: string; count?: number; error?: string }> =>
    call(() => wails().LibraryService.ExportLibrary()),
  importLibrary: (jsonData: string): Promise<{ status?: string; added?: number; updated?: number; skipped?: number; total?: number; error?: string }> =>
    call(() => wails().LibraryService.ImportLibrary(jsonData)),
};

// === Torrent Downloads ===
export const torrentDownloads = {
  add: (magnet: string, title: string, dir: string): Promise<{ id?: string; status?: string; error?: string; existing?: boolean }> =>
    call(() => wails().TorrentDownloadService.AddDownload(magnet, title, dir)),
  list: (): Promise<any[]> =>
    call(() => wails().TorrentDownloadService.ListDownloads()),
  pause: (id: string): Promise<any> =>
    call(() => wails().TorrentDownloadService.PauseDownload(id)),
  resume: (id: string): Promise<any> =>
    call(() => wails().TorrentDownloadService.ResumeDownload(id)),
  remove: (id: string): Promise<any> =>
    call(() => wails().TorrentDownloadService.RemoveDownload(id)),
  delete: (id: string): Promise<any> =>
    call(() => wails().TorrentDownloadService.DeleteDownload(id)),
  cancel: (id: string): Promise<any> =>
    call(() => wails().TorrentDownloadService.CancelDownload(id)),
  getPlayPath: (id: string): Promise<{ path?: string; title?: string; dir?: string; error?: string }> =>
    call(() => wails().TorrentDownloadService.GetPlayPath(id)),
};

// === Local Media ===
export const media = {
  scan: (): Promise<any> =>
    call(() => wails().MediaService.ScanAll()),
  scanStatus: (): Promise<any> =>
    call(() => wails().MediaService.ScanStatus()),
  getLocalPageData: (): Promise<any> =>
    call(() => wails().MediaService.GetLocalPageData()),
  getDetails: (id: string): Promise<LocalMedia | null> =>
    call(() => wails().MediaService.GetMediaDetails(id)),
  play: (id: string): Promise<{ stream_url: string; title: string; duration: number }> =>
    call(() => wails().MediaService.PlayMedia(id)),
  playPath: (filePath: string): Promise<{ stream_url?: string; title?: string; error?: string; id?: string }> =>
    call(() => wails().MediaService.PlayPath(filePath)),
  getContinueWatching: (): Promise<LocalMedia[]> =>
    call(() => wails().MediaService.GetContinueWatching()),
  getWatchHistory: (): Promise<LocalMedia[]> =>
    call(() => wails().MediaService.GetWatchHistory()),
  updateResumePosition: (id: string, seconds: number): Promise<any> =>
    call(() => wails().MediaService.UpdateResumePosition(id, seconds)),
  listSubtitles: (id: string): Promise<SubtitleTrack[]> =>
    call(() => wails().MediaService.ListLocalSubtitles(id)),
  getMediaFolders: (): Promise<any> =>
    call(() => wails().MediaService.GetMediaFolders()),
  addMediaFolder: (path: string, category: string): Promise<any> =>
    call(() => wails().MediaService.AddMediaFolder(path, category)),
  removeMediaFolder: (path: string, type: string): Promise<any> =>
    call(() => wails().MediaService.RemoveMediaFolder(path, type)),
  getMediaCount: (): Promise<{ total: number }> =>
    call(() => wails().MediaService.GetMediaCount()),
  prepareLocalHLS: (id: string): Promise<PrepareStreamResult> =>
    call(() => wails().MediaService.PrepareLocalHLS(id)),
  searchMedia: (q: string): Promise<LocalMedia[]> =>
    call(() => wails().MediaService.SearchMedia(q)),
  detectHWAccel: (): Promise<any> =>
    call(() => wails().MediaService.DetectHWAccel()),
};

// === Remote ===
export const remote = {
  getSources: (): Promise<RemoteSource[]> =>
    call(() => wails().RemoteService.GetSources()),
  addSource: (source: Partial<RemoteSource>): Promise<any> =>
    call(() => wails().RemoteService.AddSource(JSON.stringify(source))),
  removeSource: (id: string): Promise<any> =>
    call(() => wails().RemoteService.RemoveSource(id)),
  connect: (id: string, password?: string): Promise<any> =>
    call(() => wails().RemoteService.Connect(id, password || '')),
  disconnect: (id: string): Promise<any> =>
    call(() => wails().RemoteService.Disconnect(id)),
  browse: (sourceId: string, path: string): Promise<RemoteFile[]> =>
    call(() => wails().RemoteService.BrowseRemote(sourceId, path || '/')),
  getStreamURL: (sourceId: string, filePath: string): Promise<string> =>
    call(() => wails().RemoteService.GetRemoteStreamURL(sourceId, filePath)),
  listSubtitles: (sourceId: string, filePath: string): Promise<SubtitleTrack[]> =>
    call(() => wails().RemoteService.ListRemoteSubtitles(sourceId, filePath)),
  discover: (): Promise<any[]> =>
    call(() => wails().RemoteService.DiscoverDevices()),
  connectedSources: (): Promise<string[]> =>
    call(() => wails().RemoteService.ConnectedSources()),
  quickConnect: (host: string, port: number, type: string): Promise<any> =>
    call(() => wails().RemoteService.QuickConnect(host, port, type)),
  selectKeyFile: (): Promise<string> =>
    call(() => wails().RemoteService.SelectKeyFile()),
};

// === Updater ===
export const updater = {
  checkUpdate: (): Promise<any> =>
    call(() => wails().UpdaterService.CheckUpdate()),
  doUpdate: (): Promise<any> =>
    call(() => wails().UpdaterService.Update()),
  getStatus: (): Promise<any> =>
    call(() => wails().UpdaterService.UpdateStatus()),
  restart: (): Promise<void> =>
    call(() => wails().UpdaterService.RestartApp()),
};

// === Wails Runtime Events ===
export const events = {
  on: (event: string, callback: (data: any) => void): (() => void) => {
    const rt = runtime();
    if (rt?.EventsOn) {
      return rt.EventsOn(event, callback);
    }
    return () => {};
  },
  emit: (event: string, data: any): void => {
    const rt = runtime();
    if (rt?.EventsEmit) {
      rt.EventsEmit(event, data);
    }
  },
};

// === Media Server ===
export const mediaServer = {
  start: (): Promise<any> =>
    call(() => wails().MediaServer.Start()),
  stop: (): Promise<any> =>
    call(() => wails().MediaServer.Stop()),
  status: (): Promise<any> =>
    call(() => wails().MediaServer.Status()),
  setPort: (port: number): Promise<any> =>
    call(() => wails().MediaServer.SetPort(port)),
  setPin: (pin: string): Promise<any> =>
    call(() => wails().MediaServer.SetPin(pin)),
};

// === DLNA ===
export const dlna = {
  start: (): Promise<any> =>
    call(() => wails().DLNAService.Start()),
  stop: (): Promise<any> =>
    call(() => wails().DLNAService.Stop()),
  status: (): Promise<any> =>
    call(() => wails().DLNAService.Status()),
};

// === Player ===
export const player = {
  playFile: (filePath: string): Promise<any> =>
    call(() => wails().PlayerService.PlayFile(filePath)),
  openFolder: (filePath: string): Promise<any> =>
    call(() => wails().PlayerService.OpenFolder(filePath)),
};

// === Codecs ===
export const codecs = {
  check: (): Promise<any> =>
    call(() => wails().CodecService.CheckCodecs()),
  install: (): Promise<any> =>
    call(() => wails().CodecService.InstallCodecs()),
};

// Export all as a single API object for convenience
export const api = {
  app,
  license,
  settings,
  downloads,
  deps,
  search,
  tmdb,
  stream,
  library,
  torrentDownloads,
  media,
  remote,
  updater,
  events,
  codecs,
  mediaServer,
  dlna,
  player,
};

export default api;
