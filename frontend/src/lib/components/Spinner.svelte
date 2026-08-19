<script lang="ts" module>
  export type SpinnerVariant = 'orbit' | 'singularity' | 'radar' | 'warp' | 'classic';

  export const SPINNER_OPTIONS: { id: SpinnerVariant; label: string; desc: string }[] = [
    { id: 'orbit', label: 'Orbital Halo', desc: 'Soft core with orbiting lights' },
    { id: 'singularity', label: 'Singularity Pulse', desc: 'Breathing glow + dashed ring' },
    { id: 'radar', label: 'Radar Sweep', desc: 'Scan beam over a disc' },
    { id: 'warp', label: 'Warp Streaks', desc: 'Comet tails on a ring' },
    { id: 'classic', label: 'Classic', desc: 'Simple arc spinner' },
  ];

  const VALID = new Set<string>(SPINNER_OPTIONS.map((o) => o.id));

  export function normalizeSpinnerVariant(v: unknown): SpinnerVariant {
    const s = String(v || '');
    return VALID.has(s) ? (s as SpinnerVariant) : 'orbit';
  }

  export function readSpinnerVariant(): SpinnerVariant {
    try {
      return normalizeSpinnerVariant(localStorage.getItem('yaria_spinner'));
    } catch {
      return 'orbit';
    }
  }

  export function writeSpinnerVariant(v: SpinnerVariant): void {
    try {
      localStorage.setItem('yaria_spinner', normalizeSpinnerVariant(v));
    } catch {
      /* ignore */
    }
    if (typeof window !== 'undefined') {
      window.dispatchEvent(new CustomEvent('yaria-spinner-change', { detail: normalizeSpinnerVariant(v) }));
    }
  }
</script>

<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';

  let {
    size = 36,
    message = '',
    overlay = false,
    /** Force a style (settings preview). When omitted, uses user preference. */
    variant,
  }: {
    size?: number;
    message?: string;
    overlay?: boolean;
    variant?: SpinnerVariant;
  } = $props();

  let pref = $state<SpinnerVariant>(readSpinnerVariant());
  const active = $derived(variant ? normalizeSpinnerVariant(variant) : pref);
  // Mutable mirror for rAF (closure-safe)
  let activeStyle = $state<SpinnerVariant>(readSpinnerVariant());
  $effect(() => {
    activeStyle = active;
  });

  const uid = `sp${Math.random().toString(36).slice(2, 9)}`;
  const classicStroke = $derived(Math.max(2, Math.round(size / 12)));
  const vb = 48;

  let layerA: SVGGElement | undefined = $state();
  let layerB: SVGGElement | undefined = $state();
  let layerC: SVGGElement | undefined = $state();
  let classicEl: SVGSVGElement | undefined = $state();

  onMount(() => {
    const onChange = (e: Event) => {
      pref = normalizeSpinnerVariant((e as CustomEvent).detail);
    };
    window.addEventListener('yaria-spinner-change', onChange);

    let raf = 0;
    let a = 0;
    let b = 0;
    let c = 0;
    let last = performance.now();

    const tick = (now: number) => {
      const dt = now - last;
      last = now;
      // Calm pace (~0.55–0.9 rev/s) — CSS must not also spin .spinner-svg
      a = (a + (dt / 1000) * 200) % 360;
      b = (b - (dt / 1000) * 95) % 360;
      c = (c + (dt / 1000) * 260) % 360;

      if (activeStyle === 'classic' && classicEl) {
        classicEl.style.transform = `rotate(${a}deg)`;
      } else {
        if (layerA) layerA.style.transform = `rotate(${a}deg)`;
        if (layerB) layerB.style.transform = `rotate(${b}deg)`;
        if (layerC) layerC.style.transform = `rotate(${c}deg)`;
      }
      raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);

    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener('yaria-spinner-change', onChange);
    };
  });
</script>

