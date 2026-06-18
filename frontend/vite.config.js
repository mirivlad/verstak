import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig(({ mode }) => {
  const isTest = mode === 'test';

  return {
    plugins: [svelte()],
    build: {
      outDir: 'dist',
      emptyOutDir: true,
    },
    server: {
      port: isTest ? 5174 : 5173,
      strictPort: true,
    },
  };
});
