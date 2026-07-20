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

  test('indexes a real Wails file listing containing exactly two entries', async ({ page }) => {
    await page.evaluate(() => {
      window.__wailsMock.putVaultFile('TwoEntryFolder/first-exact-result.txt', 'first exact result');
      window.__wailsMock.putVaultFile('TwoEntryFolder/second-exact-result.txt', 'second exact result');
      window.__wailsMock.setListVaultFilesResponseMode('plain');
      window.dispatchEvent(new CustomEvent('verstak:files-changed'));
    });

    const input = page.locator('[data-global-search-input]');
    await input.focus();
    await input.fill('second-exact-result');
    await expect(page.locator('[data-global-search-result-path="TwoEntryFolder/second-exact-result.txt"]')).toBeVisible({ timeout: 5000 });
  });

  test('drops archived browser captures while indexing their converted note', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:workspace:Project': [{
          captureId: 'converted-search-capture',
          capturedAt: '2026-07-21T01:00:00.000Z',
          kind: 'page',
          url: 'https://example.com/converted-search-token',
          title: 'Converted Search Token',
          text: 'converted-search-token',
          workspaceRootPath: 'Project',
          globalState: 'archived',
        }],
      });
      window.__wailsMock.putVaultFile('Project/Notes/Converted_Search_Note.md', '# Converted Search Note\nconverted-search-token');
      window.dispatchEvent(new CustomEvent('verstak:files-changed'));
    });

    const input = page.locator('[data-global-search-input]');
    await input.focus();
    await input.fill('converted-search-token');
    await expect(page.locator('[data-global-search-result-path="Project/Notes/Converted_Search_Note.md"]')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('[data-global-search-result-type="Browser"]')).toHaveCount(0);
  });

  test('results panel is about three input widths wide and at least five rows tall', async ({ page }) => {
    await page.evaluate(() => {
      for (let index = 1; index <= 5; index += 1) {
        window.__wailsMock.putVaultFile(`Project/Files/panel-sizing-${index}.txt`, `panel sizing ${index}`);
      }
      window.dispatchEvent(new CustomEvent('verstak:files-changed'));
    });

    const input = page.locator('[data-global-search-input]');
    await input.focus();
    await input.fill('panel-sizing');
    const panel = page.locator('[data-global-search-results]');
    await expect(panel.locator('[data-global-search-result-type="File"]')).toHaveCount(5, { timeout: 5000 });
    const [inputBox, panelBox, rowBoxes] = await Promise.all([
      input.boundingBox(),
      panel.boundingBox(),
      panel.locator('[data-global-search-result-type="File"]').evaluateAll((rows) => rows.map((row) => row.getBoundingClientRect().height)),
    ]);
    expect(panelBox.width).toBeGreaterThanOrEqual(inputBox.width * 2.8);
    expect(panelBox.height).toBeGreaterThanOrEqual(rowBoxes.slice(0, 5).reduce((sum, height) => sum + height, 0));
  });
});
