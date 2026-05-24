<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  let container: HTMLDivElement;
  let scheduledTimeout: ReturnType<typeof setTimeout> | null = null;
  let initialTimeout: ReturnType<typeof setTimeout> | null = null;

  interface Star {
    size: number;
    x: number;
    y: number;
    dur: string;
    delay: string;
    peak: string;
    accent: boolean;
  }

  // Generate star data
  const stars: Star[] = [];

  // 120 twinkling stars
  for (let i = 0; i < 120; i++) {
    stars.push({
      size: Math.random() * 2 + 0.5,
      x: Math.random() * 100,
      y: Math.random() * 100,
      dur: (Math.random() * 5 + 4).toFixed(1),
      delay: (Math.random() * 8).toFixed(1),
      peak: (Math.random() * 0.5 + 0.15).toFixed(2),
      accent: false,
    });
  }

  // 8 accent stars
  for (let i = 0; i < 8; i++) {
    stars.push({
      size: Math.random() * 2 + 2.5,
      x: Math.random() * 100,
      y: Math.random() * 100,
      dur: (Math.random() * 6 + 5).toFixed(1),
      delay: (Math.random() * 10).toFixed(1),
      peak: '0.8',
      accent: true,
    });
  }

  function shootingStar() {
    if (!container) return;

    const star = document.createElement('div');
    star.className = 'shooting-star';

    const x = Math.random() * 80 + 10;
    const y = Math.random() * 40;
    const angle = Math.random() * 30 + 15;
    const dist = Math.random() * 250 + 150;
    const dx = Math.cos((angle * Math.PI) / 180) * dist;
    const dy = Math.sin((angle * Math.PI) / 180) * dist;
    const dur = (Math.random() * 0.8 + 0.6).toFixed(2);
    const w = Math.random() * 50 + 30;

    star.style.cssText = `left:${x}%;top:${y}%;width:${w}px;--dx:${dx}px;--dy:${dy}px;--shoot-dur:${dur}s;--shoot-delay:0s`;
    container.appendChild(star);

    setTimeout(() => star.remove(), parseFloat(dur) * 1000 + 200);
  }

  function schedule() {
    scheduledTimeout = setTimeout(() => {
      shootingStar();
      schedule();
    }, Math.random() * 10000 + 6000);
  }

  onMount(() => {
    initialTimeout = setTimeout(schedule, 3000);
  });

  onDestroy(() => {
    if (scheduledTimeout) clearTimeout(scheduledTimeout);
    if (initialTimeout) clearTimeout(initialTimeout);
  });
</script>

<div id="starfield" bind:this={container}>
  {#each stars as star}
    {#if star.accent}
      <div
        class="star"
        style="width:{star.size}px;height:{star.size}px;left:{star.x}%;top:{star.y}%;--dur:{star.dur}s;--delay:{star.delay}s;--peak:{star.peak};background:radial-gradient(circle,#fff 0%,rgba(167,139,250,0.4) 50%,transparent 70%)"
      ></div>
    {:else}
      <div
        class="star"
        style="width:{star.size}px;height:{star.size}px;left:{star.x}%;top:{star.y}%;--dur:{star.dur}s;--delay:{star.delay}s;--peak:{star.peak}"
      ></div>
    {/if}
  {/each}
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  #starfield {
    position: fixed;
    inset: 0;
    z-index: 0;
    pointer-events: none;
    overflow: hidden;
  }

  .star {
    position: absolute;
    border-radius: 50%;
    background: white;
    opacity: 0;
    will-change: opacity;
    animation: twinkle var(--dur) var(--delay) infinite ease-in-out;
  }

  :global(.shooting-star) {
    position: absolute;
    height: 1px;
    background: linear-gradient(90deg, rgba(255, 255, 255, 0.8), transparent);
    border-radius: 1px;
    opacity: 0;
    will-change: transform, opacity;
    animation: shoot var(--shoot-dur) var(--shoot-delay) linear forwards;
  }

  @keyframes twinkle {
    0%, 100% {
      opacity: 0;
    }
    50% {
      opacity: var(--peak);
    }
  }

  @keyframes shoot {
    0% {
      opacity: 1;
      transform: translate(0, 0);
    }
    70% {
      opacity: 1;
    }
    100% {
      opacity: 0;
      transform: translate(var(--dx), var(--dy));
    }
  }
</style>
