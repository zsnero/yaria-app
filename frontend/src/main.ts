import { mount } from 'svelte';
import App from './App.svelte';
import './lib/styles/global.scss';
import { applyUISettings } from './lib/stores/app';

// Apply saved UI settings on load
applyUISettings();

const app = mount(App, {
  target: document.getElementById('app')!,
});

export default app;
