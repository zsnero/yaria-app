<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../../api/wails';
  import { focusTrap } from '../../actions/index';
  import { modalTransition, backdropFade } from '../../utils/transition';

  // About
  let appVersion = $state('');
  let updateChecking = $state(false);
  let updateInfo = $state<any>(null);
  let updateError = $state('');
  let updating = $state(false);
  let updateProgress = $state(0);
  let updateMsg = $state('');

  // Legal modal
  let showLegal = $state(false);
  let legalTitle = $state('');
  let legalContent = $state('');

  // --- Privacy Policy / Terms ---
  const privacyPolicy = `<p>Last updated: May 2026</p>

<h3>Overview</h3>
<p>Yaria ("the App") is a desktop application. We respect your privacy and are committed to transparency about how the App operates.</p>

<h3>Data We Collect</h3>
<p><strong>We do not collect, transmit, or store any personal data on our servers.</strong> All user data stays on your device.</p>
<ul>
  <li><strong>License keys</strong> are validated against our server (yaria.live) to verify your subscription. The server receives your license key and a device fingerprint (a hash derived from your machine ID). No personal information is sent.</li>
  <li><strong>TMDB API key</strong> (if provided) is stored locally and used to fetch movie/TV metadata directly from The Movie Database API. We do not proxy or log these requests.</li>
  <li><strong>Search queries</strong> in Mantorex are sent directly to third-party torrent indexing websites. We do not log, proxy, or store your searches.</li>
  <li><strong>Download history, library data, watch progress, and settings</strong> are stored locally on your device in a BoltDB database under <code>~/.yaria/</code> and <code>~/.config/</code>.</li>
  <li><strong>Cookies</strong> extracted for video downloads are cached locally at <code>~/.yaria/cookies.txt</code> and are never transmitted to us.</li>
</ul>

<h3>Third-Party Services</h3>
<ul>
  <li><strong>yaria.live</strong> -- License validation only. Receives license key + device ID hash.</li>
  <li><strong>TMDB (The Movie Database)</strong> -- Metadata queries using your own API key.</li>
  <li><strong>Torrent indexing sites</strong> -- Search queries are sent directly from your device to third-party sites. We have no control over their privacy practices.</li>
  <li><strong>GitHub</strong> -- Dependency updates (yt-dlp, FFmpeg) are fetched from GitHub releases.</li>
</ul>

<h3>Data Storage</h3>
<p>All data is stored locally on your machine. You can delete all app data by removing the <code>~/.yaria/</code> and <code>~/.config/yaria/</code> directories.</p>

<h3>Analytics & Telemetry</h3>
<p>The App does not include any analytics, telemetry, crash reporting, or tracking of any kind.</p>

<h3>Changes</h3>
<p>We may update this policy. Changes will be included in future versions of the App.</p>

<h3>Contact</h3>
<p>For questions about this policy, visit <a href="https://yaria.live" target="_blank">yaria.live</a>.</p>`;

  const termsOfUse = `<p>Last updated: May 2026</p>

<h3>Acceptance</h3>
<p>By downloading, installing, or using Yaria ("the App"), you agree to these Terms of Use. If you do not agree, do not use the App.</p>

<h3>Description of Service</h3>
<p>Yaria is a video and audio downloader that wraps yt-dlp. The Pro version ("Yaria Pro") includes Mantorex, a torrent search engine and BitTorrent client.</p>

<h3>Lawful Use</h3>
<ul>
  <li>You agree to use the App <strong>only for lawful purposes</strong> and in compliance with all applicable local, national, and international laws.</li>
  <li>The App is a tool. <strong>You are solely responsible</strong> for the content you download, stream, or share using the App.</li>
  <li>Downloading or distributing copyrighted material without authorization from the copyright holder is illegal in most jurisdictions. <strong>We do not condone or encourage copyright infringement.</strong></li>
</ul>

<h3>Mantorex (Torrent Features)</h3>
<ul>
  <li>Mantorex searches third-party torrent indexing websites. We do not host, index, or control any torrent content.</li>
  <li>Search results are fetched directly from third-party sites. We make no warranties about the accuracy, legality, or safety of listed content.</li>
  <li>By using Mantorex, you accept the Legal Disclaimer presented on first use.</li>
  <li>BitTorrent is a peer-to-peer protocol. When downloading or streaming, your IP address is visible to other peers in the swarm.</li>
</ul>

<h3>Intellectual Property</h3>
<p>Yaria, the Yaria logo, and Mantorex are property of their respective owners. Third-party tools (yt-dlp, FFmpeg, aria2) are used under their respective open-source licenses.</p>

<h3>Licensing</h3>
<ul>
  <li>The free version of Yaria is available for personal use.</li>
  <li>Yaria Pro requires a valid license key purchased from yaria.live.</li>
  <li>License keys are bound to a single device and are non-transferable.</li>
  <li>We reserve the right to revoke licenses that violate these terms.</li>
</ul>

<h3>Disclaimer of Warranties</h3>
<p>The App is provided <strong>"as is"</strong> without warranty of any kind. We do not guarantee uninterrupted or error-free operation. Use at your own risk.</p>

<h3>Limitation of Liability</h3>
<p>To the maximum extent permitted by law, the developers shall not be liable for any damages arising from the use of this App, including but not limited to damages from downloaded content, legal actions, data loss, or service interruptions.</p>

<h3>Changes</h3>
<p>We reserve the right to modify these terms. Continued use after changes constitutes acceptance.</p>

<h3>Contact</h3>
<p>For questions about these terms, visit <a href="https://yaria.live" target="_blank">yaria.live</a>.</p>`;

  async function checkUpdate() {
    updateChecking = true;
    updateError = '';
    updateInfo = null;
    try {
      const info = await api.updater.checkUpdate();
      if (info.error) {
        updateError = info.error;
      } else {
        updateInfo = info;
      }
    } catch (e: any) {
      updateError = 'Check failed: ' + (e.message || '');
    }
    updateChecking = false;
  }

  async function doUpdate() {
    updating = true;
    updateProgress = 0;
    updateMsg = '';
    await api.updater.doUpdate();
    const poll = setInterval(async () => {
      try {
        const s = await api.updater.getStatus();
        updateProgress = s.progress || 0;
        updateMsg = s.message || '';
        if (!s.updating) {
          clearInterval(poll);
          updating = false;
        }
      } catch {
        clearInterval(poll);
        updating = false;
      }
    }, 800);
  }

  function showPrivacy() {
    legalTitle = 'Privacy Policy';
    legalContent = privacyPolicy;
    showLegal = true;
  }

  function showTerms() {
    legalTitle = 'Terms of Use';
    legalContent = termsOfUse;
    showLegal = true;
  }

  function closeLegal() {
    showLegal = false;
  }

  onMount(async () => {
    try {
      appVersion = await api.app.getAppVersion();
    } catch {}
  });
