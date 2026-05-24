import type { Action } from 'svelte/action';

/**
 * Fires callback when user clicks outside the element.
 */
export const clickOutside: Action<HTMLElement, () => void> = (node, callback) => {
  let cb = callback;

  function handleClick(e: MouseEvent) {
    if (!node.contains(e.target as Node)) {
      cb?.();
    }
  }

  document.addEventListener('click', handleClick, true);

  return {
    update(newCallback) {
      cb = newCallback;
    },
    destroy() {
      document.removeEventListener('click', handleClick, true);
    },
  };
};

/**
 * Traps focus within the element (for modals/dialogs).
 */
export const focusTrap: Action<HTMLElement> = (node) => {
  const focusableSelector =
    'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

  function handleKeydown(e: KeyboardEvent) {
    if (e.key !== 'Tab') return;

    const focusable = Array.from(
      node.querySelectorAll<HTMLElement>(focusableSelector)
    ).filter((el) => el.offsetParent !== null);

    if (focusable.length === 0) {
      e.preventDefault();
      return;
    }

    const first = focusable[0];
    const last = focusable[focusable.length - 1];

    if (e.shiftKey) {
      if (document.activeElement === first) {
        e.preventDefault();
        last.focus();
      }
    } else {
      if (document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  }

  node.addEventListener('keydown', handleKeydown);

  return {
    destroy() {
      node.removeEventListener('keydown', handleKeydown);
    },
  };
};

/**
 * Shows a tooltip on hover.
 */
export const tooltip: Action<HTMLElement, string> = (node, text) => {
  let tip: HTMLDivElement | null = null;
  let currentText = text;

  function show() {
    if (!currentText) return;
    tip = document.createElement('div');
    tip.className = 'svelte-tooltip';
    tip.textContent = currentText;
    Object.assign(tip.style, {
      position: 'fixed',
      zIndex: '99999',
      padding: '5px 10px',
      borderRadius: '6px',
      fontSize: '11px',
      fontWeight: '500',
      color: '#e4e4f0',
      background: 'rgba(20, 20, 40, 0.95)',
      border: '1px solid rgba(255,255,255,0.1)',
      backdropFilter: 'blur(8px)',
      pointerEvents: 'none',
      whiteSpace: 'nowrap',
      transition: 'opacity 0.15s',
      opacity: '0',
    });
    document.body.appendChild(tip);

    const rect = node.getBoundingClientRect();
    tip.style.left = `${rect.left + rect.width / 2 - tip.offsetWidth / 2}px`;
    tip.style.top = `${rect.top - tip.offsetHeight - 6}px`;

    requestAnimationFrame(() => {
      if (tip) tip.style.opacity = '1';
    });
  }

  function hide() {
    if (tip) {
      tip.remove();
      tip = null;
    }
  }

  node.addEventListener('mouseenter', show);
  node.addEventListener('mouseleave', hide);

  return {
    update(newText) {
      currentText = newText;
    },
    destroy() {
      hide();
      node.removeEventListener('mouseenter', show);
      node.removeEventListener('mouseleave', hide);
    },
  };
};

/**
 * Material-style ripple effect on click.
 */
export const ripple: Action<HTMLElement> = (node) => {
  node.style.position = node.style.position || 'relative';
  node.style.overflow = 'hidden';

  function handleClick(e: MouseEvent) {
    const rect = node.getBoundingClientRect();
    const size = Math.max(rect.width, rect.height) * 2;
    const x = e.clientX - rect.left - size / 2;
    const y = e.clientY - rect.top - size / 2;

    const rippleEl = document.createElement('span');
    Object.assign(rippleEl.style, {
      position: 'absolute',
      width: `${size}px`,
      height: `${size}px`,
      left: `${x}px`,
      top: `${y}px`,
      borderRadius: '50%',
      background: 'rgba(139, 108, 239, 0.25)',
      transform: 'scale(0)',
      animation: 'ripple-expand 0.5s ease-out forwards',
      pointerEvents: 'none',
    });

    node.appendChild(rippleEl);
    setTimeout(() => rippleEl.remove(), 600);
  }

  // Inject keyframes if not already present
  if (!document.getElementById('svelte-ripple-style')) {
    const style = document.createElement('style');
    style.id = 'svelte-ripple-style';
    style.textContent = `@keyframes ripple-expand { to { transform: scale(1); opacity: 0; } }`;
    document.head.appendChild(style);
  }

  node.addEventListener('click', handleClick);

  return {
    destroy() {
      node.removeEventListener('click', handleClick);
    },
  };
};

/**
 * Lazy loads images when they enter the viewport.
 * Pass the real src as the parameter. The element should have a placeholder src initially.
 */
export const lazyLoad: Action<HTMLImageElement, string> = (node, src) => {
  let realSrc = src;
  let loaded = false;

  function loadImage(imgSrc: string) {
    node.src = imgSrc;
    node.style.transition = 'opacity 0.3s ease';
    node.style.opacity = '0';
    node.onload = () => {
      node.style.opacity = '1';
    };
  }

  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          loaded = true;
          loadImage(realSrc);
          observer.unobserve(node);
        }
      });
    },
    { rootMargin: '200px' }
  );

  observer.observe(node);

  return {
    update(newSrc) {
      realSrc = newSrc;
      // If already visible, update the image immediately
      if (loaded) {
        loadImage(realSrc);
      }
    },
    destroy() {
      observer.disconnect();
    },
  };
};

/**
 * Fires callback on long press (default 500ms).
 */
export const longpress: Action<HTMLElement, { callback: () => void; duration?: number }> = (
  node,
  params
) => {
  let timer: ReturnType<typeof setTimeout>;
  let current = params;

  function handleMouseDown() {
    timer = setTimeout(() => {
      current.callback();
    }, current.duration || 500);
  }

  function handleMouseUp() {
    clearTimeout(timer);
  }

  node.addEventListener('mousedown', handleMouseDown);
  node.addEventListener('mouseup', handleMouseUp);
  node.addEventListener('mouseleave', handleMouseUp);

  return {
    update(newParams) {
      current = newParams;
    },
    destroy() {
      clearTimeout(timer);
      node.removeEventListener('mousedown', handleMouseDown);
      node.removeEventListener('mouseup', handleMouseUp);
      node.removeEventListener('mouseleave', handleMouseUp);
    },
  };
};

/**
 * Auto-focuses the element on mount.
 */
export const autoFocus: Action<HTMLElement> = (node) => {
  requestAnimationFrame(() => {
    node.focus();
  });
};
