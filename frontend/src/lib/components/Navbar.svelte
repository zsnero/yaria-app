<script lang="ts">
  import { activeTab, navigate } from '../stores/app';

  let tab = $state<'yaria' | 'mantorex'>('yaria');
  activeTab.subscribe(v => tab = v);

  function switchTab(t: 'yaria' | 'mantorex') {
    navigate(t === 'yaria' ? '#/yaria' : '#/mantorex');
  }

  function goDownloads() {
    navigate(tab === 'yaria' ? '#/yaria/downloads' : '#/torrent-downloads');
  }
</script>

<nav class="navbar">
  <a href="#/yaria" class="logo">
    <img src="/img/yaria-fox-nav.png" alt="Yaria" class="nav-logo-img" />
  </a>

  <div class="nav-center">
    <div class="nav-tabs">
      <button
        class="nav-tab"
        class:active={tab === 'yaria'}
        onclick={() => switchTab('yaria')}
      >
        Yaria
      </button>
      <button
        class="nav-tab"
        class:active={tab === 'mantorex'}
        onclick={() => switchTab('mantorex')}
      >
        Mantorex
      </button>
    </div>
  </div>

  <div class="nav-right">
    {#if tab === 'mantorex'}
      <div class="search-box">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" class="search-icon">
          <path d="M15.5 14h-.79l-.28-.27A6.471 6.471 0 0016 9.5 6.5 6.5 0 109.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
        </svg>
        <input
          type="text"
          placeholder="Search..."
          class="search-input"
          onkeydown={(e: KeyboardEvent) => {
            if (e.key === 'Enter') {
              const q = (e.currentTarget as HTMLInputElement).value.trim();
              if (q) navigate(`#/search?q=${encodeURIComponent(q)}`);
            }
          }}
        />
      </div>
    {/if}
    <button class="nav-link" onclick={goDownloads}>Downloads</button>
    {#if tab === 'mantorex'}
      <button class="nav-link" onclick={() => navigate('#/library')}>Library</button>
    {/if}
    <button class="nav-link" onclick={() => navigate('#/settings')}>Settings</button>
  </div>
</nav>

<style lang="scss">
  @use '../styles/variables' as *;

  .navbar {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    height: $nav-h;
    display: flex;
    align-items: center;
    padding: 0 32px;
    z-index: 100;
    background: rgba(10, 10, 20, 0.55);
    backdrop-filter: saturate(180%) blur(20px);
    -webkit-backdrop-filter: saturate(180%) blur(20px);
    border-bottom: 1px solid $border-glass;
  }

  .logo {
    display: flex;
    align-items: center;
  }

  .nav-logo-img {
    height: 34px;
    width: auto;
  }

  .nav-center {
    position: absolute;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    justify-content: center;
    z-index: 1;
    pointer-events: auto;
  }

  .nav-tabs {
    display: flex;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid $border-glass;
    border-radius: 99px;
    padding: 3px;
    gap: 2px;
  }

  .nav-tab {
    padding: 7px 22px;
    border-radius: 99px;
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.2px;
    color: $text-muted;
    transition: all 0.2s;

    &:hover {
      color: $text-dim;
    }

    &.active {
      background: rgba(139, 108, 239, 0.2);
      color: white;
      box-shadow: 0 2px 8px $accent-glow;
    }
  }

  .nav-right {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-left: auto;
    z-index: 2;
  }

  .search-box {
    display: flex;
    align-items: center;
    gap: 6px;
    background: rgba(255, 255, 255, 0.04);
    border: 1px solid $border-glass;
    border-radius: 99px;
    padding: 6px 14px;
  }

  .search-icon {
    color: $text-muted;
    flex-shrink: 0;
  }

  .search-input {
    background: none;
    border: none;
    color: $text;
    font-size: 13px;
    outline: none;
    width: 140px;

    &::placeholder {
      color: $text-muted;
    }
  }

  .nav-link {
    position: relative;
    padding: 6px 14px;
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0.2px;
    color: $text-dim;
    border-radius: 8px;
    transition: color 0.2s;

    &::after {
      content: '';
      position: absolute;
      bottom: -2px;
      left: 50%;
      width: 0;
      height: 2px;
      background: $accent;
      border-radius: 1px;
      transition: all 0.25s cubic-bezier(0.22, 1, 0.36, 1);
      transform: translateX(-50%);
    }

    &:hover {
      color: $text;

      &::after {
        width: 70%;
      }
    }
  }
</style>
