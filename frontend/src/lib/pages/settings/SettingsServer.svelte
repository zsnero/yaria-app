<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../../api/wails';
  import Spinner from '../../components/Spinner.svelte';

  // Media Server
  let mediaServerLoading = $state(true);
  let mediaServerStatus = $state<any>(null);
  let msPort = $state(8096);
  let msPin = $state('');
  let msError = $state('');

  // DLNA
  let dlnaLoading = $state(true);
  let dlnaStatus = $state<any>(null);
  let dlnaError = $state('');

  async function loadMediaServer() {
    mediaServerLoading = true;
    try {
      mediaServerStatus = await api.mediaServer.status();
    } catch {
      mediaServerStatus = null;
    }
    mediaServerLoading = false;
  }

  async function startMediaServer() {
    msError = '';
    try {
      await api.mediaServer.setPort(msPort);
      if (msPin) await api.mediaServer.setPin(msPin);
      const result = await api.mediaServer.start();
      if (result?.error) {
        msError = result.error;
      } else {
        loadMediaServer();
      }
    } catch (e: any) {
      msError = e.message || 'Failed to start';
    }
  }

  async function stopMediaServer() {
    try {
      await api.mediaServer.stop();
      loadMediaServer();
    } catch {}
  }

  async function loadDLNA() {
    dlnaLoading = true;
    try {
      dlnaStatus = await api.dlna.status();
    } catch {
      dlnaStatus = null;
    }
    dlnaLoading = false;
  }

  async function startDLNA() {
    dlnaError = '';
    try {
      const result = await api.dlna.start();
      if (result?.error) {
        dlnaError = result.error;
      } else {
        loadDLNA();
      }
    } catch (e: any) {
      dlnaError = e.message || 'Failed to start';
    }
  }

  async function stopDLNA() {
    try {
      await api.dlna.stop();
      loadDLNA();
    } catch {}
  }

  onMount(() => {
    loadMediaServer();
    loadDLNA();
  });
</script>

<div class="stg-panel active">
  <h3 class="stg-panel-title">Media Server</h3>

  <!-- LAN Server -->
  <div class="setting-group">
    <div class="setting-label">LAN Server</div>
    <div class="setting-desc">Stream media to phones, tablets, and browsers on your network.</div>
    {#if mediaServerLoading}
      <Spinner size={18} message="Loading..." />
    {:else if mediaServerStatus?.running}
      <div class="server-info">
        <div class="license-badge-row">
          <span class="badge badge-pro">RUNNING</span>
          <span>Port {mediaServerStatus.port}</span>
        </div>
        <div class="license-detail">URL: <a href={mediaServerStatus.url} target="_blank" class="link">{mediaServerStatus.url}</a></div>
        {#if mediaServerStatus.pin}
          <div class="text-muted" style="font-size:12px;">PIN protected</div>
        {:else}
          <div class="text-yellow" style="font-size:12px;">No PIN set (open access)</div>
        {/if}
      </div>
      <button class="btn btn-ghost btn-sm deactivate-btn" onclick={stopMediaServer}>Stop Server</button>
    {:else if mediaServerStatus}
      <div class="server-config">
        <div class="server-config-row">
          <div>
            <label class="appearance-label">Port</label>
            <input type="number" class="setting-input port-input" bind:value={msPort} />
          </div>
          <div>
            <label class="appearance-label">PIN (optional)</label>
            <input type="text" class="setting-input" placeholder="No PIN" maxlength="8" bind:value={msPin} />
          </div>
        </div>
      </div>
      <button class="btn btn-primary btn-sm" onclick={startMediaServer}>Start Server</button>
      {#if msError}
        <span class="msg-error inline">{msError}</span>
      {/if}
    {:else}
      <span class="text-muted">Media server not available</span>
    {/if}
  </div>

  <!-- DLNA -->
  <div class="setting-group">
    <div class="setting-label">DLNA / UPnP</div>
    <div class="setting-desc">Advertise to smart TVs, consoles, and DLNA devices.</div>
    {#if dlnaLoading}
      <Spinner size={18} message="Loading..." />
    {:else if dlnaStatus?.running}
      <div class="server-info">
        <div class="license-badge-row">
          <span class="badge badge-pro">ACTIVE</span>
          <span>Port {dlnaStatus.port}</span>
        </div>
        <div class="text-muted" style="font-size:12px;">Devices on your network can now discover and browse your media.</div>
      </div>
      <button class="btn btn-ghost btn-sm deactivate-btn" onclick={stopDLNA}>Stop DLNA</button>
    {:else if dlnaStatus}
      <button class="btn btn-primary btn-sm" onclick={startDLNA}>Start DLNA Server</button>
      {#if dlnaError}
        <span class="msg-error inline">{dlnaError}</span>
      {/if}
    {:else}
      <span class="text-muted">DLNA not available</span>
    {/if}
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

  .server-info {
    font-size: 13px;
    line-height: 2;
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

  .license-detail {
    color: $text-dim;
  }

  .link {
    color: $accent;
  }

  .server-config-row {
    display: flex;
    gap: 8px;
    align-items: flex-end;
    margin-bottom: 12px;
  }

  .port-input {
    width: 100px;
    padding: 6px 10px;
    font-size: 13px;
  }

  .appearance-label {
    font-size: 12px;
    color: $text-dim;
    display: block;
    margin-bottom: 4px;
  }

  .deactivate-btn {
    color: $red !important;
  }

  .msg-error {
    font-size: 13px;
    color: $red;

    &.inline {
      margin-left: 10px;
    }
  }

  .text-muted {
    color: $text-muted;
    font-size: 13px;
  }

  .text-yellow {
    color: $yellow;
  }
</style>
