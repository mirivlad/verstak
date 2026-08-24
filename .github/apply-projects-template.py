from pathlib import Path

replacements = {
    "internal/core/workspacetree/lifecycle.go": [
        (
            'WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.todo", "verstak.journal", "verstak.activity", "verstak.browser-inbox"}',
            'WorkspaceTools: []string{"verstak.projects", "verstak.notes", "verstak.files", "verstak.todo", "verstak.journal", "verstak.activity", "verstak.browser-inbox"}',
        ),
    ],
    "internal/core/workspace/manager.go": [
        (
            '\t\tFeatures: map[string]bool{\n\t\t\t"files":         true,\n\t\t\t"notes":         true,\n\t\t\t"todo":          true,',
            '\t\tFeatures: map[string]bool{\n\t\t\t"projects":      true,\n\t\t\t"files":         true,\n\t\t\t"notes":         true,\n\t\t\t"todo":          true,',
        ),
        (
            'WorkspaceTools: []string{"verstak.notes", "verstak.files", "verstak.todo", "verstak.journal", "verstak.activity", "verstak.browser-inbox"}',
            'WorkspaceTools: []string{"verstak.projects", "verstak.notes", "verstak.files", "verstak.todo", "verstak.journal", "verstak.activity", "verstak.browser-inbox"}',
        ),
    ],
    "internal/api/app.go": [
        (
            '\tfor feature, pluginID := range map[string]string{\n\t\t"journal":',
            '\tfor feature, pluginID := range map[string]string{\n\t\t"projects":      "verstak.projects",\n\t\t"journal":',
        ),
    ],
    "frontend/src/lib/test/wails-mock.js": [
        (
            "workspaceTools: ['verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.journal', 'verstak.activity', 'verstak.browser-inbox'],",
            "workspaceTools: ['verstak.projects', 'verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.journal', 'verstak.activity', 'verstak.browser-inbox'],",
        ),
        (
            "features: { files: true, notes: true, todo: true, journal: true, activity: true, 'browser-inbox': true },",
            "features: { projects: true, files: true, notes: true, todo: true, journal: true, activity: true, 'browser-inbox': true },",
        ),
    ],
    "frontend/e2e/workspace-templates.spec.js": [
        (
            "tools: ['verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.journal', 'verstak.activity', 'verstak.browser-inbox'],",
            "tools: ['verstak.projects', 'verstak.notes', 'verstak.files', 'verstak.todo', 'verstak.journal', 'verstak.activity', 'verstak.browser-inbox'],",
        ),
        (
            "    await expect(page.getByRole('tab', { name: 'Todos' })).toBeVisible();",
            "    await expect(page.getByRole('tab', { name: 'Project' })).toBeVisible();\n    await expect(page.getByRole('tab', { name: 'Todos' })).toBeVisible();",
        ),
        (
            "      'verstak.notes',\n      'verstak.todo',",
            "      'verstak.projects',\n      'verstak.notes',\n      'verstak.todo',",
        ),
        (
            "await expect(modal.locator('[data-workspace-tool]')).toHaveCount(8);",
            "await expect(modal.locator('[data-workspace-tool]')).toHaveCount(9);",
        ),
    ],
}

changed = False
for filename, pairs in replacements.items():
    path = Path(filename)
    text = path.read_text()
    for old, new in pairs:
        if new in text:
            continue
        count = text.count(old)
        if count != 1:
            raise SystemExit(f"{filename}: expected one match, got {count}: {old[:80]!r}")
        text = text.replace(old, new)
        changed = True
    path.write_text(text)

print("changed" if changed else "already-applied")
