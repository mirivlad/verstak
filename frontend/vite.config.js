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
    optimizeDeps: {
      include: [
        'lucide-svelte/icons/briefcase',
        'lucide-svelte/icons/chevron-down',
        'lucide-svelte/icons/chevron-right',
        'lucide-svelte/icons/circle',
        'lucide-svelte/icons/flask-conical',
        'lucide-svelte/icons/folder',
        'lucide-svelte/icons/layout-grid',
        'lucide-svelte/icons/panels-top-left',
        'lucide-svelte/icons/pencil',
        'lucide-svelte/icons/plug',
        'lucide-svelte/icons/puzzle',
        'lucide-svelte/icons/settings',
        'lucide-svelte/icons/shield',
        'lucide-svelte/icons/trash-2',
        'lucide-svelte/icons/triangle-alert',
      ],
    },
  };
});
