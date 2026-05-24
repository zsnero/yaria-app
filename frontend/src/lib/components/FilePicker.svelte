<script lang="ts">
  import { onMount } from 'svelte';
  import { deps } from '../api/wails';
  import { focusTrap, autoFocus } from '../actions/index';
  import { modalTransition, backdropFade } from '../utils/transition';
  import Spinner from './Spinner.svelte';

  let { onSelect, onClose }: {
    onSelect: (dir: string) => void;
    onClose: () => void;
  } = $props();

  interface DirEntry {
    name: string;
    path: string;
  }

  interface QuickAccess {
    name: string;
    path: string;
    icon: string;
  }

  const quickAccessLocations: QuickAccess[] = [
    { name: 'Downloads', path: '~/Downloads', icon: '📥' },
    { name: 'Mantorex', path: '~/Downloads/Mantorex', icon: '🎬' },
    { name: 'Videos', path: '~/Videos', icon: '🎞' },
    { name: 'Home', path: '~', icon: '🏠' },
    { name: 'Desktop', path: '~/Desktop', icon: '🖥' },
    { name: 'Documents', path: '~/Documents', icon: '📁' },
  ];

  let currentPath = $state('~');
  let selectedPath = $state('~');
  let directories: DirEntry[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let activeQuickAccess = $state('');

  // New folder state
  let showNewFolderInput = $state(false);
  let newFolderName = $state('');

  let breadcrumbs = $derived(buildBreadcrumbs(currentPath));

  function buildBreadcrumbs(path: string): { label: string; path: string }[] {
    const parts = path.split('/').filter((p) => p);
    const crumbs: { label: string; path: string }[] = [];
    let cumPath = '';

    for (const part of parts) {
      if (part === '~') {
        cumPath = '~';
        crumbs.push({ label: 'Home', path: '~' });
      } else {
        cumPath += '/' + part;
        crumbs.push({ label: part, path: cumPath });
      }
    }

    return crumbs;
  }

  async function navigateTo(path: string) {
    currentPath = path;
    selectedPath = path;
    loading = true;
    error = '';
    showNewFolderInput = false;
    newFolderName = '';

    try {
      const dirs = await deps.listDirectories(path);
      directories = dirs || [];
    } catch (e) {
      error = 'Cannot access this folder';
      directories = [];
    } finally {
      loading = false;
    }
  }

  function handleQuickAccess(loc: QuickAccess) {
    activeQuickAccess = loc.path;
    navigateTo(loc.path);
  }

  function handleSelect() {
    if (selectedPath) {
      onSelect(selectedPath);
    }
  }

  function handleNewFolder() {
    if (!showNewFolderInput) {
      showNewFolderInput = true;
      return;
    }

    const name = newFolderName.trim();
    if (name && currentPath) {
      selectedPath = currentPath + '/' + name;
      showNewFolderInput = false;
      newFolderName = '';
    }
  }

  function handleNewFolderKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') handleNewFolder();
    if (e.key === 'Escape') {
      showNewFolderInput = false;
      newFolderName = '';
    }
  }

  function handleOverlayClick(e: MouseEvent) {
    if (e.target === e.currentTarget) onClose();
  }

  function handleWindowKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onClose();
  }

  onMount(() => {
    navigateTo('~');
  });
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<div
  class="fp-overlay"
  onclick={handleOverlayClick}
  role="dialog"
  aria-modal="true"
  transition:backdropFade
