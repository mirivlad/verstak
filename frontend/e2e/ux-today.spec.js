import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('UX Overview workspace flow', () => {
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

  test('workspace opens with Overview before plugin tools', async ({ page }) => {
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });

    const tabs = page.getByRole('tab');
    await expect(tabs.nth(0)).toHaveText('Overview');
    await expect(tabs.nth(1)).toHaveText('Files');
    await expect(page.getByRole('tab', { name: 'Overview' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.getByRole('tab', { name: 'Today' })).toHaveCount(0);

    const overview = page.locator('[data-overview-root]');
    await expect(overview).toBeVisible();
    await expect(overview.locator('[data-overview-section="continue"]')).toContainText('Continue working');
    await expect(overview.locator('[data-overview-section="recent"]')).toContainText('Recent changes');
    await expect(overview.locator('[data-overview-section="attention"]')).toContainText('Needs attention');
    await expect(overview.locator('[data-overview-section="quick-actions"]')).toContainText('Quick actions');
    await expect(overview.locator('[data-overview-summary="notes"]')).toContainText('recent changes');
    await expect(overview.locator('[data-overview-summary="captures"]')).toContainText('unprocessed captures');
    await expect(overview).toContainText('No clear resume point yet');
    await expect(overview).toContainText('No meaningful changes for this filter yet');
  });

  test('Overview quick action opens Browser Inbox workspace tool', async ({ page }) => {
    await page.locator('[data-overview-action="browser-inbox"]').click();

    await expect(page.getByRole('tab', { name: 'Browser Inbox' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.browser-inbox-root')).toBeVisible({ timeout: 10000 });
  });

  test('Overview prioritizes resume work and filters meaningful recent changes', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:workspace:Project': [
          {
            captureId: 'overview-capture-1',
            capturedAt: '2026-06-30T08:00:00.000Z',
            kind: 'page',
            url: 'https://example.com/research',
            title: 'Research Report',
            domain: 'example.com',
            workspaceRootPath: 'Project',
          },
          {
            captureId: 'overview-capture-2',
            capturedAt: '2026-06-30T08:15:00.000Z',
            kind: 'selection',
            title: 'Quote to process',
            domain: 'example.com',
            workspaceRootPath: 'Project',
          },
        ],
      });
      await window.go.api.App.WritePluginSettings('verstak.activity', {
        'events:workspace:Project': [
          {
            activityId: 'overview-selected-file',
            occurredAt: '2026-06-30T08:50:00.000Z',
            type: 'file.selected',
            title: 'Selected file',
            summary: 'Project/draft.md',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'overview-opened-file',
            occurredAt: '2026-06-30T08:40:00.000Z',
            type: 'file.opened',
            title: 'draft.md',
            summary: 'Project/draft.md',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'overview-note',
            occurredAt: '2026-06-30T08:25:00.000Z',
            type: 'note.saved',
            title: 'Overview',
            summary: 'Project/Notes/Overview.md',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'overview-file',
            occurredAt: '2026-06-30T08:20:00.000Z',
            type: 'file.changed',
            title: 'draft.md',
            summary: 'Project/draft.md',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'overview-workspace-selected',
            occurredAt: '2026-06-30T08:10:00.000Z',
            type: 'case.selected',
            title: 'Workspace selected',
            workspaceRootPath: 'Project',
          },
        ],
      });
      await window.go.api.App.WritePluginSettings('verstak.journal', {
        'worklog:workspace:Project': [
          {
            entryId: 'overview-journal-1',
            date: '2026-06-30',
            title: 'Write project summary',
            summary: 'Turn recent captures into a worklog entry',
            minutes: 35,
            workspaceRootPath: 'Project',
          },
        ],
        'suggestions:workspace:Project': [
          {
            suggestionId: 'overview-suggestion-1',
            date: '2026-06-30',
            title: 'Project work on 2026-06-30',
            minutes: 50,
            workspaceRootPath: 'Project',
          },
        ],
      });
    });

    await page.locator('[data-overview-action="refresh"]').click();

    const overview = page.locator('[data-overview-root]');
    await expect(overview.locator('[data-overview-summary="notes"]')).toContainText('1');
    await expect(overview.locator('[data-overview-summary="files"]')).toContainText('1');
    await expect(overview.locator('[data-overview-summary="captures"]')).toContainText('2');
    await expect(overview.locator('[data-overview-summary="journal"]')).toContainText('1');
    await expect(overview.locator('[data-overview-summary="attention"]')).toContainText('3');

    const resume = overview.locator('[data-overview-section="continue"]');
    await expect(resume).toContainText('Opened file "draft.md"');
    await expect(resume).not.toContainText('Research Report');
    await resume.locator('[data-overview-action="continue-primary"]').click();

    await expect(page.getByRole('tab', { name: 'Files' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.files-root')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Overview' }).click();
    const recent = overview.locator('[data-overview-section="recent"]');
    await expect(recent).toContainText('Edited note "Overview"');
    await expect(recent).toContainText('Changed file "draft.md"');
    await expect(recent).toContainText('Captured page "Research Report"');
    await expect(recent).toContainText('Added journal entry "Write project summary"');
    await expect(recent).not.toContainText('Selected file');
    await expect(recent).not.toContainText('Workspace selected');
    await expect(recent).not.toContainText('file.opened');

    await overview.locator('[data-overview-filter="notes"]').click();
    await expect(recent).toContainText('Edited note "Overview"');
    await expect(recent).not.toContainText('Changed file "draft.md"');
    await expect(recent).not.toContainText('Research Report');

    await overview.locator('[data-overview-filter="captures"]').click();
    await expect(recent).toContainText('Captured page "Research Report"');
    await expect(recent).toContainText('Captured selection "Quote to process"');
    await expect(recent).not.toContainText('Edited note "Overview"');

    await overview.locator('[data-overview-filter="journal"]').click();
    await expect(recent).toContainText('Added journal entry "Write project summary"');
    await expect(recent).not.toContainText('Changed file "draft.md"');
  });
});
