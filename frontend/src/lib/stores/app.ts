import { writable, derived } from 'svelte/store';
import { api } from '../api/wails';

// Active tab: 'yaria' or 'mantorex'
export const activeTab = writable<'yaria' | 'mantorex'>('yaria');

// Current route
export const currentRoute = writable<string>('/yaria');
export const routeParams = writable<URLSearchParams>(new URLSearchParams());

// Pro license status
export const isPro = writable<boolean>(false);
export const proChecked = writable<boolean>(false);

// Player state
export const playerActive = writable<boolean>(false);
export const currentStreamID = writable<number | null>(null);
export const currentMagnet = writable<string | null>(null);

// Mantorex mode: 'local' | 'remote' | 'torrents'
export const mantorexMode = writable<'local' | 'remote' | 'torrents'>('torrents');

function defaultBlur(): boolean {
  // Blur OFF by default on Linux (WebKitGTK glitches), ON elsewhere
  return !navigator.platform.includes('Linux');
}

function readLocalUI() {
  return {
    font: localStorage.getItem('yaria_ui_font') || 'Inter',
    fontSize: localStorage.getItem('yaria_ui_fontsize') || '14',
    scale: localStorage.getItem('yaria_ui_scale') || '100',
    animations: localStorage.getItem('yaria_ui_animations') !== '0',
    blur: localStorage.getItem('yaria_ui_blur')
      ? localStorage.getItem('yaria_ui_blur') === '1'
      : defaultBlur(),
  };
}

// UI Settings — start from localStorage for instant paint, then hydrate from disk
export const uiSettings = writable(readLocalUI());

// Derived: is mantorex route
export const isMantorexRoute = derived(currentRoute, ($route) =>
  ['/mantorex', '/search', '/detail', '/play', '/library', '/local', '/remote', '/torrent-downloads'].some(r => $route.startsWith(r))
);

// Navigate helper
export function navigate(hash: string) {
  window.location.hash = hash;
}

function applyToDOM(settings: {
  font: string;
  fontSize: string;
  scale: string;
  animations: boolean;
  blur: boolean;
}) {
  const root = document.documentElement;

  root.style.setProperty('--app-font', settings.font + ', sans-serif');
  root.style.setProperty('--app-font-size', settings.fontSize + 'px');

  const fontZoom = parseInt(settings.fontSize) / 14;
  const scaleZoom = parseInt(settings.scale) / 100;
  const totalZoom = fontZoom * scaleZoom;
  // Prefer CSS zoom only when needed. body.zoom breaks CSS animations in WebKitGTK;
  // use a transform scale on #app when available, fall back to zoom.
  const appRoot = document.getElementById('app') || document.getElementById('app-root');
  if (appRoot) {
    document.body.style.zoom = '';
    if (totalZoom !== 1) {
      appRoot.style.transform = `scale(${totalZoom})`;
      appRoot.style.transformOrigin = 'top left';
      appRoot.style.width = `${100 / totalZoom}%`;
      appRoot.style.height = `${100 / totalZoom}%`;
    } else {
      appRoot.style.transform = '';
      appRoot.style.transformOrigin = '';
      appRoot.style.width = '';
      appRoot.style.height = '';
    }
  } else {
    document.body.style.zoom = totalZoom !== 1 ? totalZoom.toString() : '';
  }

  root.classList.toggle('no-animations', !settings.animations);
  root.classList.toggle('no-blur', !settings.blur);

  uiSettings.set(settings);
}

function writeLocalUI(settings: {
  font: string;
  fontSize: string;
  scale: string;
  animations: boolean;
  blur: boolean;
}) {
  localStorage.setItem('yaria_ui_font', settings.font);
  localStorage.setItem('yaria_ui_fontsize', settings.fontSize);
  localStorage.setItem('yaria_ui_scale', settings.scale);
  localStorage.setItem('yaria_ui_animations', settings.animations ? '1' : '0');
  localStorage.setItem('yaria_ui_blur', settings.blur ? '1' : '0');
}

// Apply UI settings from local cache (sync, for first paint)
export function applyUISettings() {
  applyToDOM(readLocalUI());
}

// Load UI settings from disk-backed config (survives rebuilds / WebView resets)
export async function loadUISettingsFromDisk(): Promise<void> {
  try {
    const res = await api.settings.getUISettings() as any;
    if (!res || res.error) return;

    // First run after upgrade: migrate localStorage → disk
    if (!res.configured) {
      const local = readLocalUI();
      writeLocalUI(local);
      applyToDOM(local);
      try {
        await api.settings.saveUISettings({
          font: local.font,
          font_size: local.fontSize,
          scale: local.scale,
          animations: local.animations,
          blur: local.blur,
        });
      } catch { /* ignore */ }
      return;
    }

    const settings = {
      font: res.font || 'Inter',
      fontSize: String(res.font_size || '14'),
      scale: String(res.scale || '100'),
      animations: res.animations !== false,
      blur: res.blur_set ? !!res.blur : defaultBlur(),
    };

    writeLocalUI(settings);
    applyToDOM(settings);
  } catch {
    // Backend not ready — keep localStorage values
  }
}

// Persist UI settings to disk + localStorage and apply immediately
export async function saveUISettings(partial: {
  font?: string;
  fontSize?: string;
  scale?: string;
  animations?: boolean;
  blur?: boolean;
}): Promise<void> {
  const current = readLocalUI();
  const next = {
    font: partial.font ?? current.font,
    fontSize: partial.fontSize ?? current.fontSize,
    scale: partial.scale ?? current.scale,
    animations: partial.animations ?? current.animations,
    blur: partial.blur ?? current.blur,
  };

  writeLocalUI(next);
  applyToDOM(next);

  try {
    await api.settings.saveUISettings({
      font: next.font,
      font_size: next.fontSize,
      scale: next.scale,
      animations: next.animations,
      blur: next.blur,
    });
  } catch {
    // Disk save failed — localStorage still has the value for this session
  }
}
