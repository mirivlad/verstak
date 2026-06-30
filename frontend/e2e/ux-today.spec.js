import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('UX Today workspace flow', () => {
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

  test('workspace opens with Today before plugin tools', async ({ page }) => {
    await expect(page.locator('.workspace-host')).toBeVisible({ timeout: 10000 });

    const tabs = page.getByRole('tab');
    await expect(tabs.nth(0)).toHaveText('Today');
    await expect(tabs.nth(1)).toHaveText('Files');
    await expect(page.getByRole('tab', { name: 'Today' })).toHaveAttribute('aria-selected', 'true');

    const today = page.locator('.today-root');
    await expect(today).toBeVisible();
    await expect(today.locator('[data-today-section="captured"]')).toContainText('Captured');
    await expect(today.locator('[data-today-section="activity"]')).toContainText('Recent Activity');
    await expect(today.locator('[data-today-section="worklog"]')).toContainText('Worklog Suggestions');
    await expect(today.locator('[data-today-section="quick-actions"]')).toContainText('Quick Actions');
    await expect(today).toContainText('No browser captures yet');
    await expect(today).toContainText('No activity events yet');
  });

  test('Today quick action opens Browser Inbox workspace tool', async ({ page }) => {
    await page.locator('[data-today-action="browser-inbox"]').click();

    await expect(page.getByRole('tab', { name: 'Browser Inbox' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.browser-inbox-root')).toBeVisible({ timeout: 10000 });
  });

  test('Today summarizes available work and highlights the next resume item', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:workspace:Project': [
          {
            captureId: 'today-capture-1',
            capturedAt: '2026-06-30T08:00:00.000Z',
            kind: 'page',
            url: 'https://example.com/research',
            title: 'Research Report',
            domain: 'example.com',
            workspaceRootPath: 'Project',
          },
          {
            captureId: 'today-capture-2',
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
            activityId: 'today-activity-1',
            occurredAt: '2026-06-30T08:25:00.000Z',
            type: 'note.saved',
            title: 'Saved research note',
            summary: 'Project/Notes/Research.md',
            workspaceRootPath: 'Project',
          },
        ],
      });
      await window.go.api.App.WritePluginSettings('verstak.journal', {
        'suggestions:workspace:Project': [
          {
            entryId: 'today-worklog-1',
            title: 'Write project summary',
            summary: 'Turn recent captures into a worklog entry',
            minutes: 35,
            workspaceRootPath: 'Project',
          },
        ],
      });
    });

    await page.locator('.today-header').getByRole('button', { name: 'Refresh' }).click();

    const today = page.locator('.today-root');
    await expect(today.locator('[data-today-summary="captures"]')).toContainText('2');
    await expect(today.locator('[data-today-summary="activity"]')).toContainText('1');
    await expect(today.locator('[data-today-summary="worklog"]')).toContainText('1');

    const resume = today.locator('[data-today-section="resume"]');
    await expect(resume).toContainText('Resume next');
    await expect(resume).toContainText('Research Report');
    await resume.locator('[data-today-action="resume-primary"]').click();

    await expect(page.getByRole('tab', { name: 'Browser Inbox' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.browser-inbox-root')).toBeVisible({ timeout: 10000 });
  });
});
