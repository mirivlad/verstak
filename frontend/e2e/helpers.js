/**
 * Shared helpers for Verstak E2E tests.
 */
import { expect } from '@playwright/test';

/** Wait for the app to finish loading (loading screen disappears) */
export async function waitForAppReady(page) {
  // App shows "Loading Verstak..." initially, then renders the main layout
  await page.waitForSelector('main', { state: 'visible', timeout: 15000 });
  // Wait a bit for all async data to load
  await page.waitForTimeout(1000);
}

/** Open the secondary Plugin Manager route from the status-bar settings menu. */
export async function openPluginManager(page) {
  await page.locator('[data-settings-menu-button]').click();
  await page.locator('[data-settings-action="plugin-manager"]').click();
  await page.waitForSelector('.plugin-manager', { state: 'visible', timeout: 10000 });
}

/** Collect all console errors since last reset */
export function setupConsoleCollector(page) {
  const errors = [];
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });
  page.on('pageerror', (err) => {
    errors.push(err.message);
  });
  return {
    getErrors: () => errors,
    assertNoErrors: () => {
      if (errors.length > 0) {
        throw new Error(`Console errors detected:\n${errors.join('\n')}`);
      }
    },
  };
}

/** Reset mock state before each test */
export async function resetMockState(page) {
  await page.evaluate(() => {
    if (window.__wailsMock) {
      window.__wailsMock.reset();
    }
  });
}

/** Set plugin status in mock */
export async function setPluginStatus(page, pluginId, status, enabled) {
  await page.evaluate(
    ({ id, st, en }) => {
      if (window.__wailsMock) {
        window.__wailsMock.setPluginStatus(id, st, en);
      }
    },
    { id: pluginId, st: status, en: enabled }
  );
}
