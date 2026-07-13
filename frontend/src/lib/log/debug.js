/**
 * Frontend debug logger.
 *
 * Enabled when:
 * 1. URL has ?debug query param, OR
 * 2. localStorage has verstak-debug = "true"
 *
 * Writes to:
 * - console (always, with [debug] prefix)
 * - localStorage buffer (last 1000 entries, key: verstak-debug-log)
 *
 * Usage:
 *   import { debug } from '../log/debug.js';
 *   debug.log('[ComponentName]', 'message', data);
 *   debug.logf('[ComponentName]', 'format %s', arg);
 *
 * To enable: open app with ?debug or run in console:
 *   localStorage.setItem('verstak-debug', 'true')
 *
 * To export log: run in console:
 *   copy(JSON.parse(localStorage.getItem('verstak-debug-log')))
 */

var ENABLED = false;
var BUFFER_KEY = 'verstak-debug-log';
var MAX_ENTRIES = 1000;

// Check enable conditions
function checkEnabled() {
  try {
    if (window.location && window.location.search && window.location.search.indexOf('debug') !== -1) {
      return true;
    }
    if (typeof localStorage !== 'undefined' && localStorage.getItem('verstak-debug') === 'true') {
      return true;
    }
  } catch (e) {
    // localStorage not available
  }
  return false;
}

ENABLED = checkEnabled();

function getTimestamp() {
  return new Date().toISOString();
}

function formatMessage(args) {
  var parts = [];
  for (var i = 0; i < args.length; i++) {
    var a = args[i];
    if (typeof a === 'object') {
      try {
        parts.push(JSON.stringify(a));
      } catch (e) {
        parts.push(String(a));
      }
    } else {
      parts.push(String(a));
    }
  }
  return parts.join(' ');
}

function writeToBuffer(entry) {
  try {
    if (typeof localStorage === 'undefined') return;
    var raw = localStorage.getItem(BUFFER_KEY);
    var log = raw ? JSON.parse(raw) : [];
    log.push(entry);
    if (log.length > MAX_ENTRIES) {
      log = log.slice(log.length - MAX_ENTRIES);
    }
    localStorage.setItem(BUFFER_KEY, JSON.stringify(log));
  } catch (e) {
    // Ignore quota errors
  }
}

function log() {
  if (!ENABLED) return;
  var msg = formatMessage(Array.prototype.slice.call(arguments));
  var entry = { ts: getTimestamp(), msg: msg };
  writeToBuffer(entry);
  console.log('[debug]', msg);
}

function logf() {
  if (!ENABLED) return;
  var args = Array.prototype.slice.call(arguments);
  var format = args.shift();
  var i = 0;
  var msg = format.replace(/%[sdfo]/g, function () {
    return i < args.length ? String(args[i++]) : '';
  });
  var entry = { ts: getTimestamp(), msg: msg };
  writeToBuffer(entry);
  console.log('[debug]', msg);
}

function getLog() {
  try {
    if (typeof localStorage === 'undefined') return [];
    var raw = localStorage.getItem(BUFFER_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch (e) {
    return [];
  }
}

function clearLog() {
  try {
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem(BUFFER_KEY);
    }
  } catch (e) {}
}

function exportLog() {
  var entries = getLog();
  return entries.map(function (e) { return e.ts + ' ' + e.msg; }).join('\n');
}

// Named export for Svelte/Vite
export var debug = {
  log: log,
  logf: logf,
  isEnabled: function () { return ENABLED; },
  enable: function (options) {
    ENABLED = true;
    if (!options || options.persist !== false) {
      try { localStorage.setItem('verstak-debug', 'true'); } catch (e) {}
    }
  },
  disable: function () {
    ENABLED = false;
    try { localStorage.removeItem('verstak-debug'); } catch (e) {}
  },
  getLog: getLog,
  clearLog: clearLog,
  exportLog: exportLog
};

// Also expose globally for console access
if (typeof window !== 'undefined') {
  window.__verstakDebug = debug;
}

if (ENABLED) {
  console.log('[debug] frontend debug logger enabled');
}
