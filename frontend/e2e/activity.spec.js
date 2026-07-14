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

  test('clearing activity requires destructive confirmation for the current case', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.activity', {
        'events:workspace:Project': [{
          activityId: 'activity-to-clear',
          occurredAt: '2026-06-30T08:00:00.000Z',
          type: 'note.saved',
          title: 'Keep until confirmed',
          workspaceRootPath: 'Project',
        }],
      });
    });

    await page.getByRole('tab', { name: 'Activity' }).click();
    const activity = page.locator('.activity-root');
    await expect(activity.locator('.activity-count')).toHaveText('1 event');

    await activity.locator('[data-activity-action="clear"]').click();
    const confirmation = activity.locator('[data-activity-clear-confirmation]');
    await expect(confirmation).toBeVisible();
    await expect(confirmation).toContainText(/Project|Дело/);
    await expect(activity.locator('.activity-count')).toHaveText('1 event');

    await confirmation.locator('[data-activity-clear-cancel]').click();
    await expect(confirmation).toHaveCount(0);
    await expect(activity.locator('.activity-count')).toHaveText('1 event');

    await activity.locator('[data-activity-action="clear"]').click();
    await activity.locator('[data-activity-clear-confirm]').click();
    await expect(activity.locator('.activity-count')).toHaveText('0 events');
  });

  test('workspace activity keeps raw events and renders factual work session candidates', async ({ page }) => {
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
            occurredAt: '2026-06-30T08:20:00.000Z',
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
    await expect(activity.locator('.activity-count')).toHaveText('3 events');
    await expect(activity.locator('[data-activity-id="activity-e2e-capture"]')).toContainText('Research Capture');
    await expect(activity.locator('[data-activity-id="activity-e2e-note"]')).toContainText('Saved note');
    await expect(activity.locator('[data-activity-id="activity-e2e-open"]')).toContainText('Selected file');

    const candidateSection = activity.locator('[data-activity-section="work-session-candidates"]');
    await expect(candidateSection).toContainText('Possible journal entries');
    const candidate = candidateSection.locator('[data-work-session-candidate]');
    await expect(candidate).toHaveCount(1);
    await expect(candidate).toContainText('Deal: Project');
    await expect(candidate).toContainText('Estimated duration: 10 min');
    await expect(candidate).toContainText('Activities: 2');
    await expect(candidate).not.toContainText('Project work on');
    await expect(candidate.locator('[data-work-session-action="review"]')).toBeVisible();
    await expect(candidate.locator('[data-work-session-action="dismiss"]')).toBeVisible();
  });

  test('Review opens an empty Journal form with selectable candidate activities', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.activity', {
        'events:workspace:Project': [
          {
            activityId: 'review-capture',
            occurredAt: '2026-06-30T08:00:00.000Z',
            type: 'browser.capture.selection',
            title: 'Research Capture',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'review-note',
            occurredAt: '2026-06-30T08:20:00.000Z',
            type: 'note.saved',
            title: 'Saved note',
            workspaceRootPath: 'Project',
          },
        ],
      });
    });

    await page.getByRole('tab', { name: 'Activity' }).click();
    await page.locator('[data-work-session-candidate] [data-work-session-action="review"]').click();

    await expect(page.getByRole('tab', { name: 'Journal' })).toHaveAttribute('aria-selected', 'true');
    const journal = page.locator('.journal-root');
    await expect(journal).toBeVisible({ timeout: 10000 });
    await expect(journal.locator('[data-journal-candidate]')).toContainText('Deal: Project');
    await expect(journal.locator('[data-journal-candidate]')).toContainText('Estimated duration: 10 min');
    await expect(journal.locator('[data-journal-input="title"]')).toHaveValue('');
    await expect(journal.locator('[data-journal-input="summary"]')).toHaveValue('');
    await expect(journal.locator('[data-journal-input="minutes"]')).toHaveValue('10');

    const activityInputs = journal.locator('[data-journal-candidate-activity]');
    await expect(activityInputs).toHaveCount(2);
    await expect(activityInputs.nth(0)).toBeChecked();
    await expect(activityInputs.nth(1)).toBeChecked();
    await journal.locator('[data-journal-input="title"]').fill('Review research capture');
    await journal.locator('[data-journal-input="summary"]').fill('Read the capture and updated the project note.');
    await activityInputs.nth(1).uncheck();
    await journal.locator('[data-journal-action="save-entry"]').click();

    await expect(journal).toContainText('Review research capture');
    await expect(journal).toContainText('10 min');
    const stored = await page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginSettings('verstak.journal');
      return Array.isArray(result) ? result[0]['worklog:workspace:Project'] : result['worklog:workspace:Project'];
    });
    await expect(stored).toEqual([expect.objectContaining({
      title: 'Review research capture',
      summary: 'Read the capture and updated the project note.',
      sourceCandidateId: expect.any(String),
      activityIds: ['review-capture'],
    })]);
  });
});
