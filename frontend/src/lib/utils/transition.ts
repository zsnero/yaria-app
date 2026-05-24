import { cubicOut } from 'svelte/easing';

/**
 * Page transition: slide + fade.
 * Usage: <div transition:pageSlide={{ direction: 'right' }}>
 */
export function pageSlide(
  _node: HTMLElement,
  {
    duration = 250,
    direction = 'right',
  }: {
    duration?: number;
    direction?: 'left' | 'right' | 'up' | 'down';
  } = {},
) {
  const xMap: Record<string, number> = { left: -20, right: 20, up: 0, down: 0 };
  const yMap: Record<string, number> = { left: 0, right: 0, up: -20, down: 20 };

  return {
    duration,
    easing: cubicOut,
    css: (t: number) => {
      const x = (1 - t) * (xMap[direction] || 0);
      const y = (1 - t) * (yMap[direction] || 0);
      return `opacity: ${t}; transform: translate(${x}px, ${y}px);`;
    },
  };
}

/**
 * Stagger children animation for grids/lists.
 * Usage: <div transition:stagger={{ index: i }}>
 */
export function stagger(
  _node: HTMLElement,
  {
    duration = 300,
    delay = 50,
    index = 0,
  }: {
    duration?: number;
    delay?: number;
    index?: number;
  } = {},
) {
  return {
    duration,
    delay: index * delay,
    easing: cubicOut,
    css: (t: number) =>
      `opacity: ${t}; transform: translateY(${(1 - t) * 12}px);`,
  };
}

/**
 * Scale + fade for modals/overlays.
 * Usage: <div transition:modalTransition>
 */
export function modalTransition(
  _node: HTMLElement,
  { duration = 200 }: { duration?: number } = {},
) {
  return {
    duration,
    easing: cubicOut,
    css: (t: number) =>
      `opacity: ${t}; transform: scale(${0.95 + t * 0.05});`,
  };
}

/**
 * Simple backdrop fade.
 * Usage: <div transition:backdropFade>
 */
export function backdropFade(
  _node: HTMLElement,
  { duration = 200 }: { duration?: number } = {},
) {
  return {
    duration,
    css: (t: number) => `opacity: ${t};`,
  };
}
