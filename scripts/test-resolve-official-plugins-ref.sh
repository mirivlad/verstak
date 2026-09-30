#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RESOLVER="$ROOT/scripts/resolve-official-plugins-ref.sh"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

fail() {
  echo "official plugins ref resolver: $*" >&2
  exit 1
}

resolve() {
  VERSTAK_OFFICIAL_PLUGINS_PIN_FILE="$WORK/pin" \
  VERSTAK_RELEASE_NOTES_DIR="$WORK/notes" \
  "$RESOLVER" "$@"
}

mkdir -p "$WORK/notes"
printf 'v0.1.5\n' > "$WORK/pin"

# A tag push passes only the desktop version: the pin decides, not the tag name.
[[ "$(resolve v0.2.6)" == "v0.1.5" ]] || fail "desktop tag must resolve to the pinned plugins version"

# A manual run may override the pin.
[[ "$(resolve v0.2.6 f49878284e6a)" == "f49878284e6a" ]] || fail "explicit override must win"

# Notes that name the packaged version pass.
printf '# Verstak v0.2.6\n\n**Official plugins / Официальные плагины:** v0.1.5.\n' > "$WORK/notes/v0.2.6.md"
[[ "$(resolve v0.2.6)" == "v0.1.5" ]] || fail "matching notes must pass"

# Notes that promise a different plugins release must stop the build.
printf '# Verstak v0.2.6\n\n**Official plugins / Официальные плагины:** v0.1.4.\n' > "$WORK/notes/v0.2.6.md"
if resolve v0.2.6 >/dev/null 2>&1; then
  fail "notes declaring another plugins version must fail"
fi
if resolve v0.2.6 main >/dev/null 2>&1; then
  fail "an override that contradicts the notes must fail"
fi

# The pin must be a release tag, never a moving branch.
printf 'main\n' > "$WORK/pin"
if resolve v0.2.7 >/dev/null 2>&1; then
  fail "a branch name in the pin must be rejected"
fi
rm -f "$WORK/pin"
if resolve v0.2.7 >/dev/null 2>&1; then
  fail "a missing pin must be rejected"
fi

# The real pin in this repository must itself be valid.
"$RESOLVER" v0.0.0-contract >/dev/null || fail "repository OFFICIAL_PLUGINS_VERSION is invalid"

echo "official plugins ref resolver test passed"
