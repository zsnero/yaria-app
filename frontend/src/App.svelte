<script lang="ts">
  import { onMount, type Component } from 'svelte';
  import { currentRoute, routeParams, activeTab, mantorexMode, isPro, proChecked, applyUISettings, loadUISettingsFromDisk } from './lib/stores/app';
  import { api } from './lib/api/wails';
  import { ensureLinuxMediaDefaults } from './lib/utils/torrent';
  import Navbar from './lib/components/Navbar.svelte';
  import Starfield from './lib/components/Starfield.svelte';
  import ProGate from './lib/components/ProGate.svelte';
  import MantorexLegal from './lib/components/MantorexLegal.svelte';
  import Toast from './lib/components/Toast.svelte';
  import { saveScrollPosition, restoreScrollPosition } from './lib/stores/scroll';

  // Free pages (always present)
  import YariaHome from './lib/pages/YariaHome.svelte';
  import YariaDownloads from './lib/pages/YariaDownloads.svelte';
  import Settings from './lib/pages/Settings.svelte';

  // Pro pages/components are closed-source — loaded only when present on disk.
  const proPageGlob = import.meta.glob('./lib/pages/{MantorexHome,SearchResults,Detail,Player,Library,LocalHome,RemoteHome,TorrentDownloads}.svelte');
  const proCompGlob = import.meta.glob('./lib/components/ModeSwitcher.svelte');

  let MantorexHome = $state<Component | null>(null);
  let SearchResults = $state<Component | null>(null);
  let Detail = $state<Component | null>(null);
  let Player = $state<Component | null>(null);
  let Library = $state<Component | null>(null);
  let LocalHome = $state<Component | null>(null);
  let RemoteHome = $state<Component | null>(null);
  let TorrentDownloads = $state<Component | null>(null);
  let ModeSwitcher = $state<Component | null>(null);

  let route = $state('/yaria');
  let params = $state(new URLSearchParams());

  let proStatus = $state(false);
  let proCheckDone = $state(false);
  let mantorexLegalAccepted = $state(false);
  let mantorexLegalChecked = $state(false);

  let isPlayerRoute = $derived(route === '/play');
  const mantorexRoutes = ['/mantorex', '/home', '/local', '/remote', '/search', '/detail', '/library', '/torrent-downloads'];
  let showModeSwitcher = $derived(!isPlayerRoute && mantorexRoutes.some(r => route.startsWith(r)) && !!ModeSwitcher);
  let isProRoute = $derived(mantorexRoutes.some(r => route.startsWith(r)));
  let needsProGate = $derived(isProRoute && proCheckDone && !proStatus);
  // Legal notice only for Torrents (not Local / Remote / Library)
  const torrentLegalRoutes = ['/mantorex', '/home', '/search', '/detail', '/torrent-downloads'];
  let isTorrentLegalRoute = $derived(torrentLegalRoutes.some(r => route.startsWith(r)));
  let needsMantorexLegal = $derived(
    isTorrentLegalRoute && proCheckDone && proStatus && mantorexLegalChecked && !mantorexLegalAccepted
  );

  let playerMounted = $state(false);
  let playerProps = $state<{
    magnet?: string;
    title?: string;
    poster?: string;
    localId?: string;
    filePath?: string;
    remoteStream?: string;
    remoteSourceId?: string;
    remotePath?: string;
  }>({});

  async function loadComponent(glob: Record<string, () => Promise<unknown>>, path: string): Promise<Component | null> {
    const loader = glob[path];
    if (!loader) return null;
    try {
      const mod = await loader() as { default: Component };
      return mod.default;
    } catch {
      return null;
    }
  }

  async function loadProModules() {
    MantorexHome = await loadComponent(proPageGlob, './lib/pages/MantorexHome.svelte');
    SearchResults = await loadComponent(proPageGlob, './lib/pages/SearchResults.svelte');
    Detail = await loadComponent(proPageGlob, './lib/pages/Detail.svelte');
    Player = await loadComponent(proPageGlob, './lib/pages/Player.svelte');
    Library = await loadComponent(proPageGlob, './lib/pages/Library.svelte');
    LocalHome = await loadComponent(proPageGlob, './lib/pages/LocalHome.svelte');
    RemoteHome = await loadComponent(proPageGlob, './lib/pages/RemoteHome.svelte');
    TorrentDownloads = await loadComponent(proPageGlob, './lib/pages/TorrentDownloads.svelte');
    ModeSwitcher = await loadComponent(proCompGlob, './lib/components/ModeSwitcher.svelte');
  }

  /** On first paint only: empty / default hash → user-chosen startup tab */
  function applyStartupTabIfNeeded() {
    const raw = (window.location.hash || '').trim();
    const isDefault =
      raw === '' ||
      raw === '#' ||
      raw === '#/' ||
      raw === '#/yaria' ||
      raw === '#/yaria/';
    if (!isDefault) return;
    let tab = 'yaria';
    try {
      tab = localStorage.getItem('yaria_startup_tab') || 'yaria';
    } catch { /* ignore */ }
    // Prefer disk-backed value when already loaded this session
    // (loadUISettingsFromDisk may have updated localStorage)
    if (tab === 'mantorex') {
      window.location.hash = '#/local';
    } else if (raw === '' || raw === '#' || raw === '#/') {
      window.location.hash = '#/yaria';
    }
  }

  function handleHashChange() {
    const hash = window.location.hash || '#/yaria';
    const [path, queryStr] = hash.substring(1).split('?');

    if (route !== '/play') {
      saveScrollPosition(route);
    }

    route = path;
    params = new URLSearchParams(queryStr || '');

    currentRoute.set(route);
    routeParams.set(params);

    const yariaRoutes = ['/yaria', '/yaria/downloads', '/settings'];
    activeTab.set(yariaRoutes.some(r => route.startsWith(r)) ? 'yaria' : 'mantorex');

    if (route === '/local') mantorexMode.set('local');
    else if (route === '/remote') mantorexMode.set('remote');
    else if (route.startsWith('/mantorex') || route === '/home') mantorexMode.set('torrents');

    if (route === '/play') {
      playerProps = {
        magnet: params.get('magnet') || undefined,
        title: params.get('title') || undefined,
        poster: params.get('poster') || undefined,
        localId: params.get('local') || undefined,
        filePath: params.get('file') || undefined,
        remoteStream: params.get('remoteStream') || undefined,
        remoteSourceId: params.get('remoteSource') || undefined,
        remotePath: params.get('remotePath') || undefined,
      };
      playerMounted = true;
    } else {
      playerMounted = false;
    }

    if (route !== '/play') {
      restoreScrollPosition(route);
    }
  }

  function onPlayerStop() {
    playerMounted = false;
  }

  function onProActivated() {
    proStatus = true;
    isPro.set(true);
    // Fresh trial/key → show legal notice if not yet accepted
    loadMantorexLegal();
  }

  function onMantorexLegalAccepted() {
    mantorexLegalAccepted = true;
  }

  async function loadMantorexLegal() {
    try {
      if (localStorage.getItem('yaria_mantorex_legal_v1') === '1') {
        mantorexLegalAccepted = true;
        return;
      }
    } catch { /* ignore */ }
    try {
      const ui = await api.settings.getUISettings();
      mantorexLegalAccepted = !!ui?.mantorex_legal_accepted;
      if (mantorexLegalAccepted) {
        try { localStorage.setItem('yaria_mantorex_legal_v1', '1'); } catch { /* ignore */ }
      }
      try {
        const backend = ui?.player_backend === 'libmpv' ? 'libmpv' : 'webview';
        localStorage.setItem('yaria_player_backend', backend);
      } catch { /* ignore */ }
      try {
        const av = await api.mpv.available().catch(() => ({ available: false }));
        if (av?.available) localStorage.setItem('yaria_hevc_ok', '1');
        else localStorage.removeItem('yaria_hevc_ok');
      } catch { /* ignore */ }
    } catch {
      mantorexLegalAccepted = false;
    }
  }

  // First-run / background dependency install progress
  let setupVisible = $state(false);
  let setupMessage = $state('');
  let setupPercent = $state(0);
  let setupName = $state('');
  let setupDone = $state(false);
  let setupError = $state('');
  /** Only true after a real download/extract started — prevents "Ready" on every launch */
  let setupHadWork = $state(false);
  let setupCleanups: (() => void)[] = [];

  function markSetupWork(name: string, message: string, percent?: number) {
    setupHadWork = true;
    setupVisible = true;
    setupDone = false;
    setupError = '';
    if (name) setupName = name;
    if (message) setupMessage = message;
    if (percent != null && percent > setupPercent) setupPercent = percent;
  }

  function finishSetup(message?: string) {
    if (!setupHadWork || !setupVisible) {
      setupVisible = false;
      return;
    }
    setupDone = true;
    setupPercent = 100;
    if (message) setupMessage = message;
    setTimeout(() => {
      setupVisible = false;
      setupHadWork = false;
    }, 2000);
  }

  async function runAppSetup() {
    ensureLinuxMediaDefaults();
    applyUISettings();
    await loadUISettingsFromDisk();
    // Cold start: honor preferred startup tab when no real route is in the hash
    applyStartupTabIfNeeded();
    handleHashChange();
    window.addEventListener('hashchange', handleHashChange);

    await loadProModules();

    try {
      const pro = await api.license.isPro();
      proStatus = !!pro;
      isPro.set(proStatus);
    } catch {
      proStatus = false;
    }
    proCheckDone = true;
    proChecked.set(true);
    await loadMantorexLegal();
    mantorexLegalChecked = true;

    // Banner ONLY when real install work happens (never on idle "already ready" launches)
    setupCleanups.push(api.events.on('setup-progress', (data: any) => {
      if (!data) return;
      const phase = String(data.phase || '');
      const msg = String(data.message || '').toLowerCase();
      // Backend stays silent when nothing to install; still ignore soft "already" lines
      if (msg.includes('already') && !setupHadWork) return;
        if (phase === 'install' || phase === 'start') {
          // "start" alone is not enough — require install phase or active download wording
          if (
            phase === 'install' ||
            msg.includes('download') ||
            msg.includes('extract') ||
            msg.includes('setting up missing') ||
            msg.includes('resuming')
          ) {
            markSetupWork(data.name || '', data.message || '', Number(data.percent) || 0);
          }
        }
      if (!setupHadWork) return;
      if (data.message) setupMessage = data.message;
      if (data.name) setupName = data.name;
      if (data.percent != null) setupPercent = Number(data.percent) || setupPercent;
      setupError = data.error || '';
      if (data.done) finishSetup(data.message || 'Setup complete');
    }));
    setupCleanups.push(api.events.on('deps-install-progress', (data: any) => {
      if (!data) return;
      const st = String(data.status || '');
      const msg = String(data.message || '').toLowerCase();
      if (st === 'complete' && (msg.includes('already') || !setupHadWork)) return;
      if (st === 'downloading' || st === 'extracting' || msg.includes('resuming')) {
        markSetupWork(data.name || '', data.message || '', Number(data.percent) || 0);
      } else if (st === 'error' && setupHadWork) {
        const name = String(data.name || '').toLowerCase();
        // libmpv is optional — never pin a red "Setup issue" for it
        if (name.includes('mpv') || msg.includes('libmpv') || msg.includes('native player') || msg.includes('optional')) {
          finishSetup('Native player optional — using WebView');
          return;
        }
        setupVisible = true;
        setupError = data.message || 'Install failed';
        setupMessage = data.message || setupMessage;
      } else if (st === 'complete' && setupHadWork) {
        const doneMsg = String(data.message || '');
        if (doneMsg.toLowerCase().includes('skipped (optional)')) {
          finishSetup('Ready (WebView player)');
        } else {
          finishSetup(doneMsg || 'Installed');
        }
      }
    }));
    setupCleanups.push(api.events.on('deps-progress', (data: any) => {
      if (!data?.message) return;
      const msg = String(data.message);
      const low = msg.toLowerCase();
      if (low.includes('you can install it manually') ||
          low.includes('skipping') ||
          low.includes('not available') ||
          low.includes('warning:') ||
          low.includes('npm not found') ||
          low.includes('webtorrent') ||
          low.includes('already') ||
          low.includes('found yt-dlp') ||
          low.includes('ready')) {
        return;
      }
      // Strict: only active fetch/install lines
      if (!(low.includes('downloading') || low.startsWith('installing') || low.includes('extracting') || low.includes('fetching'))) {
        return;
      }
      markSetupWork('yt-dlp', msg, Math.max(setupPercent, 20));
    }));
    setupCleanups.push(api.events.on('deps-ready', () => {
      // Never open the banner here — only close if we were actually installing
      if (setupHadWork) finishSetup('Download tools ready');
    }));
    setupCleanups.push(api.events.on('deps-error', (data: any) => {
      const err = String(data?.error || '');
      if (err.toLowerCase().includes('webtorrent')) return;
      if (setupHadWork) {
        setupVisible = true;
        setupError = err || 'Setup failed';
      }
    }));

    try {
      await api.deps.ensureAll();
    } catch { /* optional */ }
    try {
      await api.downloads.initDeps();
    } catch { /* optional */ }
  }

  onMount(() => {
    void runAppSetup();
    return () => {
      window.removeEventListener('hashchange', handleHashChange);
      setupCleanups.forEach((fn) => fn());
      setupCleanups = [];
    };
  });
