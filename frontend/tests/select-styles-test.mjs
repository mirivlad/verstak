import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const source = await readFile(new URL('../src/lib/ui/Select.svelte', import.meta.url), 'utf8');

assert.match(source, /\.vt-select\s*\{[^}]*appearance:\s*none/, 'shared select must hide the native arrow');
assert.match(source, /\.vt-select option\s*\{[^}]*background/, 'shared select options must use the application surface');

console.log('workspace select style contract passed');
