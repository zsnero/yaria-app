<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../../api/wails';
  import { isPro, saveUISettings, loadUISettingsFromDisk } from '../../stores/app';
  import { toastSuccess, toastError } from '../../stores/toast';
  import { autoFocus } from '../../actions/index';
  import Spinner from '../../components/Spinner.svelte';
  import ConfirmDialog from '../../components/ConfirmDialog.svelte';
  import AppSelect from '../../components/AppSelect.svelte';
  import FilePicker from '../../components/FilePicker.svelte';
  import { ensureLinuxMediaDefaults } from '../../utils/torrent';

  // License
  let licenseLoading = $state(true);
  let licenseInfo = $state<any>(null);
  let deviceInfo = $state<any>(null);
  let licenseKey = $state('');
  let licenseError = $state('');
  let licenseSuccess = $state('');
  let activating = $state(false);

  // TMDB
  let tmdbKey = $state('');
  let tmdbError = $state('');
  let tmdbUsingDefault = $state(false);
  let tmdbDebounce: ReturnType<typeof setTimeout> | null = null;

  // Proxy
  let proxyType = $state('none');
  let proxyAddr = $state('');
  let proxyDebounce: ReturnType<typeof setTimeout> | null = null;

  // Format filter (Linux only) — default OFF (show HEVC) after linux media defaults
  const isLinux = navigator.platform.indexOf('Linux') !== -1;
  if (isLinux) ensureLinuxMediaDefaults();
  let formatFilterEnabled = $state(localStorage.getItem('yaria_show_all_formats') !== '1');

  // Appearance
  let uiFont = $state(localStorage.getItem('yaria_ui_font') || 'Roboto');
  let uiFontSize = $state(localStorage.getItem('yaria_ui_fontsize') || '14');
  let uiScale = $state(localStorage.getItem('yaria_ui_scale') || '100');
  let uiAnimations = $state(localStorage.getItem('yaria_ui_animations') === '1');
  let uiBlur = $state(
    localStorage.getItem('yaria_ui_blur')
      ? localStorage.getItem('yaria_ui_blur') === '1'
      : !navigator.platform.includes('Linux')
  );

  // Video player backend (applies to local / torrents / remote)
  let playerBackend = $state<'webview' | 'libmpv'>('webview');
  let mpvAvailable = $state(false);
  let mpvReason = $state('');
  let playerBackendSaving = $state(false);
  let playerBackendMsg = $state('');
  // Native mpv tuning (defaults match previous hardcoded behavior)
  let playerHwdec = $state('auto-safe');
  let playerCache = $state('normal');
  let playerHqScale = $state(false);
  let playerDeinterlace = $state(false);
  let playerLoadUserConfig = $state(false);
  let playerOptsSaving = $state(false);

  // App start tab
  let startupTab = $state<'yaria' | 'mantorex'>(
    localStorage.getItem('yaria_startup_tab') === 'mantorex' ? 'mantorex' : 'yaria'
  );
  let startupTabSaving = $state(false);

  // Font options
  const fontOptions = [
    { value: 'Roboto', label: 'Roboto (Default)' },
    { value: 'Inter', label: 'Inter' },
    { value: 'system-ui', label: 'System Default' },
    { value: "'SF Pro Display', -apple-system, BlinkMacSystemFont", label: 'SF Pro (macOS)' },
    { value: "'Segoe UI'", label: 'Segoe UI (Windows)' },
    { value: "'JetBrains Mono', monospace", label: 'JetBrains Mono' },
    { value: "'Fira Code', monospace", label: 'Fira Code' },
    { value: 'monospace', label: 'Monospace' },
  ];

  const fontSizeOptions = [
    { value: '12', label: 'Small (12px)' },
    { value: '13', label: 'Compact (13px)' },
    { value: '14', label: 'Default (14px)' },
    { value: '15', label: 'Medium (15px)' },
    { value: '16', label: 'Large (16px)' },
    { value: '18', label: 'Extra Large (18px)' },
  ];

  // Confirm dialog
  let showConfirm = $state(false);
  let confirmMessage = $state('');
  let confirmCallback = $state<(() => void) | null>(null);

  // Library backup
  let backupBusy = $state(false);
  let backupMsg = $state('');
  let backupErr = $state('');
  let showExportPicker = $state(false);
  let showImportPicker = $state(false);
  let exportDefaultName = $state('yaria-library-backup.json');

  // --- Handlers ---
  function handleTMDBInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    tmdbKey = val;
    tmdbError = '';
    tmdbUsingDefault = false;
    if (tmdbDebounce) clearTimeout(tmdbDebounce);
    tmdbDebounce = setTimeout(async () => {
      try {
        await api.settings.saveTMDBKey(val.trim());
        // Empty field → falls back to built-in key
        if (!val.trim()) {
          const info = await api.settings.getTMDBKey();
          tmdbUsingDefault = !!info?.using_default;
        }
        toastSuccess('Settings saved');
      } catch (err: any) {
        toastError('Failed to save: ' + (err.message || ''));
      }
    }, 800);
  }

  async function handleProxyTypeChange() {
    try {
      await api.settings.saveProxy(proxyType, proxyAddr);
      toastSuccess('Settings saved');
    } catch (err: any) {
      toastError('Failed to save: ' + (err.message || ''));
    }
  }

  function handleProxyAddrInput() {
    if (proxyDebounce) clearTimeout(proxyDebounce);
    proxyDebounce = setTimeout(async () => {
      try {
        await api.settings.saveProxy(proxyType, proxyAddr.trim());
        toastSuccess('Settings saved');
      } catch (err: any) {
        toastError('Failed to save: ' + (err.message || ''));
      }
    }, 800);
  }

  function handleFormatFilter() {
    if (formatFilterEnabled) {
      localStorage.removeItem('yaria_show_all_formats');
    } else {
      localStorage.setItem('yaria_show_all_formats', '1');
    }
  }

  function handleFontChange() {
    saveUISettings({ font: uiFont });
  }

  function handleFontSizeChange() {
    saveUISettings({ fontSize: uiFontSize });
  }

  function handleScaleChange() {
    saveUISettings({ scale: uiScale });
  }

  function handleAnimationsChange() {
    saveUISettings({ animations: uiAnimations });
  }

  function handleBlurChange() {
    saveUISettings({ blur: uiBlur });
  }

  async function loadPlayerBackend() {
    try {
      const [ui, av] = await Promise.all([
        api.settings.getUISettings(),
        api.mpv.available().catch(() => ({ available: false, reason: 'MpvService unavailable' })),
      ]);
      playerBackend = ui.player_backend === 'libmpv' ? 'libmpv' : 'webview';
      startupTab = ui.startup_tab === 'mantorex' ? 'mantorex' : 'yaria';
      if (ui.player_hwdec === 'no' || ui.player_hwdec === 'auto' || ui.player_hwdec === 'auto-safe') {
        playerHwdec = ui.player_hwdec;
      }
      if (ui.player_cache === 'low' || ui.player_cache === 'normal' || ui.player_cache === 'high') {
        playerCache = ui.player_cache;
      }
      playerHqScale = !!ui.player_hq_scale;
      playerDeinterlace = !!ui.player_deinterlace;
      playerLoadUserConfig = !!ui.player_load_user_config;
      try {
        localStorage.setItem('yaria_startup_tab', startupTab);
        localStorage.setItem('yaria_player_backend', playerBackend);
        if (av?.available) localStorage.setItem('yaria_hevc_ok', '1');
        else localStorage.removeItem('yaria_hevc_ok');
      } catch { /* ignore */ }
      mpvAvailable = !!av?.available;
      mpvReason = (av as any)?.reason || '';
    } catch {
      playerBackend = 'webview';
      mpvAvailable = false;
    }
  }

  async function savePlayerOptions(partial: Record<string, string | boolean>) {
    playerOptsSaving = true;
    try {
      await api.settings.saveUISettings(partial as any);
      toastSuccess('Player settings saved — applies on next playback');
    } catch (e: any) {
      toastError(e?.message || 'Failed to save player settings');
    }
    playerOptsSaving = false;
  }

  async function setStartupTab(tab: 'yaria' | 'mantorex') {
    startupTabSaving = true;
    try {
      startupTab = tab;
      try { localStorage.setItem('yaria_startup_tab', tab); } catch { /* ignore */ }
      await api.settings.saveUISettings({ startup_tab: tab });
      toastSuccess('Startup tab saved');
    } catch (e: any) {
      toastError(e?.message || 'Failed to save');
    }
    startupTabSaving = false;
  }

  async function setPlayerBackend(backend: 'webview' | 'libmpv') {
    playerBackendSaving = true;
    playerBackendMsg = '';
    try {
      if (backend === 'libmpv' && !mpvAvailable) {
        playerBackendMsg = 'Downloading native player…';
        await api.deps.installMpv();
        // Re-check after a short wait (download runs async)
        for (let i = 0; i < 60; i++) {
          await new Promise((r) => setTimeout(r, 1000));
          const av = await api.mpv.available().catch(() => ({ available: false }));
          if (av?.available) {
            mpvAvailable = true;
            break;
          }
        }
        if (!mpvAvailable) {
          playerBackendMsg = 'Could not install native player automatically. WebView still works. Optional: sudo pacman -S mpv';
          toastError(playerBackendMsg);
          playerBackendSaving = false;
          return;
        }
        playerBackendMsg = 'Native player installed. Restart the app if playback fails to load libmpv.';
      }
      playerBackend = backend;
      await api.settings.saveUISettings({ player_backend: backend });
      try {
        localStorage.setItem('yaria_player_backend', backend);
      } catch {
        /* ignore */
      }
      if (backend === 'libmpv') {
        playerBackendMsg = playerBackendMsg || 'Native player selected. Applies on next playback.';
      } else {
        playerBackendMsg = 'WebView player selected. Applies on next playback.';
      }
      toastSuccess('Settings saved');
    } catch (e: any) {
      playerBackendMsg = e?.message || 'Failed to save';
      toastError(playerBackendMsg);
    }
    playerBackendSaving = false;
  }

  function resetUIDefaults() {
    uiFont = 'Roboto';
    uiFontSize = '14';
    uiScale = '100';
    uiAnimations = false;
    uiBlur = !navigator.platform.includes('Linux');
    saveUISettings({
      font: uiFont,
      fontSize: uiFontSize,
      scale: uiScale,
      animations: uiAnimations,
      blur: uiBlur,
    });
  }

  function collectPlayerPositions(): Record<string, string> {
    const out: Record<string, string> = {};
    try {
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (!key) continue;
        // Resume positions + played magnets
        if (
          key.startsWith('pos_') ||
          key.startsWith('local_') ||
          key === 'yaria_played_magnets' ||
          key.includes('magnet')
        ) {
          const val = localStorage.getItem(key);
          if (val) out[key] = val;
        }
      }
    } catch { /* ignore */ }
    return out;
  }

  function restorePlayerPositions(positions: Record<string, string> | undefined) {
    if (!positions || typeof positions !== 'object') return 0;
    let n = 0;
    try {
      for (const [key, val] of Object.entries(positions)) {
        if (typeof val === 'string') {
          localStorage.setItem(key, val);
          n++;
        }
      }
    } catch { /* ignore */ }
    return n;
  }

  function openExportPicker() {
    const date = new Date().toISOString().slice(0, 10);
    exportDefaultName = `yaria-library-backup-${date}.json`;
    backupMsg = '';
    backupErr = '';
    showExportPicker = true;
  }

  function openImportPicker() {
    backupMsg = '';
    backupErr = '';
    showImportPicker = true;
  }

  async function exportLibraryTo(path: string) {
    showExportPicker = false;
    if (!path) return;
    backupBusy = true;
    backupMsg = '';
    backupErr = '';
    try {
      const res = await api.library.exportLibrary();
      if (res?.error) {
        backupErr = res.error;
        toastError(res.error);
        return;
      }
      let payload: any;
      try {
        payload = JSON.parse(res.data || '{}');
      } catch {
        payload = { library: { items: [] }, version: 1 };
      }
      payload.player_positions = collectPlayerPositions();
      payload.app = payload.app || 'yaria';
      payload.version = payload.version || 1;

      const text = JSON.stringify(payload, null, 2);
      const writeRes = await api.deps.writeTextFile(path, text);
      if (writeRes?.error) {
        backupErr = writeRes.error;
        toastError(writeRes.error);
        return;
      }

      const count = res.count ?? payload?.library?.items?.length ?? 0;
      backupMsg = `Exported ${count} item${count === 1 ? '' : 's'} to ${path}`;
      toastSuccess('Library exported');
    } catch (e: any) {
      backupErr = e?.message || 'Export failed';
      toastError(backupErr);
    } finally {
      backupBusy = false;
    }
  }

  async function importLibraryFrom(path: string) {
    showImportPicker = false;
    if (!path) return;
    backupBusy = true;
    backupMsg = '';
    backupErr = '';
    try {
      const fileRes = await api.deps.readTextFile(path);
      if (fileRes?.error || !fileRes?.data) {
        backupErr = fileRes?.error || 'Could not read file';
        toastError(backupErr);
        return;
      }
      const text = fileRes.data;
      let parsed: any = null;
      try {
        parsed = JSON.parse(text);
      } catch {
        backupErr = 'Invalid backup file (not JSON)';
        toastError(backupErr);
        return;
      }

      const res = await api.library.importLibrary(text);
      if (res?.error) {
        backupErr = res.error;
        toastError(res.error);
        return;
      }

      const posN = restorePlayerPositions(parsed?.player_positions);
      const added = res.added ?? 0;
      const updated = res.updated ?? 0;
      const parts = [
        added ? `${added} added` : '',
        updated ? `${updated} updated` : '',
        posN ? `${posN} resume positions` : '',
      ].filter(Boolean);
      backupMsg = parts.length
        ? `Import complete: ${parts.join(', ')}.`
        : 'Import complete (nothing new to merge).';
      toastSuccess(backupMsg);
    } catch (err: any) {
      backupErr = err?.message || 'Import failed';
      toastError(backupErr);
    } finally {
      backupBusy = false;
    }
  }

  async function activateLicense() {
    const key = licenseKey.trim();
    if (!key) { licenseError = 'Please enter a license key'; return; }
    activating = true;
    licenseError = '';
    licenseSuccess = '';
    try {
      const result = await api.license.activate(key);
      if (result.error) {
        licenseError = result.error;
      } else if (result.valid) {
        licenseSuccess = 'Pro activated! Mantorex is now available.';
        isPro.set(true);
        setTimeout(() => loadLicense(), 1500);
      } else {
        licenseError = 'Invalid license key';
      }
    } catch (e: any) {
      licenseError = e.message || 'Activation failed';
    }
    activating = false;
  }

  async function deactivateLicense() {
    confirmMessage = 'Deactivate your Pro license on this device?';
    confirmCallback = async () => {
      showConfirm = false;
      try {
        await api.license.deactivate();
        isPro.set(false);
        loadLicense();
      } catch {}
    };
    showConfirm = true;
  }

  function handleLicenseKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') activateLicense();
  }

  async function loadLicense() {
    licenseLoading = true;
    try {
      const [info, device] = await Promise.all([
        api.license.check(),
        api.license.getDeviceInfo(),
      ]);
      licenseInfo = info;
      deviceInfo = device;
    } catch {
      licenseInfo = null;
      deviceInfo = null;
    }
    licenseLoading = false;
  }

  onMount(async () => {
    loadLicense();

    // Hydrate UI prefs from disk (source of truth across rebuilds)
    await loadUISettingsFromDisk();
    uiFont = localStorage.getItem('yaria_ui_font') || 'Roboto';
    uiFontSize = localStorage.getItem('yaria_ui_fontsize') || '14';
    uiScale = localStorage.getItem('yaria_ui_scale') || '100';
    uiAnimations = localStorage.getItem('yaria_ui_animations') === '1';
    uiBlur = localStorage.getItem('yaria_ui_blur')
      ? localStorage.getItem('yaria_ui_blur') === '1'
      : !navigator.platform.includes('Linux');

    // Load TMDB key (built-in default is never shown; only user override is)
    api.settings.getTMDBKey().then((tmdbInfo: any) => {
      tmdbUsingDefault = !!tmdbInfo?.using_default;
      if (tmdbInfo?.configured && !tmdbInfo?.using_default) {
        tmdbKey = tmdbInfo.key || '';
      } else {
        tmdbKey = '';
      }
    }).catch(() => {});

    // Load proxy
    api.settings.getProxy().then((proxy: any) => {
      if (proxy.type) proxyType = proxy.type;
      if (proxy.addr) proxyAddr = proxy.addr;
    }).catch(() => {});

    loadPlayerBackend();
  });
