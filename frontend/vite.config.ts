import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import path from 'path';

export default defineConfig({
  plugins: [svelte()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      '@components': path.resolve(__dirname, 'src/lib/components'),
      '@pages': path.resolve(__dirname, 'src/lib/pages'),
      '@stores': path.resolve(__dirname, 'src/lib/stores'),
      '@api': path.resolve(__dirname, 'src/lib/api'),
      '@types': path.resolve(__dirname, 'src/lib/types'),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: false,
    minify: true,
  },
  server: {
    port: 5173,
  },
});
