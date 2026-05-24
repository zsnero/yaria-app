<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../../api/wails';
  import Spinner from '../../components/Spinner.svelte';
  import ConfirmDialog from '../../components/ConfirmDialog.svelte';

  // Codecs
  let codecLoading = $state(true);
  let codecInfo = $state<any>(null);
  let codecInstalling = $state(false);
  let codecInstallMsg = $state('');
  let codecInstallError = $state(false);

  // Cache
  let cacheLoading = $state(true);
  let cacheStats = $state<any>(null);
  let cacheResult = $state('');
  let cacheResultError = $state(false);

  // Confirm dialog
  let showConfirm = $state(false);
  let confirmMessage = $state('');
  let confirmCallback = $state<(() => void) | null>(null);

  async function loadCodecs() {
    codecLoading = true;
    try {
      codecInfo = await api.codecs.check();
    } catch {
      codecInfo = null;
    }
    codecLoading = false;
  }

  async function loadCacheStats() {
    cacheLoading = true;
    cacheResult = '';
    try {
      cacheStats = await api.settings.getCacheStats();
    } catch {
      cacheStats = null;
    }
    cacheLoading = false;
  }

  async function installCodecs() {
    codecInstalling = true;
    codecInstallMsg = 'A password prompt may appear...';
    codecInstallError = false;
    try {
      const result = await api.codecs.install();
      if (result.error) {
        codecInstallMsg = result.error;
        codecInstallError = true;
      } else {
        codecInstallMsg = 'Installed! Restart the app for full codec support.';
        codecInstallError = false;
        setTimeout(() => loadCodecs(), 2000);
      }
    } catch (e: any) {
      codecInstallMsg = 'Failed: ' + (e.message || '');
      codecInstallError = true;
    }
    codecInstalling = false;
  }

  async function clearCache(type: string) {
    if (type === 'all') {
      confirmMessage = 'This will delete ALL cached data. Continue?';
      confirmCallback = async () => {
        showConfirm = false;
        await doClearCache('all');
      };
      showConfirm = true;
    } else {
      await doClearCache(type);
    }
  }

  async function doClearCache(type: string) {
    try {
      const result = await api.settings.clearCache(type);
      cacheResult = `Cleared ${result.removed} items, freed ${result.freed}`;
      cacheResultError = false;
      setTimeout(() => loadCacheStats(), 2000);
    } catch (err: any) {
      cacheResult = `Failed: ${err.message}`;
      cacheResultError = true;
    }
  }

  onMount(() => {
    loadCodecs();
    loadCacheStats();
  });
</script>

<div class="stg-panel active">
  <h3 class="stg-panel-title">Mantorex / Torrents</h3>

  <!-- Codecs -->
  <div class="setting-group">
    <div class="setting-label">Video Codecs</div>
    <div class="setting-desc">Codecs required for playing streams in the built-in player.</div>
    {#if codecLoading}
      <Spinner size={18} message="Checking..." />
    {:else if codecInfo?.supported === false}
      <span class="text-muted">Codec check not available on this platform</span>
    {:else if codecInfo}
      <div class="codec-list">
        {#each codecInfo.codecs || [] as c}
          <div class="codec-row">
            <span class="text-dim">{c.name}</span>
            <span>
              {#if c.installed}
                <span class="text-green">Installed</span>
              {:else}
                <span class="text-red">Missing</span>
              {/if}
              <span class="codec-package">{c.package}</span>
            </span>
          </div>
        {/each}
      </div>
      {#if !codecInfo.all_installed}
        <button
          class="btn btn-primary btn-sm"
          onclick={installCodecs}
          disabled={codecInstalling}
        >
          {codecInstalling ? 'Installing...' : 'Install Missing Codecs'}
        </button>
        {#if codecInstallMsg}
          <div class="codec-install-msg" class:error={codecInstallError}>{codecInstallMsg}</div>
        {/if}
      {/if}
    {:else}
      <span class="text-muted">Could not check codec status</span>
    {/if}
  </div>

  <!-- Cache -->
  <div class="setting-group">
    <div class="setting-label">Torrent Cache</div>
    <div class="setting-desc">Remove cached torrent data, partial files, and metadata.</div>
    {#if cacheLoading}
      <Spinner size={18} message="Scanning..." />
    {:else if cacheStats}
      <div class="cache-stats">
        <div>Partial downloads: <span class="val">{cacheStats.partial_files} files</span> ({cacheStats.partial_size})</div>
        <div>Metadata files: <span class="val">{cacheStats.meta_files} files</span> ({cacheStats.meta_size})</div>
        <div>Data folders: <span class="val">{cacheStats.data_dirs} folders</span> ({cacheStats.dir_size})</div>
        <div class="cache-total">Total: {cacheStats.total_size}</div>
        <div class="cache-dir">{cacheStats.data_dir}</div>
      </div>
      <div class="cache-actions">
        <button class="btn btn-ghost btn-sm" onclick={() => clearCache('partial')}>Partial</button>
        <button class="btn btn-ghost btn-sm" onclick={() => clearCache('meta')}>Metadata</button>
        <button class="btn btn-ghost btn-sm" onclick={() => clearCache('dirs')}>Data</button>
        <button class="btn btn-sm clear-all-btn" onclick={() => clearCache('all')}>Clear All</button>
      </div>
      {#if cacheResult}
        <div class="cache-result" class:error={cacheResultError}>{cacheResult}</div>
      {/if}
    {:else}
      <span class="text-muted">Could not load cache info</span>
    {/if}
  </div>
</div>

{#if showConfirm}
  <ConfirmDialog
    message={confirmMessage}
    onConfirm={() => { if (confirmCallback) confirmCallback(); }}
    onCancel={() => { showConfirm = false; }}
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
  }

  .codec-list {
    font-size: 13px;
    line-height: 2;
    margin-top: 10px;
    margin-bottom: 12px;
  }

  .codec-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .codec-package {
    color: $text-muted;
    font-size: 11px;
    margin-left: 6px;
  }

  .codec-install-msg {
    font-size: 13px;
    color: $text-muted;
    margin-top: 8px;

    &.error {
      color: $red;
    }
  }

  .cache-stats {
    font-size: 13px;
    color: $text-dim;
    line-height: 1.8;
    margin-bottom: 10px;

    .val {
      color: $text;
    }
  }

  .cache-total {
    margin-top: 6px;
    font-weight: 600;
    color: $text;
  }

  .cache-dir {
    font-size: 12px;
    color: $text-muted;
    margin-top: 4px;
  }

  .cache-actions {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .clear-all-btn {
    background: $red;
    color: white;
    box-shadow: none;
  }

  .cache-result {
    margin-top: 8px;
    font-size: 13px;
    color: $green;

    &.error {
      color: $red;
    }
  }

  .text-dim {
    color: $text-dim;
  }

  .text-muted {
    color: $text-muted;
    font-size: 13px;
  }

  .text-green {
    color: $green;
  }

  .text-red {
    color: $red;
  }
</style>
