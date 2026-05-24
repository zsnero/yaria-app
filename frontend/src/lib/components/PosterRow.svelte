<script lang="ts">
  import { fly } from 'svelte/transition';
  import PosterCard from './PosterCard.svelte';

  let { title, items, onItemClick = undefined }: {
    title: string;
    items: any[];
    onItemClick?: ((item: any) => void) | undefined;
  } = $props();

  let scrollContainer: HTMLDivElement | undefined = $state();
  let showLeftBtn = $state(false);
  let showRightBtn = $state(false);
  let isHovering = $state(false);

  function updateScrollButtons() {
    if (!scrollContainer) return;
    const { scrollLeft, scrollWidth, clientWidth } = scrollContainer;
    showLeftBtn = scrollLeft > 10;
    showRightBtn = scrollLeft + clientWidth < scrollWidth - 10;
  }

  $effect(() => {
    if (scrollContainer && items?.length) {
      // Wait for content to render
      requestAnimationFrame(() => updateScrollButtons());
    }
  });

  function scrollBy(direction: 'left' | 'right') {
    if (!scrollContainer) return;
    const amount = scrollContainer.clientWidth * 0.75;
    scrollContainer.scrollBy({
      left: direction === 'left' ? -amount : amount,
      behavior: 'smooth',
    });
  }
</script>

{#if items && items.length > 0}
  <div
    class="content-section"
    onmouseenter={() => (isHovering = true)}
    onmouseleave={() => (isHovering = false)}
  >
    <h2 class="section-title">{title}</h2>
    <div class="poster-row-wrap">
      {#if isHovering && showLeftBtn}
        <button
          class="scroll-btn scroll-btn-left"
          onclick={() => scrollBy('left')}
          aria-label="Scroll left"
          transition:fly={{ x: -20, duration: 200 }}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="15 18 9 12 15 6"></polyline>
          </svg>
        </button>
      {/if}

      <div
        class="poster-row"
        bind:this={scrollContainer}
        onscroll={updateScrollButtons}
      >
        {#each items as item, i (item.id || item.title + '_' + i)}
          <PosterCard
            title={item.title || item.name || ''}
            year={item.year || ''}
            rating={item.rating}
            poster={item.poster}
            type={item.type || item.media_type || 'movie'}
            onClick={onItemClick ? () => onItemClick(item) : undefined}
            index={i}
            progress={item.progress}
          />
        {/each}
      </div>

      {#if isHovering && showRightBtn}
        <button
          class="scroll-btn scroll-btn-right"
          onclick={() => scrollBy('right')}
          aria-label="Scroll right"
          transition:fly={{ x: 20, duration: 200 }}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="9 18 15 12 9 6"></polyline>
          </svg>
        </button>
      {/if}
    </div>
  </div>
{/if}

<style lang="scss">
  @use '../styles/variables' as *;

  .content-section {
    margin-bottom: 32px;
    position: relative;
  }

  .section-title {
    font-size: 20px;
    font-weight: 700;
    color: $text;
    margin-bottom: 18px;
    padding-left: 4px;
    letter-spacing: -0.3px;
  }

  .poster-row-wrap {
    position: relative;
  }

  .poster-row {
    display: flex;
    gap: 16px;
    overflow-x: auto;
    overflow-y: hidden;
    padding-bottom: 4px;
    scroll-behavior: smooth;

    // Hide scrollbar completely (Netflix style)
    scrollbar-width: none;
    -ms-overflow-style: none;

    &::-webkit-scrollbar {
      display: none;
    }
  }

  .scroll-btn {
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    z-index: 10;
    width: 36px;
    height: 36px;
    border-radius: 50%;
    background: rgba(5, 5, 16, 0.85);
    border: 1px solid $border-glass-hover;
    color: $text;
    @include flex-center;
    cursor: pointer;
    transition: background 0.2s, border-color 0.2s;
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);

    &:hover {
      background: rgba(20, 20, 40, 0.95);
      border-color: rgba($accent, 0.4);
      color: $accent-hover;
    }
  }

  .scroll-btn-left {
    left: -4px;
  }

  .scroll-btn-right {
    right: -4px;
  }
</style>
