// Contrast is easy to lose one token at a time and hard to notice going. This
// reads the real design-system values and checks the pairs that actually meet
// on screen, so a token nudged for looks fails here rather than in use.
//
// Text is held to 4.5:1. Control boundaries -- the thing that tells you where
// an input ends -- are held to 3:1. Decorative separators are not checked:
// nothing depends on seeing them, and pretending otherwise would force
// heavier lines than the design wants.
import fs from 'node:fs';
import path from 'node:path';

const css = fs.readFileSync(path.resolve('frontend/src/lib/ui/design-system.css'), 'utf8');

function token(name) {
  const match = new RegExp(`--${name}:\\s*(#[0-9a-fA-F]{6})\\s*;`).exec(css);
  if (!match) throw new Error(`design-system.css has no opaque colour token --${name}`);
  return match[1];
}

function relativeLuminance(hex) {
  const channels = [1, 3, 5].map((offset) => parseInt(hex.slice(offset, offset + 2), 16) / 255);
  const [r, g, b] = channels.map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(foreground, background) {
  const a = relativeLuminance(foreground);
  const b = relativeLuminance(background);
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
}

const background = token('vt-color-background');
const surface = token('vt-color-surface');
const surfaceMuted = token('vt-color-surface-muted');
const input = token('vt-color-input');
const borderStrong = token('vt-color-border-strong');

const checks = [
  ['text-primary on background', token('vt-color-text-primary'), background, 4.5],
  ['text-primary on surface', token('vt-color-text-primary'), surface, 4.5],
  ['text-secondary on background', token('vt-color-text-secondary'), background, 4.5],
  ['text-secondary on surface', token('vt-color-text-secondary'), surface, 4.5],
  ['text-muted on background', token('vt-color-text-muted'), background, 4.5],
  ['text-muted on surface', token('vt-color-text-muted'), surface, 4.5],
  ['placeholder on input', token('vt-color-placeholder'), input, 4.5],
  ['accent on background', token('vt-color-accent'), background, 4.5],
  ['danger on background', token('vt-color-danger'), background, 4.5],
  ['warning on background', token('vt-color-warning'), background, 4.5],
  ['control border on background', borderStrong, background, 3],
  ['control border on surface', borderStrong, surface, 3],
  ['control border on muted surface', borderStrong, surfaceMuted, 3],
  ['control border on input', borderStrong, input, 3],
];

const failures = [];
for (const [name, foreground, back, required] of checks) {
  const ratio = contrast(foreground, back);
  if (ratio < required) {
    failures.push(`${name}: ${foreground} on ${back} is ${ratio.toFixed(2)}:1, needs ${required}:1`);
  }
}

if (failures.length > 0) {
  console.error('contrast check failed:');
  failures.forEach((failure) => console.error('  ' + failure));
  process.exit(1);
}

console.log(`contrast: ${checks.length} token pairs meet their threshold`);
