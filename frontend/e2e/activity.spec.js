import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Activity workflow', () => {
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

  test('workspace activity explains empty event flow', async ({ page }) => {
    await page.getByRole('tab', { name: 'Activity' }).click();

    const activity = page.locator('.activity-root');
    await expect(activity).toBeVisible({ timeout: 10000 });
    await expect(activity.locator('.activity-title')).toContainText('Activity');
    await expect(activity.locator('.activity-count')).toHaveText('0 events');
    await expect(activity.locator('.activity-empty')).toContainText('No activity events yet');
    await expect(activity.locator('.activity-empty')).toContainText('File changes, browser captures, and conversions will appear here');
    await expect(activity.locator('[data-activity-action="clear"]')).toBeDisabled();
  });

  test('workspace activity renders stored events and worklog suggestions', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.activity', {
        'events:workspace:Project': [
          {
            activityId: 'activity-e2e-capture',
            occurredAt: '2026-06-30T08:00:00.000Z',
            type: 'browser.capture.selection',
            title: 'Research Capture',
            summary: 'Selected text from the article',
            sourcePluginId: 'verstak.browser-inbox',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'activity-e2e-note',
            occurredAt: '2026-06-30T08:25:00.000Z',
            type: 'note.saved',
            title: 'Saved note',
            summary: 'Project/Notes/Research Capture.md',
            sourcePluginId: 'verstak.files',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'activity-e2e-open',
            occurredAt: '2026-06-30T08:30:00.000Z',
            type: 'file.opened',
            title: 'Selected file',
            summary: 'Project/Notes/Research Capture.md',
            sourcePluginId: 'verstak.files',
            workspaceRootPath: 'Project',
          },
        ],
      });
    });

    await page.getByRole('tab', { name: 'Activity' }).click();

    const activity = page.locator('.activity-root');
    await expect(activity.locator('.activity-count')).toHaveText('2 events');
    await expect(activity.locator('[data-activity-id="activity-e2e-capture"]')).toContainText('Research Capture');
    await expect(activity.locator('[data-activity-id="activity-e2e-note"]')).toContainText('Saved note');
    await expect(activity.locator('[data-activity-id="activity-e2e-open"]')).toHaveCount(0);
    await expect(activity.locator('[data-activity-section="worklog-suggestions"]')).toContainText('Worklog suggestions');
    await expect(activity.locator('[data-worklog-suggestion="worklog:Project:2026-06-30"]')).toContainText('Project work on 2026-06-30');
    await expect(activity.locator('[data-worklog-suggestion="worklog:Project:2026-06-30"]')).toContainText('50 min');
  });
});
