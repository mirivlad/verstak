import { test, expect } from '@playwright/test';
import { resetMockState, waitForAppReady } from './helpers.js';

test.describe('Progressive global search index', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await waitForAppReady(page);
    await resetMockState(page);
  });

  test('publishes a nested filename before delayed content reads finish', async ({ page }) => {
    await page.evaluate(() => {
      window.__wailsMock.putVaultFile('Project/Files/test.txt', 'content arrives later');
      window.__wailsMock.setReadTextDelay(1500);
      window.dispatchEvent(new CustomEvent('verstak:files-changed'));
    });
    const input = page.locator('[data-global-search-input]');
    await input.focus();
    await input.fill('test.txt');
    await expect(page.locator('[data-global-search-result-path="Project/Files/test.txt"]')).toBeVisible({ timeout: 3000 });
    await expect(page.locator('[data-global-search-results]')).not.toContainText('No results');
  });

  test('refresh signal makes a newly created file searchable', async ({ page }) => {
    const input = page.locator('[data-global-search-input]');
    await input.focus();
    await page.waitForTimeout(300);
    await page.evaluate(() => {
      window.__wailsMock.putVaultFile('Project/Files/after-refresh.txt', 'new file');
      window.dispatchEvent(new CustomEvent('verstak:files-changed'));
    });
    await input.fill('after-refresh');
    await expect(page.locator('[data-global-search-result-path="Project/Files/after-refresh.txt"]')).toBeVisible({ timeout: 5000 });
  });
});
