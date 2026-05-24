// Core application types

export interface TorrentResult {
  title: string;
  source: string;
  seeders: number;
  leechers: number;
  size: string;
  magnet: string;
}

export interface MetaResult {
  id: number;
  title: string;
  name?: string;
  year?: string;
  type: 'movie' | 'tv';
  poster?: string;
  poster_hq?: string;
  backdrop?: string;
  rating?: number;
  overview?: string;
  cast?: string;
  genres?: string;
  media_type?: string;
}

export interface LibraryItem {
  id: string;
  tmdb_id?: number;
  title: string;
  year?: string;
  type: string;
  poster?: string;
  poster_hq?: string;
  backdrop?: string;
  rating?: number;
  overview?: string;
  genres?: string[];
  watch_progress?: WatchProgress;
  last_watched?: string;
}

export interface WatchProgress {
  time_seconds: number;
  duration_seconds: number;
  season?: number;
  episode?: number;
}

export interface LocalMedia {
  id: string;
  title: string;
  name?: string;
  path: string;
  poster?: string;
  thumbnail?: string;
  year?: string;
  rating?: number;
  resolution?: string;
  duration?: number;
  resume_pos?: number;
  watched?: boolean;
  codec?: string;
  audio_codec?: string;
  category?: string;
}

export interface TVShowGroup {
  title: string;
  poster?: string;
  year?: string;
  rating?: number;
  tmdb_id?: number;
  episodes: LocalMedia[];
}

export interface RemoteSource {
  id: string;
  name: string;
  host: string;
  port: number;
  type: 'ssh' | 'smb' | 'ftp' | 'http';
  username?: string;
  base_path?: string;
}

export interface RemoteFile {
  name: string;
  path: string;
  is_dir: boolean;
  size: number;
  size_str?: string;
  is_video?: boolean;
}

export interface StreamStatus {
  state: string;
  active: boolean;
  playable?: boolean;
  buffer_pct?: number;
  stream_id?: number;
  percent?: number;
  downloaded?: string;
  total?: string;
  peers?: number;
  file_name?: string;
  error?: string;
}

export interface DepInfo {
  name: string;
  desc?: string;
  installed: boolean;
  path?: string;
  message?: string;
}

export interface SubtitleTrack {
  name: string;
  lang: string;
  url: string;
  source?: string;
}

export type PlayerType = 'torrent' | 'local' | 'remote';
export type PlaybackMode = 'direct' | 'hls' | 'vod';

export interface PrepareStreamResult {
  mode: PlaybackMode;
  ready?: boolean;
  duration?: number;
  error?: string;
}
