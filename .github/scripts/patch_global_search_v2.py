from pathlib import Path


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label} anchor count: {count}")
    return text.replace(old, new, 1)

path = Path('frontend/src/lib/shell/GlobalSearch.svelte')
text = path.read_text()
text = replace_once(
    text,
    """  function workspaceTitle(node) {\n    return node?.title || node?.name || node?.id || node?.rootPath || '';\n  }\n\n  function workspaceName(node) {\n    return node?.rootPath || node?.name || node?.id || '';\n  }\n""",
    """  function workspaceTitle(node) {\n    return node?.title || node?.name || node?.id || node?.path || node?.rootPath || '';\n  }\n\n  function workspaceName(node) {\n    return node?.path || node?.rootPath || node?.name || node?.id || '';\n  }\n\n  function collectWorkspaceNodes(nodes, output = []) {\n    (nodes || []).forEach(node => {\n      if (node?.kind === 'workspace') output.push(node);\n      if (Array.isArray(node?.children) && node.children.length) collectWorkspaceNodes(node.children, output);\n    });\n    return output;\n  }\n""",
    'workspace helpers',
)
text = replace_once(
    text,
    """    const tree = await resultOrEmpty(App.GetWorkspaceTree(), { nodes: [] });\n    const nodes = Array.isArray(tree.nodes) ? tree.nodes : [];\n""",
    """    const tree = await resultOrEmpty(App.GetWorkspaceTreeV2(), { roots: [] });\n    const nodes = collectWorkspaceNodes(Array.isArray(tree.roots) ? tree.roots : []);\n""",
    'workspace tree source',
)
text = replace_once(
    text,
    "keywords: `${node.id || ''} ${node.rootPath || ''}`,",
    "keywords: `${node.id || ''} ${node.path || node.rootPath || ''}`,",
    'workspace keywords',
)
path.write_text(text)

path = Path('frontend/tests/shell-source-contract-test.mjs')
text = path.read_text()
old = """  '__filesHistoryByWorkspace',\n]) {\n"""
new = """  '__filesHistoryByWorkspace',\n  'App.GetWorkspaceTree()',\n]) {\n"""
idx = text.rfind(old)
if idx < 0:
    raise SystemExit('GlobalSearch forbidden list tail not found')
text = text[:idx] + text[idx:].replace(old, new, 1)
marker = """assertIncludes(\n  globalSearch,\n  'item?.categoryLabel || provider?.label',\n"""
addition = """assertIncludes(\n  globalSearch,\n  'App.GetWorkspaceTreeV2()',\n  'GlobalSearch should index Deals from the semantic workspace tree',\n);\nassertIncludes(\n  globalSearch,\n  'collectWorkspaceNodes',\n  'GlobalSearch should recursively index Deals nested under semantic folders',\n);\n\n"""
text = replace_once(text, marker, addition + marker, 'GlobalSearch V2 assertions')
path.write_text(text)

path = Path('frontend/e2e/global-search-results.spec.js')
text = path.read_text()
marker = """  test('publishes a nested filename before delayed content reads finish', async ({ page }) => {\n"""
test = """  test('indexes a Deal nested under semantic folders', async ({ page }) => {\n    await page.evaluate(() => {\n      window.go.api.App.GetWorkspaceTreeV2 = async () => ({\n        roots: [{\n          kind: 'folder',\n          id: 'clients',\n          key: 'folder:clients',\n          name: 'Clients',\n          path: 'Clients',\n          children: [{\n            kind: 'folder',\n            id: 'active',\n            key: 'folder:active',\n            name: 'Active',\n            path: 'Clients/Active',\n            children: [{\n              kind: 'workspace',\n              id: 'nested-deal',\n              key: 'workspace:nested-deal',\n              name: 'Acme Nested Deal',\n              path: 'Clients/Active/Acme',\n              children: [],\n            }],\n          }],\n        }],\n        currentWorkspaceId: '',\n        revision: 2,\n        warnings: [],\n      });\n    });\n\n    const input = page.locator('[data-global-search-input]');\n    await input.focus();\n    await input.fill('Acme Nested Deal');\n    const result = page.locator('[data-global-search-result-type=\\\"Workspace\\\"]', { hasText: 'Acme Nested Deal' });\n    await expect(result).toBeVisible({ timeout: 3000 });\n  });\n\n"""
text = replace_once(text, marker, test + marker, 'nested Deal E2E insertion')
path.write_text(text)
