from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)


mock_path = Path("frontend/src/lib/test/wails-mock.js")
text = mock_path.read_text()

text = replace_once(
    text,
    "import filePreviewSource from '../../../../../verstak-official-plugins/plugins/file-preview/frontend/src/index.js?raw';\n",
    "import filePreviewSource from '../../../../../verstak-official-plugins/plugins/file-preview/frontend/src/index.js?raw';\n"
    "import trashSource from '../../../../../verstak-official-plugins/plugins/trash/frontend/src/index.js?raw';\n",
    "Trash source import",
)

start = text.index("  function trashPluginBundle() {\n")
end = text.index("\n  function simplePluginBundle(", start)
text = text[:start] + text[end:]

text = replace_once(
    text,
    "      if (pluginId === trashManifest.id && assetPath === trashManifest.frontend.entry) {\n"
    "        return Promise.resolve(trashPluginBundle());\n"
    "      }\n",
    "      if (pluginId === trashManifest.id && assetPath === trashManifest.frontend.entry) {\n"
    "        return Promise.resolve(trashSource);\n"
    "      }\n",
    "Trash asset loader",
)

for forbidden in ("function trashPluginBundle()", "Promise.resolve(trashPluginBundle())"):
    if forbidden in text:
        raise SystemExit(f"stale synthetic Trash frontend remains: {forbidden}")
mock_path.write_text(text)

contract_path = Path("frontend/tests/shell-source-contract-test.mjs")
contract = contract_path.read_text()
anchor = """assertExcludes(
  wailsMock,
  'function filesPluginBundle()',
  'E2E should not maintain a divergent synthetic Files implementation',
);

"""
addition = anchor + """assertIncludes(
  wailsMock,
  "plugins/trash/frontend/src/index.js?raw",
  'E2E should exercise the real official Trash frontend bundle',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(trashSource)',
  'E2E Trash asset loading should return the real official source',
);
assertExcludes(
  wailsMock,
  'function trashPluginBundle()',
  'E2E should not maintain a divergent synthetic Trash implementation',
);
assertExcludes(
  wailsMock,
  'Promise.resolve(trashPluginBundle())',
  'E2E Trash asset loader should never return a synthetic Trash bundle',
);

"""
contract = replace_once(contract, anchor, addition, "Trash source contract")
contract_path.write_text(contract)
