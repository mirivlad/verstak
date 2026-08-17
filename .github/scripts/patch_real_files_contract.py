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
# Keep the behavior expectations, but target the real modal contract and the
# current transfer contract.
path = Path("frontend/e2e/files-plugin.spec.js")
text = path.read_text()
text = text.replace("page.locator('[data-files-create-confirm]')", "page.locator('[data-files-create-modal] .files-modal-btn.confirm')")
text = text.replace("page.locator('[data-files-rename-confirm]')", "page.locator('[data-files-rename-modal] .files-modal-btn.confirm')")
old = """      { overwrite: false },\n    ]]);\n"""
new = """      expect.objectContaining({ overwrite: false, transferId: expect.any(String) }),\n    ]]);\n"""
text = replace_once(text, old, new, "copy transfer options")
path.write_text(text)

# Guard the important part: real Files source, manifest and entry must move together.
path = Path("frontend/tests/shell-source-contract-test.mjs")
text = path.read_text()
old = """assertIncludes(\n  wailsMock,\n  \"plugins/files/frontend/src/index.js?raw\",\n  'E2E should exercise the real official Files frontend bundle',\n);\n"""
new = old + """assertIncludes(\n  wailsMock,\n  \"plugins/files/plugin.json\",\n  'E2E should exercise the real official Files manifest and permissions',\n);\nassertIncludes(\n  wailsMock,\n  'manifest: filesManifest',\n  'E2E Files plugin state should use the real official manifest',\n);\nassertIncludes(\n  wailsMock,\n  \"assetPath === filesManifest.frontend.entry\",\n  'E2E Files bundle should be loaded through the entry declared by the real manifest',\n);\n"""
text = replace_once(text, old, new, "Files manifest source guard")
path.write_text(text)
