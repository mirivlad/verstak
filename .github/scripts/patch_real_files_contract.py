from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


def replace_files_state(text, indent, label):
    start = (
        f"{indent}'verstak.files': {{\n"
        f"{indent}  status: 'loaded',\n"
        f"{indent}  enabled: true,\n"
        f"{indent}  manifest: {{\n"
    )
    end = (
        f"{indent}  rootPath: '/tmp/verstak-test/plugins/files',\n"
        f"{indent}  error: ''\n"
        f"{indent}}},\n"
    )
    count = text.count(start)
    if count != 1:
        raise SystemExit(f"{label} start count: {count}")
    start_idx = text.index(start)
    end_idx = text.index(end, start_idx) + len(end)
    replacement = (
        f"{indent}'verstak.files': {{\n"
        f"{indent}  status: 'loaded',\n"
        f"{indent}  enabled: true,\n"
        f"{indent}  manifest: filesManifest,\n"
        f"{indent}  rootPath: '/tmp/verstak-test/plugins/files',\n"
        f"{indent}  error: ''\n"
        f"{indent}}},\n"
    )
    return text[:start_idx] + replacement + text[end_idx:]


# E2E must use the real official Files manifest together with the real frontend
# source; otherwise permissions, contributions and frontend entry drift independently.
path = Path("frontend/src/lib/test/wails-mock.js")
text = path.read_text()
old = "import filesSource from '../../../../../verstak-official-plugins/plugins/files/frontend/src/index.js?raw';\n"
new = old + "import filesManifest from '../../../../../verstak-official-plugins/plugins/files/plugin.json';\n"
text = replace_once(text, old, new, "Files manifest import")

# The mock has both its initial state and a reset() fixture. They must point to
# the same real manifest or every test reset silently restores the old contract.
text = replace_files_state(text, "    ", "initial Files plugin state")
text = replace_files_state(text, "        ", "reset Files plugin state")

# The host asks for the entry declared by the manifest. Do not pin the mock to
# the historical dist path now that the official manifest owns this contract.
old = "      if (pluginId === 'verstak.files' && assetPath === 'frontend/dist/index.js') {\n"
new = "      if (pluginId === 'verstak.files' && assetPath === filesManifest.frontend.entry) {\n"
text = replace_once(text, old, new, "Files frontend entry")
path.write_text(text)

# Existing Files E2E was written against the removed synthetic implementation.
# Keep behavior coverage, but target the real UI and host contracts.
path = Path("frontend/e2e/files-plugin.spec.js")
text = path.read_text()
text = text.replace("page.locator('[data-files-create-confirm]')", "page.locator('[data-files-create-modal] .files-modal-btn.confirm')")
text = text.replace("page.locator('[data-files-rename-confirm]')", "page.locator('[data-files-rename-modal] .files-modal-btn.confirm')")

old = """      { overwrite: false },\n    ]]);\n"""
new = """      expect.objectContaining({ overwrite: false, transferId: expect.any(String) }),\n    ]]);\n"""
text = replace_once(text, old, new, "copy transfer options")

# Playwright locator.dragTo can stall on HTML5 drag/drop flows that depend on a
# custom MIME payload. Dispatch the browser's real DragEvents with one shared
# DataTransfer instead: Files still creates the payload in dragstart and the
# real WorkspaceTree/TreeNode consumes it in dragover/drop.
old = """async function openFilesTool(page) {\n  await page.getByRole('tab', { name: 'Files' }).click();\n  await expect(page.locator('.files-root')).toBeVisible({ timeout: 10000 });\n}\n\n"""
new = old + """async function dragFileToWorkspace(page, fileName, workspaceName) {\n  await page.evaluate(({ fileName, workspaceName }) => {\n    const source = document.querySelector(`[data-file-name=\"${CSS.escape(fileName)}\"]`);\n    const targetLabel = Array.from(document.querySelectorAll('.wt-label'))\n      .find((node) => node.textContent.trim() === workspaceName);\n    const target = targetLabel?.closest('.wt-node');\n    if (!source || !target) throw new Error(`missing drag endpoints: ${fileName} -> ${workspaceName}`);\n\n    const dataTransfer = new DataTransfer();\n    source.dispatchEvent(new DragEvent('dragstart', { bubbles: true, cancelable: true, dataTransfer }));\n    target.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true, dataTransfer }));\n    target.dispatchEvent(new DragEvent('drop', { bubbles: true, cancelable: true, dataTransfer }));\n    source.dispatchEvent(new DragEvent('dragend', { bubbles: true, cancelable: true, dataTransfer }));\n  }, { fileName, workspaceName });\n}\n\n"""
text = replace_once(text, old, new, "Files drag helper")

old = """    const source = page.locator('[data-file-name=\"project-only.txt\"]');\n    const target = page.locator('.wt-node').filter({ hasText: 'Test' });\n    await source.dragTo(target);\n"""
if text.count(old) != 2:
    raise SystemExit(f"Files cross-Deal drag blocks: {text.count(old)}")
text = text.replace(old, """    await dragFileToWorkspace(page, 'project-only.txt', 'Test');\n""")

# Creating a real file intentionally opens it in the editor. Assert that
# behavior, then return to Files before continuing the list/rename flow.
old = """    await page.locator('[data-files-create-input]').fill('Log.md');\n    await page.locator('[data-files-create-modal] .files-modal-btn.confirm').click();\n    await expect(page.locator('[data-file-name=\"Log.md\"]')).toBeVisible();\n\n    await page.locator('[data-file-name=\"Log.md\"]').click();\n"""
new = """    await page.locator('[data-files-create-input]').fill('Log.md');\n    await page.locator('[data-files-create-modal] .files-modal-btn.confirm').click();\n    await expect(page.locator('[data-resource-path=\"Project/Daily/Log.md\"]')).toBeVisible({ timeout: 10000 });\n\n    await page.locator('.wt-label').filter({ hasText: 'Project' }).click();\n    await page.getByRole('tab', { name: 'Files' }).click();\n    await expect(page.locator('.files-breadcrumb')).toContainText('Daily', { timeout: 10000 });\n    await expect(page.locator('[data-file-name=\"Log.md\"]')).toBeVisible();\n    await page.locator('[data-file-name=\"Log.md\"]').click();\n"""
text = replace_once(text, old, new, "created Markdown opens editor")

invalid_count = text.count("toContainText('Invalid characters')")
if invalid_count != 2:
    raise SystemExit(f"Files invalid-character expectations: {invalid_count}")
text = text.replace("toContainText('Invalid characters')", "toContainText(/invalid characters/i)")
path.write_text(text)

# Guard the important part: real Files source, manifest and entry must move together.
path = Path("frontend/tests/shell-source-contract-test.mjs")
text = path.read_text()
old = """assertIncludes(\n  wailsMock,\n  \"plugins/files/frontend/src/index.js?raw\",\n  'E2E should exercise the real official Files frontend bundle',\n);\n"""
new = old + """assertIncludes(\n  wailsMock,\n  \"plugins/files/plugin.json\",\n  'E2E should exercise the real official Files manifest and permissions',\n);\nassertIncludes(\n  wailsMock,\n  'manifest: filesManifest',\n  'E2E Files plugin state should use the real official manifest',\n);\nassertIncludes(\n  wailsMock,\n  \"assetPath === filesManifest.frontend.entry\",\n  'E2E Files bundle should be loaded through the entry declared by the real manifest',\n);\n"""
text = replace_once(text, old, new, "Files manifest source guard")
path.write_text(text)
