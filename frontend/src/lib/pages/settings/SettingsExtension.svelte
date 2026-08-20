<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../../api/wails';
  import { toastSuccess, toastError } from '../../stores/toast';

  let extEnabled = $state(true);
  let extRunning = $state(false);
  let extHost = $state('127.0.0.1');
  let extPort = $state(19547);
  let extToken = $state('');
  let extBusy = $state(false);
  let tokenVisible = $state(false);
  let loadError = $state('');

  function applyExtStatus(s: any) {
    if (!s || s.error) {
      if (s?.error) loadError = String(s.error);
      return;
    }
    loadError = '';
    extEnabled = !!s.enabled;
    extRunning = !!s.running;
    extHost = s.host || '127.0.0.1';
    if (s.port) extPort = s.port;
    if (typeof s.token === 'string') extToken = s.token;
  }

  async function loadExtension() {
    try {
      if (!api.extension?.getStatus) {
        loadError = 'Bridge API not available — rebuild/restart Yaria';
        return;
      }
      const s = await api.extension.getStatus();
      applyExtStatus(s);
    } catch (e: any) {
      loadError = e?.message || 'Could not load extension status';
    }
  }

  async function toggleExtension() {
    extBusy = true;
    try {
      const s = await api.extension.setEnabled(!extEnabled);
      if (s?.error) throw new Error(s.error);
      applyExtStatus(s);
      toastSuccess(extEnabled ? 'Bridge on' : 'Bridge off');
    } catch (err: any) {
      toastError(err.message || 'Failed to update');
      await loadExtension();
    } finally {
      extBusy = false;
    }
  }

  async function savePort() {
    extBusy = true;
    try {
      const s = await api.extension.setPort(Number(extPort));
      if (s?.error) throw new Error(s.error);
      applyExtStatus(s);
      toastSuccess('Port saved');
    } catch (err: any) {
      toastError(err.message || 'Failed to save port');
      await loadExtension();
    } finally {
      extBusy = false;
    }
  }

  async function regenToken() {
    extBusy = true;
    try {
      const s = await api.extension.regenerateToken();
      if (s?.error) throw new Error(s.error);
      applyExtStatus(s);
      tokenVisible = true;
      toastSuccess('New token generated — update Yaria Bridge');
    } catch (err: any) {
      toastError(err.message || 'Failed to regenerate token');
    } finally {
      extBusy = false;
    }
  }

  async function copyToken() {
    try {
      await navigator.clipboard.writeText(extToken);
      toastSuccess('Token copied');
    } catch {
      toastError('Could not copy');
    }
  }

  onMount(() => {
    loadExtension();
  });
</script>

<div class="stg-panel active">
  <h3 class="stg-panel-title">Yaria Bridge</h3>
  <p class="intro">
    Pair the free <strong>Yaria Bridge</strong> browser add-on so media from web pages
    appears on the main <strong>Downloads</strong> page.
  </p>

  {#if loadError}
    <div class="banner err">{loadError}</div>
  {/if}

  <div class="setting-group">
    <div class="setting-label">Bridge integration</div>
    <div class="setting-desc">
      When on, Yaria listens on localhost for Yaria Bridge. When off, the connection is stopped.
    </div>
    <div class="ext-row">
      <button
        class="btn btn-sm"
        class:btn-primary={extEnabled}
        class:btn-ghost={!extEnabled}
        disabled={extBusy}
        onclick={toggleExtension}
      >
        {extEnabled ? 'Enabled' : 'Disabled'}
      </button>
      <span class="ext-pill" class:on={extRunning} class:off={!extRunning}>
        {extRunning ? 'Bridge running' : 'Bridge stopped'}
      </span>
    </div>
  </div>

  <div class="setting-group">
    <div class="setting-label">Bridge address</div>
    <div class="setting-desc">Default port is 19547. Match this in Yaria Bridge settings.</div>
    <div class="ext-row">
      <code class="ext-code">{extHost}:</code>
      <input class="ext-input" type="number" min="1" max="65535" bind:value={extPort} disabled={extBusy} />
      <button class="btn btn-ghost btn-sm" disabled={extBusy} onclick={savePort}>Save port</button>
    </div>
  </div>

  <div class="setting-group">
    <div class="setting-label">Pairing token</div>
    <div class="setting-desc">Paste this token into Yaria Bridge settings.</div>
    <div class="ext-row">
      <code class="ext-token"
        >{tokenVisible ? extToken || '—' : extToken ? '•'.repeat(Math.min(extToken.length, 32)) : '—'}</code
      >
      <button class="btn btn-ghost btn-sm" disabled={!extToken} onclick={() => (tokenVisible = !tokenVisible)}>
        {tokenVisible ? 'Hide' : 'Show'}
      </button>
      <button class="btn btn-ghost btn-sm" disabled={!extToken} onclick={copyToken}>Copy</button>
      <button class="btn btn-ghost btn-sm" disabled={extBusy} onclick={regenToken}>Regenerate</button>
    </div>
  </div>
</div>

<style lang="scss">
  @use '../../styles/variables' as *;

  .stg-panel-title {
    font-size: 20px;
    font-weight: 700;
    color: $text;
    margin-bottom: 8px;
  }

  .intro {
    font-size: 13px;
    color: $text-dim;
    line-height: 1.55;
    margin: 0 0 22px;
  }

  .banner {
    margin-bottom: 18px;
    padding: 10px 12px;
    border-radius: 10px;
    font-size: 12px;

    &.err {
      color: #fca5a5;
      background: rgba(248, 113, 113, 0.1);
      border: 1px solid rgba(248, 113, 113, 0.25);
    }
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

  .ext-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
  }

  .ext-pill {
    font-size: 11px;
    padding: 4px 10px;
    border-radius: 999px;
    border: 1px solid rgba(255, 255, 255, 0.08);

    &.on {
      color: $green;
      border-color: rgba(52, 211, 153, 0.35);
    }

    &.off {
      color: $text-muted;
    }
  }

  .ext-code,
  .ext-token {
    font-size: 12px;
    color: $accent-hover;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 8px;
    padding: 8px 10px;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ext-token {
    flex: 1;
    min-width: 140px;
  }

  .ext-input {
    width: 100px;
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.08);
    background: rgba(255, 255, 255, 0.04);
    color: $text;
    padding: 8px 10px;
    font: inherit;
  }

  .btn {
    border: 0;
    border-radius: 8px;
    padding: 8px 12px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    color: $text;
  }

  .btn-sm {
    padding: 7px 12px;
  }

  .btn-primary {
    background: $accent;
    color: #fff;
  }

  .btn-ghost {
    background: transparent;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