</script>

<div class="stg-panel active">
  <h3 class="stg-panel-title">General</h3>

  <!-- License -->
  <div class="setting-group">
    <div class="setting-label">Yaria Pro License</div>
    {#if licenseLoading}
      <Spinner size={18} message="Checking..." />
    {:else if licenseInfo?.valid && (licenseInfo?.plan === 'pro' || licenseInfo?.plan === 'trial')}
      <div class="license-info">
        <div class="license-badge-row">
          {#if licenseInfo.plan === 'trial'}
            <span class="badge badge-trial">TRIAL</span>
            <span>Active — free Pro trial</span>
          {:else}
            <span class="badge badge-pro">PRO</span>
            <span>Active</span>
          {/if}
        </div>
        {#if licenseInfo.expires_at}
          <div class="license-detail">Expires: <span class="val">{licenseInfo.expires_at}</span></div>
        {/if}
        <div class="license-detail">Email: <span class="val">{licenseInfo.email || '-'}</span></div>
        <div class="license-detail">Key: <span class="val">{licenseInfo.key ? licenseInfo.key.substring(0, 8) + '...' : '-'}</span></div>
        <div class="license-detail">Device: <span class="val">{deviceInfo?.device_name || '-'}</span></div>
        <div class="license-device-id">ID: {deviceInfo?.device_id || '-'}</div>
      </div>
      <button class="btn btn-ghost btn-sm deactivate-btn" onclick={deactivateLicense}>Deactivate License</button>
    {:else}
      <div class="license-info">
        <div class="license-badge-row">
          <span class="badge badge-free">FREE</span>
          <span class="text-dim">Mantorex features require Pro</span>
        </div>
        {#if deviceInfo}
          <div class="license-device-id">Device: {deviceInfo.device_name} ({deviceInfo.device_id})</div>
        {/if}
      </div>
      <div class="license-activate-row">
        <input
          type="text"
          class="setting-input"
          placeholder="Enter license key"
          bind:value={licenseKey}
          onkeydown={handleLicenseKeydown}
          use:autoFocus
        />
        <button
          class="btn btn-primary btn-sm"
          onclick={activateLicense}
          disabled={activating}
        >
          {activating ? 'Activating...' : 'Activate'}
        </button>
      </div>
      {#if licenseError}
        <div class="msg-error">{licenseError}</div>
      {/if}
      {#if licenseSuccess}
        <div class="msg-success">{licenseSuccess}</div>
      {/if}
      <p class="license-hint">Get a license at <a href="https://yaria.live" target="_blank">yaria.live</a></p>
    {/if}
  </div>

  <!-- TMDB -->
  <div class="setting-group">
    <div class="setting-label">TMDB API Key</div>
    <div class="setting-desc">
      Enables trending content, posters, and metadata.
      {#if tmdbUsingDefault}
        Using the built-in key — optional: enter your own to override.
      {:else}
        Get a free key at <a href="https://www.themoviedb.org/settings/api" target="_blank">themoviedb.org</a>.
      {/if}
    </div>
    <input
      type="text"
      class="setting-input"
      placeholder={tmdbUsingDefault ? 'Built-in key active (optional override)' : 'Enter your TMDB API key'}
      value={tmdbKey}
      oninput={handleTMDBInput}
    />
    {#if tmdbError}
      <div class="setting-saved error">{tmdbError}</div>
    {/if}
  </div>

  <!-- Proxy -->
  <div class="setting-group">
    <div class="setting-label">Proxy</div>
    <div class="setting-desc">Route network traffic through a proxy server.</div>
    <AppSelect
      bind:value={proxyType}
      options={[
        { value: 'none', label: 'No Proxy' },
        { value: 'http', label: 'HTTP Proxy' },
        { value: 'socks5', label: 'SOCKS5 Proxy' },
      ]}
      onchange={handleProxyTypeChange}
    />
    {#if proxyType !== 'none'}
      <input
        type="text"
        class="setting-input proxy-addr"
        placeholder="e.g. http://127.0.0.1:8080"
        bind:value={proxyAddr}
        oninput={handleProxyAddrInput}
      />
    {/if}
  </div>

  <!-- Format Filter (Linux only) -->
  {#if isLinux}
    <div class="setting-group">
      <div class="setting-label">Video Format Filter</div>
      <div class="setting-desc">Optional: hide HEVC/x265/10-bit releases (mainly for WebView). Off by default on Linux because Native (libmpv) is the default player and plays these formats. Turn on only if you use WebView and want a cleaner list.</div>
      <label class="toggle-row">
        <input
          type="checkbox"
          bind:checked={formatFilterEnabled}
          onchange={handleFormatFilter}
        />
        <span class="text-dim">Hide unplayable formats (HEVC, x265, 10-bit)</span>
      </label>
    </div>
  {/if}

  <!-- Appearance -->
  <div class="setting-group">
    <div class="setting-label">Appearance</div>
    <div class="setting-desc">Customize the look and feel of the app.</div>
    <div class="appearance-grid">
      <div>
        <label class="appearance-label">Font Family</label>
        <AppSelect bind:value={uiFont} options={fontOptions} onchange={handleFontChange} />
      </div>
      <div>
        <label class="appearance-label">Font Size</label>
        <AppSelect bind:value={uiFontSize} options={fontSizeOptions} onchange={handleFontSizeChange} />
      </div>
    </div>
    <div class="scale-row">
      <label class="appearance-label">UI Scale</label>
      <div class="scale-control">
        <input
          type="range"
          min="75"
          max="150"
          step="5"
          bind:value={uiScale}
          oninput={handleScaleChange}
        />
        <span class="scale-value">{uiScale}%</span>
      </div>
    </div>
    <label class="toggle-row">
      <input type="checkbox" bind:checked={uiAnimations} onchange={handleAnimationsChange} />
      <span class="text-dim">Enable animations</span>
    </label>
    <label class="toggle-row">
      <input type="checkbox" bind:checked={uiBlur} onchange={handleBlurChange} />
      <span class="text-dim">Enable glassmorphism (blur effects)</span>
    </label>
    <div class="reset-row">
      <button class="btn btn-ghost btn-sm reset-btn" onclick={resetUIDefaults}>Reset to Defaults</button>
    </div>
  </div>

  <!-- Startup tab -->
  <div class="setting-group">
    <div class="setting-label">Open on startup</div>
    <div class="setting-desc">
      Which main tab to show when the app launches. Deep links and the last page from the same session are not overridden.
    </div>
    <div class="player-backend-row">
      <button
        class="btn btn-sm"
        class:btn-primary={startupTab === 'yaria'}
        class:btn-ghost={startupTab !== 'yaria'}
        onclick={() => setStartupTab('yaria')}
        disabled={startupTabSaving}
      >
        Yaria
      </button>
      <button
        class="btn btn-sm"
        class:btn-primary={startupTab === 'mantorex'}
        class:btn-ghost={startupTab !== 'mantorex'}
        onclick={() => setStartupTab('mantorex')}
        disabled={startupTabSaving}
      >
        Mantorex (Local)
      </button>
    </div>
  </div>

  <!-- Video player (all modes: local, torrents, remote) -->
  <div class="setting-group">
    <div class="setting-label">Video Player</div>
    <div class="setting-desc">
      Used for Local, Mantorex torrents, and Remote playback. WebView is the built-in browser player.
      Native (libmpv) embeds mpv for wider codec support.
    </div>
    <div class="player-backend-row">
      <button
        class="btn btn-sm"
        class:btn-primary={playerBackend === 'webview'}
        class:btn-ghost={playerBackend !== 'webview'}
        onclick={() => setPlayerBackend('webview')}
        disabled={playerBackendSaving}
      >
        WebView (default)
      </button>
      <button
        class="btn btn-sm"
        class:btn-primary={playerBackend === 'libmpv'}
        class:btn-ghost={playerBackend !== 'libmpv'}
        onclick={() => setPlayerBackend('libmpv')}
        disabled={playerBackendSaving}
        title={mpvAvailable ? 'Use embedded libmpv' : 'Will download libmpv if missing'}
      >
        Native (libmpv)
      </button>
    </div>
    {#if !mpvAvailable}
      <div class="text-muted player-backend-hint">
        {mpvReason || 'libmpv not found yet — choosing Native will try to download it automatically.'}
      </div>
    {/if}
    {#if playerBackendMsg}
      <div class="msg-success" style="margin-top:8px">{playerBackendMsg}</div>
    {/if}
  </div>

  <!-- Native player tuning — only when Native (libmpv) is selected -->
  {#if playerBackend === 'libmpv'}
    <div class="setting-group">
      <div class="setting-label">Player (Native mpv)</div>
      <div class="setting-desc">
        Options for the Native player. Defaults match the built-in safe profile.
        Changes apply the next time you start playback.
      </div>

      <div class="setting-sublabel">Hardware decoding</div>
      <AppSelect
        bind:value={playerHwdec}
        disabled={playerOptsSaving}
        options={[
          { value: 'auto-safe', label: 'Auto (safe) — default' },
          { value: 'auto', label: 'Auto (aggressive)' },
          { value: 'no', label: 'Off (software)' },
        ]}
        onchange={() => savePlayerOptions({ player_hwdec: playerHwdec })}
      />

      <div class="setting-sublabel" style="margin-top:12px">Stream cache</div>
      <div class="text-muted" style="font-size:12px;margin-bottom:6px">
        Larger cache can reduce freezes on slow torrents (uses more RAM).
      </div>
      <AppSelect
        bind:value={playerCache}
        disabled={playerOptsSaving}
        options={[
          { value: 'low', label: 'Low' },
          { value: 'normal', label: 'Normal — default' },
          { value: 'high', label: 'High' },
        ]}
        onchange={() => savePlayerOptions({ player_cache: playerCache })}
      />

      <label class="toggle-row" style="margin-top:14px">
        <input
          type="checkbox"
          bind:checked={playerHqScale}
          disabled={playerOptsSaving}
          onchange={() => savePlayerOptions({ player_hq_scale: playerHqScale })}
        />
        <span class="text-dim">High-quality scaling (more GPU)</span>
      </label>

      <label class="toggle-row">
        <input
          type="checkbox"
          bind:checked={playerDeinterlace}
          disabled={playerOptsSaving}
          onchange={() => savePlayerOptions({ player_deinterlace: playerDeinterlace })}
        />
        <span class="text-dim">Deinterlace</span>
      </label>

      <label class="toggle-row">
        <input
          type="checkbox"
          bind:checked={playerLoadUserConfig}
          disabled={playerOptsSaving}
          onchange={() => savePlayerOptions({ player_load_user_config: playerLoadUserConfig })}
        />
        <span class="text-dim">Load user mpv config (~/.config/mpv)</span>
      </label>
      <div class="text-muted" style="font-size:12px;margin-top:4px">
        Off by default. Your system mpv.conf can break in-app embedding (window, controls, keys).
        Embed-safe options are still forced when enabled.
      </div>
    </div>
  {/if}

  <!-- Library Backup -->
  {#if $isPro}
    <div class="setting-group">
      <div class="setting-label">Library Backup</div>
      <div class="setting-desc">
        Export your Mantorex library (titles, posters, watch progress) to a file.
        On a new computer, import that file to restore everything as you left it.
      </div>
      <div class="backup-actions">
        <button class="btn btn-primary btn-sm" onclick={openExportPicker} disabled={backupBusy}>
          {backupBusy ? 'Working…' : 'Export Library'}
        </button>
        <button class="btn btn-ghost btn-sm" onclick={openImportPicker} disabled={backupBusy}>
          Import Library
        </button>
      </div>
      {#if backupMsg}
        <div class="msg-success">{backupMsg}</div>
      {/if}
      {#if backupErr}
        <div class="msg-error">{backupErr}</div>
      {/if}
    </div>
  {/if}
</div>

<!-- Confirm Dialog for deactivation -->
{#if showConfirm}
  <ConfirmDialog
    message={confirmMessage}
    onConfirm={() => { if (confirmCallback) confirmCallback(); }}
    onCancel={() => { showConfirm = false; }}
  />
{/if}

{#if showExportPicker}
  <FilePicker
    mode="save"
    title="Export Library"
    fileExt="json"
    defaultFileName={exportDefaultName}
    onSelect={exportLibraryTo}
    onClose={() => { showExportPicker = false; }}
  />
{/if}

{#if showImportPicker}
  <FilePicker
    mode="open"
    title="Import Library"
    fileExt="json"
    onSelect={importLibraryFrom}
    onClose={() => { showImportPicker = false; }}
  />
{/if}

<style lang="scss">
  @use '../../styles/variables' as *;

  .stg-panel-title {
    font-size: 20px;
    font-weight: 700;
    color: $text;
    margin-bottom: 24px;
  }

  .setting-group {
    margin-bottom: 28px;
    padding-bottom: 24px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.04);

    &:last-child {
      border-bottom: none;
    }
  }

  .setting-label {
    font-size: 14px;
    font-weight: 600;
    color: $text;
    margin-bottom: 6px;
  }

  .setting-desc {
    font-size: 12px;
    color: $text-muted;
    margin-bottom: 12px;
    line-height: 1.6;

    a {
      color: $accent;
    }
  }

  .setting-saved {
    font-size: 13px;
    color: $green;
    margin-top: 6px;

    &.error {
      color: $red;
    }
  }

  .proxy-addr {
    margin-top: 8px;
  }

  .license-info {
    font-size: 13px;
    line-height: 1.8;
    margin-bottom: 12px;
  }

  .license-badge-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .badge {
    font-size: 11px;
    font-weight: 700;
    padding: 3px 10px;
    border-radius: 99px;
  }

  .badge-pro {
    background: $green;
    color: #000;
  }

  .badge-trial {
    background: $accent;
    color: #fff;
  }

  .badge-free {
    background: $text-muted;
    color: #000;
  }

  .license-detail {
    color: $text-dim;

    .val {
      color: $text;
    }
  }

  .license-device-id {
    color: $text-muted;
    font-size: 11px;
    margin-top: 4px;
  }

  .license-activate-row {
    display: flex;
    gap: 8px;
    align-items: center;
    margin-bottom: 8px;

    :global(.setting-input) {
      flex: 1;
    }
  }

  .license-hint {
    color: $text-muted;
    font-size: 12px;
    margin-top: 12px;

    a {
      color: $accent;
    }
  }

  .msg-error {
    font-size: 13px;
    color: $red;
    margin-top: 8px;
  }

  .msg-success {
    font-size: 13px;
    color: $green;
    margin-top: 8px;
  }

  .deactivate-btn {
    color: $red !important;
  }

  .setting-sublabel {
    font-size: 13px;
    font-weight: 600;
    color: $text;
    margin-top: 4px;
    margin-bottom: 6px;
  }

  .toggle-row {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-top: 12px;
    cursor: pointer;

    input[type="checkbox"] {
      cursor: pointer;
    }

    .text-dim {
      font-size: 13px;
    }
  }

  .appearance-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin-top: 10px;
  }

  .appearance-label {
    font-size: 12px;
    color: $text-dim;
    display: block;
    margin-bottom: 4px;
  }

  .scale-row {
    margin-top: 12px;
  }

  .scale-control {
    display: flex;
    align-items: center;
    gap: 10px;

    input[type="range"] {
      flex: 1;
    }
  }

  .scale-value {
    font-size: 13px;
    color: $text-dim;
    min-width: 35px;
  }

  .reset-row {
    margin-top: 16px;
  }

  .reset-btn {
    color: $red !important;
  }

  .player-backend-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .player-backend-hint {
    margin-top: 8px;
    font-size: 12px;
  }

  .backup-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
  }

  .text-dim {
    color: $text-dim;
  }
</style>
