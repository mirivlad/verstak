import { defineConfig } from '@playwright/test';
import baseConfig from './playwright.config.js';
import { fileURLToPath } from 'url';
import { dirname, resolve } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export default defineConfig({
  ...baseConfig,
  testMatch: '**/*.visual.js',
  retries: 0,
  globalTimeout: process.env.CI ? 5 * 60 * 1000 : undefined,
  reporter: [['list']],
  outputDir: resolve(__dirname, 'e2e-results/visual-runtime'),
  use: {
    ...baseConfig.use,
    trace: 'off',
    screenshot: 'off',
    video: 'off',
    viewport: { width: 1200, height: 820 },
  },
});
