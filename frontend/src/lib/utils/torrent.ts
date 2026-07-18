import type { TorrentResult } from '../types';

// Check if we're on Linux
export const isLinux = typeof navigator !== 'undefined' && navigator.platform.indexOf('Linux') !== -1;

// Filter out HEVC/x265/10-bit torrents on Linux (unless user opted out)
const unsupportedRE = /\b(hevc|[hx]\.?265|10[.\-\s]?bit)\b/i;
export function filterPlayableTorrents(torrents: TorrentResult[]): TorrentResult[] {
  if (!isLinux || !torrents || !Array.isArray(torrents)) return torrents;
  if (localStorage.getItem('yaria_show_all_formats') === '1') return torrents;
  return torrents.filter(t => !unsupportedRE.test(t.title || ''));
}

// Parse size string to bytes (e.g., "1.5 GB" -> 1610612736)
export function parseSizeBytes(sizeStr: string): number {
  if (!sizeStr) return 0;
  const s = sizeStr.toUpperCase();
  const num = parseFloat(s);
  if (isNaN(num)) return 0;
  if (s.includes('TB')) return num * 1024 * 1024 * 1024 * 1024;
  if (s.includes('GB')) return num * 1024 * 1024 * 1024;
  if (s.includes('MB')) return num * 1024 * 1024;
  if (s.includes('KB') || s.includes('KIB')) return num * 1024;
  return num;
}

// Extract season number from torrent title
export function extractSeason(title: string): string {
  if (!title) return '';
  const t = title.toUpperCase();

  // Complete / Batch
  if (/\b(COMPLETE|BATCH)\b/.test(t)) return 'Complete';

  // S01 format
  let m = t.match(/\bS(\d{1,2})/);
  if (m) return m[1].replace(/^0+/, '') || '0';

  // "Season 1" / "Season 01"
  m = t.match(/\bSEASON[\s._-]*(\d{1,2})\b/);
  if (m) return m[1].replace(/^0+/, '') || '0';

  // "2nd Season", "3rd Season"
  m = t.match(/\b(\d{1,2})(ST|ND|RD|TH)\s*SEASON\b/);
  if (m) return m[1];

  // "Part 1", "Cour 1"
  m = t.match(/\b(?:PART|COUR)[\s._-]*(\d{1,2})\b/);
  if (m) return m[1];

  return '';
}

// Extract episode number from torrent title
export function extractEpisode(title: string): string {
  if (!title) return '';
  const t = title;

  // S01E02
  let m = t.match(/[Ss]\d{1,2}[Ee](\d{1,3})/);
  if (m) return m[1].replace(/^0+/, '') || '0';

  // EP01, Ep 01
  m = t.match(/\b[Ee][Pp][\s._-]*(\d{1,3})\b/);
  if (m) return m[1].replace(/^0+/, '') || '0';

  // Anime: " - 01 " pattern (space-dash-space-number-space)
  m = t.match(/\s-\s(\d{2,3})\s/);
  if (m) return m[1].replace(/^0+/, '') || '0';

  return '';
}

// Extract quality tag from torrent title
export function extractQuality(title: string): string {
  if (!title) return '';
  const t = title.toUpperCase();
  if (/\b(2160P|4K|UHD)\b/.test(t)) return '4K';
  if (/\b1080P\b/.test(t)) return '1080p';
  if (/\b720P\b/.test(t)) return '720p';
  if (/\b480P\b/.test(t)) return '480p';
  if (/\bBLU[\s-]?RAY\b/.test(t)) return 'BluRay';
  if (/\bWEB[\s-]?DL\b/.test(t)) return 'WEB-DL';
  if (/\bWEBRIP\b/.test(t)) return 'WEBRip';
  if (/\bWEB\b/.test(t)) return 'WEB';
  if (/\bHDTV\b/.test(t)) return 'HDTV';
  if (/\bCAM\b/.test(t)) return 'CAM';
  return '';
}

