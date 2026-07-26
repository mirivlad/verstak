import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

// Settings had no home: language lived in a status-bar dropdown, each plugin's
// settings opened as a modal inside the Plugin Manager, and nothing told a user
// where to look for a given option.
test.describe('Settings window', () => {
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

  async function openSettings(page) {
    await page.locator('[data-settings-menu-button]').click();
    await expect(page.locator('[data-settings-window]')).toBeVisible();
  }

  test('the gear opens a settings window listing built-in and plugin sections', async ({ page }) => {
    await openSettings(page);

    await expect(page.locator('[data-settings-section="general"]')).toBeVisible();
    await expect(page.locator('[data-settings-section="plugins"]')).toBeVisible();
    // Plugin settings panels are sections here, not entries in a menu that
    // grows with every plugin installed.
    await expect(page.locator('[data-settings-section^="plugin:"]').first()).toBeVisible();

    await expect(page.locator('[data-settings-language="system"]')).toBeVisible();
  });

  test('search narrows the section list by what a setting is called', async ({ page }) => {
    await openSettings(page);
    const sections = page.locator('[data-settings-section]');
    const total = await sections.count();
    expect(total).toBeGreaterThan(2);

    // "language" names a setting, not a section; finding General through it is
    // the point of the search.
    await page.locator('[data-settings-search]').fill('language');
    await expect(sections).toHaveCount(1);
    await expect(page.locator('[data-settings-section="general"]')).toBeVisible();

    await page.locator('[data-settings-search]').fill('zzzz no such setting');
    await expect(page.locator('[data-settings-no-matches]')).toBeVisible();

    await page.locator('[data-settings-search]').fill('');
    await expect(sections).toHaveCount(total);
  });

  test('the section list is a vertical tablist reachable from the keyboard', async ({ page }) => {
    await openSettings(page);
    const general = page.locator('[data-settings-section="general"]');
    await expect(general).toHaveAttribute('aria-selected', 'true');

    await general.focus();
    await page.keyboard.press('ArrowDown');
    await expect(page.locator('[data-settings-section="plugins"]')).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('[data-settings-section="plugins"]')).toBeFocused();

    await page.keyboard.press('Home');
    await expect(general).toHaveAttribute('aria-selected', 'true');

    // Only the selected tab is in the tab order; the arrows move within.
    await expect(general).toHaveAttribute('tabindex', '0');
    await expect(page.locator('[data-settings-section="plugins"]')).toHaveAttribute('tabindex', '-1');
  });

  test('the section you left is the section you come back to', async ({ page }) => {
    await openSettings(page);
    await page.locator('[data-settings-section="plugins"]').click();
    await expect(page.locator('[data-settings-section="plugins"]')).toHaveAttribute('aria-selected', 'true');

    await page.locator('[data-settings-window-close]').click();
    await expect(page.locator('[data-settings-window]')).toHaveCount(0);

    await openSettings(page);
    await expect(page.locator('[data-settings-section="plugins"]')).toHaveAttribute('aria-selected', 'true');
  });

  test('Escape closes the window', async ({ page }) => {
    await openSettings(page);
    await page.keyboard.press('Escape');
    await expect(page.locator('[data-settings-window]')).toHaveCount(0);
  });

  test('a plugin asking for its settings opens them here, at its own section', async ({ page }) => {
    // Ask the shell the way api.ui.openSettings does.
    await page.evaluate(() => {
      window.dispatchEvent(new CustomEvent('verstak:open-settings', {
        detail: { pluginId: 'verstak.sync', panelId: 'verstak.sync.settings' },
      }));
    });
    await expect(page.locator('[data-settings-window]')).toBeVisible();
    await expect(page.locator('[data-settings-section="plugin:verstak.sync:verstak.sync.settings"]'))
      .toHaveAttribute('aria-selected', 'true');
  });
});