</script>

<div class="stg-panel active">
  <h3 class="stg-panel-title">About & Legal</h3>
  <div class="setting-group">
    <div class="setting-label">Yaria</div>
    <div class="setting-desc about-desc">
      {#if appVersion}
        Yaria Desktop App v{appVersion}
      {:else}
        Yaria Desktop App
      {/if}
      <br />Video & audio downloader + media center.
    </div>

    <!-- Update -->
    <div class="update-section">
      {#if !updateInfo?.available}
        <button
          class="btn btn-ghost btn-sm"
          onclick={checkUpdate}
          disabled={updateChecking}
        >
          {updateChecking ? 'Checking...' : 'Check for Updates'}
        </button>
        {#if updateError}
          <span class="msg-error inline">{updateError}</span>
        {/if}
        {#if updateInfo && !updateInfo.available}
          <span class="text-green inline">You're on the latest version (v{updateInfo.current})</span>
        {/if}
      {:else}
        <div class="update-available">
          <span class="text-green bold">Update available!</span>
          <span class="text-muted">v{updateInfo.current} → v{updateInfo.latest}</span>
        </div>
        {#if !updating}
          <button class="btn btn-primary btn-sm" onclick={doUpdate}>Update Now</button>
        {:else}
          <div class="update-progress">
            <div class="progress-bar">
              <div class="progress-fill" style="width: {updateProgress}%"></div>
            </div>
            <p class="update-msg">{updateMsg}</p>
          </div>
        {/if}
      {/if}
    </div>

    <!-- Legal -->
    <div class="legal-btns">
      <button class="btn btn-ghost btn-sm" onclick={showPrivacy}>Privacy Policy</button>
      <button class="btn btn-ghost btn-sm" onclick={showTerms}>Terms of Use</button>
    </div>
  </div>
</div>

<!-- Legal Modal -->
{#if showLegal}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_interactive_supports_focus -->
  <div class="legal-overlay" onclick={(e) => { if (e.target === e.currentTarget) closeLegal(); }} role="dialog" aria-modal="true" transition:backdropFade>
    <div class="legal-modal" use:focusTrap transition:modalTransition>
      <div class="legal-header">
        <h3>{legalTitle}</h3>
        <button class="legal-close" onclick={closeLegal}>&times;</button>
      </div>
      <div class="legal-body">
        {@html legalContent}
      </div>
    </div>
  </div>
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

  .about-desc {
    margin-top: 6px;
  }

  .update-section {
    margin-top: 14px;
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 10px;
  }

  .update-available {
    font-size: 13px;
    margin-bottom: 12px;
    width: 100%;

    .bold {
      font-weight: 600;
    }
  }

  .update-progress {
    margin-top: 12px;
    width: 100%;
  }

  .progress-bar {
    height: 4px;
    background: rgba(255, 255, 255, 0.06);
    border-radius: 2px;
    overflow: hidden;
    margin-bottom: 6px;
  }

  .progress-fill {
    height: 100%;
    background: $accent;
    border-radius: 2px;
    transition: width 0.3s;
  }

  .update-msg {
    font-size: 12px;
    color: $text-muted;
  }

  .legal-btns {
    display: flex;
    gap: 8px;
    margin-top: 14px;
    flex-wrap: wrap;
  }

  .msg-error {
    font-size: 13px;
    color: $red;

    &.inline {
      margin-left: 10px;
    }
  }

  .text-green {
    color: $green;
  }

  .text-muted {
    color: $text-muted;
    font-size: 13px;
  }

  .inline {
    display: inline;
    margin-left: 10px;
  }

  // Legal Modal
  .legal-overlay {
    position: fixed;
    inset: 0;
    z-index: 1000;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    animation: pageIn 0.2s ease-out;
  }

  .legal-modal {
    width: 640px;
    max-width: 92vw;
    max-height: 80vh;
    background: #12121e;
    border: 1px solid $border-glass;
    border-radius: $radius;
    display: flex;
    flex-direction: column;
    box-shadow: 0 24px 80px rgba(0, 0, 0, 0.6);
  }

  .legal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 18px 24px;
    border-bottom: 1px solid $border-glass;

    h3 {
      font-size: 18px;
      font-weight: 700;
    }
  }

  .legal-close {
    font-size: 22px;
    color: $text-muted;
    padding: 4px 8px;
    border-radius: 50%;
    transition: color 0.2s;

    &:hover {
      color: $text;
    }
  }

  .legal-body {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
    font-size: 13px;
    color: $text-dim;
    line-height: 1.7;

    :global(h3) {
      font-size: 15px;
      font-weight: 700;
      color: $text;
      margin: 20px 0 8px;

      &:first-child {
        margin-top: 0;
      }
    }

    :global(ul) {
      padding-left: 20px;
      margin: 8px 0;
    }

    :global(li) {
      margin-bottom: 6px;
    }

    :global(code) {
      background: rgba(255, 255, 255, 0.06);
      padding: 1px 6px;
      border-radius: 3px;
      font-size: 12px;
    }

    :global(p) {
      margin-bottom: 10px;
    }

    :global(a) {
      color: $accent;
    }
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
