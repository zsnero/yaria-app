<script lang="ts">
  import { onMount, type Component } from 'svelte';
  import { currentRoute, routeParams, activeTab, mantorexMode, isPro, proChecked, applyUISettings, loadUISettingsFromDisk } from './lib/stores/app';
  import { api } from './lib/api/wails';
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
    } catch {
      mantorexLegalAccepted = false;
    }
  }

  let proStatus = $state(false);
  let proCheckDone = $state(false);
  let mantorexLegalAccepted = $state(false);
  let mantorexLegalChecked = $state(false);

  // First-run / background dependency install progress
  let setupVisible = $state(false);
  let setupMessage = $state('');
  let setupPercent = $state(0);
  let setupName = $state('');
  let setupDone = $state(false);
  let setupError = $state('');
  let setupCleanups: (() => void)[] = [];

  onMount(async () => {
    handleHashChange();
    applyUISettings();
    await loadUISettingsFromDisk();
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

    // Only show banner when something is actually installing (not on every launch)
    setupCleanups.push(api.events.on('setup-progress', (data: any) => {
      if (!data) return;
      // Ignore silent no-op completions
      if (data.done && !setupVisible) return;
      if (data.phase === 'start' || data.phase === 'install') {
        setupVisible = true;
      }
      if (!setupVisible && !data.done) setupVisible = true;
      setupMessage = data.message || '';
      setupPercent = Number(data.percent) || 0;
      setupName = data.name || '';
      setupDone = !!data.done;
      setupError = data.error || '';
      if (data.done && setupVisible) {
        setTimeout(() => { setupVisible = false; }, 2200);
      }
    }));
    setupCleanups.push(api.events.on('deps-install-progress', (data: any) => {
      if (!data) return;
      // Only surface real download/extract activity
      const st = data.status || '';
      if (st === 'complete' && !setupVisible) return;
      if (st === 'downloading' || st === 'extracting' || st === 'error') {
        setupVisible = true;
      }
      if (!setupVisible) return;
      setupName = data.name || setupName;
      setupMessage = data.message || setupMessage;
      if (data.percent != null) setupPercent = Number(data.percent) || setupPercent;
      if (st === 'error') setupError = data.message || 'Install failed';
      if (st === 'complete') {
        setupMessage = data.message || 'Installed';
        setupPercent = 100;
        setTimeout(() => { setupVisible = false; }, 2200);
      }
    }));
    // yt-dlp first-time lines only (ignore if nothing is installing)
    setupCleanups.push(api.events.on('deps-progress', (data: any) => {
      if (!data?.message || setupDone) return;
      const msg = String(data.message).toLowerCase();
      // Only show when actually downloading/installing tools
      if (!msg.includes('download') && !msg.includes('install') && !msg.includes('extract') && !msg.includes('fetch')) {
        return;
      }
      setupVisible = true;
      setupName = 'yt-dlp';
      setupMessage = data.message;
    }));

    try {
      await api.deps.ensureAll();
    } catch { /* optional */ }
    try {
      await api.downloads.initDeps();
    } catch { /* optional */ }

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
    animation: spin 0.7s linear infinite;
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