</script>

<Starfield />

{#if !isPlayerRoute}
  <Navbar />
{/if}

{#if setupVisible && !isPlayerRoute}
  <div class="setup-banner" class:done={setupDone} class:error={!!setupError}>
    <div class="setup-banner-inner">
      {#if !setupDone && !setupError}
        <div class="setup-spinner"></div>
      {/if}
      <div class="setup-text">
        <div class="setup-title">
          {#if setupError}
            Setup issue
          {:else if setupDone}
            Ready
          {:else}
            Setting up Yaria…
          {/if}
        </div>
        <div class="setup-msg">
          {#if setupError}
            {setupError}
          {:else if setupName}
            {setupName}: {setupMessage}
          {:else}
            {setupMessage || 'Preparing dependencies…'}
          {/if}
        </div>
      </div>
      {#if !setupDone}
        <div class="setup-pct">{setupPercent}%</div>
      {/if}
      <button type="button" class="setup-dismiss" onclick={() => { setupVisible = false; }}>×</button>
    </div>
    {#if !setupDone && !setupError}
      <div class="setup-bar"><div class="setup-bar-fill" style="width:{setupPercent}%"></div></div>
    {/if}
  </div>
{/if}

{#if playerMounted && Player}
  <div class="player-mount" class:hidden={!isPlayerRoute}>
    <Player
      magnet={playerProps.magnet}
      title={playerProps.title}
      poster={playerProps.poster}
      localId={playerProps.localId}
      filePath={playerProps.filePath}
      remoteStream={playerProps.remoteStream}
      remoteSourceId={playerProps.remoteSourceId}
      remotePath={playerProps.remotePath}
      onStop={onPlayerStop}
    />
  </div>
{/if}

<main id="app-content" class:player-active={isPlayerRoute}>
  {#if showModeSwitcher && ModeSwitcher}
    <ModeSwitcher />
  {/if}
  {#if !isPlayerRoute}
    {#if route === '/yaria' || route === '/yaria/'}
      <YariaHome />
    {:else if route === '/yaria/downloads'}
      <YariaDownloads />
    {:else if route === '/settings'}
      <Settings />
    {:else if route === '/mantorex' || route === '/mantorex/' || route === '/home'}
      {#if needsProGate}
        <ProGate onActivated={onProActivated} />
      {:else if needsMantorexLegal}
        <MantorexLegal onAccepted={onMantorexLegalAccepted} />
      {:else if MantorexHome}
        <MantorexHome />
      {:else}
        <div class="loading-screen"><p class="dim">Mantorex is not available in this build.</p></div>
      {/if}
    {:else if route === '/search'}
      {#if needsProGate}
        <ProGate onActivated={onProActivated} />
      {:else if needsMantorexLegal}
        <MantorexLegal onAccepted={onMantorexLegalAccepted} />
      {:else if SearchResults}
        <SearchResults query={params.get('q') || ''} />
      {/if}
    {:else if route === '/detail'}
      {#if needsProGate}
        <ProGate onActivated={onProActivated} />
      {:else if needsMantorexLegal}
        <MantorexLegal onAccepted={onMantorexLegalAccepted} />
      {:else if Detail}
        <Detail
          title={params.get('title') || ''}
          year={params.get('year') || ''}
          mediaType={params.get('type') || 'movie'}
          poster={params.get('poster') || ''}
          cast={params.get('cast') || ''}
          id={params.get('id') || ''}
        />
      {/if}
    {:else if route === '/library'}
      {#if needsProGate}
        <ProGate onActivated={onProActivated} />
      {:else if Library}
        <Library />
      {/if}
    {:else if route === '/local'}
      {#if needsProGate}
        <ProGate onActivated={onProActivated} />
      {:else if LocalHome}
        <LocalHome />
      {/if}
    {:else if route === '/remote'}
      {#if needsProGate}
        <ProGate onActivated={onProActivated} />
      {:else if RemoteHome}
        <RemoteHome />
      {/if}
    {:else if route === '/torrent-downloads'}
      {#if needsProGate}
        <ProGate onActivated={onProActivated} />
      {:else if needsMantorexLegal}
        <MantorexLegal onAccepted={onMantorexLegalAccepted} />
      {:else if TorrentDownloads}
        <TorrentDownloads />
      {/if}
    {:else}
      <div class="loading-screen">
        <p class="dim">Page not found: {route}</p>
      </div>
    {/if}
  {/if}
</main>

<Toast />

<style lang="scss">
  @use './lib/styles/variables' as *;

  #app-content {
    min-height: calc(100vh - $nav-h);
    padding-top: $nav-h;

    &.player-active {
      display: none;
    }
  }

  .page-wrapper {
    min-height: calc(100vh - $nav-h);
  }

  .player-mount.hidden {
    position: fixed;
    top: 0;
    left: 0;
    width: 0;
    height: 0;
    overflow: hidden;
    pointer-events: none;
    visibility: hidden;
  }

  .loading-screen {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 40vh;
  }

  .dim {
    color: $text-dim;
  }

  .setup-banner {
    position: fixed;
    top: calc(#{$nav-h} + 12px);
    right: 16px;
    z-index: 200;
    width: min(380px, calc(100vw - 32px));
    background: rgba(12, 12, 24, 0.92);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.45);
    backdrop-filter: blur(16px);
    overflow: hidden;
    animation: setupIn 0.25s ease;

    &.done {
      border-color: rgba(52, 211, 153, 0.35);
    }
    &.error {
      border-color: rgba(248, 113, 113, 0.4);
    }
  }

  .setup-banner-inner {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 12px 14px 10px;
  }

  .setup-spinner {
    width: 16px;
    height: 16px;
    margin-top: 2px;
    border: 2px solid rgba(255, 255, 255, 0.15);
    border-top-color: $accent;
    border-radius: 50%;
    /* use global yaria-spin so no-animations exceptions keep it moving */
    animation: yaria-spin 0.75s linear infinite;
    flex-shrink: 0;
  }

  .setup-text {
    flex: 1;
    min-width: 0;
  }

  .setup-title {
    font-size: 13px;
    font-weight: 600;
    color: $text;
  }

  .setup-msg {
    font-size: 12px;
    color: $text-muted;
    margin-top: 2px;
    line-height: 1.4;
    word-break: break-word;
  }

  .setup-pct {
    font-size: 11px;
    color: $text-dim;
    font-variant-numeric: tabular-nums;
    flex-shrink: 0;
  }

  .setup-dismiss {
    background: none;
    border: none;
    color: $text-muted;
    font-size: 18px;
    line-height: 1;
    cursor: pointer;
    padding: 0 2px;
    flex-shrink: 0;
    &:hover { color: $text; }
  }

  .setup-bar {
    height: 3px;
    background: rgba(255, 255, 255, 0.06);
  }

  .setup-bar-fill {
    height: 100%;
    background: $accent;
    transition: width 0.25s ease;
  }

  @keyframes setupIn {
    from { opacity: 0; transform: translateY(-6px); }
    to { opacity: 1; transform: translateY(0); }
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
