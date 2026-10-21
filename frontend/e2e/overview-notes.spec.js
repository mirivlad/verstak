import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector } from './helpers.js';

async function overview(page) {
  await page.locator('.wt-label').filter({ hasText: /^Project$/ }).click();
  await page.getByRole('tab', { name: 'Overview', exact: true }).click();
  await expect(page.locator('[data-overview-root]')).toBeVisible();
}

test.beforeEach(async ({ page }) => {
  await page.goto('/');
  await waitForAppReady(page);
  await page.evaluate(() => {
    window.__wailsMock.putVaultFile('Project/Notes/Alpha.md', '# Alpha\n\nFirst note body.');
    window.__wailsMock.putVaultFile('Project/Notes/Beta.md', '# Beta\n\nSecond note body.');
  });
  await page.getByRole('tab', { name: 'Notes', exact: true }).click();
  await overview(page);
});

test('Overview resumes distinct notes, opens the named resource and keeps the summary linked to the list', async ({ page }) => {
  const consoleCollector = setupConsoleCollector(page);
  for (const [title, body] of [['Beta', 'Second note body.'], ['Alpha', 'First note body.']]) {
    const card = page.locator('.overview-continue-item').filter({ has: page.locator('strong', { hasText: new RegExp(`^${title}$`) }) });
    await expect(card).toBeVisible();
    await expect(card.locator('.overview-continue-item-action')).toHaveText('Open');
    await card.click();
    await expect(page.locator('[data-editor-mode]')).toHaveAttribute('data-resource-path', `Project/Notes/${title}.md`);
    await expect(page.locator('[data-preview]')).toContainText(body);
    await overview(page);
  }

  await page.locator('[data-overview-summary="notes"]').click();
  await expect(page.locator('.notes-root')).toContainText('Alpha');
  await expect(page.locator('.notes-root')).toContainText('Beta');
  await expect(page.locator('[data-editor-mode]')).toHaveCount(0);

  await overview(page);
  await page.locator('.overview-resource-row').getByRole('button', { name: 'Open: Overview.md', exact: true }).click();
  await expect(page.locator('[data-editor-mode]')).toHaveAttribute('data-resource-path', 'Project/Notes/Overview.md');
  consoleCollector.assertNoErrors();
});

test('a note deleted after loading Overview leaves a useful error and the remaining Notes list', async ({ page }) => {
  const consoleCollector = setupConsoleCollector(page);
  const card = page.locator('.overview-continue-item').filter({ has: page.locator('strong', { hasText: /^Beta$/ }) });
  await expect(card).toBeVisible();
  await page.evaluate(async () => {
    const [, error] = await window.go.api.App.TrashVaultPath('verstak.notes', 'Project/Notes/Beta.md');
    if (error) throw new Error(error);
  });
  await card.click();
  await expect(page.locator('.notes-root')).toContainText('Could not open the note');
  await expect(page.locator('.notes-root')).toContainText('Alpha');
  await expect(page.locator('[data-editor-mode]')).toHaveCount(0);
  consoleCollector.assertNoErrors();
});
