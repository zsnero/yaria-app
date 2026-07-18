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
  let tmdbDebounce: ReturnType<typeof setTimeout> | null = null;

  // Proxy
  let proxyType = $state('none');
  let proxyAddr = $state('');
  let proxyDebounce: ReturnType<typeof setTimeout> | null = null;

  // Format filter (Linux only)
  const isLinux = navigator.platform.indexOf('Linux') !== -1;
  let formatFilterEnabled = $state(localStorage.getItem('yaria_show_all_formats') !== '1');

  // Appearance
  let uiFont = $state(localStorage.getItem('yaria_ui_font') || 'Inter');
  let uiFontSize = $state(localStorage.getItem('yaria_ui_fontsize') || '14');
  let uiScale = $state(localStorage.getItem('yaria_ui_scale') || '100');
  let uiAnimations = $state(localStorage.getItem('yaria_ui_animations') !== '0');
  let uiBlur = $state(
    localStorage.getItem('yaria_ui_blur')
      ? localStorage.getItem('yaria_ui_blur') === '1'
      : !navigator.platform.includes('Linux')
  );

  // Font options
  const fontOptions = [
    { value: 'Inter', label: 'Inter (Default)' },
    { value: 'system-ui', label: 'System Default' },
    { value: "'SF Pro Display', -apple-system, BlinkMacSystemFont", label: 'SF Pro (macOS)' },
    { value: "'Segoe UI'", label: 'Segoe UI (Windows)' },
    { value: 'Roboto', label: 'Roboto' },
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
    if (tmdbDebounce) clearTimeout(tmdbDebounce);
    tmdbDebounce = setTimeout(async () => {
      try {
        await api.settings.saveTMDBKey(val.trim());
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

  function resetUIDefaults() {
    uiFont = 'Inter';
    uiFontSize = '14';
    uiScale = '100';
    uiAnimations = true;
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
    uiFont = localStorage.getItem('yaria_ui_font') || 'Inter';
    uiFontSize = localStorage.getItem('yaria_ui_fontsize') || '14';
    uiScale = localStorage.getItem('yaria_ui_scale') || '100';
    uiAnimations = localStorage.getItem('yaria_ui_animations') !== '0';
    uiBlur = localStorage.getItem('yaria_ui_blur')
      ? localStorage.getItem('yaria_ui_blur') === '1'
      : !navigator.platform.includes('Linux');

    // Load TMDB key
    api.settings.getTMDBKey().then((tmdbInfo: any) => {
      if (tmdbInfo?.configured) tmdbKey = tmdbInfo.key;
    }).catch(() => {});

    // Load proxy
    api.settings.getProxy().then((proxy: any) => {
      if (proxy.type) proxyType = proxy.type;
      if (proxy.addr) proxyAddr = proxy.addr;
    }).catch(() => {});
  });
</script>

<div class="stg-panel active">
  <h3 class="stg-panel-title">General</h3>

  <!-- License -->
  <div class="setting-group">
    <div class="setting-label">Yaria Pro License</div>
    {#if licenseLoading}
      <Spinner size={18} message="Checking..." />
    {:else if licenseInfo?.valid && licenseInfo?.plan === 'pro'}
      <div class="license-info">
        <div class="license-badge-row">
          <span class="badge badge-pro">PRO</span>
          <span>Active</span>
        </div>
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
    <div class="setting-desc">Enables trending content, posters, and metadata. Get a free key at <a href="https://www.themoviedb.org/settings/api" target="_blank">themoviedb.org</a>.</div>
    <input
      type="text"
      class="setting-input"
      placeholder="Enter your TMDB API key"
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
      <div class="setting-desc">Linux cannot reliably play HEVC/x265/10-bit video. When enabled, these formats are hidden from torrent listings. Disable to show all formats (for downloading or if you have working codec support).</div>
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