{#snippet body()}
  <div class="spinner-container">
    {#if active === 'classic'}
      <svg
        bind:this={classicEl}
        class="spinner-svg spinner-classic"
        width={size}
        height={size}
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <circle class="spinner-track" cx="12" cy="12" r="10" fill="none" stroke-width={classicStroke} />
        <circle class="spinner-arc" cx="12" cy="12" r="10" fill="none" stroke-width={classicStroke} />
      </svg>
    {:else if active === 'orbit'}
      <svg class="spinner-svg" width={size} height={size} viewBox="0 0 {vb} {vb}" aria-hidden="true">
        <circle class="orbit-core" cx="24" cy="24" r="5" />
        <circle class="orbit-ring" cx="24" cy="24" r="16" fill="none" />
        <circle class="orbit-ring dim" cx="24" cy="24" r="11" fill="none" />
        <g bind:this={layerA} class="spin-layer">
          <circle class="orbit-comet" cx="24" cy="8" r="2.6" />
          <circle cx="24" cy="8" r="1.1" fill="#fff" opacity="0.9" />
        </g>
        <g bind:this={layerB} class="spin-layer">
          <circle class="orbit-moon" cx="24" cy="35" r="1.5" />
        </g>
      </svg>
    {:else if active === 'singularity'}
      <svg class="spinner-svg" width={size} height={size} viewBox="0 0 {vb} {vb}" aria-hidden="true">
        <defs>
          <radialGradient id="singGlow-{uid}" cx="50%" cy="50%" r="50%">
            <stop offset="0%" stop-color="#f0abfc" stop-opacity="0.85" />
            <stop offset="50%" stop-color="#a78bfa" stop-opacity="0.35" />
            <stop offset="100%" stop-color="#a78bfa" stop-opacity="0" />
          </radialGradient>
        </defs>
        <circle class="sing-glow" cx="24" cy="24" r="15" fill="url(#singGlow-{uid})" />
        <circle class="sing-core" cx="24" cy="24" r="3.2" />
        <g bind:this={layerA} class="spin-layer">
          <circle class="sing-ring" cx="24" cy="24" r="17" fill="none" />
        </g>
      </svg>
    {:else if active === 'radar'}
      <svg class="spinner-svg" width={size} height={size} viewBox="0 0 {vb} {vb}" aria-hidden="true">
        <circle class="radar-disc" cx="24" cy="24" r="18" />
        <circle class="radar-ring" cx="24" cy="24" r="11" fill="none" />
        <g bind:this={layerA} class="spin-layer">
          <defs>
            <linearGradient id="radarSw-{uid}" x1="0" y1="0" x2="1" y2="0">
              <stop offset="0%" stop-color="#67e8f9" stop-opacity="0" />
              <stop offset="100%" stop-color="#67e8f9" stop-opacity="0.5" />
            </linearGradient>
          </defs>
          <path d="M24 24 L24 6 A18 18 0 0 1 39.5 31 Z" fill="url(#radarSw-{uid})" />
          <line x1="24" y1="24" x2="24" y2="6" class="radar-beam" />
        </g>
        <circle class="radar-dot" cx="24" cy="24" r="2" />
      </svg>
    {:else}
      <!-- warp -->
      <svg class="spinner-svg" width={size} height={size} viewBox="0 0 {vb} {vb}" aria-hidden="true">
        <g bind:this={layerC} class="spin-layer">
          <circle class="warp-arc a" cx="24" cy="24" r="16" fill="none" />
          <circle class="warp-arc b" cx="24" cy="24" r="16" fill="none" />
          <circle class="warp-arc c" cx="24" cy="24" r="16" fill="none" />
        </g>
      </svg>
    {/if}
    {#if message}
      <p class="spinner-message">{message}</p>
    {/if}
  </div>
{/snippet}

{#if overlay}
  <div class="spinner-overlay" transition:fade={{ duration: 200 }}>
    {@render body()}
  </div>
{:else}
  <div transition:fade={{ duration: 150 }}>
    {@render body()}
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
    overflow: visible;
  }

  .spinner-classic {
    transform-origin: center center;
  }

  .spin-layer {
    transform-origin: 24px 24px;
  }

  .spinner-track {
    stroke: rgba(255, 255, 255, 0.12);
  }

  .spinner-arc {
    stroke: #{$accent};
    stroke-linecap: round;
    stroke-dasharray: 40 80;
  }

  /* Orbit */
  .orbit-core {
    fill: rgba($accent, 0.45);
    animation: orbit-breathe 2.4s ease-in-out infinite;
  }
  .orbit-ring {
    stroke: rgba($accent, 0.28);
    stroke-width: 1.4;
    &.dim {
      stroke: rgba($accent, 0.14);
      stroke-width: 1;
    }
  }
  .orbit-comet {
    fill: $accent;
    filter: drop-shadow(0 0 3px rgba($accent, 0.9));
  }
  .orbit-moon {
    fill: #67e8f9;
    opacity: 0.85;
  }
  @keyframes orbit-breathe {
    0%,
    100% {
      opacity: 0.4;
    }
    50% {
      opacity: 0.85;
    }
  }

  /* Singularity */
  .sing-glow {
    animation: sing-pulse 2s ease-in-out infinite;
    transform-origin: 24px 24px;
  }
  .sing-core {
    fill: #fff;
    opacity: 0.9;
  }
  .sing-ring {
    stroke: #67e8f9;
    stroke-width: 1.3;
    stroke-dasharray: 5 7;
    opacity: 0.85;
  }
  @keyframes sing-pulse {
    0%,
    100% {
      opacity: 0.55;
      transform: scale(0.88);
    }
    50% {
      opacity: 1;
      transform: scale(1.05);
    }
  }

  /* Radar */
  .radar-disc {
    fill: rgba(103, 232, 249, 0.07);
    stroke: rgba(103, 232, 249, 0.28);
    stroke-width: 1;
  }
  .radar-ring {
    stroke: rgba(103, 232, 249, 0.2);
    stroke-width: 1;
  }
  .radar-beam {
    stroke: #67e8f9;
    stroke-width: 1.5;
  }
  .radar-dot {
    fill: #67e8f9;
  }

  /* Warp */
  .warp-arc {
    stroke-linecap: round;
    stroke-width: 2.2;
    &.a {
      stroke: $accent;
      stroke-dasharray: 18 82;
      opacity: 0.95;
    }
    &.b {
      stroke: #67e8f9;
      stroke-dasharray: 10 90;
      stroke-dashoffset: 40;
      opacity: 0.7;
    }
    &.c {
      stroke: #f0abfc;
      stroke-dasharray: 6 94;
      stroke-dashoffset: 70;
      opacity: 0.5;
    }
  }

  .spinner-message {
    color: $text-dim;
    font-size: 13px;
    margin: 0;
  }

  :global(.no-animations) {
    .orbit-core,
    .sing-glow {
      animation: none !important;
    }
  }
</style>
