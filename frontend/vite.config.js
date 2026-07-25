import { resolve } from 'node:path';
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig(({ mode }) => {
  const isTest = mode === 'test';

  return {
    plugins: [svelte()],
    define: {
      // Statically false outside `--mode test`, so Rollup eliminates the
      // branch that imports the Wails mock and the mock never reaches a
      // shipped bundle.
      __VERSTAK_TEST_MOCK__: JSON.stringify(isTest),
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true,
    },
    server: {
      port: isTest ? 5174 : 5173,
      strictPort: true,
      fs: {
        allow: [resolve(__dirname, '..', '..')],
      },
    },
  };
});
