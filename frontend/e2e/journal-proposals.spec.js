import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState, readJournalText } from './helpers.js';

// Activity's view is never opened in this spec. That is the point: the Journal
// reaches possible entries through the plugin's activation, not through
// whatever happens to be mounted.
async function seedActivity(page) {
  await page.evaluate(async () => {
    const PROJECT_ID = '11111111-1111-4111-8111-111111111111';
    await window.go.api.App.WritePluginDataNDJSON('verstak.activity', 'activity-events', [
      {
        activityId: 'proposal-note',
        type: 'note.saved',
        title: 'Saved note',
        occurredAt: '2026-07-20T09:00:00.000Z',
        sourcePluginId: 'verstak.notes',
        workspaceRootPath: 'Project',
        workspaceId: PROJECT_ID,
        payload: { workspaceRootPath: 'Project', workspaceId: PROJECT_ID },
      },
      {
        activityId: 'proposal-file',
        type: 'file.changed',
        title: 'Changed file',
        occurredAt: '2026-07-20T09:15:00.000Z',
        sourcePluginId: 'verstak.files',
        workspaceRootPath: 'Project',
        workspaceId: PROJECT_ID,
        payload: { workspaceRootPath: 'Project', workspaceId: PROJECT_ID },
      },
    ]);
  });
}

test('the global Journal lists proposals without Activity being open', async ({ page }) => {
  const consoleCollector = setupConsoleCollector(page);
  await resetMockState(page);
  await page.goto('/');
  await waitForAppReady(page);
  await seedActivity(page);

  await page.locator('.sidebar .plugin-item').filter({ hasText: 'Journal' }).click();
  const journal = page.locator('.journal-root');
  await expect(journal.locator('.journal-title')).toHaveText('Journal');

  const proposal = journal.locator('[data-journal-proposal]');
  await expect(proposal).toHaveCount(1);
  await expect(proposal).toContainText('Project');
  await expect(proposal).toContainText('15 min');

  await proposal.locator('[data-journal-action="review-proposal"]').click();
  await expect(journal.locator('[data-journal-candidate]')).toBeVisible();
  await expect(journal.locator('[data-journal-input="workspaceRootPath"]')).toHaveValue('Project');
  await expect(journal.locator('[data-journal-input="minutes"]')).toHaveValue('15');

  await journal.locator('[data-journal-input="title"]').fill('Worked on the project');
  await journal.locator('[data-journal-action="save-entry"]').click();

  await expect.poll(async () => readJournalText(page, 'Project')).toContain('### Worked on the project');
  // Accepted, so it is no longer a proposal.
  await expect(journal.locator('[data-journal-proposal]')).toHaveCount(0);
  consoleCollector.assertNoErrors();
});
