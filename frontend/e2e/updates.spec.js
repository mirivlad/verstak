import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

const RELEASE = {
  latest: 'v9.9.9',
  newer: true,
  url: 'https://github.com/mirivlad/verstak/releases/tag/v9.9.9',
};

// Checking for a new version reaches the network. It is off until the user
// turns it on, a manual check is one click, and a found release is a link to
// its page -- nothing is downloaded.
test.describe('Update check', () => {
  let consoleCollector;

  test.beforeEach(async ({ page }) => {
    consoleCollector = setupConsoleCollector(page);
    await resetMockState(page);
    await page.goto('/');
    await waitForAppReady(page);
  });

  test.afterEach(async () => {
    consoleCollector.assertNoErrors();
  });

  async function openGeneralSettings(page) {
    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-section="general"]').click();
  }

  test('startup check is off by default and the toggle persists', async ({ page }) => {
    await openGeneralSettings(page);
    const toggle = page.locator('[data-settings-update-check-toggle]');
    await expect(toggle).not.toBeChecked();
    expect(await page.evaluate(() => window.__wailsMockUpdateChecks || 0)).toBe(0);

    await toggle.check();
    await expect.poll(() => page.evaluate(async () => (await window.go.api.App.GetAppSettings()).checkForUpdates)).toBe(true);
  });

  test('manual check reports a newer release and opens its page', async ({ page }) => {
    await page.evaluate((release) => { window.__VERSTAK_MOCK_UPDATE__ = release; }, RELEASE);
    await openGeneralSettings(page);
    await page.locator('[data-settings-update-check-now]').click();

    await expect(page.locator('[data-settings-update-status]')).toContainText('v9.9.9');
    await page.locator('[data-settings-update-open]').click();
    await expect.poll(() => page.evaluate(() => window.__wailsMockOpenedUpdatePages || [])).toEqual([RELEASE.url]);
  });

  test('a failed check says so instead of claiming the version is current', async ({ page }) => {
    await openGeneralSettings(page);
    await page.locator('[data-settings-update-check-now]').click();
    await expect(page.locator('[data-settings-update-status]')).toHaveClass(/is-error/);
    await expect(page.locator('[data-settings-update-open]')).toHaveCount(0);
  });

  test('a startup announcement shows a status bar link to the release', async ({ page }) => {
    await expect(page.locator('[data-status-update-available]')).toHaveCount(0);
    await page.evaluate(async (release) => {
      window.__VERSTAK_MOCK_UPDATE__ = release;
      const [result] = await window.go.api.App.CheckForUpdates();
      window.dispatchEvent(new CustomEvent('verstak:update-available', { detail: result }));
    }, RELEASE);

    const badge = page.locator('[data-status-update-available="v9.9.9"]');
    await expect(badge).toBeVisible();
    await badge.click();
    await expect.poll(() => page.evaluate(() => window.__wailsMockOpenedUpdatePages || [])).toEqual([RELEASE.url]);
  });
});
