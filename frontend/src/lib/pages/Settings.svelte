<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '../api/wails';

  import SettingsGeneral from './settings/SettingsGeneral.svelte';
  import SettingsDownloader from './settings/SettingsDownloader.svelte';
  import SettingsTorrents from './settings/SettingsTorrents.svelte';
  import SettingsMedia from './settings/SettingsMedia.svelte';
  import SettingsServer from './settings/SettingsServer.svelte';
  import SettingsAbout from './settings/SettingsAbout.svelte';

  let activeSection = $state('general');
  let settingsPro = $state(false);
  let proLoaded = $state(false);
  let appVersion = $state('');

  const baseSections = [
    { id: 'general', label: 'General', icon: 'gear' },
    { id: 'downloader', label: 'Downloader', icon: 'download' },
  ];

  const proSections = [
    { id: 'torrents', label: 'Torrents', icon: 'bolt' },
    { id: 'media', label: 'Local Media', icon: 'folder' },
    { id: 'server', label: 'Media Server', icon: 'server' },
  ];

  const aboutSection = { id: 'about', label: 'About', icon: 'info' };

  let sections = $derived([
    ...baseSections,
    ...(settingsPro ? proSections : []),
    aboutSection,
  ]);

  function goBack() {
    window.history.back();
  }

  onMount(async () => {
    try {
      settingsPro = await api.license.isPro();
    } catch {
      settingsPro = false;
    }
    proLoaded = true;

    try {
      appVersion = await api.app.getAppVersion();
    } catch {}
  });
</script>

<div class="settings-page">
  <div class="stg-layout">
    <!-- Sidebar -->
    <div class="stg-sidebar">
      <button class="stg-back" onclick={goBack}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/></svg>
        Back
      </button>
      <div class="stg-nav">
        {#each sections as section}
          <button
            class="stg-nav-item"
            class:active={activeSection === section.id}
            onclick={() => activeSection = section.id}
          >
            {#if section.icon === 'gear'}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58a.49.49 0 0 0 .12-.61l-1.92-3.32a.49.49 0 0 0-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.48.48 0 0 0-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.49.49 0 0 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.07.62-.07.94s.02.64.07.94l-2.03 1.58a.49.49 0 0 0-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2z"/></svg>
            {:else if section.icon === 'download'}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M19 9h-4V3H9v6H5l7 7 7-7zM5 18v2h14v-2H5z"/></svg>
            {:else if section.icon === 'bolt'}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M11 21h-1l1-7H7.5c-.88 0-.33-.75-.31-.78C8.48 10.94 10.42 7.54 13.01 3h1l-1 7h3.51c.4 0 .62.19.4.66C12.97 17.55 11 21 11 21z"/></svg>
            {:else if section.icon === 'folder'}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M10 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg>
            {:else if section.icon === 'server'}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M4 1h16c1.1 0 2 .9 2 2v4c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V3c0-1.1.9-2 2-2zm0 8h16c1.1 0 2 .9 2 2v4c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2v-4c0-1.1.9-2 2-2zm0 8h16c1.1 0 2 .9 2 2v2c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2v-2c0-1.1.9-2 2-2zM7 4.5a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3zm0 8a1.5 1.5 0 1 0 0 3 1.5 1.5 0 0 0 0-3z"/></svg>
            {:else if section.icon === 'info'}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>
            {/if}
            {section.label}
          </button>
        {/each}
      </div>
      <div class="stg-version">Yaria{appVersion ? ` v${appVersion}` : ''}</div>
    </div>

    <!-- Content -->
    <div class="stg-content">
      {#if activeSection === 'general'}
        <SettingsGeneral />
      {:else if activeSection === 'downloader'}
        <SettingsDownloader />
      {:else if activeSection === 'torrents' && settingsPro}
        <SettingsTorrents />
      {:else if activeSection === 'media' && settingsPro}
        <SettingsMedia />
      {:else if activeSection === 'server' && settingsPro}
        <SettingsServer />
      {:else if activeSection === 'about'}
        <SettingsAbout />
      {/if}
    </div>
  </div>
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  .settings-page {
    animation: pageIn 0.3s ease-out;
    padding: 24px;
    max-width: 1100px;
    margin: 0 auto;
  }

  .stg-layout {
    display: flex;
    gap: 0;
    min-height: calc(100vh - #{$nav-h} - 48px);
    @include glass;
    border-radius: $radius;
    overflow: hidden;
  }

  // Sidebar
  .stg-sidebar {
    width: 200px;
    flex-shrink: 0;
    padding: 20px 0;
    border-right: 1px solid $border-glass;
    display: flex;
    flex-direction: column;
  }

  .stg-back {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 20px;
    font-size: 13px;
    color: $text-dim;
    transition: color 0.2s;
    margin-bottom: 12px;

    &:hover {
      color: $text;
    }
  }

  .stg-nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 0 8px;
  }

  .stg-nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 9px 14px;
    border-radius: 8px;
    font-size: 13px;
    color: $text-dim;
    text-align: left;
    transition: all 0.15s;

    &:hover {
      background: $bg-glass-hover;
      color: $text;
    }

    &.active {
      background: rgba($accent, 0.12);
      color: $accent-hover;
    }

    :global(svg) {
      flex-shrink: 0;
      opacity: 0.7;
    }
  }

  .stg-version {
    margin-top: auto;
    padding: 12px 20px;
    font-size: 11px;
    color: $text-muted;
  }

  // Content
  .stg-content {
    flex: 1;
    padding: 28px 32px;
    overflow-y: auto;
    max-height: calc(100vh - #{$nav-h} - 48px);
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
