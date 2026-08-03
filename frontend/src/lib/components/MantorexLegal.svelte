<script lang="ts">
  import { api } from '../api/wails';

  let { onAccepted }: { onAccepted: () => void } = $props();

  let agreed = $state(false);
  let saving = $state(false);
  let error = $state('');

  async function accept() {
    if (!agreed || saving) return;
    saving = true;
    error = '';
    try {
      try {
        localStorage.setItem('yaria_mantorex_legal_v1', '1');
      } catch { /* ignore */ }
      await api.settings.saveUISettings({ mantorex_legal_accepted: true });
      onAccepted();
    } catch (e: any) {
      error = e?.message || 'Could not save acceptance';
    }
    saving = false;
  }

  function decline() {
    window.location.hash = '#/yaria';
  }
</script>

<div class="legal-overlay" role="dialog" aria-modal="true" aria-labelledby="legal-title">
  <div class="legal-card">
    <h2 id="legal-title" class="legal-title">Legal notice</h2>
    <p class="legal-intro">
      The <strong>Torrents</strong> section can search, stream, and download content via BitTorrent
      and related protocols. Before you continue, please read and accept the following:
    </p>

    <div class="legal-body">
      <ul>
        <li>
          Yaria / Mantorex is provided as a <strong>tool only</strong>. You are solely responsible for
          how you use it and for any content you search, stream, download, share, or store.
        </li>
        <li>
          You must comply with all applicable laws in your country or region, including copyright,
          trademark, and export rules. Do not use Mantorex to obtain or distribute material you do
          not have the right to use.
        </li>
        <li>
          Some networks block or restrict BitTorrent. Using torrents may violate your ISP, school,
          workplace, or local regulations. You accept any risk of notices, throttling, or legal action.
        </li>
        <li>
          Files from untrusted torrents may contain malware (for example fake “movie” <code>.exe</code>
          files). Only open media from sources you trust.
        </li>
        <li>
          The developers of Yaria are not liable for misuse of this software or for third-party content
          accessed through it.
        </li>
      </ul>
    </div>

    <label class="legal-check">
      <input type="checkbox" bind:checked={agreed} />
      <span>I have read this notice and I agree to use torrent features only in accordance with the law.</span>
    </label>

    {#if error}
      <p class="legal-error">{error}</p>
    {/if}

    <div class="legal-actions">
      <button type="button" class="btn btn-ghost" onclick={decline} disabled={saving}>
        Decline
      </button>
      <button
        type="button"
        class="btn btn-primary"
        onclick={accept}
        disabled={!agreed || saving}
      >
        {saving ? 'Saving…' : 'Accept and continue'}
      </button>
    </div>
  </div>
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  .legal-overlay {
    position: fixed;
    inset: 0;
    z-index: 300;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: rgba(5, 5, 16, 0.82);
    backdrop-filter: blur(8px);
  }

  .legal-card {
    width: min(560px, 100%);
    max-height: min(90vh, 720px);
    overflow: auto;
    background: rgba(18, 18, 32, 0.98);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 16px;
    padding: 28px 28px 24px;
    box-shadow: 0 24px 64px rgba(0, 0, 0, 0.55);
  }

  .legal-title {
    font-size: 22px;
    font-weight: 700;
    color: $text;
    margin: 0 0 12px;
  }

  .legal-intro {
    font-size: 14px;
    color: $text-dim;
    line-height: 1.55;
    margin: 0 0 16px;
  }

  .legal-body {
    font-size: 13px;
    color: $text-muted;
    line-height: 1.6;
    margin-bottom: 18px;
    padding: 14px 16px;
    background: rgba(255, 255, 255, 0.03);
    border-radius: 10px;
    border: 1px solid rgba(255, 255, 255, 0.06);

    ul {
      margin: 0;
      padding-left: 1.15em;
    }

    li {
      margin-bottom: 10px;
      &:last-child {
        margin-bottom: 0;
      }
    }

    code {
      font-size: 12px;
      color: $accent-hover;
    }

    strong {
      color: $text-dim;
      font-weight: 600;
    }
  }

  .legal-check {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    font-size: 13px;
    color: $text;
    cursor: pointer;
    margin-bottom: 18px;
    line-height: 1.45;

    input {
      margin-top: 3px;
      flex-shrink: 0;
      accent-color: $accent;
    }
  }

  .legal-error {
    color: $red;
    font-size: 12px;
    margin: -8px 0 12px;
  }

  .legal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    flex-wrap: wrap;
  }
</style>
