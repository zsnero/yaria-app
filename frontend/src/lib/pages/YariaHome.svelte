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
  let containerFormat = $state('mp4');
  let formats = $state<{ video: any[]; audio: any[] }>({ video: [], audio: [] });

  const containerFormats = ['mp4', 'mkv', 'webm'];

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

  // Background downloads (moved from main view when user clicks "Download Another")
  interface BgDownload {
    id: string;
    title: string;
    thumbnail: string;
    status: string;
    percent: number;
    speed: string;
    eta: string;
    error: string;
  }
  let bgDownloads = $state<BgDownload[]>([]);

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
        // FileSize is a pre-formatted string from yt-dlp (e.g. "175.47MiB") — display as-is
        size: typeof f.filesize === 'string' && f.filesize ? f.filesize : '',
      }));
    }
    // Don't show default resolutions — just "Best" until loadFormats completes
    return [];
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
      if (dlPollInterval) { clearInterval(dlPollInterval); dlPollInterval = null; }
    };
  });

  async function initDeps() {
    depsReady = false;
    depsError = '';
    depsMessage = 'Preparing download tools...';

    // Always call DownloadService.InitDeps() to create the yt-dlp wrapper (d.dl).
    // DepsService.CheckDeps() only checks if binaries exist on disk —
    // it does NOT initialize the internal downloader needed for StartDownload/FetchMetadata.
    // InitDeps() returns immediately; the goroutine finishes fast when binaries exist.
    // Set depsReady immediately (like old JS); startDownload has error handling if d.dl isn't ready yet.
    try {
      await api.downloads.initDeps();
      depsReady = true;
      depsMessage = '';
    } catch (err: any) {
      depsError = err?.message || 'Failed to initialize';
    }
  }

  async function loadDownloadDir() {
    // Use localStorage override if set, otherwise fetch default from Go backend
    try {
      const saved = localStorage.getItem('yaria_download_dir');
      if (saved) {
        downloadDir = saved;
        return;
      }
    } catch { /* ignore */ }
    try {
      downloadDir = await api.downloads.getDownloadDir() || '';
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
      const result = await api.downloads.listFormats(fetchUrl);
      if (result && !result.error) {
        formats = {
          video: result.video || [],
          audio: result.audio || [],
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

  let dlPollInterval: ReturnType<typeof setInterval> | null = null;

  function startPollIfNeeded() {
    if (dlPollInterval) return; // already polling
    dlPollInterval = setInterval(async () => {
      try {
        const downloads = await api.downloads.list();
        if (!downloads) return;
        // Build set of IDs we care about
        const trackedIds = new Set<string>();
        if (dlId) trackedIds.add(dlId);
        for (const bg of bgDownloads) trackedIds.add(bg.id);

        if (trackedIds.size === 0) {
          // Nothing to track
          if (dlPollInterval) { clearInterval(dlPollInterval); dlPollInterval = null; }
          return;
        }

        for (const dl of downloads) {
          if (trackedIds.has(dl.id)) {
            updateActiveDownload(dl);
          }
        }

        // Stop polling if all tracked downloads are terminal
        const allDone = [...trackedIds].every(id => {
          // Check foreground
          if (id === dlId) {
            const s = dlStatus.toLowerCase();
            return s === 'complete' || s === 'error' || s === 'cancelled';
          }
          // Check background
          const bg = bgDownloads.find(d => d.id === id);
          if (bg) {
            return bg.status === 'complete' || bg.status === 'error' || bg.status === 'cancelled';
          }
          return true;
        });
        if (allDone) {
          if (dlPollInterval) { clearInterval(dlPollInterval); dlPollInterval = null; }
        }
      } catch { /* ignore */ }
    }, 1000);
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
      // Check for existing/in-progress download of same URL
      try {
        const existing = await api.downloads.checkExistingDownload(url, downloadDir);
        if (existing?.exists) {
          if (!confirm((existing.message || 'Download already exists') + '\n\nDownload anyway?')) {
            downloading = false;
            return;
          }
        }
      } catch { /* ignore check */ }

      const result = await api.downloads.start(url, selectedFormat, downloadDir, audioOnly, audioFormat, audioOnly ? '' : containerFormat);

      // Check for error in result (e.g. "downloader not initialized")
      if (result?.error) {
        downloading = false;
        fetchError = result.error;
        return;
      }

      dlId = result?.id || '';
      if (!dlId) {
        downloading = false;
        fetchError = 'Download failed to start (no ID returned)';
        return;
      }

      toastSuccess('Download started');

      // Start polling for progress (covers foreground + background downloads)
      startPollIfNeeded();
    } catch (err: any) {
      downloading = false;
      fetchError = err?.message || 'Download failed to start';
    }
  }

  function updateActiveDownload(data: any) {
    if (!data?.id) return;

    // Update background download if it matches
    const bgIdx = bgDownloads.findIndex(d => d.id === data.id);
    if (bgIdx >= 0) {
      const bg = bgDownloads[bgIdx];
      if (data.percent != null) bg.percent = Math.min(data.percent, 100);
      if (data.speed) bg.speed = data.speed;
      if (data.eta) bg.eta = data.eta;
      if (data.status) bg.status = data.status;
      if (data.error) bg.error = data.error;
      if (data.title && data.title !== data.url) bg.title = data.title;
      bgDownloads = [...bgDownloads]; // trigger reactivity
      return;
    }

    // Update current (foreground) download
    if (dlId && data.id !== dlId) return;

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
    } else if (data.status === 'downloading' || data.status === 'processing') {
      dlMessage = '';
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
    // Move current download to background if it's still active
    if (dlId && dlStatus && dlStatus !== 'COMPLETE' && dlStatus !== 'ERROR' && dlStatus !== 'CANCELLED') {
      bgDownloads = [...bgDownloads, {
        id: dlId,
        title: meta?.title || url,
        thumbnail: meta?.thumbnail || '',
        status: dlStatus.toLowerCase(),
        percent: dlPercent,
        speed: dlSpeed,
        eta: dlEta,
        error: '',
      }];
    }

    // Don't clear the poll — it now updates background downloads too
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
    containerFormat = 'mp4';
    formats = { video: [], audio: [] };
    url = '';
    setTimeout(() => urlInput?.focus(), 50);
  }

  async function cancelBgDownload(id: string) {
    try {
      await api.downloads.cancel(id);
      const bg = bgDownloads.find(d => d.id === id);
      if (bg) bg.status = 'cancelled';
      bgDownloads = [...bgDownloads];
    } catch { /* ignore */ }
  }

  function dismissBgDownload(id: string) {
    bgDownloads = bgDownloads.filter(d => d.id !== id);
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
                  {#if opt.size}
                    <div class="format-card-ext">{opt.size}</div>
                  {/if}
                </button>
              {/each}
            {/if}
          </div>

          <!-- Container format (video only) -->
          {#if !audioOnly}
            <div class="format-header" style="margin-top:12px;">
              <span class="format-section-title">Format</span>
            </div>
            <div class="format-grid">
              {#each containerFormats as fmt}
                <button
                  class="format-card format-card-sm"
                  class:selected={containerFormat === fmt}
                  onclick={() => { containerFormat = fmt; }}
                  use:ripple
                >
                  <div class="format-card-label">.{fmt}</div>
                </button>
              {/each}
            </div>
          {/if}
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

  <!-- Background downloads -->
  {#if bgDownloads.length > 0}
    <div class="bg-downloads">
      {#each bgDownloads as bg (bg.id)}
        <div class="bg-dl-card" transition:slide={{ duration: 200 }}>
          <div class="bg-dl-row">
            {#if bg.thumbnail}
              <img class="bg-dl-thumb" src={bg.thumbnail} alt="" onerror={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none'; }} />
            {/if}
            <div class="bg-dl-info">
              <div class="bg-dl-title">{bg.title}</div>
              <div class="bg-dl-status-row">
                <span class="download-status {getStatusClass(bg.status)}">{bg.status.toUpperCase()}</span>
                <span class="bg-dl-pct">{bg.percent.toFixed(1)}%</span>
                {#if bg.speed && bg.status === 'downloading'}
                  <span class="bg-dl-speed">{bg.speed}</span>
                {/if}
                {#if bg.eta && bg.status === 'downloading'}
                  <span class="bg-dl-eta">ETA: {bg.eta}</span>
                {/if}
              </div>
              <div class="download-progress-bar">
                <div class="download-progress-fill" style="width: {bg.percent}%"></div>
              </div>
            </div>
            <div class="bg-dl-actions">
              {#if bg.status === 'downloading' || bg.status === 'metadata' || bg.status === 'queued'}
                <button class="btn btn-ghost btn-xs" onclick={() => cancelBgDownload(bg.id)} title="Cancel">x</button>
              {:else}
                <button class="btn btn-ghost btn-xs" onclick={() => dismissBgDownload(bg.id)} title="Dismiss">x</button>
              {/if}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
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

  .format-card-sm {
    padding: 6px 14px;
    min-width: 55px;
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

  // Background downloads
  .bg-downloads {
    margin-top: 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .bg-dl-card {
    @include glass;
    border-radius: $radius-sm;
    padding: 12px 16px;
  }

  .bg-dl-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .bg-dl-thumb {
    width: 48px;
    height: 28px;
    object-fit: cover;
    border-radius: 4px;
    flex-shrink: 0;
    background: rgba(255, 255, 255, 0.02);
  }

  .bg-dl-info {
    flex: 1;
    min-width: 0;
  }

  .bg-dl-title {
    font-size: 12px;
    font-weight: 500;
    color: $text;
    @include truncate;
    margin-bottom: 4px;
  }

  .bg-dl-status-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;
    flex-wrap: wrap;
  }

  .bg-dl-pct {
    font-size: 11px;
    font-weight: 600;
    color: $text;
  }

  .bg-dl-speed,
  .bg-dl-eta {
    font-size: 11px;
    color: $text-dim;
  }

  .bg-dl-actions {
    flex-shrink: 0;
  }

  .btn-xs {
    padding: 2px 8px;
    font-size: 12px;
    line-height: 1;
    min-width: 24px;
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
