import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test('global Journal creates an entry in the selected Deal', async ({ page }) => {
  const consoleCollector = setupConsoleCollector(page);
  await resetMockState(page);
  await page.goto('/');
  await waitForAppReady(page);
  await page.evaluate(async () => {
    await window.go.api.App.WritePluginSettings('verstak.journal', {
      'worklog:workspace:Project': [{ entryId: 'existing-project-entry', workspaceRootPath: 'Project', date: '2026-07-14', title: 'Existing entry', minutes: 5 }],
    });
  });

  await page.locator('.sidebar .plugin-item').filter({ hasText: 'Journal' }).click();
  const journal = page.locator('.journal-root');
  await expect(journal.locator('.journal-title')).toHaveText('Journal');
  await journal.locator('[data-journal-action="add"]').click();
  await journal.locator('[data-journal-input="workspaceRootPath"]').selectOption('Project');
  await journal.locator('[data-journal-input="title"]').fill('Prepare project handoff');
  await journal.locator('[data-journal-input="minutes"]').fill('30');
  await journal.locator('[data-journal-action="save-entry"]').click();

  await expect.poll(async () => page.evaluate(async () => {
    const result = await window.go.api.App.ReadPluginSettings('verstak.journal');
    const settings = Array.isArray(result) ? result[0] : result;
    return settings['worklog:workspace:Project']?.[0]?.title;
  })).toBe('Prepare project handoff');
  await expect(journal).toContainText('Prepare project handoff');

  await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
  await page.getByRole('tab', { name: 'Journal' }).click();
  await expect(page.locator('.journal-root')).toContainText('Prepare project handoff');
  consoleCollector.assertNoErrors();
});
