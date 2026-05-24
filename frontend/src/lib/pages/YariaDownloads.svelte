<script lang="ts">
  import { onMount } from 'svelte';
  import { slide } from 'svelte/transition';
  import { tweened } from 'svelte/motion';
  import { cubicOut } from 'svelte/easing';
  import { api } from '../api/wails';
  import ConfirmDialog from '../components/ConfirmDialog.svelte';
  import Spinner from '../components/Spinner.svelte';

  interface DownloadItem {
    id: string;
    title?: string;
    url?: string;
    thumbnail?: string;
    status: string;
    percent?: number;
    speed?: string;
    eta?: string;
    file_size?: number;
    started_at?: string;
    download_dir?: string;
    file_path?: string;
    error?: string;
  }

  let downloads = $state<DownloadItem[]>([]);
  let loading = $state(true);
  let confirmMessage = $state('');
  let confirmAction = $state<(() => void) | null>(null);

  // Tweened progress values per download item
  type TweenedStore = ReturnType<typeof tweened<number>>;
  let progressStores: Map<string, TweenedStore> = new Map();
  let progressValues = $state<Record<string, number>>({});

  function ensureProgressStore(id: string, initial = 0) {
    if (!progressStores.has(id)) {
      const store = tweened<number>(initial, { duration: 400, easing: cubicOut });
      progressStores.set(id, store);
      store.subscribe(v => {
        progressValues = { ...progressValues, [id]: v };
      });
    }
  }

  function updateProgress(id: string, value: number) {
    ensureProgressStore(id, value);
    progressStores.get(id)!.set(value);
  }

  function getProgressValue(id: string, fallback: number): number {
    return progressValues[id] ?? fallback;
  }

  let active = $derived(
    downloads.filter(d => ['downloading', 'queued', 'metadata', 'processing'].includes(d.status))
  );
  let history = $derived(
    downloads.filter(d => ['complete', 'error', 'cancelled'].includes(d.status))
  );
  let isEmpty = $derived(downloads.length === 0);

  // Cleanups
  let eventCleanups: (() => void)[] = [];
  let refreshInterval: ReturnType<typeof setInterval> | null = null;

  onMount(() => {
    refreshList();

    // Auto-refresh every 3 seconds
    refreshInterval = setInterval(refreshList, 3000);

    // Subscribe to download progress
    const off = api.events.on('download-progress', (data: any) => {
      updateItemUI(data);
    });
    eventCleanups.push(off);

    return () => {
      eventCleanups.forEach(fn => fn());
      eventCleanups = [];
      if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
      }
    };
  });

  async function refreshList() {
    try {
      const result = await api.downloads.list();
      downloads = Array.isArray(result) ? result : [];
      // Sync tweened progress stores
      for (const dl of downloads) {
        if (dl.percent != null) {
          updateProgress(dl.id, Math.min(dl.percent, 100));
        }
      }
    } catch {
      // keep existing data on error
    } finally {
      loading = false;
    }
  }

  function updateItemUI(data: any) {
    if (!data?.id) return;

    const idx = downloads.findIndex(d => d.id === data.id);
    if (idx === -1) {
      // New item, refresh the list
      refreshList();
      return;
    }

    // Update tweened progress
    if (data.percent != null) {
      updateProgress(data.id, Math.min(data.percent, 100));
    }

    // Update in place
    downloads = downloads.map(d => {
      if (d.id !== data.id) return d;
      return {
        ...d,
        ...(data.percent != null ? { percent: Math.min(data.percent, 100) } : {}),
        ...(data.speed ? { speed: data.speed } : {}),
        ...(data.eta ? { eta: data.eta } : {}),
        ...(data.status ? { status: data.status } : {}),
        ...(data.error ? { error: data.error } : {}),
      };
    });

    // Refresh on terminal states
    if (['complete', 'error', 'cancelled'].includes(data.status)) {
      setTimeout(refreshList, 500);
    }
  }

  async function clearCompleted() {
    try {
      for (const dl of history) {
        await api.downloads.remove(dl.id);
      }
      refreshList();
    } catch { /* ignore */ }
  }

  async function cancelDownload(id: string) {
    try {
      await api.downloads.cancel(id);
      refreshList();
    } catch { /* ignore */ }
  }

  async function removeDownload(id: string) {
    try {
      await api.downloads.remove(id);
      refreshList();
    } catch { /* ignore */ }
  }

  function deleteDownload(id: string) {
    confirmMessage = 'Delete downloaded files from disk?';
    confirmAction = async () => {
      confirmMessage = '';
      confirmAction = null;
      try {
        await api.downloads.remove(id);
        refreshList();
      } catch { /* ignore */ }
    };
  }

  function cancelConfirm() {
    confirmMessage = '';
    confirmAction = null;
  }

  function fmtBytes(bytes: number): string {
    if (!bytes || bytes <= 0) return '';
    if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB';
    if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB';
    if (bytes >= 1024) return (bytes / 1024).toFixed(1) + ' KB';
    return bytes + ' B';
  }

  function getStatusClass(status: string): string {
    switch (status) {
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

<div class="yaria-downloads-page">
  <!-- Header -->
  <div class="yd-header">
    <div>
      <h2 class="section-heading">Downloads</h2>
      <p class="section-subtitle">Manage your video downloads</p>
    </div>
    <button
      class="btn btn-ghost btn-sm"
      onclick={clearCompleted}
      disabled={history.length === 0}
    >
      Clear Completed
    </button>
  </div>

  {#if loading && downloads.length === 0}
    <Spinner message="Loading downloads..." />
  {:else if isEmpty}
    <div class="no-results">
      No downloads yet. Go to the Yaria tab to start downloading.
    </div>
  {:else}
    <!-- Active downloads -->
    {#if active.length > 0}
      <h3 class="yd-section-title">Active</h3>
      {#each active as dl (dl.id)}
        {@const progressVal = getProgressValue(dl.id, dl.percent ?? 0)}
        <div class="download-item" transition:slide={{ duration: 250 }}>
          {#if dl.thumbnail}
            <img
              class="dl-item-thumb"
              src={dl.thumbnail}
              alt=""
              onerror={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none'; }}
            />
          {/if}
          <div class="download-item-info">
            <div class="download-item-title">{dl.title || dl.url || 'Download'}</div>
            <div class="download-item-details">
              <span class="download-status {getStatusClass(dl.status)}">{dl.status.toUpperCase()}</span>
              {#if (dl.percent ?? 0) > 0}
                <span class="dl-item-pct">{dl.percent?.toFixed(1)}%</span>
              {/if}
              {#if dl.speed}
                <span class="dl-item-speed">{dl.speed}</span>
              {/if}
              {#if dl.eta}
                <span class="dl-item-eta">ETA: {dl.eta}</span>
              {/if}
              {#if dl.file_size && dl.file_size > 0}
                <span class="dl-item-size">{fmtBytes(dl.file_size)}</span>
              {/if}
              {#if dl.started_at}
                <span class="dl-item-time">{dl.started_at}</span>
              {/if}
            </div>
            <div class="download-progress-bar">
              <div
                class="download-progress-fill"
                style="width: {Math.min(progressVal, 100)}%"
              ></div>
            </div>
            {#if dl.error}
              <div class="dl-item-error">{dl.error}</div>
            {/if}
          </div>
          <div class="download-item-actions">
            <button
              class="btn btn-ghost btn-sm"
              onclick={() => cancelDownload(dl.id)}
            >
              Cancel
            </button>
          </div>
        </div>
      {/each}
    {/if}

    <!-- History -->
    {#if history.length > 0}
      <h3 class="yd-section-title" class:with-margin={active.length > 0}>History</h3>
      {#each history as dl (dl.id)}
        <div class="download-item" transition:slide={{ duration: 250 }}>
          {#if dl.thumbnail}
            <img
              class="dl-item-thumb"
              src={dl.thumbnail}
              alt=""
              onerror={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none'; }}
            />
          {/if}
          <div class="download-item-info">
            <div class="download-item-title">{dl.title || dl.url || 'Download'}</div>
            <div class="download-item-details">
              <span class="download-status {getStatusClass(dl.status)}">{dl.status.toUpperCase()}</span>
              {#if dl.file_size && dl.file_size > 0}
                <span class="dl-item-size">{fmtBytes(dl.file_size)}</span>
              {/if}
              {#if dl.started_at}
                <span class="dl-item-time">{dl.started_at}</span>
              {/if}
            </div>
            {#if dl.download_dir}
              <div class="dl-item-location" title={dl.file_path || ''}>{dl.download_dir}</div>
            {/if}
            {#if dl.error}
              <div class="dl-item-error">{dl.error}</div>
            {/if}
          </div>
          <div class="download-item-actions">
            {#if dl.status === 'complete'}
              <button class="btn btn-primary btn-sm" title="Play">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M8 5v14l11-7z"/>
                </svg>
              </button>
            {/if}
            <button
              class="btn btn-ghost btn-sm"
              onclick={() => removeDownload(dl.id)}
              title="Remove from list"
            >
              Remove
            </button>
            {#if dl.status === 'complete'}
              <button
                class="btn btn-ghost btn-sm dl-delete-btn"
                onclick={() => deleteDownload(dl.id)}
                title="Delete files from disk"
              >
                Delete
              </button>
            {/if}
          </div>
        </div>
      {/each}
    {/if}
  {/if}
</div>

{#if confirmMessage && confirmAction}
  <ConfirmDialog
    message={confirmMessage}
    onConfirm={confirmAction}
    onCancel={cancelConfirm}
  />
{/if}

<style lang="scss">
  @use '../styles/variables' as *;

  .yaria-downloads-page {
    padding: 32px 24px 48px;
    max-width: 800px;
    margin: 0 auto;
    animation: pageIn 0.3s ease;
  }

  .yd-header {
    @include flex-between;
    margin-bottom: 24px;
  }

  .section-heading {
    font-size: 20px;
    font-weight: 600;
    color: $text;
    margin-bottom: 4px;
  }

  .section-subtitle {
    font-size: 13px;
    color: $text-muted;
  }

  .yd-section-title {
    font-size: 13px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: $text-dim;
    margin-bottom: 12px;

    &.with-margin {
      margin-top: 24px;
    }
  }

  .download-item {
    @include glass;
    border-radius: $radius-sm;
    padding: 16px;
    margin-bottom: 10px;
    display: flex;
    gap: 14px;
    align-items: flex-start;
    transition: background 0.15s;

    &:hover {
      @include glass-hover;
    }
  }

  .dl-item-thumb {
    width: 80px;
    height: 45px;
    object-fit: cover;
    border-radius: 6px;
    flex-shrink: 0;
    background: rgba(255, 255, 255, 0.02);
  }

  .download-item-info {
    flex: 1;
    min-width: 0;
  }

  .download-item-title {
    font-size: 13px;
    font-weight: 500;
    color: $text;
    margin-bottom: 6px;
    @include truncate;
  }

  .download-item-details {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
    margin-bottom: 8px;
    font-size: 12px;
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

  .dl-item-pct {
    font-weight: 600;
    color: $text;
  }

  .dl-item-speed {
    color: $text-dim;
  }

  .dl-item-eta {
    color: $text-dim;
  }

  .dl-item-size {
    color: $text-muted;
  }

  .dl-item-time {
    color: $text-muted;
    font-size: 11px;
  }

  .dl-item-location {
    font-size: 11px;
    color: $text-muted;
    margin-bottom: 4px;
    @include truncate;
  }

  .dl-item-error {
    font-size: 12px;
    color: $red;
    margin-top: 4px;
  }

  .download-progress-bar {
    width: 100%;
    height: 3px;
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

  .download-item-actions {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
    align-items: center;
  }

  .dl-delete-btn {
    color: $red !important;

    &:hover {
      background: rgba($red, 0.1) !important;
    }
  }

  .no-results {
    text-align: center;
    padding: 48px 0;
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
</style>
