import fs from 'node:fs';
import path from 'node:path';

const bindingsPath = path.resolve('frontend/wailsjs/go/api/App.js');
const bindings = fs.readFileSync(bindingsPath, 'utf8');

for (const method of [
  'ReplacePluginNotifications',
  'ClearPluginNotifications',
  'PluginSelectImportDirectory',
  'PluginSelectImportArchive',
  'PluginListImportEntries',
  'PluginReadImportText',
  'PluginApplyImportPlan',
  'PluginCancelImport',
  'PluginCloseImportSource',
  'PlaceWorkspaceTreeNodeV2',
]) {
  if (!bindings.includes(`export function ${method}(`)) {
    throw new Error(`Wails binding does not export ${method}`);
  }
}

console.log('Wails notification and import bindings are present');
