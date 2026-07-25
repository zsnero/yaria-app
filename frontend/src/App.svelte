<script lang="ts">
  import { onMount, type Component } from 'svelte';
  import { currentRoute, routeParams, activeTab, mantorexMode, isPro, proChecked, applyUISettings, loadUISettingsFromDisk } from './lib/stores/app';
  import { api } from './lib/api/wails';
  import Navbar from './lib/components/Navbar.svelte';
  import Starfield from './lib/components/Starfield.svelte';
  import ProGate from './lib/components/ProGate.svelte';
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
  }

  let proStatus = $state(false);
  let proCheckDone = $state(false);

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

    return () => window.removeEventListener('hashchange', handleHashChange);
  });
</script>

<Starfield />

{#if !isPlayerRoute}
  <Navbar />
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
      {:else if MantorexHome}
        <MantorexHome />
      {:else}
        <div class="loading-screen"><p class="dim">Mantorex is not available in this build.</p></div>
      {/if}
    {:else if route === '/search'}
      {#if needsProGate}
        <ProGate onActivated={onProActivated} />
      {:else if SearchResults}
        <SearchResults query={params.get('q') || ''} />
      {/if}
    {:else if route === '/detail'}
      {#if needsProGate}
        <ProGate onActivated={onProActivated} />
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
</style>
