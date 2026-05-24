<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../../api/wails';
  import Spinner from '../../components/Spinner.svelte';
  import FilePicker from '../../components/FilePicker.svelte';

  // Media folders
  let mediaFolders = $state<any>(null);
  let mediaFoldersLoading = $state(true);
  let mediaFoldersError = $state('');
  let scanning = $state(false);
  let scanMsg = $state('');
  let showFilePicker = $state(false);
  let filePickerCategory = $state('movie');

  // HW Acceleration
  let hwAccelLoading = $state(true);
  let hwAccelInfo = $state<any>(null);

  async function loadMediaFolders() {
    mediaFoldersLoading = true;
    mediaFoldersError = '';
    try {
      const folders = await api.media.getMediaFolders();
      if (folders.error) {
        mediaFoldersError = 'Media library not available in this build';
        mediaFolders = null;
      } else {
        mediaFolders = folders;
      }
    } catch {
      mediaFolders = null;
      mediaFoldersError = 'Could not load media folders';
    }
    mediaFoldersLoading = false;
  }

  async function loadHWAccel() {
    hwAccelLoading = true;
    try {
      hwAccelInfo = await api.media.detectHWAccel();
    } catch {
      hwAccelInfo = null;
    }
    hwAccelLoading = false;
  }

  function addFolder(category: string) {
    filePickerCategory = category;
    showFilePicker = true;
  }

  async function handleFolderSelected(dir: string) {
    showFilePicker = false;
    if (dir) {
      await api.media.addMediaFolder(dir, filePickerCategory);
      loadMediaFolders();
    }
  }

  async function removeFolder(path: string, type: string) {
    await api.media.removeMediaFolder(path, type);
    loadMediaFolders();
  }

  async function scanMedia() {
    scanning = true;
    scanMsg = '';
    try {
      await api.media.scan();
      const poll = setInterval(async () => {
        try {
          const s = await api.media.scanStatus();
          scanMsg = s?.message || '';
          if (!s?.scanning) {
            clearInterval(poll);
            scanning = false;
          }
        } catch {
          clearInterval(poll);
          scanning = false;
        }
      }, 1000);
    } catch {
      scanning = false;
    }
  }

  onMount(() => {
    loadMediaFolders();
    loadHWAccel();
  });
</script>

<div class="stg-panel active">
  <h3 class="stg-panel-title">Local Media</h3>

  <!-- Folders -->
  <div class="setting-group">
    <div class="setting-label">Media Folders</div>
    <div class="setting-desc">Directories scanned for your media library with TMDB metadata.</div>
    {#if mediaFoldersLoading}
      <Spinner size={18} message="Loading..." />
    {:else if mediaFoldersError}
      <span class="text-muted">{mediaFoldersError}</span>
    {:else if mediaFolders}
      {@const movieDirs = mediaFolders.movie_dirs || []}
      {@const tvDirs = mediaFolders.tv_dirs || []}
      {@const videoDirs = mediaFolders.video_dirs || []}

      {#each [{ label: 'Movies', type: 'movie', dirs: movieDirs }, { label: 'TV Shows', type: 'tv', dirs: tvDirs }, { label: 'Other Videos', type: 'video', dirs: videoDirs }] as group}
        <div class="folder-group">
          <div class="folder-group-label">{group.label}</div>
          {#if group.dirs.length === 0}
            <div class="text-muted italic">No folders configured</div>
          {/if}
          {#each group.dirs as dir}
            <div class="folder-item">
              <span class="folder-path" title={dir}>{dir}</span>
              <button class="btn btn-ghost btn-sm remove-btn" onclick={() => removeFolder(dir, group.type)}>Remove</button>
            </div>
          {/each}
          <button class="btn btn-ghost btn-sm" onclick={() => addFolder(group.type)}>+ Add Folder</button>
        </div>
      {/each}

      <div class="scan-row">
        <button
          class="btn btn-primary btn-sm"
          onclick={scanMedia}
          disabled={scanning}
        >
          {scanning ? 'Scanning...' : 'Scan Now'}
        </button>
        {#if scanMsg}
          <span class="scan-msg">{scanMsg}</span>
        {/if}
      </div>
    {/if}
  </div>

  <!-- HW Accel -->
  <div class="setting-group">
    <div class="setting-label">Hardware Acceleration</div>
    <div class="setting-desc">Detected hardware video encoders for transcoding.</div>
    {#if hwAccelLoading}
      <Spinner size={18} message="Detecting..." />
    {:else if hwAccelInfo?.available}
      <div class="codec-list">
        {#each hwAccelInfo.encoders || [] as enc}
          <div class="codec-row">
            <span class="text-dim">{enc.name}</span>
            {#if enc.available}
              <span class="text-green">Available</span>
            {:else}
              <span class="text-muted">Not found</span>
            {/if}
          </div>
        {/each}
      </div>
    {:else if hwAccelInfo}
      <span class="text-muted">FFmpeg not found. Install FFmpeg to enable transcoding.</span>
    {:else}
      <span class="text-muted">Could not detect hardware acceleration</span>
    {/if}
  </div>
</div>

{#if showFilePicker}
  <FilePicker
    onSelect={handleFolderSelected}
    onClose={() => { showFilePicker = false; }}
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

  .folder-group {
    margin-top: 12px;
    margin-bottom: 12px;
  }

  .folder-group-label {
    color: $text-dim;
    font-weight: 600;
    font-size: 13px;
    margin-bottom: 4px;
  }

  .folder-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
    padding: 2px 0;
  }

  .folder-path {
    color: $text;
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .remove-btn {
    color: $red !important;
    flex-shrink: 0;
    font-size: 11px;
  }

  .scan-row {
    margin-top: 16px;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .scan-msg {
    font-size: 12px;
    color: $text-muted;
  }

  .italic {
    font-style: italic;
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
</style>
