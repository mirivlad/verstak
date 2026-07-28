#!/usr/bin/env bash
# Checks a built release before anybody installs it.
#
# Everything here answers a question that has actually gone wrong: a package
# that was never rebuilt, an installed plugin a version behind its source, a
# binary that reports a version nobody released. What it cannot check --
# whether the thing works on a machine that is not this one -- is in
# docs/RELEASE_CHECKLIST.md, and that part is not optional either.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RELEASE_DIR="${VERSTAK_RELEASE_DIR:-$ROOT/release}"
BINARY="$ROOT/build/bin/verstak-desktop"
PLUGINS_DIR="$ROOT/build/bin/plugins"
FAILED=0

report() {
  if [ "$2" -eq 0 ]; then
    echo "  ✅ $1"
  else
    echo "  ❌ $1"
    FAILED=1
  fi
}

note() { echo "     $1"; }

echo "=== release smoke ==="

# ── The binary ──────────────────────────────────────────────────────────────
echo "[binary]"
if [ -x "$BINARY" ]; then
  report "build/bin/verstak-desktop exists" 0
else
  report "build/bin/verstak-desktop exists — run scripts/build.sh" 1
fi

# ── Packaged plugins ────────────────────────────────────────────────────────
#
# The build copies plugins from build/bin/plugins. Every one of them says what
# left the plugin build; a package that no longer matches is either a stale
# copy or a partial one, and both have shipped before.
echo "[plugins]"
if [ ! -d "$PLUGINS_DIR" ]; then
  report "build/bin/plugins exists" 1
else
  plugin_count=0
  unchecked=0
  for plugin_dir in "$PLUGINS_DIR"/*/; do
    [ -d "$plugin_dir" ] || continue
    name="$(basename "$plugin_dir")"
    plugin_count=$((plugin_count + 1))
    if [ ! -f "$plugin_dir/checksums.txt" ]; then
      report "$name carries checksums.txt" 1
      unchecked=$((unchecked + 1))
      continue
    fi
    if (cd "$plugin_dir" && sha256sum --quiet --check checksums.txt >/dev/null 2>&1); then
      # Files listed match. Files nobody listed are the other half of the
      # question: a leftover from an older package is exactly as wrong.
      extra="$(cd "$plugin_dir" && find . -type f ! -name checksums.txt -printf '%P\n' | LC_ALL=C sort | comm -23 - <(cut -d' ' -f3- checksums.txt | LC_ALL=C sort))"
      if [ -n "$extra" ]; then
        report "$name matches its checksums" 1
        note "not part of the package: $(echo "$extra" | tr '\n' ' ')"
      else
        report "$name matches its checksums" 0
      fi
    else
      report "$name matches its checksums" 1
      note "$(cd "$plugin_dir" && sha256sum --check checksums.txt 2>&1 | grep -v ': OK$' | head -5)"
    fi
  done
  if [ "$plugin_count" -eq 0 ]; then
    report "packaged plugins found" 1
  else
    note "$plugin_count plugin package(s), $unchecked without checksums"
  fi
fi

# ── Release artifacts ───────────────────────────────────────────────────────
echo "[artifacts]"
if [ ! -d "$RELEASE_DIR" ]; then
  report "release/ exists — run the package scripts" 1
else
  found=0
  for pattern in '*.deb' '*.AppImage' '*.zip'; do
    while IFS= read -r artifact; do
      [ -n "$artifact" ] || continue
      found=$((found + 1))
      size="$(du -h "$artifact" | cut -f1)"
      # A package that is a few kilobytes is a package that failed quietly.
      if [ "$(stat -c %s "$artifact")" -lt 1000000 ]; then
        report "$(basename "$artifact") ($size)" 1
        note "under 1 MB — the packaging step probably failed"
      else
        report "$(basename "$artifact") ($size)" 0
      fi
    done < <(find "$RELEASE_DIR" -maxdepth 1 -name "$pattern" -type f)
  done
  [ "$found" -gt 0 ] || report "release/ contains an installable artifact" 1

  if [ -f "$RELEASE_DIR/SHA256SUMS" ]; then
    if (cd "$RELEASE_DIR" && sha256sum --quiet --check SHA256SUMS >/dev/null 2>&1); then
      report "SHA256SUMS matches the artifacts" 0
    else
      report "SHA256SUMS matches the artifacts" 1
    fi
  else
    report "release/SHA256SUMS exists" 1
  fi
fi

# ── Release notes ───────────────────────────────────────────────────────────
#
# A release nobody described is a release nobody can decide whether to install.
echo "[notes]"
version="$(cd "$ROOT" && git describe --tags --abbrev=0 2>/dev/null || true)"
if [ -z "$version" ]; then
  report "a tag to describe" 1
  note "no tag found; release notes are checked against the newest tag"
elif [ -f "$ROOT/release-notes/$version.md" ]; then
  report "release-notes/$version.md exists" 0
else
  report "release-notes/$version.md exists" 1
fi

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo "✅ release smoke passed"
  echo "   What a script cannot check is in docs/RELEASE_CHECKLIST.md."
else
  echo "❌ release smoke failed"
fi
exit "$FAILED"
