<script lang="ts">
  import { onMount } from 'svelte';
  import { slide } from 'svelte/transition';
  import { api } from '../api/wails';
  import { autoFocus } from '../actions/index';
  import { ripple } from '../actions/index';
  import { toastSuccess } from '../stores/toast';
  import Spinner from '../components/Spinner.svelte';
  import Skeleton from '../components/Skeleton.svelte';
  import FilePicker from '../components/FilePicker.svelte';

  // State
  let url = $state('');
  let depsReady = $state(false);
  let depsMessage = $state('Preparing download tools...');
  let depsError = $state('');
  let fetching = $state(false);
  let fetchError = $state('');

  // Metadata
  let meta = $state<any>(null);

  // Format selection
  let audioOnly = $state(false);
  let selectedFormat = $state('best');
  let audioFormat = $state('mp3');
  let formats = $state<{ video: any[]; audio: any[] }>({ video: [], audio: [] });

  // Download
  let downloadDir = $state('');
  let showFilePicker = $state(false);
  let downloading = $state(false);
  let dlId = $state('');
  let dlStatus = $state('');
  let dlPercent = $state(0);
  let dlSpeed = $state('');
  let dlEta = $state('');
  let dlMessage = $state('');
  let dlMessageColor = $state('');

  // Cleanups
  let eventCleanups: (() => void)[] = [];
  let urlInput: HTMLInputElement | undefined = $state();

  // Format grid options
  const defaultResolutions = ['2160p', '1440p', '1080p', '720p', '480p', '360p'];
  const audioFormats = ['mp3', 'm4a', 'opus', 'wav', 'flac'];

  let videoFormatOptions = $derived.by(() => {
    if (formats.video.length > 0) {
      return formats.video.map((f: any, i: number) => ({
        key: f.resolution || f.format_id || `video-${i}`,
        label: f.resolution || f.format_note || f.format_id || `video-${i}`,
        ext: formatFilesize(f.filesize) || f.ext || '',
      }));
    }
    return defaultResolutions.map(r => ({ key: r, label: r, ext: '' }));
  });

  onMount(() => {
    initDeps();
    loadDownloadDir();

    // Subscribe to events
    const offReady = api.events.on('deps-ready', () => {
      depsReady = true;
      depsMessage = '';
      depsError = '';
    });
    eventCleanups.push(offReady);

    const offProgress = api.events.on('deps-progress', (data: any) => {
      if (data?.message) {
        depsMessage = data.message;
      }
    });
    eventCleanups.push(offProgress);

    const offError = api.events.on('deps-error', (data: any) => {
      depsError = data?.error || 'Setup failed';
      depsMessage = '';
    });
    eventCleanups.push(offError);

    const offDlProgress = api.events.on('download-progress', (data: any) => {
      updateActiveDownload(data);
    });
    eventCleanups.push(offDlProgress);

    return () => {
      eventCleanups.forEach(fn => fn());
      eventCleanups = [];
    };
  });

  async function initDeps() {
    depsReady = false;
    depsError = '';
    depsMessage = 'Preparing download tools...';

    try {
      const check = await api.deps.check();
      if (check?.all_ready) {
        depsReady = true;
        depsMessage = '';
        return;
      }
    } catch { /* continue to init */ }

    try {
      await api.downloads.initDeps();
      depsReady = true;
      depsMessage = '';
    } catch (err: any) {
      depsError = err?.message || 'Failed to initialize';
    }
  }

  async function loadDownloadDir() {
    // Default download dir from last used or empty
    try {
      const saved = localStorage.getItem('yaria_download_dir');
      if (saved) downloadDir = saved;
    } catch { /* ignore */ }
  }

  function cleanUrl(raw: string): string {
    return raw.trim().replace(/\\([?=&#])/g, '$1');
  }

  async function doFetch() {
    const cleaned = cleanUrl(url);
    if (!cleaned) return;
    url = cleaned;
    meta = null;
    selectedFormat = 'best';
    fetchError = '';
    fetching = true;

    try {
      meta = await api.downloads.getMetadata(cleaned);
      // Load formats in background
      loadFormats(cleaned);
    } catch (err: any) {
      fetchError = err?.message || 'Failed to fetch metadata';
    } finally {
      fetching = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') doFetch();
  }

  async function loadFormats(fetchUrl: string) {
    try {
      const result = await api.downloads.getMetadata(fetchUrl);
      if (result?.formats) {
        formats = {
          video: result.formats.filter((f: any) => f.vcodec && f.vcodec !== 'none') || [],
          audio: result.formats.filter((f: any) => f.acodec && f.acodec !== 'none' && (!f.vcodec || f.vcodec === 'none')) || [],
        };
      }
    } catch { /* ignore */ }
  }

  function selectVideoFormat(key: string) {
    selectedFormat = key;
  }

  function selectAudioFormat(fmt: string) {
    audioFormat = fmt;
    selectedFormat = fmt;
  }

  function toggleAudioOnly() {
    audioOnly = !audioOnly;
    if (audioOnly) {
      selectedFormat = audioFormat;
    } else {
      selectedFormat = 'best';
    }
  }

  function handleBrowse() {
    showFilePicker = true;
  }

  function handleDirSelected(dir: string) {
    showFilePicker = false;
    if (dir) {
      downloadDir = dir;
      try { localStorage.setItem('yaria_download_dir', dir); } catch { /* ignore */ }
    }
  }

  async function startDownload() {
    if (!selectedFormat || !url) return;
    downloading = true;
    dlStatus = 'DOWNLOADING';
    dlPercent = 0;
    dlSpeed = '';
    dlEta = '';
    dlMessage = '';
    dlMessageColor = '';

    try {
      const result = await api.downloads.start(url, selectedFormat, downloadDir, audioOnly, audioFormat);
      dlId = result?.id || '';
      toastSuccess('Download started');
    } catch (err: any) {
      downloading = false;
      fetchError = err?.message || 'Download failed to start';
    }
  }

  function updateActiveDownload(data: any) {
    if (!data?.id || (dlId && data.id !== dlId)) return;

    if (data.percent != null) dlPercent = Math.min(data.percent, 100);
    if (data.speed) dlSpeed = data.speed;
    if (data.eta) dlEta = data.eta;
    if (data.status) {
      dlStatus = data.status.toUpperCase();
    }
    if (data.status === 'complete') {
      dlMessage = 'Download complete!';
      dlMessageColor = 'var(--green, #34d399)';
    } else if (data.status === 'error') {
      dlMessage = data.error || 'Download failed';
      dlMessageColor = 'var(--red, #f87171)';
    } else if (data.status === 'metadata') {
      dlMessage = 'Fetching metadata...';
      dlMessageColor = '';
    }
  }

  async function cancelDownload() {
    if (!dlId) return;
    try {
      await api.downloads.cancel(dlId);
      dlStatus = 'CANCELLED';
    } catch { /* ignore */ }
  }

  function downloadAnother() {
    meta = null;
    downloading = false;
    dlId = '';
    dlStatus = '';
    dlPercent = 0;
    dlSpeed = '';
    dlEta = '';
    dlMessage = '';
    dlMessageColor = '';
    fetchError = '';
    selectedFormat = 'best';
    formats = { video: [], audio: [] };
    url = '';
    setTimeout(() => urlInput?.focus(), 50);
  }

  function formatFilesize(bytes: any): string {
    if (!bytes) return '';
    const n = typeof bytes === 'number' ? bytes : parseFloat(bytes);
    if (isNaN(n) || n === 0) return typeof bytes === 'string' ? bytes : '';
    if (n > 1073741824) return (n / 1073741824).toFixed(1) + ' GB';
    if (n > 1048576) return (n / 1048576).toFixed(1) + ' MB';
    if (n > 1024) return (n / 1024).toFixed(1) + ' KB';
    return n + ' B';
  }

  function formatDuration(seconds: number): string {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${String(s).padStart(2, '0')}`;
  }

  function getStatusClass(status: string): string {
    switch (status.toLowerCase()) {
      case 'downloading': return 'status-downloading';
      case 'complete': return 'status-complete';
      case 'error': return 'status-error';
      case 'cancelled': return 'status-cancelled';
      case 'processing': return 'status-processing';
      case 'queued': case 'metadata': return 'status-queued';
      default: return '';
    }
  }
</script>

<div class="yaria-home-page">
  <!-- Hero -->
  <div class="yaria-hero">
    <img src="img/yaria-fox-crop.png" alt="" class="yaria-hero-icon" />
    <img src="img/yaria-text-crop.png" alt="Yaria" class="yaria-hero-wordmark" />
    <p class="yaria-subtitle">Video & Audio Downloader</p>
  </div>

  <!-- URL Input -->
  <div class="download-url-section">
    <div class="download-input-wrap">
      <input
        type="text"
        class="download-input"
        placeholder="Paste a video URL..."
        autocomplete="off"
        bind:value={url}
        bind:this={urlInput}
        onkeydown={handleKeydown}
        disabled={!depsReady}
        use:autoFocus
      />
      <button
        class="btn btn-primary"
        onclick={doFetch}
        disabled={!depsReady || fetching}
      >
        {fetching ? 'Fetching...' : 'Fetch'}
      </button>
    </div>

    <!-- Deps status -->
    {#if !depsReady && !depsError}
      <div class="download-deps-status">
        <Spinner size={18} />
        <span class="deps-msg">{depsMessage}</span>
      </div>
    {/if}
    {#if depsError}
      <div class="download-deps-status">
        <span class="deps-error">Setup failed: {depsError}</span>
        <button class="btn btn-ghost btn-sm" onclick={initDeps}>Retry</button>
      </div>
    {/if}
  </div>

  <!-- Content area -->
  <div class="dl-content">
    {#if fetching}
      <div class="dl-card">
        <div class="dl-card-loading">
          <Skeleton variant="card" width="100%" height="200px" />
        </div>
      </div>
    {:else if fetchError && !meta}
      <div class="dl-card">
        <div class="no-results">Failed: {fetchError}</div>
      </div>
    {:else if downloading}
      <!-- Inline progress card -->
      <div class="dl-card">
        <div class="dl-card-header">
          {#if meta?.thumbnail}
            <img class="dl-card-thumb" src={meta.thumbnail} alt="" onerror={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none'; }} />
          {/if}
          <div class="dl-card-info">
            <h3 class="dl-card-title">{meta?.title || url}</h3>
            <div class="dl-inline-progress">
              <div class="dl-inline-status">
                <span class="download-status {getStatusClass(dlStatus)}">{dlStatus}</span>
                <span class="dl-inline-pct">{dlPercent.toFixed(1)}%</span>
                {#if dlSpeed}
                  <span class="dl-inline-speed">{dlSpeed}</span>
                {/if}
                {#if dlEta}
                  <span class="dl-inline-eta">ETA: {dlEta}</span>
                {/if}
              </div>
              <div class="download-progress-bar">
                <div class="download-progress-fill" style="width: {dlPercent}%"></div>
              </div>
            </div>
          </div>
        </div>
        <div class="dl-card-actions">
          {#if dlMessage}
            <span class="dl-inline-msg" style:color={dlMessageColor}>{dlMessage}</span>
          {:else}
            <span></span>
          {/if}
          <div class="dl-card-btns">
            {#if dlStatus !== 'COMPLETE' && dlStatus !== 'CANCELLED' && dlStatus !== 'ERROR'}
              <button class="btn btn-ghost btn-sm" onclick={cancelDownload}>Cancel</button>
            {/if}
            <button class="btn btn-primary btn-sm" onclick={downloadAnother}>Download Another</button>
          </div>
        </div>
      </div>
    {:else if meta}
      <!-- Metadata card with format selection -->
      <div class="dl-card">
        <div class="dl-card-header">
          {#if meta.thumbnail}
            <img class="dl-card-thumb" src={meta.thumbnail} alt="" onerror={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none'; }} />
          {/if}
          <div class="dl-card-info">
            <h3 class="dl-card-title">{meta.title || 'Unknown Title'}</h3>
            {#if meta.uploader}
              <p class="dl-card-uploader">{meta.uploader}</p>
            {/if}
            {#if meta.duration}
              <p class="dl-card-duration">{formatDuration(meta.duration)}</p>
            {/if}
          </div>
        </div>

        <!-- Format selection -->
        <div class="dl-card-formats">
          <div class="format-header">
            <span class="format-section-title">Quality</span>
            <div class="audio-toggle-wrap">
              <label class="audio-toggle">
                <input type="checkbox" checked={audioOnly} onchange={toggleAudioOnly} />
                <span class="audio-toggle-slider"></span>
              </label>
              <span class="audio-toggle-label">Audio Only</span>
            </div>
          </div>

          <div class="format-grid" transition:slide={{ duration: 250 }}>
            {#if audioOnly}
              {#each audioFormats as fmt}
                <button
                  class="format-card"
                  class:selected={audioFormat === fmt}
                  onclick={() => selectAudioFormat(fmt)}
                  use:ripple
                >
                  <div class="format-card-label">{fmt.toUpperCase()}</div>
                </button>
              {/each}
            {:else}
              <button
                class="format-card"
                class:selected={selectedFormat === 'best'}
                onclick={() => selectVideoFormat('best')}
                use:ripple
              >
                <div class="format-card-label">Best</div>
                <div class="format-card-ext">Auto</div>
              </button>
              {#each videoFormatOptions as opt}
                <button
                  class="format-card"
                  class:selected={selectedFormat === opt.key}
                  onclick={() => selectVideoFormat(opt.key)}
                  use:ripple
                >
                  <div class="format-card-label">{opt.label}</div>
                  {#if opt.ext}
                    <div class="format-card-ext">{opt.ext}</div>
                  {/if}
                </button>
              {/each}
            {/if}
          </div>
        </div>

        <!-- Download dir + button -->
        <div class="dl-card-actions">
          <div class="dl-card-dir">
            <input
              type="text"
              class="setting-input dir-input"
              value={downloadDir}
              readonly
              placeholder="Download directory"
            />
            <button class="btn btn-ghost btn-sm" onclick={handleBrowse}>Browse</button>
          </div>
          <button
            class="btn btn-primary dl-start-btn"
            onclick={startDownload}
          >
            Download
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>

{#if showFilePicker}
  <FilePicker
    onSelect={handleDirSelected}
    onClose={() => { showFilePicker = false; }}
  />
{/if}

<style lang="scss">
  @use '../styles/variables' as *;

  .yaria-home-page {
    padding: 32px 24px 48px;
    max-width: 720px;
    margin: 0 auto;
    animation: pageIn 0.3s ease;
  }

  .yaria-hero {
    text-align: center;
    padding: 48px 0 36px;
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .yaria-hero-icon {
    width: 120px;
    height: 120px;
    object-fit: contain;
    margin-bottom: 12px;
  }

  .yaria-hero-wordmark {
    height: 36px;
    object-fit: contain;
    margin-bottom: 6px;
  }

  .yaria-subtitle {
    color: $text-dim;
    font-size: 15px;
    letter-spacing: 0.5px;
  }

  .download-url-section {
    margin-bottom: 24px;
  }

  .download-input-wrap {
    display: flex;
    gap: 10px;
  }

  .download-input {
    flex: 1;
    padding: 12px 16px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid $border-glass;
    border-radius: $radius-sm;
    color: $text;
    font-size: 14px;
    outline: none;
    transition: border-color 0.2s;

    &:focus {
      border-color: rgba($accent, 0.3);
    }

    &:disabled {
      opacity: 0.5;
    }
  }

  .download-deps-status {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    padding: 0 4px;
  }

  .deps-msg {
    color: $text-muted;
    font-size: 13px;
  }

  .deps-error {
    color: $red;
    font-size: 13px;
  }

  .dl-content {
    margin-top: 8px;
  }

  .dl-card {
    @include glass;
    border-radius: $radius;
    padding: 0;
    overflow: hidden;
    animation: modalIn 0.2s ease;
  }

  .dl-card-loading {
    padding: 40px 20px;
    @include flex-center;
  }

  .dl-card-header {
    display: flex;
    gap: 16px;
    padding: 20px;
    align-items: flex-start;
  }

  .dl-card-thumb {
    width: 160px;
    height: 90px;
    object-fit: cover;
    border-radius: 8px;
    flex-shrink: 0;
    background: rgba(255, 255, 255, 0.02);
  }

  .dl-card-info {
    flex: 1;
    min-width: 0;
  }

  .dl-card-title {
    font-size: 15px;
    font-weight: 600;
    color: $text;
    margin-bottom: 4px;
    @include truncate;
  }

  .dl-card-uploader {
    font-size: 12px;
    color: $text-dim;
    margin-bottom: 2px;
  }

  .dl-card-duration {
    font-size: 12px;
    color: $text-muted;
  }

  .dl-card-formats {
    padding: 16px 20px;
    border-top: 1px solid $border-glass;
  }

  .format-header {
    @include flex-between;
    margin-bottom: 12px;
  }

  .format-section-title {
    font-size: 13px;
    font-weight: 500;
    color: $text;
  }

  .audio-toggle-wrap {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .audio-toggle {
    position: relative;
    display: inline-block;
    width: 36px;
    height: 20px;
    cursor: pointer;

    input {
      opacity: 0;
      width: 0;
      height: 0;
    }
  }

  .audio-toggle-slider {
    position: absolute;
    inset: 0;
    background: rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    transition: background 0.2s;

    &::before {
      content: '';
      position: absolute;
      left: 3px;
      bottom: 3px;
      width: 14px;
      height: 14px;
      background: $text;
      border-radius: 50%;
      transition: transform 0.2s;
    }
  }

  .audio-toggle input:checked + .audio-toggle-slider {
    background: $accent;

    &::before {
      transform: translateX(16px);
    }
  }

  .audio-toggle-label {
    font-size: 12px;
    color: $text-dim;
  }

  .format-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .format-card {
    @include glass;
    border-radius: 8px;
    padding: 10px 16px;
    cursor: pointer;
    transition: all 0.15s;
    min-width: 70px;
    text-align: center;

    &:hover {
      @include glass-hover;
    }

    &.selected {
      background: rgba($accent, 0.15);
      border-color: rgba($accent, 0.4);
    }
  }

  .format-card-label {
    font-size: 13px;
    font-weight: 500;
    color: $text;
  }

  .format-card-ext {
    font-size: 10px;
    color: $text-muted;
    margin-top: 2px;
  }

  .dl-card-actions {
    padding: 16px 20px;
    border-top: 1px solid $border-glass;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .dl-card-dir {
    display: flex;
    gap: 8px;
  }

  .dir-input {
    flex: 1;
  }

  .dl-start-btn {
    padding: 12px 48px;
    font-size: 14px;
    align-self: center;
  }

  // Inline progress
  .dl-inline-progress {
    margin-top: 8px;
  }

  .dl-inline-status {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 6px;
    flex-wrap: wrap;
  }

  .dl-inline-pct {
    font-size: 13px;
    font-weight: 600;
    color: $text;
  }

  .dl-inline-speed,
  .dl-inline-eta {
    font-size: 12px;
    color: $text-dim;
  }

  .dl-inline-msg {
    font-size: 12px;
    color: $text-muted;
  }

  .dl-card-btns {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .download-progress-bar {
    width: 100%;
    height: 4px;
    background: rgba(255, 255, 255, 0.06);
    border-radius: 2px;
    overflow: hidden;
  }

  .download-progress-fill {
    height: 100%;
    background: $accent;
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .download-status {
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    padding: 2px 8px;
    border-radius: 4px;
  }

  .status-downloading {
    background: rgba($accent, 0.15);
    color: $accent-hover;
  }

  .status-complete {
    background: rgba($green, 0.15);
    color: $green;
  }

  .status-error {
    background: rgba($red, 0.15);
    color: $red;
  }

  .status-cancelled {
    background: rgba(255, 255, 255, 0.05);
    color: $text-muted;
  }

  .status-processing {
    background: rgba($cyan, 0.15);
    color: $cyan;
  }

  .status-queued {
    background: rgba(255, 255, 255, 0.05);
    color: $text-dim;
  }

  .no-results {
    padding: 20px;
    text-align: center;
    color: $text-muted;
    font-size: 14px;
  }

  @keyframes pageIn {
    from {
      opacity: 0;
      transform: translateY(4px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes modalIn {
    from {
      opacity: 0;
      transform: scale(0.98);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }
</style>
