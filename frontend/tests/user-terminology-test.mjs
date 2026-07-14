import en from '../src/lib/i18n/catalogs/en.js';
import ru from '../src/lib/i18n/catalogs/ru.js';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const forbidden = [
  ['English catalog', /\bworkspace(?:s)?\b/i],
  ['Russian catalog', /рабоч(?:ее|ие|ем|его|их)\s+пространств(?:о|а|е)?/i],
];

for (const [name, pattern] of forbidden) {
  const violations = Object.entries(name === 'English catalog' ? en : ru)
    .filter(([, value]) => pattern.test(value));
  if (violations.length) {
    throw new Error(`${name} still exposes workspace terminology: ${violations.map(([key]) => key).join(', ')}`);
  }
}

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const overview = fs.readFileSync(path.join(root, 'src/lib/shell/TodaySurface.svelte'), 'utf8');
for (const phrase of ['Workspace opened', 'Workspace activity', 'Workspace overview note']) {
  if (overview.includes(phrase)) {
    throw new Error(`Overview exposes workspace terminology: ${phrase}`);
  }
}

console.log('desktop user terminology uses Deal/Дело');
