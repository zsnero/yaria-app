<script lang="ts">
  import { stagger } from '../utils/transition';
  import { ripple, tooltip } from '../actions/index';

  const NO_POSTER = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='150' height='225'%3E%3Crect fill='%231a1a2e' width='150' height='225' rx='8'/%3E%3Ctext fill='%23555570' x='75' y='112' text-anchor='middle' font-size='12' font-family='sans-serif'%3ENo Poster%3C/text%3E%3C/svg%3E";

  let { title, year = '', rating = undefined, poster = '', type = 'movie', onClick = undefined, index = 0, progress = undefined }: {
    title: string;
    year?: string;
    rating?: number | undefined;
    poster?: string;
    type?: string;
    onClick?: (() => void) | undefined;
    index?: number;
    progress?: number | undefined;
  } = $props();

  let imgSrc = $derived(poster || NO_POSTER);
  let ratingDisplay = $derived(rating ? rating.toFixed(1) : '');

  function handleImgError(e: Event) {
    const img = e.target as HTMLImageElement;
    img.src = NO_POSTER;
  }

  function handleKeydown(e: KeyboardEvent) {
    if ((e.key === 'Enter' || e.key === ' ') && onClick) {
      e.preventDefault();
      onClick();
    }
  }
</script>

<div
  class="poster-card"
  use:ripple
  use:tooltip={title}
  in:stagger={{ index }}
  onclick={onClick}
  onkeydown={handleKeydown}
  role={onClick ? 'button' : undefined}
  tabindex={onClick ? 0 : undefined}
>
  <div class="poster-img-wrap">
    <img
      src={imgSrc}
      alt={title}
      loading="lazy"
      onerror={handleImgError}
    />
    {#if progress !== undefined && progress > 0}
      <div class="progress-bar">
        <div class="progress-fill" style="width: {Math.min(100, Math.max(0, progress))}%"></div>
      </div>
    {/if}
  </div>
  <div class="card-info">
    <div class="card-title">{title}</div>
    <div class="card-meta">
      {#if year}
        <span class="yr">{year}</span>
      {/if}
      {#if ratingDisplay}
        <span class="rt">{ratingDisplay}</span>
      {/if}
      {#if type === 'tv'}
        <span class="badge-tv">TV</span>
      {/if}
    </div>
  </div>
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  .poster-card {
    flex-shrink: 0;
    width: 180px;
    cursor: pointer;
    transition: transform 0.3s cubic-bezier(0.22, 1, 0.36, 1), box-shadow 0.3s ease;
    border-radius: $radius-sm;
    overflow: hidden;
    position: relative;

    &:hover {
      transform: scale(1.05) translateY(-4px);
      box-shadow: 0 12px 32px rgba(0, 0, 0, 0.5);
    }

    &:focus-visible {
      outline: 2px solid $accent;
      outline-offset: 2px;
    }
  }

  .poster-img-wrap {
    position: relative;

    img {
      width: 100%;
      height: 270px;
      object-fit: cover;
      display: block;
      border-radius: $radius-sm $radius-sm 0 0;
      background: rgba(255, 255, 255, 0.02);
    }
  }

  .progress-bar {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 3px;
    background: rgba(0, 0, 0, 0.5);
  }

  .progress-fill {
    height: 100%;
    background: $accent;
    border-radius: 0 2px 2px 0;
    transition: width 0.3s ease;
  }

  .card-info {
    padding: 8px 10px 10px;
    background: $bg-glass;
    border: 1px solid $border-glass;
    border-top: none;
    border-radius: 0 0 $radius-sm $radius-sm;
  }

  .card-title {
    font-size: 12px;
    font-weight: 500;
    color: $text;
    @include truncate;
  }

  .card-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 4px;
    font-size: 11px;
    color: $text-muted;
  }

  .yr {
    color: $text-dim;
  }

  .rt {
    color: $yellow;
    font-weight: 600;
  }

  .badge-tv {
    background: rgba($accent, 0.2);
    color: $accent-hover;
    padding: 1px 5px;
    border-radius: 4px;
    font-size: 10px;
    font-weight: 600;
  }
</style>
