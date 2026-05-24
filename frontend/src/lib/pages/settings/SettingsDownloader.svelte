<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../../api/wails';
  import { toastSuccess, toastError } from '../../stores/toast';

  let speedLimit = $state(0);
  let maxConcurrent = $state(3);

  const speedOptions = [
    { value: 0, label: 'Unlimited' },
    { value: 524288, label: '512 KB/s' },
    { value: 1048576, label: '1 MB/s' },
    { value: 2097152, label: '2 MB/s' },
    { value: 5242880, label: '5 MB/s' },
    { value: 10485760, label: '10 MB/s' },
  ];

  async function handleSpeedChange() {
    try {
      await api.downloads.setSpeedLimit(speedLimit);
      toastSuccess('Settings saved');
    } catch (err: any) {
      toastError('Failed to save: ' + (err.message || ''));
    }
  }

  async function handleConcurrentChange() {
    try {
      await api.downloads.setMaxConcurrent(maxConcurrent);
      toastSuccess('Settings saved');
    } catch (err: any) {
      toastError('Failed to save: ' + (err.message || ''));
    }
  }

  onMount(async () => {
    try {
      const limit = await api.downloads.getSpeedLimit();
      if (limit) speedLimit = limit;
    } catch {}

    try {
      const maxC = await api.downloads.getMaxConcurrent();
      if (maxC) maxConcurrent = maxC;
    } catch {}
  });
</script>

<div class="stg-panel active">
  <h3 class="stg-panel-title">Yaria Downloader</h3>
  <div class="setting-group">
    <div class="setting-label">Download Speed Limit</div>
    <div class="setting-desc">Limit bandwidth for video and audio downloads.</div>
    <select class="setting-input" bind:value={speedLimit} onchange={handleSpeedChange}>
      {#each speedOptions as opt}
        <option value={opt.value}>{opt.label}</option>
      {/each}
    </select>
  </div>
  <div class="setting-group">
    <div class="setting-label">Concurrent Downloads</div>
    <div class="setting-desc">Maximum number of simultaneous downloads.</div>
    <select class="setting-input" bind:value={maxConcurrent} onchange={handleConcurrentChange}>
      {#each [1, 2, 3, 5, 10] as n}
        <option value={n}>{n}</option>
      {/each}
    </select>
  </div>
</div>

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
</style>