// Pick the best torrent for Smart Play
export function pickBestTorrent(list: TorrentResult[], searchTitle: string): TorrentResult | null {
  if (!list || list.length === 0) return null;

  const yearMatch = (searchTitle || '').match(/\b((?:19|20)\d{2})\b/);
  const wantYear = yearMatch ? yearMatch[1] : '';
  const normSearch = (searchTitle || '')
    .toLowerCase()
    .replace(/\b(?:19|20)\d{2}\b/g, ' ')
    .replace(/[._\-:]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
  const stopWords = ['the', 'a', 'an', 'of', 'and', 'in', 'on', 'at', 'to', 'for'];
  const searchTokens = normSearch.split(' ').filter(w => w.length > 1 && !stopWords.includes(w));

  function checkRelevance(torrentTitle: string): boolean {
    if (!normSearch || searchTokens.length === 0) return true;
    const norm = (torrentTitle || '').toLowerCase().replace(/[._\-:]/g, ' ').replace(/\s+/g, ' ').trim();
    // Reject wrong-year remakes / shared titles
    if (wantYear) {
      const years = [...(torrentTitle || '').matchAll(/\b((?:19|20)\d{2})\b/g)].map(m => m[1]);
      if (years.length > 0 && !years.includes(wantYear)) return false;
    }
    let matched = 0;
    for (const tok of searchTokens) {
      if (norm.includes(tok)) matched++;
    }
    return matched >= Math.ceil(searchTokens.length * 0.8);
  }

  function scoreTorrent(t: TorrentResult): number {
    const titleUp = (t.title || '').toUpperCase();
    const sizeBytes = parseSizeBytes(t.size);
    const sizeGB = sizeBytes / (1024 * 1024 * 1024);
    let score = 0;

    // Title relevance (critical, can disqualify)
    if (!checkRelevance(t.title)) return -1;

    // Hard disqualifiers
    const isHEVC = /\b(HEVC|H\.?265|X\.?265|10[\.\-\s]?BIT)\b/.test(titleUp);
    if (isLinux && isHEVC) return -1;
    if (/\b(SAMPLE|TRAILER|PASSWORD|RAR|ZIP)\b/.test(titleUp)) return -1;
    if (/\b(CAM|HDCAM|TELECINE|TELESYNC)\b/.test(titleUp) && !/\bWEB/.test(titleUp)) return -1;
    if (sizeGB > 0 && sizeGB < 0.3) return -1;

    // PRIMARY METRIC: Startup Speed Score (0-40 pts)
    if (t.seeders > 0 && sizeGB > 0) {
      const startupSpeed = t.seeders / sizeGB;
      score += Math.min(40, Math.log2(startupSpeed + 1) * 5);
    } else if (t.seeders > 0) {
      score += Math.min(25, Math.log2(t.seeders + 1) * 4);
    }

    // QUALITY (0-20 pts)
    if (titleUp.includes('1080P')) score += 20;
    else if (titleUp.includes('720P')) score += 18;
    else if (titleUp.includes('2160P') || titleUp.includes('4K')) score += 8;
    else if (titleUp.includes('BLURAY') || titleUp.includes('BLU-RAY')) score += 15;
    else if (titleUp.includes('WEBDL') || titleUp.includes('WEB-DL') || titleUp.includes('WEBRIP')) score += 17;
    else if (titleUp.includes('HDTV')) score += 10;
    else if (titleUp.includes('480P')) score += 5;
    else score += 8;

    // CODEC (0-10 pts)
    if (/\b(H\.?264|X\.?264|AVC)\b/.test(titleUp)) score += 10;

    // SEED HEALTH (0-8 pts)
    const ratio = t.seeders / Math.max(1, t.leechers);
    score += Math.min(8, ratio * 1.5);

    // SIZE SWEET SPOT bonus (0-5 pts)
    if (sizeGB >= 1 && sizeGB <= 3) score += 5;
    else if (sizeGB > 3 && sizeGB <= 5) score += 2;

    // PENALTIES
    if (/\b(DUAL|MULTi)\b/.test(titleUp)) score -= 15;
    if (/\b(FRENCH|SPANISH|GERMAN|ITALIAN|RUSSIAN|PORTUGUESE|HINDI|TAMIL|TELUGU|KOREAN|JAPANESE|ARABIC|TURKISH|POLISH|DUTCH|CHINESE|LATINO|CASTELLANO)\b/.test(titleUp)) {
      if (!/\bENG(LISH)?\b/.test(titleUp)) score -= 25;
    }

    // Container
    if (titleUp.includes('.MP4')) score += 3;
    if (titleUp.includes('.MKV')) score -= 2;

    return score;
  }

  let best: TorrentResult | null = null;
  let bestScore = -Infinity;

  for (const t of list) {
    const s = scoreTorrent(t);
    if (s > bestScore) {
      bestScore = s;
      best = t;
    }
  }

  return best;
}

// Format duration seconds to "Xh Xm" or "Xm"
export function fmtDuration(seconds: number): string {
  if (!seconds || seconds <= 0) return '';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

// NO_POSTER placeholder SVG
export const NO_POSTER = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='150' height='225'%3E%3Crect fill='%231a1a2e' width='150' height='225' rx='8'/%3E%3Ctext fill='%23555570' x='75' y='112' text-anchor='middle' font-size='12' font-family='sans-serif'%3ENo Poster%3C/text%3E%3C/svg%3E";

// Strip release tags from torrent/file names for TMDB lookup & library display.
// "Filth.2013.Triple.BDRip.1080p..." → { title: "Filth", year: "2013" }
export function cleanReleaseTitle(raw: string): { title: string; year: string } {
  if (!raw) return { title: '', year: '' };
  let s = raw
    .replace(/[\[\](){}]/g, ' ')
    .replace(/[._]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();

  const yearMatch = s.match(/\b((?:19|20)\d{2})\b/);
  const year = yearMatch ? yearMatch[1] : '';

  // For release-style names with a year mid-string, title is usually before the year
  // e.g. "Filth 2013 Triple BDRip 1080p" → "Filth"
  if (yearMatch && yearMatch.index !== undefined && yearMatch.index > 0) {
    const before = s.slice(0, yearMatch.index).trim();
    if (before.length >= 2) {
      s = before;
    }
  }

  // Cut at first quality / codec / source token
  const cutRe =
    /\b(?:\d{3,4}p|4k|uhd|hdr|dv|sdr|x264|x265|h264|h265|hevc|avc|av1|bluray|blu[\s-]?ray|bdrip|brrip|webrip|web[\s-]?dl|webdl|web|hdtv|dvdrip|remux|proper|repack|internal|cam|telesync|ts|hdcam|yts|yify|rarbg|sparks|amiable|geckos|ctrlhd|flux|ntb|eztv|ettv|mkv|mp4|avi|multi|dual|audio|subs?|extended|unrated|directors?|cut|truehd|dts|atmos|aac|ac3|dd5|ddpa?|ma|hdma|10bit|8bit|complete|pack|season|s\d{1,2}e\d{1,3}|s\d{1,2}|triple|remastered)\b/i;
  const cut = s.search(cutRe);
  if (cut > 0) s = s.slice(0, cut).trim();

  // Remove year from title portion if still present
  if (year) {
    s = s.replace(new RegExp(`\\b${year}\\b`), ' ').replace(/\s+/g, ' ').trim();
  }

  // Drop trailing junk like " - GROUP"
  s = s.replace(/\s+-\s+[A-Za-z0-9]+$/, '').trim();

  return { title: s || raw, year };
}
