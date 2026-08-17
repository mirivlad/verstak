from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


path = Path("frontend/src/lib/test/wails-mock.js")
text = path.read_text()

old = "import defaultEditorSource from '../../../../../verstak-official-plugins/plugins/default-editor/frontend/src/index.js?raw';\n"
new = old + "import filesSource from '../../../../../verstak-official-plugins/plugins/files/frontend/src/index.js?raw';\n"
text = replace_once(text, old, new, "official Files raw import")

start_marker = "  function filesPluginBundle() {\n"
end_marker = "  function trashPluginBundle() {\n"
if text.count(start_marker) != 1:
    raise SystemExit(f"synthetic Files bundle start count: {text.count(start_marker)}")
if text.count(end_marker) != 1:
    raise SystemExit(f"Trash bundle marker count: {text.count(end_marker)}")
start = text.index(start_marker)
end = text.index(end_marker, start)
text = text[:start] + text[end:]

old = """      if (pluginId === 'verstak.files' && assetPath === 'frontend/dist/index.js') {\n        return Promise.resolve(filesPluginBundle());\n      }\n"""
new = """      if (pluginId === 'verstak.files' && assetPath === 'frontend/dist/index.js') {\n        return Promise.resolve(filesSource);\n      }\n"""
text = replace_once(text, old, new, "Files asset resolver")
path.write_text(text)

path = Path("frontend/tests/shell-source-contract-test.mjs")
text = path.read_text()
old = "const pluginManager = read('frontend/src/lib/plugin-manager/PluginManager.svelte');\n"
new = old + "const wailsMock = read('frontend/src/lib/test/wails-mock.js');\n"
text = replace_once(text, old, new, "wails mock source binding")

marker = "const syncManifest = JSON.parse(read('../verstak-official-plugins/plugins/sync/plugin.json'));\n\n"
addition = """assertIncludes(\n  wailsMock,\n  \"plugins/files/frontend/src/index.js?raw\",\n  'E2E should exercise the real official Files frontend bundle',\n);\nassertIncludes(\n  wailsMock,\n  'Promise.resolve(filesSource)',\n  'E2E Files asset loading should return the real official Files source',\n);\nassertExcludes(\n  wailsMock,\n  'function filesPluginBundle()',\n  'E2E should not maintain a divergent synthetic Files implementation',\n);\n\n"""
text = replace_once(text, marker, marker + addition, "Files mock contract guard")
path.write_text(text)
