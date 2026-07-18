<script lang="ts">
  import { onMount } from 'svelte';
  import { deps } from '../api/wails';
  import { focusTrap, autoFocus } from '../actions/index';
  import { modalTransition, backdropFade } from '../utils/transition';
  import Spinner from './Spinner.svelte';

  let {
    onSelect,
    onClose,
    mode = 'directory',
    title = '',
    fileExt = '',
    defaultFileName = '',
  }: {
    onSelect: (path: string) => void;
    onClose: () => void;
    /** directory = pick folder; open = pick existing file; save = folder + filename */
    mode?: 'directory' | 'open' | 'save';
    title?: string;
    /** Comma-separated extensions without dots, e.g. "json" */
    fileExt?: string;
    defaultFileName?: string;
  } = $props();

  interface DirEntry {
    name: string;
    path: string;
    is_dir?: boolean;
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
  let entries: DirEntry[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let activeQuickAccess = $state('');
  let fileName = $state(defaultFileName || '');
  let selectedFile = $state('');

  // New folder state
  let showNewFolderInput = $state(false);
  let newFolderName = $state('');

  let breadcrumbs = $derived(buildBreadcrumbs(currentPath));

  let headerTitle = $derived(
    title ||
      (mode === 'open'
        ? 'Open File'
        : mode === 'save'
          ? 'Save File'
          : 'Choose Download Location')
  );

  let confirmLabel = $derived(
    mode === 'open' ? 'Open' : mode === 'save' ? 'Save' : 'Select This Folder'
  );

  let canConfirm = $derived.by(() => {
    if (mode === 'directory') return !!selectedPath;
    if (mode === 'open') return !!selectedFile;
    if (mode === 'save') return !!selectedPath && !!fileName.trim();
    return false;
  });

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
    selectedFile = '';
    loading = true;
    error = '';
    showNewFolderInput = false;
    newFolderName = '';

    try {
      if (mode === 'directory') {
        const dirs = await deps.listDirectories(path);
        entries = (dirs || []).map((d) => ({ ...d, is_dir: true }));
      } else {
        const list = await deps.listEntries(path, fileExt);
        entries = list || [];
      }
    } catch {
      error = 'Cannot access this folder';
      entries = [];
    } finally {
      loading = false;
    }
  }

  function handleQuickAccess(loc: QuickAccess) {
    activeQuickAccess = loc.path;
    navigateTo(loc.path);
  }

  function handleEntryClick(entry: DirEntry) {
    if (mode === 'directory' || entry.is_dir) {
      navigateTo(entry.path);
      return;
    }
    // File
    selectedFile = entry.path;
    selectedPath = currentPath;
    if (mode === 'save') {
      fileName = entry.name;
    }
  }

  function handleEntryDblClick(entry: DirEntry) {
    if (entry.is_dir) {
      navigateTo(entry.path);
      return;
    }
    if (mode === 'open') {
      selectedFile = entry.path;
      handleSelect();
    }
  }

  function handleSelect() {
    if (mode === 'directory') {
      if (selectedPath) onSelect(selectedPath);
      return;
    }
    if (mode === 'open') {
      if (selectedFile) onSelect(selectedFile);
      return;
    }
    // save
    const name = fileName.trim();
    if (!selectedPath || !name) return;
    let finalName = name;
    if (fileExt) {
      const primary = fileExt.split(',')[0].trim().replace(/^\./, '');
      if (primary && !finalName.toLowerCase().endsWith('.' + primary.toLowerCase())) {
        finalName += '.' + primary;
      }
    }
    const join = selectedPath.endsWith('/') ? selectedPath + finalName : selectedPath + '/' + finalName;
    onSelect(join);
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
    if (defaultFileName) fileName = defaultFileName;
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
      <h3>{headerTitle}</h3>
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
          {:else if entries.length === 0}
            <div class="fp-list-msg">Empty folder</div>
          {:else}
            {#each entries as entry}
              <button
                class="fp-dir-item"
                class:selected={selectedFile === entry.path}
                onclick={() => handleEntryClick(entry)}
                ondblclick={() => handleEntryDblClick(entry)}
              >
                <span class="fp-dir-icon">{entry.is_dir === false ? '📄' : '📁'}</span>
                <span class="fp-dir-name">{entry.name}</span>
              </button>
            {/each}
          {/if}
        </div>

        {#if mode === 'save'}
          <div class="fp-filename-row">
            <label class="fp-filename-label" for="fp-filename">File name</label>
            <input
              id="fp-filename"
              type="text"
              class="fp-path-input"
              bind:value={fileName}
              placeholder="backup.json"
            />
          </div>
        {/if}
      </div>
    </div>

    <div class="fp-footer">
      <div class="fp-new-folder">
        {#if mode !== 'open'}
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
        {/if}
      </div>
      <div class="fp-actions">
        <button class="btn btn-ghost btn-sm" onclick={onClose}>Cancel</button>
        <button
          class="btn btn-primary btn-sm"
          disabled={!canConfirm}
          onclick={handleSelect}
        >
          {confirmLabel}
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
    background: rgba(0, 0, 0, 0.8);
  }

  .fp-modal {
    background: #14141f;
    border: 1px solid $border-glass-hover;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.6);
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
    width: 100%;
    padding: 8px 10px;
    border-radius: 8px;
    font-size: 13px;
    color: $text-dim;
    text-align: left;
    transition: background 0.15s, color 0.15s;

    &:hover,
    &.active {
      background: rgba(255, 255, 255, 0.06);
      color: $text;
    }
  }

  .fp-quick-icon {
    font-size: 14px;
  }

  .fp-current-path {
    font-size: 11px;
    color: $text-muted;
    word-break: break-all;
    line-height: 1.4;
  }

  .fp-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    padding: 16px;
    gap: 10px;
  }

  .fp-breadcrumb {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 4px;
    font-size: 12px;
  }

  .fp-bc-item {
    color: $text-dim;
    background: none;
    border: none;
    padding: 0;
    font-size: 12px;
    cursor: pointer;

    &.active {
      color: $text;
      font-weight: 600;
      cursor: default;
    }

    &:not(.active):hover {
      color: $accent;
    }
  }

  .fp-bc-sep {
    color: $text-muted;
  }

  .fp-path-input-wrap {
    width: 100%;
  }

  .fp-path-input {
    width: 100%;
    padding: 8px 12px;
    border-radius: 8px;
    border: 1px solid $border-glass;
    background: rgba(0, 0, 0, 0.25);
    color: $text;
    font-size: 13px;
  }

  .fp-list {
    flex: 1;
    min-height: 180px;
    max-height: 320px;
    overflow-y: auto;
    border: 1px solid $border-glass;
    border-radius: 10px;
    background: rgba(0, 0, 0, 0.15);
  }

  .fp-list-msg {
    @include flex-center;
    padding: 32px;
    color: $text-muted;
    font-size: 13px;
    gap: 10px;
  }

  .fp-error {
    color: $red;
  }

  .fp-dir-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 10px 14px;
    border: none;
    background: transparent;
    color: $text;
    font-size: 13px;
    text-align: left;
    cursor: pointer;
    border-bottom: 1px solid rgba(255, 255, 255, 0.03);

    &:hover,
    &.selected {
      background: rgba(139, 108, 239, 0.12);
    }

    &.selected {
      color: $accent-hover;
    }
  }

  .fp-dir-icon {
    font-size: 15px;
  }

  .fp-dir-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .fp-filename-row {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .fp-filename-label {
    font-size: 11px;
    color: $text-muted;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .fp-footer {
    @include flex-between;
    padding: 14px 20px;
    border-top: 1px solid $border-glass;
    gap: 12px;
  }

  .fp-new-folder {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .fp-new-input {
    width: 160px;
    padding: 6px 10px;
    border-radius: 8px;
    border: 1px solid $border-glass;
    background: rgba(0, 0, 0, 0.25);
    color: $text;
    font-size: 13px;
  }

  .fp-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }
</style>
