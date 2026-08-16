import { test, expect } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { resolve } from 'node:path';
import { waitForAppReady, resetMockState } from '../helpers.js';

const VISUAL_DIR = resolve(process.cwd(), 'e2e-results/visual');

async function prepare(page) {
  await resetMockState(page);
  await page.goto('/');
  await waitForAppReady(page);
  await page.locator('[data-settings-menu-button]').click();
  await page.locator('[data-settings-language="ru"]').click();
  await page.keyboard.press('Escape');
  await expect(page.locator('[data-overview-root]')).toBeVisible();
  await mkdir(VISUAL_DIR, { recursive: true });
}

async function shot(page, name) {
  await page.screenshot({
    path: resolve(VISUAL_DIR, name),
    fullPage: true,
    animations: 'disabled',
  });
}

test.describe('Visual audit: Overview', () => {
  test('empty Deal in Russian', async ({ page }) => {
    await prepare(page);
    await expect(page.locator('[data-overview-section="continue"]')).toContainText('Пока неясно, с чего продолжить');
    await shot(page, 'overview-empty-ru.png');
  });

  test('populated Deal in Russian', async ({ page }) => {
    await prepare(page);

    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.browser-inbox', {
        'captures:global': [
          {
            captureId: 'visual-capture-page',
            capturedAt: '2026-08-16T09:40:00.000Z',
            kind: 'page',
            url: 'https://docs.example.test/release',
            domain: 'docs.example.test',
            title: 'План выпуска',
            workspaceRootPath: 'Project',
          },
          {
            captureId: 'visual-capture-selection',
            capturedAt: '2026-08-16T09:20:00.000Z',
            kind: 'selection',
            domain: 'example.test',
            title: 'Фрагмент документации',
            workspaceRootPath: 'Project',
          },
        ],
      });

      await window.go.api.App.WritePluginSettings('verstak.activity', {
        'events:workspace:Project': [
          {
            activityId: 'visual-note',
            occurredAt: '2026-08-16T10:10:00.000Z',
            type: 'note.saved',
            title: 'План миграции',
            summary: 'Project/Notes/migration.md',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'visual-file',
            occurredAt: '2026-08-16T10:00:00.000Z',
            type: 'file.changed',
            title: 'server.conf',
            summary: 'Project/Files/server.conf',
            workspaceRootPath: 'Project',
          },
          {
            activityId: 'visual-opened',
            occurredAt: '2026-08-16T09:55:00.000Z',
            type: 'file.opened',
            title: 'server.conf',
            summary: 'Project/Files/server.conf',
            workspaceRootPath: 'Project',
          },
        ],
        'work-session-candidates:workspace:Project': [
          {
            candidateId: 'visual-work-session',
            workspaceRootPath: 'Project',
            startedAt: '2026-08-16T09:50:00.000Z',
            endedAt: '2026-08-16T10:10:00.000Z',
            estimatedMinutes: 20,
            activityCount: 2,
            activityIds: ['visual-file', 'visual-note'],
          },
        ],
      });

      await window.go.api.App.WritePluginSettings('verstak.journal', {
        'worklog:workspace:Project': [
          {
            entryId: 'visual-journal',
            date: '2026-08-16',
            title: 'Подготовка миграции',
            summary: 'Проверил конфигурацию и план переноса.',
            minutes: 45,
            workspaceRootPath: 'Project',
          },
        ],
      });

      await window.go.api.App.WritePluginSettings('verstak.todo', {
        'todos:global': [
          {
            id: 'visual-overdue-todo',
            title: 'Проверить резервную копию',
            workspaceRootPath: 'Project',
            workspaceName: 'Project',
            status: 'open',
            priority: 'high',
            dueAt: '2026-08-15',
            createdAt: '2026-08-14T08:00:00.000Z',
            updatedAt: '2026-08-16T08:00:00.000Z',
          },
        ],
      });
    });

    const overview = page.locator('[data-overview-root]');
    await overview.locator('[data-overview-action="refresh"]').click();
    await expect(overview).toContainText('План миграции');
    await expect(overview).toContainText('Проверить резервную копию');
    await expect(overview).toContainText('План выпуска');
    await shot(page, 'overview-populated-ru.png');
  });
});
