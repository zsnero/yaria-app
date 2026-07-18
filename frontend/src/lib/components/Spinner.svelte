<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  let { size = 24, message = '', overlay = false }: {
    size?: number;
    message?: string;
    overlay?: boolean;
  } = $props();

  const stroke = $derived(Math.max(2, Math.round(size / 12)));
  let svgEl: SVGSVGElement | undefined = $state();

  // JS-driven rotation — reliable in WebKitGTK/Wails where CSS animations
  // and sometimes SMIL fail (especially with UI scale / zoom).
  onMount(() => {
    let raf = 0;
    let angle = 0;
    let last = performance.now();

    const tick = (now: number) => {
      const dt = now - last;
      last = now;
      // 0.75s per revolution => 480 deg/s
      angle = (angle + (dt / 750) * 360) % 360;
      if (svgEl) {
        svgEl.style.transform = `rotate(${angle}deg)`;
      }
      raf = requestAnimationFrame(tick);
    };

    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  });
</script>

{#if overlay}
  <div class="spinner-overlay" transition:fade={{ duration: 200 }}>
    <div class="spinner-container">
      <svg
        bind:this={svgEl}
        class="spinner-svg"
        width={size}
        height={size}
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <circle class="spinner-track" cx="12" cy="12" r="10" fill="none" stroke-width={stroke} />
        <circle class="spinner-arc" cx="12" cy="12" r="10" fill="none" stroke-width={stroke} />
      </svg>
      {#if message}
        <p class="spinner-message">{message}</p>
      {/if}
    </div>
  </div>
{:else}
  <div class="spinner-container" transition:fade={{ duration: 150 }}>
    <svg
      bind:this={svgEl}
      class="spinner-svg"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <circle class="spinner-track" cx="12" cy="12" r="10" fill="none" stroke-width={stroke} />
      <circle class="spinner-arc" cx="12" cy="12" r="10" fill="none" stroke-width={stroke} />
    </svg>
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
    z-index: 10;
  }

  .spinner-container {
    @include flex-center;
    flex-direction: column;
    gap: 12px;
    padding: 20px;
  }

  .spinner-svg {
    display: block;
    transform-origin: center center;
  }

  .spinner-track {
    stroke: rgba(255, 255, 255, 0.12);
  }

  .spinner-arc {
    stroke: #{$accent};
    stroke-linecap: round;
    stroke-dasharray: 40 80;
  }

  .spinner-message {
    color: $text-dim;
    font-size: 13px;
    margin: 0;
  }
</style>
