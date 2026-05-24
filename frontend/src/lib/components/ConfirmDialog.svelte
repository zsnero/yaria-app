<script lang="ts">
  import { focusTrap } from '../actions/index';
  import { modalTransition, backdropFade } from '../utils/transition';

  let { message, onConfirm, onCancel = undefined }: {
    message: string;
    onConfirm: () => void;
    onCancel?: (() => void) | undefined;
  } = $props();

  let cancelBtn: HTMLButtonElement | undefined = $state();

  $effect(() => {
    if (cancelBtn) {
      cancelBtn.focus();
    }
  });

  function handleOverlayClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      cancel();
    }
  }

  function confirm() {
    onConfirm();
  }

  function cancel() {
    if (onCancel) onCancel();
  }

  function handleWindowKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') cancel();
  }
</script>

<svelte:window onkeydown={handleWindowKeydown} />

<div
  class="app-confirm-overlay"
  onclick={handleOverlayClick}
  role="dialog"
  aria-modal="true"
  transition:backdropFade
>
  <div class="app-confirm-modal" use:focusTrap transition:modalTransition>
    <p class="app-confirm-msg">{message}</p>
    <div class="app-confirm-actions">
      <button class="btn btn-ghost" bind:this={cancelBtn} onclick={cancel}>Cancel</button>
      <button class="btn btn-primary" onclick={confirm}>Confirm</button>
    </div>
  </div>
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  .app-confirm-overlay {
    position: fixed;
    inset: 0;
    z-index: 9999;
    @include flex-center;
    background: rgba(0, 0, 0, 0.6);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
  }

  .app-confirm-modal {
    @include glass;
    border-radius: $radius;
    padding: 28px 32px;
    max-width: 420px;
    width: 90%;
  }

  .app-confirm-msg {
    color: $text;
    font-size: 14px;
    line-height: 1.6;
    margin-bottom: 24px;
    white-space: pre-line;
  }

  .app-confirm-actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }
</style>
