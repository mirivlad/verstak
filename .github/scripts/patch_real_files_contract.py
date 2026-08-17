from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


# E2E must use the real official Files manifest together with the real frontend
# source; otherwise permissions/contributions drift independently.
path = Path("frontend/src/lib/test/wails-mock.js")
text = path.read_text()
old = "import filesSource from '../../../../../verstak-official-plugins/plugins/files/frontend/src/index.js?raw';\n"
new = old + "import filesManifest from '../../../../../verstak-official-plugins/plugins/files/plugin.json';\n"
text = replace_once(text, old, new, "Files manifest import")

start = """    'verstak.files': {\n      status: 'loaded',\n      enabled: true,\n      manifest: {\n"""
end = """      rootPath: '/tmp/verstak-test/plugins/files',\n      error: ''\n    },\n"""
if text.count(start) != 1:
    raise SystemExit(f"Files plugin state start count: {text.count(start)}")
start_idx = text.index(start)
end_idx = text.index(end, start_idx) + len(end)
replacement = """    'verstak.files': {\n      status: 'loaded',\n      enabled: true,\n      manifest: filesManifest,\n      rootPath: '/tmp/verstak-test/plugins/files',\n      error: ''\n    },\n"""
text = text[:start_idx] + replacement + text[end_idx:]
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

# Guard the important part: real Files source and manifest must move together.
path = Path("frontend/tests/shell-source-contract-test.mjs")
text = path.read_text()
old = """assertIncludes(\n  wailsMock,\n  \"plugins/files/frontend/src/index.js?raw\",\n  'E2E should exercise the real official Files frontend bundle',\n);\n"""
new = old + """assertIncludes(\n  wailsMock,\n  \"plugins/files/plugin.json\",\n  'E2E should exercise the real official Files manifest and permissions',\n);\nassertIncludes(\n  wailsMock,\n  'manifest: filesManifest',\n  'E2E Files plugin state should use the real official manifest',\n);\n"""
text = replace_once(text, old, new, "Files manifest source guard")
path.write_text(text)