>
  <div class="fp-modal" use:focusTrap transition:modalTransition>
    <div class="fp-header">
      <h3>Choose Download Location</h3>
      <button class="fp-close" onclick={onClose} title="Cancel">&times;</button>
    </div>

    <div class="fp-body">
      <div class="fp-sidebar">
        <div class="fp-section-title">Quick Access</div>
        <div class="fp-quick">
          {#each quickAccessLocations as loc}
            <button
              class="fp-quick-item"
              class:active={activeQuickAccess === loc.path}
              onclick={() => handleQuickAccess(loc)}
            >
              <span class="fp-quick-icon">{loc.icon}</span>
              <span>{loc.name}</span>
            </button>
          {/each}
        </div>

        <div class="fp-section-title" style="margin-top: 16px;">Current</div>
        <div class="fp-current-path">{currentPath}</div>
      </div>

      <div class="fp-main">
        <div class="fp-breadcrumb">
          {#each breadcrumbs as crumb, i}
            {#if i === breadcrumbs.length - 1}
              <span class="fp-bc-item active">{crumb.label}</span>
            {:else}
              <button class="fp-bc-item" onclick={() => navigateTo(crumb.path)}>
                {crumb.label}
              </button>
              <span class="fp-bc-sep">/</span>
            {/if}
          {/each}
        </div>

        <div class="fp-path-input-wrap">
          <input
            type="text"
            class="fp-path-input"
            value={currentPath}
            use:autoFocus
            readonly
          />
        </div>

        <div class="fp-list">
          {#if loading}
            <div class="fp-list-msg">
              <Spinner size={20} message="Loading..." />
            </div>
          {:else if error}
            <div class="fp-list-msg fp-error">{error}</div>
          {:else if directories.length === 0}
            <div class="fp-list-msg">Empty folder</div>
          {:else}
            {#each directories as dir}
              <button class="fp-dir-item" onclick={() => navigateTo(dir.path)}>
                <span class="fp-dir-icon">📁</span>
                <span class="fp-dir-name">{dir.name}</span>
              </button>
            {/each}
          {/if}
        </div>
      </div>
    </div>

    <div class="fp-footer">
      <div class="fp-new-folder">
        {#if showNewFolderInput}
          <input
            type="text"
            class="fp-new-input"
            placeholder="New folder name..."
            bind:value={newFolderName}
            onkeydown={handleNewFolderKeydown}
          />
        {/if}
        <button class="btn btn-ghost btn-sm" onclick={handleNewFolder}>New Folder</button>
      </div>
      <div class="fp-actions">
        <button class="btn btn-ghost btn-sm" onclick={onClose}>Cancel</button>
        <button
          class="btn btn-primary btn-sm"
          disabled={!selectedPath}
          onclick={handleSelect}
        >
          Select This Folder
        </button>
      </div>
    </div>
  </div>
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  .fp-overlay {
    position: fixed;
    inset: 0;
    z-index: 9999;
    @include flex-center;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
  }

  .fp-modal {
    @include glass;
    border-radius: $radius;
    width: 680px;
    max-width: 95vw;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
  }

  .fp-header {
    @include flex-between;
    padding: 18px 24px;
    border-bottom: 1px solid $border-glass;

    h3 {
      font-size: 16px;
      font-weight: 600;
      color: $text;
    }
  }

  .fp-close {
    font-size: 22px;
    color: $text-muted;
    padding: 4px 8px;
    border-radius: 6px;
    transition: color 0.2s;

    &:hover {
      color: $text;
    }
  }

  .fp-body {
    display: flex;
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .fp-sidebar {
    width: 180px;
    flex-shrink: 0;
    padding: 16px;
    border-right: 1px solid $border-glass;
    overflow-y: auto;
  }

  .fp-section-title {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: $text-muted;
    margin-bottom: 8px;
  }

  .fp-quick {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .fp-quick-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 10px;
    border-radius: 8px;
    font-size: 12px;
    color: $text-dim;
    text-align: left;
    transition: all 0.15s;

    &:hover {
      background: $bg-glass-hover;
      color: $text;
    }

    &.active {
      background: rgba($accent, 0.15);
      color: $accent-hover;
    }
  }

  .fp-quick-icon {
    font-size: 14px;
    flex-shrink: 0;
  }

  .fp-current-path {
    font-size: 11px;
    color: $text-muted;
    word-break: break-all;
    padding: 4px 0;
  }

  .fp-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .fp-breadcrumb {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 12px 16px;
    border-bottom: 1px solid $border-glass;
    flex-wrap: wrap;
  }

  .fp-bc-item {
    font-size: 12px;
    color: $text-muted;
    padding: 2px 6px;
    border-radius: 4px;
    transition: color 0.15s;

    &:hover:not(.active) {
      color: $accent-hover;
    }

    &.active {
      color: $text;
      font-weight: 500;
    }
  }

  .fp-bc-sep {
    font-size: 11px;
    color: $text-muted;
  }

  .fp-path-input-wrap {
    padding: 8px 16px;
    border-bottom: 1px solid $border-glass;
  }

  .fp-path-input {
    width: 100%;
    padding: 6px 10px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid $border-glass;
    border-radius: 6px;
    color: $text;
    font-size: 12px;
    font-family: monospace;
    outline: none;

    &:focus {
      border-color: rgba($accent, 0.3);
    }
  }

  .fp-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
    min-height: 200px;
    max-height: 360px;
  }

  .fp-list-msg {
    @include flex-center;
    flex-direction: column;
    padding: 20px;
    color: $text-muted;
    font-size: 13px;
  }

  .fp-error {
    color: $red;
  }

  .fp-dir-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px 12px;
    border-radius: 8px;
    text-align: left;
    font-size: 13px;
    color: $text-dim;
    transition: all 0.15s;

    &:hover {
      background: $bg-glass-hover;
      color: $text;
    }
  }

  .fp-dir-icon {
    font-size: 16px;
    flex-shrink: 0;
  }

  .fp-dir-name {
    @include truncate;
  }

  .fp-footer {
    @include flex-between;
    padding: 14px 20px;
    border-top: 1px solid $border-glass;
  }

  .fp-new-folder {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .fp-new-input {
    padding: 5px 10px;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid $border-glass;
    border-radius: 6px;
    color: $text;
    font-size: 12px;
    outline: none;
    width: 160px;

    &:focus {
      border-color: rgba($accent, 0.3);
    }
  }

  .fp-actions {
    display: flex;
    gap: 8px;
  }
</style>
