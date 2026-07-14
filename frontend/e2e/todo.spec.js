import { test, expect } from '@playwright/test';
import { waitForAppReady, setupConsoleCollector, resetMockState } from './helpers.js';

test.describe('Todo plugin workflow', () => {
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

  test('workspace Todo supports CRUD, reminders, and manual Journal conversion', async ({ page }) => {
    await page.getByRole('tab', { name: 'Todos' }).click();

    const todos = page.locator('.todo-root');
    await expect(todos).toBeVisible({ timeout: 10000 });
    await expect(todos).toContainText('Todos · Project');

    await todos.locator('[data-todo-action="add"]').click();
    await todos.locator('[data-todo-input="title"]').fill('Prepare project review');
    await todos.locator('[data-todo-input="description"]').fill('Collect factual review notes.');
    await todos.locator('[data-todo-input="priority"]').selectOption('high');
    await todos.locator('[data-todo-input="dueAt"]').fill('2000-01-01');
    await todos.locator('[data-todo-input="reminderDate"]').fill('2000-01-01');
    const reminderHour = todos.locator('[data-todo-input="reminderHour"]');
    const reminderMinute = todos.locator('[data-todo-input="reminderMinute"]');
    await expect(reminderHour).toHaveCSS('appearance', 'none');
    await expect(reminderMinute).toHaveCSS('appearance', 'none');
    await reminderHour.selectOption('09');
    await reminderMinute.selectOption('30');
    await todos.locator('[data-todo-action="save"]').click();

    await expect(todos).toContainText('Overdue');
    await expect(todos).toContainText('Reminder due');
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginSettings('verstak.todo');
      const settings = Array.isArray(result) ? result[0] : result;
      const todo = settings['todos:global'].find((item) => item.title === 'Prepare project review');
      return todo && [todo.workspaceRootPath, todo.priority, todo.dueAt, todo.reminderAt].join('|');
    })).toBe('Project|high|2000-01-01|2000-01-01T09:30');

    await todos.locator('[data-todo-action="edit"]').click();
    await todos.locator('[data-todo-input="title"]').fill('Prepare project review updated');
    await todos.locator('[data-todo-action="save"]').click();
    await expect(todos).toContainText('Prepare project review updated');

    await todos.locator('[data-todo-action="mark-done"]').click();
    await expect(todos.locator('[data-todo-action="create-journal-entry"]')).toBeVisible();
    await todos.locator('[data-todo-action="create-journal-entry"]').click();

    const journal = page.locator('.journal-root');
    await expect(page.getByRole('tab', { name: 'Journal' })).toHaveAttribute('aria-selected', 'true');
    await expect(journal).toBeVisible({ timeout: 10000 });
    await expect(journal).toContainText('Create journal entry from completed todo');
    await expect(journal.locator('[data-journal-input="title"]')).toHaveValue('Prepare project review updated');
    await expect(journal.locator('[data-journal-input="summary"]')).toHaveValue('Collect factual review notes.');
    await expect(journal.locator('[data-journal-input="minutes"]')).toHaveValue('0');

    await journal.locator('[data-journal-input="title"]').fill('Prepare project review handoff');
    await journal.locator('[data-journal-input="summary"]').fill('Reviewed factual project notes before handoff.');
    await journal.locator('[data-journal-action="save-entry"]').click();
    await expect.poll(async () => page.evaluate(async () => {
      const todoResult = await window.go.api.App.ReadPluginSettings('verstak.todo');
      const todoSettings = Array.isArray(todoResult) ? todoResult[0] : todoResult;
      const todo = todoSettings['todos:global'].find((item) => item.title === 'Prepare project review updated');
      const journalResult = await window.go.api.App.ReadPluginSettings('verstak.journal');
      const journalSettings = Array.isArray(journalResult) ? journalResult[0] : journalResult;
      const entry = journalSettings['worklog:workspace:Project'].find((item) => item.sourceTodoId === todo.id);
      return entry && [entry.title, entry.summary, entry.minutes].join('|');
    })).toBe('Prepare project review handoff|Reviewed factual project notes before handoff.|0');

    await page.getByRole('tab', { name: 'Todos' }).click();
    await page.locator('.todo-root [data-todo-action="delete"]').click();
    await expect.poll(async () => page.evaluate(async () => {
      const result = await window.go.api.App.ReadPluginSettings('verstak.todo');
      const settings = Array.isArray(result) ? result[0] : result;
      return settings['todos:global'].some((item) => item.title === 'Prepare project review updated');
    })).toBe(false);
  });

  test('global Todos filter by workspace and Overview only exposes current workspace attention', async ({ page }) => {
    await page.evaluate(async () => {
      await window.go.api.App.WritePluginSettings('verstak.todo', {
        'todos:global': [
          {
            id: 'todo-project-open',
            title: 'Project deadline',
            workspaceRootPath: 'Project',
            status: 'open',
            priority: 'high',
            dueAt: '2000-01-01',
            reminderAt: '2000-01-01T09:00',
            createdAt: '2026-06-30T08:00:00.000Z',
            updatedAt: '2026-06-30T08:00:00.000Z',
          },
          {
            id: 'todo-project-done',
            title: 'Project completed item',
            workspaceRootPath: 'Project',
            status: 'done',
            priority: 'normal',
            createdAt: '2026-06-30T08:01:00.000Z',
            updatedAt: '2026-06-30T08:01:00.000Z',
            completedAt: '2026-06-30T08:02:00.000Z',
          },
          {
            id: 'todo-test-open',
            title: 'Test workspace item',
            workspaceRootPath: 'Test',
            status: 'open',
            priority: 'normal',
            dueAt: '2000-01-01',
            createdAt: '2026-06-30T08:03:00.000Z',
            updatedAt: '2026-06-30T08:03:00.000Z',
          },
        ],
      });
    });

    const overview = page.locator('[data-overview-root]');
    await overview.locator('[data-overview-action="refresh"]').click();
    const attention = overview.locator('[data-overview-section="attention"]');
    await expect(attention).toContainText('Project deadline');
    await expect(attention).not.toContainText('Test workspace item');
    await attention.locator('.overview-attention-row', { hasText: 'Project deadline' }).getByRole('button', { name: 'Open Todos' }).click();
    await expect(page.getByRole('tab', { name: 'Todos' })).toHaveAttribute('aria-selected', 'true');

    await page.locator('.sidebar .plugin-item').filter({ hasText: 'Todos' }).click();
    const globalTodos = page.locator('.todo-root');
    await expect(globalTodos).toContainText('Project deadline');
    await expect(globalTodos).toContainText('Test workspace item');

    await globalTodos.locator('[data-todo-filter="workspace"]').selectOption('Project');
    await expect(globalTodos).toContainText('Project deadline');
    await expect(globalTodos).toContainText('Project completed item');
    await expect(globalTodos).not.toContainText('Test workspace item');

    await globalTodos.locator('[data-todo-filter="status"]').selectOption('done');
    await expect(globalTodos).toContainText('Project completed item');
    await expect(globalTodos).not.toContainText('Project deadline');

    await page.locator('.wt-label').filter({ hasText: 'Test' }).click();
    await page.getByRole('tab', { name: 'Todos' }).click();
    const workspaceTodos = page.locator('.todo-root');
    await expect(workspaceTodos).toContainText('Test workspace item');
    await expect(workspaceTodos).not.toContainText('Project deadline');
  });
});
