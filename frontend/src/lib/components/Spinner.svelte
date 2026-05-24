<script lang="ts">
  import { fade } from 'svelte/transition';

  let { size = 24, message = '', overlay = false }: {
    size?: number;
    message?: string;
    overlay?: boolean;
  } = $props();
</script>

{#if overlay}
  <div class="spinner-overlay" transition:fade={{ duration: 200 }}>
    <div class="spinner-container">
      <div
        class="spinner"
        style="width: {size}px; height: {size}px; border-width: {Math.max(2, size / 12)}px;"
      ></div>
      {#if message}
        <p class="spinner-message">{message}</p>
      {/if}
    </div>
  </div>
{:else}
  <div class="spinner-container" transition:fade={{ duration: 150 }}>
    <div
      class="spinner"
      style="width: {size}px; height: {size}px; border-width: {Math.max(2, size / 12)}px;"
    ></div>
    {#if message}
      <p class="spinner-message">{message}</p>
    {/if}
  </div>
{/if}

<style lang="scss">
  @use '../styles/variables' as *;

  .spinner-overlay {
    position: absolute;
    inset: 0;
    @include flex-center;
    background: rgba(5, 5, 16, 0.6);
    backdrop-filter: blur(2px);
    -webkit-backdrop-filter: blur(2px);
    z-index: 10;
  }

  .spinner-container {
    @include flex-center;
    flex-direction: column;
    gap: 12px;
    padding: 20px;
  }

  .spinner {
    border: 2px solid rgba(255, 255, 255, 0.1);
    border-top-color: $accent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;

    :global(.no-animations) & {
      animation: none;
    }
  }

  .spinner-message {
    color: $text-dim;
    font-size: 13px;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
