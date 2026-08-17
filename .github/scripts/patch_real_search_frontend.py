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
    "import trashSource from '../../../../../verstak-official-plugins/plugins/trash/frontend/src/index.js?raw';\n",
    "import trashSource from '../../../../../verstak-official-plugins/plugins/trash/frontend/src/index.js?raw';\n"
    "import searchSource from '../../../../../verstak-official-plugins/plugins/search/frontend/src/index.js?raw';\n",
    "Search source import",
)

start = text.index("  function searchPluginBundle() {\n")
end = text.index("\n  function platformTestBundle() {", start)
text = text[:start] + text[end:]

text = replace_once(
    text,
    "      if (pluginId === searchManifest.id && assetPath === searchManifest.frontend.entry) {\n"
    "        return Promise.resolve(searchPluginBundle());\n"
    "      }\n",
    "      if (pluginId === searchManifest.id && assetPath === searchManifest.frontend.entry) {\n"
    "        return Promise.resolve(searchSource);\n"
    "      }\n",
    "Search asset loader",
)

for forbidden in ("function searchPluginBundle()", "Promise.resolve(searchPluginBundle())"):
    if forbidden in text:
        raise SystemExit(f"stale synthetic Search frontend remains: {forbidden}")
mock_path.write_text(text)

contract_path = Path("frontend/tests/shell-source-contract-test.mjs")
contract = contract_path.read_text()
anchor = """assertExcludes(
  wailsMock,
  'Promise.resolve(trashPluginBundle())',
  'E2E Trash asset loader should never return a synthetic Trash bundle',
);

"""
addition = anchor + """assertIncludes(
  wailsMock,
  "plugins/search/frontend/src/index.js?raw",
  'E2E should exercise the real official Search frontend bundle',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(searchSource)',
  'E2E Search asset loading should return the real official source',
);
assertExcludes(
  wailsMock,
  'function searchPluginBundle()',
  'E2E should not maintain a divergent synthetic Search implementation',
);
assertExcludes(
  wailsMock,
  'Promise.resolve(searchPluginBundle())',
  'E2E Search asset loader should never return a synthetic Search bundle',
);

"""
contract = replace_once(contract, anchor, addition, "Search source contract")
contract_path.write_text(contract)
