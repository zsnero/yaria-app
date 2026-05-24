<script lang="ts">
  import { fly } from 'svelte/transition';
  import { toasts, dismissToast } from '../stores/toast';
  import type { ToastMessage } from '../stores/toast';

  const typeColors: Record<ToastMessage['type'], string> = {
    success: '#34d399',
    error: '#f87171',
    info: '#8b6cef',
    warning: '#fbbf24',
  };
</script>

<div class="toast-container">
  {#each $toasts as t (t.id)}
    <button
      class="toast toast-{t.type}"
      style="--toast-accent: {typeColors[t.type]}"
      transition:fly={{ x: 300, duration: 300 }}
      onclick={() => dismissToast(t.id)}
    >
      <span class="toast-icon">
        {#if t.type === 'success'}
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
            <path d="M5 8l2 2 4-4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        {:else if t.type === 'error'}
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
            <path d="M6 6l4 4M10 6l-4 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
          </svg>
        {:else if t.type === 'info'}
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <circle cx="8" cy="8" r="7" stroke="currentColor" stroke-width="1.5"/>
            <path d="M8 7v4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            <circle cx="8" cy="5" r="0.75" fill="currentColor"/>
          </svg>
        {:else}
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
            <path d="M8 1.5l6.5 12H1.5L8 1.5z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
            <path d="M8 6.5v3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
            <circle cx="8" cy="11.5" r="0.75" fill="currentColor"/>
          </svg>
        {/if}
      </span>
      <span class="toast-message">{t.message}</span>
      <span class="toast-close">
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
          <path d="M3 3l6 6M9 3l-6 6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
      </span>
    </button>
  {/each}
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  .toast-container {
    position: fixed;
    bottom: 20px;
    right: 20px;
    z-index: 10000;
    display: flex;
    flex-direction: column;
    gap: 8px;
    pointer-events: none;
    max-width: 360px;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    border-radius: $radius-sm;
    background: rgba(10, 10, 25, 0.88);
    border: 1px solid rgba(255, 255, 255, 0.08);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    color: $text;
    font-size: 13px;
    cursor: pointer;
    pointer-events: auto;
    text-align: left;
    width: 100%;
    border-left: 3px solid var(--toast-accent);
    transition: background 0.15s ease;

    &:hover {
      background: rgba(15, 15, 35, 0.92);
    }
  }

  .toast-icon {
    flex-shrink: 0;
    color: var(--toast-accent);
    display: flex;
    align-items: center;
  }

  .toast-message {
    flex: 1;
    min-width: 0;
    line-height: 1.4;
  }

  .toast-close {
    flex-shrink: 0;
    color: $text-muted;
    display: flex;
    align-items: center;
    opacity: 0.6;
    transition: opacity 0.15s;

    .toast:hover & {
      opacity: 1;
    }
  }
</style>
