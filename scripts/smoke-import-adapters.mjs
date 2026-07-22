#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { webcrypto } from 'node:crypto';

const [inputPath, outputPath, pluginRoot, requestedFormat] = process.argv.slice(2);
if (!inputPath || !outputPath || !pluginRoot || !requestedFormat) process.exit(2);
if (!globalThis.crypto) globalThis.crypto = webcrypto;

const moduleURL = (relativePath) => pathToFileURL(path.join(pluginRoot, 'plugins', 'import', 'frontend', 'src', relativePath)).href;
const [{ detectCandidates }, { buildDokuWikiGraph }, { buildObsidianGraph }, { proposePlan, serializeApplyPlan, validateEditablePlan }] = await Promise.all([
  import(moduleURL('model/source.js')),
  import(moduleURL('dokuwiki/adapter.js')),
  import(moduleURL('obsidian/adapter.js')),
  import(moduleURL('model/plan.js')),
]);

const input = JSON.parse(fs.readFileSync(inputPath, 'utf8'));
const api = {
  imports: {
    listEntries: async () => ({ entries: input.entries, nextCursor: '', fingerprint: input.session.fingerprint }),
    readText: async (_sourceHandle, entryId) => {
      if (!Object.prototype.hasOwnProperty.call(input.texts, entryId)) throw new Error('text-entry-unavailable');
      return input.texts[entryId];
    },
  },
};
const candidates = detectCandidates(input.entries);
const candidate = candidates.find((item) => item.format === requestedFormat);
if (!candidate) throw new Error('source-format-not-detected');
const graph = requestedFormat === 'dokuwiki'
  ? await buildDokuWikiGraph(api, input.session, candidate)
  : await buildObsidianGraph(api, input.session, candidate);
const plan = proposePlan(graph, new Date('2026-07-23T04:30:00Z'), { locale: 'en' });
const validation = validateEditablePlan(plan);
const aggregate = {
  format: graph.format,
  nodes: graph.nodes.length,
  notes: graph.nodes.filter((node) => node.role === 'note').length,
  assets: graph.nodes.filter((node) => node.role === 'asset').length,
  warnings: graph.warnings.length,
  planNodes: plan.nodes.length,
  validationCount: validation.length,
};
const output = { aggregate, plan: serializeApplyPlan(plan, input.session.fingerprint) };
fs.writeFileSync(outputPath, JSON.stringify(output), { encoding: 'utf8', mode: 0o600 });
process.stdout.write(JSON.stringify(aggregate));
