import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// The Wails mock imports the real official plugin bundles from the sibling
// repository, which is what makes the e2e suite exercise the actual plugins
// rather than stand-ins. Rollup resolves that import graph even in a
// production build, where the mock is dead code and nothing from it ships --
// so a checkout of verstak-desktop alone could not be built at all, and CI
// never once got past this step.
//
// Outside `--mode test` a missing sibling now yields empty content. The mock
// is eliminated anyway, so the difference is invisible in the output and
// production-bundle-test.mjs still proves it. In test mode nothing is
// substituted: e2e without the real plugin sources would be testing nothing,
// and should fail loudly.
function optionalSiblingPluginSources(isTest) {
  return {
    name: 'verstak-optional-sibling-plugin-sources',
    // Ahead of Vite's own resolver, which fails on the missing file first.
    enforce: 'pre',
    resolveId(source, importer) {
      if (isTest || !source.includes('verstak-official-plugins') || !source.endsWith('?raw')) return null;
      const target = resolve(importer ? resolve(importer, '..') : __dirname, source.slice(0, -4));
      return existsSync(target) ? null : source;
    },
    load(id) {
      if (isTest || !id.includes('verstak-official-plugins') || !id.endsWith('?raw')) return null;
      return 'export default "";';
    },
  };
}

export default defineConfig(({ mode }) => {
  const isTest = mode === 'test';

  return {
    plugins: [svelte(), optionalSiblingPluginSources(isTest)],
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
