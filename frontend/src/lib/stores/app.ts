import { writable, derived } from 'svelte/store';

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

// UI Settings
export const uiSettings = writable({
  font: localStorage.getItem('yaria_ui_font') || 'Inter',
  fontSize: localStorage.getItem('yaria_ui_fontsize') || '14',
  scale: localStorage.getItem('yaria_ui_scale') || '100',
  animations: localStorage.getItem('yaria_ui_animations') !== '0',
  blur: localStorage.getItem('yaria_ui_blur') !== '0',
});

// Derived: is mantorex route
export const isMantorexRoute = derived(currentRoute, ($route) =>
  ['/mantorex', '/search', '/detail', '/play', '/library', '/local', '/remote', '/torrent-downloads'].some(r => $route.startsWith(r))
);

// Navigate helper
export function navigate(hash: string) {
  window.location.hash = hash;
}

// Apply UI settings
export function applyUISettings() {
  const root = document.documentElement;
  const settings = {
    font: localStorage.getItem('yaria_ui_font') || 'Inter',
    fontSize: localStorage.getItem('yaria_ui_fontsize') || '14',
    scale: localStorage.getItem('yaria_ui_scale') || '100',
    animations: localStorage.getItem('yaria_ui_animations') !== '0',
    // Blur OFF by default on Linux (WebKitGTK glitches), ON elsewhere (macOS/Windows)
    blur: localStorage.getItem('yaria_ui_blur')
      ? localStorage.getItem('yaria_ui_blur') === '1'
      : !navigator.platform.includes('Linux'),
  };

  root.style.setProperty('--app-font', settings.font + ', sans-serif');
  root.style.setProperty('--app-font-size', settings.fontSize + 'px');

  const fontZoom = parseInt(settings.fontSize) / 14;
  const scaleZoom = parseInt(settings.scale) / 100;
  const totalZoom = fontZoom * scaleZoom;
  document.body.style.zoom = totalZoom !== 1 ? totalZoom.toString() : '';

  root.classList.toggle('no-animations', !settings.animations);
  root.classList.toggle('no-blur', !settings.blur);

  uiSettings.set(settings);
}
