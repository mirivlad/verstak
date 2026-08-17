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
    "import searchSource from '../../../../../verstak-official-plugins/plugins/search/frontend/src/index.js?raw';\n",
    "import searchSource from '../../../../../verstak-official-plugins/plugins/search/frontend/src/index.js?raw';\n"
    "import platformTestSource from '../../../../../verstak-official-plugins/plugins/platform-test/frontend/src/index.js?raw';\n",
    "Platform Test source import",
)

# Platform Test is the final synthetic frontend helper before the mock bridge
# API section, so use that named section boundary rather than guessing which
# helper function should follow it.
start = text.index("  function platformTestBundle() {\n")
end = text.index("\n\n  // ── Mock API", start)
text = text[:start] + text[end:]

text = replace_once(
    text,
    "      if (pluginId === platformTestManifest.id && assetPath === platformTestManifest.frontend.entry) {\n"
    "        return Promise.resolve(platformTestBundle());\n"
    "      }\n",
    "      if (pluginId === platformTestManifest.id && assetPath === platformTestManifest.frontend.entry) {\n"
    "        return Promise.resolve(platformTestSource);\n"
    "      }\n",
    "Platform Test asset loader",
)

for forbidden in ("function platformTestBundle()", "Promise.resolve(platformTestBundle())"):
    if forbidden in text:
        raise SystemExit(f"stale synthetic Platform Test frontend remains: {forbidden}")
mock_path.write_text(text)

contract_path = Path("frontend/tests/shell-source-contract-test.mjs")
contract = contract_path.read_text()
anchor = """assertExcludes(
  wailsMock,
  'Promise.resolve(searchPluginBundle())',
  'E2E Search asset loader should never return a synthetic Search bundle',
);

"""
addition = anchor + """assertIncludes(
  wailsMock,
  "plugins/platform-test/frontend/src/index.js?raw",
  'E2E should exercise the real official Platform Test frontend bundle',
);
assertIncludes(
  wailsMock,
  'Promise.resolve(platformTestSource)',
  'E2E Platform Test asset loading should return the real official source',
);
assertExcludes(
  wailsMock,
  'function platformTestBundle()',
  'E2E should not maintain a divergent synthetic Platform Test implementation',
);
assertExcludes(
  wailsMock,
  'Promise.resolve(platformTestBundle())',
  'E2E Platform Test asset loader should never return a synthetic Platform Test bundle',
);

"""
contract = replace_once(contract, anchor, addition, "Platform Test source contract")
contract_path.write_text(contract)
