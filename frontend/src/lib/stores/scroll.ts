import { writable } from 'svelte/store';

const positions = writable<Record<string, number>>({});

export function saveScrollPosition(route: string) {
  positions.update(p => ({ ...p, [route]: window.scrollY }));
}

export function restoreScrollPosition(route: string) {
  let pos = 0;
  positions.subscribe(p => { pos = p[route] || 0; })();
  if (pos > 0) {
    requestAnimationFrame(() => window.scrollTo(0, pos));
  }
}
