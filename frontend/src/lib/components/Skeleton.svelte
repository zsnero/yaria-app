<script lang="ts">
  interface Props {
    variant?: 'row' | 'card' | 'rect' | 'circle' | 'text';
    width?: string;
    height?: string;
    count?: number;
  }

  let {
    variant = 'row',
    width = '100%',
    height,
    count = 1,
  }: Props = $props();

  const defaultHeight: Record<string, string> = {
    row: '180px',
    card: '240px',
    rect: '120px',
    circle: '48px',
    text: '16px',
  };

  let h = $derived(height || defaultHeight[variant] || '120px');
</script>

{#each Array(count) as _, i}
  <div
    class="skeleton skeleton-{variant}"
    style:width={variant === 'circle' ? h : width}
    style:height={h}
    style:animation-delay="{i * 0.15}s"
  >
    {#if variant === 'row'}
      <div class="skeleton-row-inner">
        <div class="skeleton-poster"></div>
        <div class="skeleton-texts">
          <div class="skeleton-line" style="width: 70%"></div>
          <div class="skeleton-line short" style="width: 40%"></div>
        </div>
        <div class="skeleton-poster"></div>
        <div class="skeleton-poster"></div>
        <div class="skeleton-poster"></div>
        <div class="skeleton-poster"></div>
        <div class="skeleton-poster"></div>
      </div>
    {/if}
  </div>
{/each}

<style lang="scss">
  @use '../styles/variables' as *;

  .skeleton {
    border-radius: $radius-sm;
    background: rgba(255, 255, 255, 0.03);
    overflow: hidden;
    position: relative;

    &::after {
      content: '';
      position: absolute;
      inset: 0;
      background: linear-gradient(
        90deg,
        transparent 0%,
        rgba(255, 255, 255, 0.04) 40%,
        rgba(255, 255, 255, 0.06) 50%,
        rgba(255, 255, 255, 0.04) 60%,
        transparent 100%
      );
      animation: shimmer 1.8s ease-in-out infinite;
    }
  }

  .skeleton-circle {
    border-radius: 50%;
  }

  .skeleton-card {
    border-radius: $radius;
    border: 1px solid rgba(255, 255, 255, 0.04);
  }

  .skeleton-row {
    margin-bottom: 24px;
    padding: 16px;
  }

  .skeleton-row-inner {
    display: flex;
    gap: 14px;
    align-items: flex-start;
  }

  .skeleton-poster {
    width: 120px;
    height: 140px;
    flex-shrink: 0;
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.04);
  }

  .skeleton-texts {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 120px;
    flex-shrink: 0;
  }

  .skeleton-line {
    height: 14px;
    border-radius: 4px;
    background: rgba(255, 255, 255, 0.06);

    &.short {
      height: 10px;
    }
  }

  @keyframes shimmer {
    0% {
      transform: translateX(-100%);
    }
    100% {
      transform: translateX(100%);
    }
  }
</style>
