import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const source = await readFile(new URL('../src/lib/shell/WorkspaceTree.svelte', import.meta.url), 'utf8');

assert.match(source, /\.workspace-create-field select\s*\{[^}]*appearance:\s*none/, 'workspace template select must hide the native arrow');
assert.match(source, /\.workspace-create-field select option\s*\{[^}]*background/, 'workspace template options must use the application surface');

console.log('workspace select style contract passed');
