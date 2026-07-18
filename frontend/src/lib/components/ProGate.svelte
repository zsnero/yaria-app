<script lang="ts">
  import { api } from '../api/wails';
  import { toastSuccess } from '../stores/toast';

  let { onActivated }: { onActivated: () => void } = $props();

  let licenseKey = $state('');
  let activating = $state(false);
  let startingTrial = $state(false);
  let error = $state('');
  let trialUsed = $state(false);

  async function activate() {
    const key = licenseKey.trim();
    if (!key) return;
    activating = true;
    error = '';
    try {
      const result = await api.license.activate(key);
      if (result?.valid || result?.success) {
        toastSuccess('Pro license activated');
        onActivated();
      } else {
        error = result?.error || result?.message || 'Invalid license key';
      }
    } catch (err: any) {
      error = err?.message || 'Activation failed';
    }
    activating = false;
  }

  async function startTrial() {
    startingTrial = true;
    error = '';
    try {
      const result = await api.license.startTrial();
      if (result?.valid) {
        toastSuccess('30-day Pro trial started');
        onActivated();
      } else {
        const msg = result?.error || result?.message || 'Could not start trial';
        error = msg;
        if (/already used/i.test(msg)) trialUsed = true;
      }
    } catch (err: any) {
      error = err?.message || 'Could not start trial';
      if (/already used/i.test(error)) trialUsed = true;
    }
    startingTrial = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') activate();
  }
</script>

<div class="pro-gate">
  <div class="pro-gate-card">
    <div class="pro-gate-icon">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="currentColor" style="color: var(--accent, #8b6cef);">
        <path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zm-6 9c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm3.1-9H8.9V6c0-1.71 1.39-3.1 3.1-3.1 1.71 0 3.1 1.39 3.1 3.1v2z"/>
      </svg>
    </div>
    <h2 class="pro-gate-title">Pro Feature</h2>
    <p class="pro-gate-desc">
      Mantorex features require Pro. Try free for 30 days, or activate a license key.
      Search, stream, and download torrents, access local media, remote servers, and your library.
    </p>

    {#if !trialUsed}
      <button
        class="btn btn-primary trial-btn"
        onclick={startTrial}
        disabled={startingTrial || activating}
      >
        {startingTrial ? 'Starting trial…' : 'Start 30-day free trial'}
      </button>
      <div class="pro-gate-or">or enter a license key</div>
    {:else}
      <p class="pro-gate-trial-used">Your free trial on this device has ended. Enter a license key to continue.</p>
    {/if}

    <div class="pro-gate-form">
      <input
        type="text"
        class="pro-gate-input"
        placeholder="Enter license key..."
        bind:value={licenseKey}
        onkeydown={handleKeydown}
        disabled={activating || startingTrial}
      />
      <button
        class="btn btn-primary"
        onclick={activate}
        disabled={activating || startingTrial || !licenseKey.trim()}
      >
        {activating ? 'Activating...' : 'Activate'}
      </button>
    </div>

    {#if error}
      <p class="pro-gate-error">{error}</p>
    {/if}

    <p class="pro-gate-hint">
      Get a license key at <span class="pro-gate-link">yaria.live</span>
    </p>
  </div>
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  .pro-gate {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 60vh;
    padding: 32px;
  }

  .pro-gate-card {
    @include glass;
    border-radius: $radius;
    padding: 48px 40px;
    max-width: 480px;
    width: 100%;
    text-align: center;
    animation: modalIn 0.3s ease;
  }

  .pro-gate-icon {
    margin-bottom: 16px;
  }

  .pro-gate-title {
    font-size: 22px;
    font-weight: 700;
    color: $text;
    margin-bottom: 10px;
  }

  .pro-gate-desc {
    font-size: 13px;
    color: $text-dim;
    line-height: 1.6;
    margin-bottom: 24px;
  }

  .trial-btn {
    width: 100%;
    padding: 12px 16px;
    font-size: 14px;
    font-weight: 600;
    margin-bottom: 16px;
  }

  .pro-gate-or {
    font-size: 12px;
    color: $text-muted;
    margin-bottom: 14px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .pro-gate-trial-used {
    font-size: 13px;
    color: $text-dim;
    line-height: 1.5;
    margin-bottom: 18px;
  }

  .pro-gate-form {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
  }

  .pro-gate-input {
    flex: 1;
    padding: 10px 14px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid $border-glass;
    border-radius: $radius-sm;
    color: $text;
    font-size: 14px;
    outline: none;

    &:focus {
      border-color: rgba($accent, 0.3);
    }

    &:disabled {
      opacity: 0.5;
    }
  }

  .pro-gate-error {
    font-size: 12px;
    color: $red;
    margin-bottom: 8px;
  }

  .pro-gate-hint {
    font-size: 12px;
    color: $text-muted;
    margin-top: 16px;
  }

  .pro-gate-link {
    color: $accent;
  }

  @keyframes modalIn {
    from { opacity: 0; transform: scale(0.96); }
    to { opacity: 1; transform: scale(1); }
  }
</style>
