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
    await expect(tabs.nth(1)).toHaveText('Notes');
    await expect(page.getByRole('tab', { name: 'Overview' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.getByRole('tab', { name: 'Today' })).toHaveCount(0);

    const overview = page.locator('[data-overview-root]');
    await expect(overview).toBeVisible();
    await expect(overview.locator('[data-overview-section="continue"]')).toContainText('Continue working');
    await expect(overview.locator('[data-overview-section="continue"]')).toContainText('Pick up where you left off.');
    await expect(overview.locator('[data-overview-section="summary"]')).toBeVisible();
    await expect(overview.locator('[data-overview-section="recent"]')).toContainText('Recent changes');
    await expect(overview.locator('[data-overview-section="attention"]')).toHaveCount(0);
    await expect(overview.locator('[data-overview-section="key-resources"]')).toBeVisible();
    await expect(overview.locator('[data-overview-section="quick-actions"]')).toHaveCount(0);

    const summaryCards = overview.locator('button[data-overview-summary]');
    await expect(summaryCards).toHaveCount(5);
    await expect(overview.locator('[data-overview-summary="attention"]')).toHaveCount(0);
    await expect(overview.locator('[data-overview-summary="notes"]')).toContainText('1 note');
    await expect(overview.locator('[data-overview-summary="captures"]')).toContainText('0 captures to review');
    await expect(overview.locator('[data-overview-summary="activity"]')).toContainText('0 recorded events');
    await expect(overview.locator('[data-overview-summary="journal"]')).toContainText('0 journal entries');
    await expect(overview).toContainText('No clear resume point yet');
    await expect(overview).toContainText('No meaningful changes for this filter yet');
  });

  test('Overview summary cards navigate to their corresponding workspace tools', async ({ page }) => {
    const overview = page.locator('[data-overview-root]');
    await overview.locator('[data-overview-summary="notes"]').click();
    await expect(page.getByRole('tab', { name: 'Notes' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.notes-root')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Overview' }).click();
    await overview.locator('[data-overview-summary="files"]').click();
    await expect(page.getByRole('tab', { name: 'Files' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.files-root')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Overview' }).click();
    await overview.locator('[data-overview-summary="captures"]').click();

    await expect(page.getByRole('tab', { name: 'Browser' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.browser-inbox-root')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Overview' }).click();
    await overview.locator('[data-overview-summary="activity"]').click();
    await expect(page.getByRole('tab', { name: 'Activity' })).toHaveAttribute('aria-selected', 'true');

    await page.getByRole('tab', { name: 'Overview' }).click();
    await overview.locator('[data-overview-summary="journal"]').click();
    await expect(page.getByRole('tab', { name: 'Journal' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.journal-root')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Overview' }).click();
    await expect(overview.locator('[data-overview-summary="attention"]')).toHaveCount(0);
  });

  test('Overview keeps the Notes total factual when the Files plugin is unavailable', async ({ page }) => {
    await page.evaluate(() => window.__wailsMock.setPluginStatus('verstak.files', 'disabled', false));
    await page.locator('[data-overview-action="refresh"]').click();

    await expect(page.locator('[data-overview-summary="notes"]')).toContainText('1 note');
  });

  test('Overview hides Browser cards and actions when the current Deal does not include it', async ({ page }) => {
    await page.locator('button[title="New Deal"]').click();
    const modal = page.locator('[data-workspace-create-modal]');
    await modal.locator('[data-workspace-name]').fill('MinimalOverview');
    await modal.locator('[data-workspace-template]').selectOption('minimal');
    await modal.getByRole('button', { name: 'Create Deal' }).click();
    await expect(page.getByRole('tab', { name: 'Browser' })).toHaveCount(0);

    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:global': [{
          captureId: 'hidden-browser-capture',
          capturedAt: '2026-07-14T08:00:00.000Z',
          kind: 'page',
          title: 'Inbox material must stay hidden',
          workspaceRootPath: 'MinimalOverview',
        }],
      });
    });
    const overview = page.locator('[data-overview-root]');
    await overview.locator('[data-overview-action="refresh"]').click();

    await expect(overview.locator('[data-overview-summary="captures"]')).toHaveCount(0);
    await expect(overview.locator('[data-overview-action="browser-inbox"]')).toHaveCount(0);
    await expect(overview).not.toContainText('Inbox material must stay hidden');
    await expect(overview.locator('.overview-side')).toHaveCount(0);
    const layoutBox = await overview.locator('.overview-layout').boundingBox();
    const mainBox = await overview.locator('.overview-main').boundingBox();
    expect(mainBox.width / layoutBox.width).toBeGreaterThan(0.95);
  });

  test('Overview refreshes when Browser is disabled through plugin state changes', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:global': [{
          captureId: 'disabled-browser-capture',
          capturedAt: '2026-07-14T08:00:00.000Z',
          kind: 'page',
          title: 'Disabled inbox material',
          workspaceRootPath: 'Project',
        }],
      });
      window.__wailsMock.setPluginStatus('verstak.browser-inbox', 'disabled', false);
      window.dispatchEvent(new CustomEvent('verstak:plugins-changed'));
    });

    const overview = page.locator('[data-overview-root]');
    await expect(page.getByRole('tab', { name: 'Browser' })).toHaveCount(0);
    await expect(overview.locator('[data-overview-summary="captures"]')).toHaveCount(0);
    await expect(overview.locator('[data-overview-action="browser-inbox"]')).toHaveCount(0);
  });

  test('Overview prioritizes resume work and filters meaningful recent changes', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:global': [
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
          {
            captureId: 'overview-client-capture',
            capturedAt: '2026-06-30T08:30:00.000Z',
            kind: 'page',
            title: 'Client capture must stay global',
            workspaceRootPath: 'ClientA',
          },
          {
            captureId: 'overview-unassigned-capture',
            capturedAt: '2026-06-30T08:35:00.000Z',
            kind: 'page',
            title: 'Unassigned capture must stay global',
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
        'work-session-candidates:workspace:Project': [
          {
            candidateId: 'work-session:Project:overview-note:overview-file',
            workspaceRootPath: 'Project',
            startedAt: '2026-06-30T08:15:00.000Z',
            endedAt: '2026-06-30T08:25:00.000Z',
            estimatedMinutes: 10,
            activityCount: 2,
            activityIds: ['overview-file', 'overview-note'],
            activities: [
              { activityId: 'overview-file', type: 'file.changed', occurredAt: '2026-06-30T08:20:00.000Z' },
              { activityId: 'overview-note', type: 'note.saved', occurredAt: '2026-06-30T08:25:00.000Z' },
            ],
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
      });
    });

    await page.locator('[data-overview-action="refresh"]').click();

    const overview = page.locator('[data-overview-root]');
    await expect(overview.locator('[data-overview-summary="notes"]')).toContainText('1 note');
    await expect(overview.locator('[data-overview-summary="files"]')).toContainText('1 recent change');
    await expect(overview.locator('[data-overview-summary="captures"]')).toContainText('2');
    await expect(overview.locator('[data-overview-summary="activity"]')).toContainText('5 recorded events');
    await expect(overview.locator('[data-overview-summary="journal"]')).toContainText('1');
    await expect(overview.locator('[data-overview-summary="attention"]')).toHaveCount(0);
    const attention = overview.locator('[data-overview-section="attention"]');
    await expect(attention).toContainText('Possible journal entry');
    await expect(attention).toContainText('10 min');
    await expect(attention).toContainText('2 activities');
    await attention.locator('.overview-attention-row', { hasText: 'Possible journal entry' }).getByRole('button').click();
    await expect(page.getByRole('tab', { name: 'Journal' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.journal-root [data-journal-candidate]')).toContainText('Deal: Project');
    await page.locator('.journal-modal-actions').getByRole('button', { name: 'Cancel' }).click();
    await page.getByRole('tab', { name: 'Overview' }).click();

    const resume = overview.locator('[data-overview-section="continue"]');
    const candidates = resume.locator('[data-overview-continue-item]');
    await expect(candidates).toHaveCount(3);
    await expect(candidates.nth(0)).toContainText('Overview');
    await expect(candidates.nth(1)).toContainText('draft.md');
    await expect(candidates.nth(2)).toContainText('Write project summary');
    await expect(resume).not.toContainText('Quote to process');
    await expect(resume).not.toContainText('Research Report');
    await candidates.nth(0).click();

    await expect(page.getByRole('tab', { name: 'Notes' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.notes-root')).toBeVisible({ timeout: 10000 });

    await page.getByRole('tab', { name: 'Overview' }).click();
    const recent = overview.locator('[data-overview-section="recent"]');
    await expect(recent).toContainText('Overview');
    await expect(recent).toContainText('draft.md');
    await expect(recent).toContainText('Research Report');
    await expect(recent).toContainText('Write project summary');
    await expect(recent).not.toContainText('Selected file');
    await expect(recent).not.toContainText('Workspace selected');
    await expect(recent).not.toContainText('file.opened');
    await expect(recent).not.toContainText('Client capture must stay global');
    await expect(recent).not.toContainText('Unassigned capture must stay global');
    await expect(recent.locator('[data-overview-recent-item]')).toHaveCount(5);
    await expect(recent.locator('[data-overview-recent-item] button')).toHaveCount(0);

    await overview.locator('[data-overview-filter="notes"]').click();
    await expect(recent).toContainText('Overview');
    await expect(recent).not.toContainText('draft.md');
    await expect(recent).not.toContainText('Research Report');

    await overview.locator('[data-overview-filter="captures"]').click();
    await expect(recent).toContainText('Research Report');
    await expect(recent).toContainText('Quote to process');
    await expect(recent).not.toContainText('Overview');

    await overview.locator('[data-overview-filter="journal"]').click();
    await expect(recent).toContainText('Write project summary');
    await expect(recent).not.toContainText('draft.md');
  });

  // The Activity card claimed a number the Activity tool would never show,
  // because it counted globally scoped events -- browser domain activity, of
  // which a real vault had 180 -- into every Deal. 184 on the card, 4 in the
  // list.
  test('the Activity card counts this Deal, not global browser activity', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.activity', {
        'events:workspace:Project': [
          {
            activityId: 'in-this-deal-1',
            occurredAt: '2026-06-30T08:10:00.000Z',
            type: 'note.saved',
            title: 'One',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'in-this-deal-2',
            occurredAt: '2026-06-30T08:11:00.000Z',
            type: 'note.saved',
            title: 'Two',
            workspaceRootPath: 'Project',
          },
          // Same event twice: the Activity tool keys its list on the id.
          {
            activityId: 'in-this-deal-2',
            occurredAt: '2026-06-30T08:11:00.000Z',
            type: 'note.saved',
            title: 'Two',
            workspaceRootPath: 'Project',
          },
          // Belongs to another Deal.
          {
            activityId: 'other-deal',
            occurredAt: '2026-06-30T08:12:00.000Z',
            type: 'note.saved',
            title: 'Elsewhere',
            workspaceRootPath: 'SomewhereElse',
          },
          // Global: a visited domain belongs to no Deal.
          {
            activityId: 'global-domain-1',
            occurredAt: '2026-06-30T08:13:00.000Z',
            type: 'browser.activity.domain',
            title: 'github.com',
          },
          {
            activityId: 'global-domain-2',
            occurredAt: '2026-06-30T08:14:00.000Z',
            type: 'browser.activity.domain',
            title: 'example.com',
          },
        ],
      });
    });

    const overview = page.locator('[data-overview-root]');
    await overview.locator('[data-overview-action="refresh"]').click();
    await expect(overview.locator('[data-overview-summary="activity"]')).toContainText('2 recorded events');

    // And the card agrees with what clicking it shows.
    await overview.locator('[data-overview-summary="activity"]').click();
    await expect(page.getByRole('tab', { name: 'Activity' })).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('.activity-root .activity-row')).toHaveCount(2);
  });

  test('Overview localizes activity labels without exposing internal event names', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.activity', {
        'events:workspace:Project': [{
          activityId: 'overview-russian-note',
          occurredAt: '2026-06-30T08:25:00.000Z',
          type: 'note.saved',
          title: 'Локализация',
          summary: 'Project/Notes/Localization.md',
          workspaceRootPath: 'Project',
        }],
      });
    });
    await page.locator('[data-settings-menu-button]').click();
    await page.locator('[data-settings-language="ru"]').click();
    // Settings are a window now, not a dropdown over the current view, so
    // getting back to the Overview is a step.
    await page.keyboard.press('Escape');

    const overview = page.locator('[data-overview-root]');
    await overview.locator('[data-overview-action="refresh"]').click();
    const recent = overview.locator('[data-overview-section="recent"]');
    await expect(recent).toContainText('Изменена заметка — Локализация');
    await expect(recent).not.toContainText('note.saved');
    await expect(recent).not.toContainText('Edited note');
  });
});
