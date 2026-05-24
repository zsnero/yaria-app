import { writable } from 'svelte/store';

export interface Toast {
  id: number;
  message: string;
  type: 'info' | 'success' | 'error' | 'warning';
  duration: number;
}

// Alias for backward compatibility
export type ToastMessage = Toast;

let nextId = 0;

export const toasts = writable<Toast[]>([]);

export function toast(message: string, type: Toast['type'] = 'info', duration = 4000) {
  const id = nextId++;
  toasts.update((t) => [...t, { id, message, type, duration }]);
  setTimeout(() => dismissToast(id), duration);
  return id;
}

export function dismissToast(id: number) {
  toasts.update((t) => t.filter((item) => item.id !== id));
}

export function toastSuccess(message: string, duration = 4000) {
  return toast(message, 'success', duration);
}

export function toastError(message: string, duration = 5000) {
  return toast(message, 'error', duration);
}

export function toastInfo(message: string, duration = 4000) {
  return toast(message, 'info', duration);
}

export function toastWarning(message: string, duration = 5000) {
  return toast(message, 'warning', duration);
}
