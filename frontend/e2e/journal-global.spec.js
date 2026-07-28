import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, readJournalText } from './helpers.js';

test('global Journal creates an entry in the selected Deal', async ({ page }) => {
  const consoleCollector = setupConsoleCollector(page);
  await resetMockState(page);
  await page.goto('/');
  await waitForAppReady(page);
  // A worklog that was already there, in the Deal, the way the journal keeps it.
  await page.evaluate(async () => {
    await window.go.api.App.CreateVaultFolder('verstak.journal', 'Project/Журнал');
    await window.go.api.App.WriteVaultTextFile('verstak.journal', 'Project/Журнал/2026-07.md', [
      '---', 'verstak: worklog', 'version: 1', 'deal: "Project"', 'month: 2026-07', '---',
      '', '# Journal', '', '## 2026-07-14', '', '### Existing entry', '', '5 min · non-billable', '',
      '<!-- verstak-entry {"entryId":"existing-project-entry","minutes":5,"billable":false} -->', '',
    ].join('\n'), { createIfMissing: true, overwrite: true, service: true });
  });

  await page.locator('.sidebar .plugin-item').filter({ hasText: 'Journal' }).click();
  const journal = page.locator('.journal-root');
  await expect(journal.locator('.journal-title')).toHaveText('Journal');
  await journal.locator('[data-journal-action="add"]').click();
  await journal.locator('[data-journal-input="workspaceRootPath"]').selectOption('Project');
  await journal.locator('[data-journal-input="title"]').fill('Prepare project handoff');
  await journal.locator('[data-journal-input="minutes"]').fill('30');
  await journal.locator('[data-journal-action="save-entry"]').click();

  await expect.poll(async () => readJournalText(page, 'Project')).toContain('### Prepare project handoff');
  await expect(journal).toContainText('Prepare project handoff');

  await page.locator('.wt-label').filter({ hasText: 'Project' }).click();
  await page.getByRole('tab', { name: 'Journal' }).click();
  await expect(page.locator('.journal-root')).toContainText('Prepare project handoff');
  consoleCollector.assertNoErrors();
});
