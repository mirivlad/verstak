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
LINUX_BUNDLE="${VERSTAK_LINUX_BUNDLE_DIR:-$ROOT/build/linux-amd64}"
WINDOWS_BUNDLE="${VERSTAK_WINDOWS_OUTPUT_DIR:-$ROOT/build/windows-amd64}"
VERSION="${1:-}"
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

check_plugin_root() {
  local label="$1"
  local root="$2"
  if [ ! -d "$root" ]; then
    report "$label shipping plugins exist" 1
    return
  fi
  if [ -e "$root/platform-test" ]; then
    report "$label excludes platform-test" 1
  else
    report "$label excludes platform-test" 0
  fi

  local plugin_count=0
  local unchecked=0
  for plugin_dir in "$root"/*/; do
    [ -d "$plugin_dir" ] || continue
    local name
    name="$(basename "$plugin_dir")"
    plugin_count=$((plugin_count + 1))
    if [ ! -f "$plugin_dir/checksums.txt" ]; then
      report "$label/$name carries checksums.txt" 1
      unchecked=$((unchecked + 1))
      continue
    fi
    if (cd "$plugin_dir" && sha256sum --quiet --check checksums.txt >/dev/null 2>&1); then
      local extra
      extra="$(cd "$plugin_dir" && find . -type f ! -name checksums.txt -printf '%P\n' | LC_ALL=C sort | comm -23 - <(cut -d' ' -f3- checksums.txt | LC_ALL=C sort))"
      if [ -n "$extra" ]; then
        report "$label/$name matches its checksums" 1
        note "not part of the package: $(echo "$extra" | tr '\n' ' ')"
      else
        report "$label/$name matches its checksums" 0
      fi
    else
      report "$label/$name matches its checksums" 1
      note "$(cd "$plugin_dir" && sha256sum --check checksums.txt 2>&1 | grep -v ': OK$' | head -5)"
    fi
  done
  if [ "$plugin_count" -eq 0 ]; then
    report "$label has packaged plugins" 1
  else
    note "$label: $plugin_count plugin package(s), $unchecked without checksums"
  fi
}

echo "=== release smoke ==="

# ── Staged shipping bundles ────────────────────────────────────────────────
# These survive the whole release pipeline. build/bin is deliberately not
# checked: the later Windows Wails -clean build replaces that transient staging
# directory after the Linux package has already been produced.
echo "[shipping bundles]"
if [ -x "$LINUX_BUNDLE/verstak-desktop" ]; then
  report "Linux shipping binary exists" 0
else
  report "Linux shipping binary exists" 1
fi
if [ -s "$WINDOWS_BUNDLE/verstak-desktop.exe" ]; then
  report "Windows shipping binary exists" 0
else
  report "Windows shipping binary exists" 1
fi
check_plugin_root "Linux" "$LINUX_BUNDLE/plugins"
check_plugin_root "Windows" "$WINDOWS_BUNDLE/plugins"

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
# A release nobody described is a release nobody can decide whether to install.
echo "[notes]"
version="$VERSION"
if [ -z "$version" ]; then
  version="$(cd "$ROOT" && git describe --tags --abbrev=0 2>/dev/null || true)"
fi
if [ -z "$version" ]; then
  report "a release version to describe" 1
  note "pass a version to release-smoke.sh or create a tag"
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
