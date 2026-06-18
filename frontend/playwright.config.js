import { defineConfig, devices } from '@playwright/test';
import { fileURLToPath } from 'url';
import { dirname, resolve } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const FRONTEND_PORT = 5174;
const FRONTEND_URL = `http://localhost:${FRONTEND_PORT}`;
const LOOPBACK_NO_PROXY = 'localhost,127.0.0.1,::1';

process.env.NO_PROXY = process.env.NO_PROXY
  ? `${process.env.NO_PROXY},${LOOPBACK_NO_PROXY}`
  : LOOPBACK_NO_PROXY;
process.env.no_proxy = process.env.no_proxy
  ? `${process.env.no_proxy},${LOOPBACK_NO_PROXY}`
  : LOOPBACK_NO_PROXY;

export default defineConfig({
  testDir: resolve(__dirname, 'e2e'),
  testMatch: '**/*.spec.js',
  timeout: 30000,
  expect: { timeout: 10000 },
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: [
    ['list'],
    ['json', { outputFile: resolve(__dirname, 'e2e-results/test-results.json') }],
  ],
  outputDir: resolve(__dirname, 'e2e-results'),
  use: {
    baseURL: FRONTEND_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on-first-retry',
    headless: true,
    viewport: { width: 1280, height: 720 },
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: `npx vite --mode test --port ${FRONTEND_PORT}`,
    url: FRONTEND_URL,
    reuseExistingServer: !process.env.CI,
    timeout: 60000,
    env: {
      NO_PROXY: process.env.NO_PROXY,
      no_proxy: process.env.no_proxy,
    },
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
