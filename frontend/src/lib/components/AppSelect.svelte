<script lang="ts">
  type Option = { value: string | number; label: string };

  let {
    value = $bindable<string | number>(''),
    options = [] as Option[],
    class: className = '',
    disabled = false,
    onchange = undefined as (() => void) | undefined,
  } = $props();

  let open = $state(false);
  let rootEl: HTMLDivElement | undefined = $state();
  let listEl: HTMLDivElement | undefined = $state();

  const selectedLabel = $derived(
    options.find((o) => String(o.value) === String(value))?.label ?? String(value ?? '')
  );

  function toggle(e?: MouseEvent) {
    e?.stopPropagation();
    if (disabled) return;
    open = !open;
  }

  function pick(opt: Option) {
    value = opt.value;
    open = false;
    onchange?.();
  }

  function onDocPointerDown(e: PointerEvent) {
    if (!open || !rootEl) return;
    if (!rootEl.contains(e.target as Node)) open = false;
  }

  function onKeydown(e: KeyboardEvent) {
    if (!open) {
      if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
        e.preventDefault();
        open = true;
      }
      return;
    }
    if (e.key === 'Escape') {
      e.preventDefault();
      open = false;
      return;
    }
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      const idx = options.findIndex((o) => String(o.value) === String(value));
      const next =
        e.key === 'ArrowDown'
          ? Math.min(options.length - 1, Math.max(0, idx) + 1)
          : Math.max(0, (idx < 0 ? 0 : idx) - 1);
      if (options[next]) pick(options[next]);
    }
  }

  $effect(() => {
    if (!open) return;
    // Scroll selected into view
    queueMicrotask(() => {
      const sel = listEl?.querySelector('.app-select-option.is-selected') as HTMLElement | null;
      sel?.scrollIntoView({ block: 'nearest' });
    });
  });
</script>

<svelte:window onpointerdown={onDocPointerDown} />

<div
  class="app-select {className}"
  class:is-open={open}
  class:is-disabled={disabled}
  bind:this={rootEl}
  role="combobox"
  aria-expanded={open}
  aria-haspopup="listbox"
  tabindex={disabled ? -1 : 0}
  onkeydown={onKeydown}
>
  <button type="button" class="app-select-trigger" {disabled} onclick={(e) => toggle(e)}>
    <span class="app-select-label">{selectedLabel}</span>
    <span class="app-select-chevron" aria-hidden="true">
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M6 9l6 6 6-6" />
      </svg>
    </span>
  </button>

  {#if open}
    <div class="app-select-menu" role="listbox" bind:this={listEl}>
      {#each options as opt}
        <button
          type="button"
          class="app-select-option"
          class:is-selected={String(opt.value) === String(value)}
          role="option"
          aria-selected={String(opt.value) === String(value)}
          onclick={() => pick(opt)}
        >
          {opt.label}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style lang="scss">
  @use '../styles/variables' as *;

  .app-select {
    position: relative;
    width: 100%;
    outline: none;
  }

  .app-select-trigger {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 12px 14px;
    background: rgba(20, 20, 30, 0.95);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: $radius-sm;
    color: $text;
    font-size: 14px;
    font-family: inherit;
    cursor: pointer;
    text-align: left;
    transition: border-color 0.15s, box-shadow 0.15s;
  }

  .app-select.is-open .app-select-trigger,
  .app-select-trigger:hover {
    border-color: rgba(255, 255, 255, 0.14);
  }

  .app-select:focus-within .app-select-trigger {
    border-color: rgba($accent, 0.45);
    box-shadow: 0 0 0 2px rgba($accent, 0.12);
  }

  .app-select.is-disabled .app-select-trigger {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .app-select-label {
    @include truncate;
    flex: 1;
  }

  .app-select-chevron {
    color: $text-dim;
    display: flex;
    flex-shrink: 0;
    transition: transform 0.15s;
  }

  .app-select.is-open .app-select-chevron {
    transform: rotate(180deg);
  }

  .app-select-menu {
    position: absolute;
    left: 0;
    right: 0;
    top: calc(100% + 6px);
    z-index: 10050;
    max-height: 240px;
    overflow-y: auto;
    padding: 6px;
    background: #141422;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: $radius-sm;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.55);
  }

  .app-select-option {
    width: 100%;
    display: block;
    padding: 10px 12px;
    border: none;
    border-radius: 8px;
    background: transparent;
    color: $text;
    font-size: 13px;
    font-family: inherit;
    text-align: left;
    cursor: pointer;
    transition: background 0.12s;
  }

  .app-select-option:hover {
    background: rgba(255, 255, 255, 0.06);
  }

  .app-select-option.is-selected {
    background: rgba($accent, 0.18);
    color: #fff;
  }
</style>
