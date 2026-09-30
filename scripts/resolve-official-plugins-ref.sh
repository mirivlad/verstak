#!/usr/bin/env bash
# Prints the official plugins git ref a desktop release must package.
#
# The ref is pinned in OFFICIAL_PLUGINS_VERSION at the repository root. It used
# to be derived from the desktop tag, which only worked while both repositories
# happened to share version numbers; once desktop reached v0.2.x and plugins
# stayed on v0.1.x, a pushed desktop tag asked for a plugins tag that does not
# exist. A manual workflow run may still override the pin explicitly.
#
# When the release notes declare the plugins version ("**Official plugins ...**
# vX.Y.Z"), it must match what is packaged, so the notes cannot promise one
# plugins release while the installer ships another.
#
# usage: resolve-official-plugins-ref.sh <desktop-version> [override-ref]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PIN_FILE="${VERSTAK_OFFICIAL_PLUGINS_PIN_FILE:-$ROOT/OFFICIAL_PLUGINS_VERSION}"
RELEASE_NOTES_DIR="${VERSTAK_RELEASE_NOTES_DIR:-$ROOT/release-notes}"
VERSION="${1:?desktop version required}"
OVERRIDE="${2:-}"

if [[ ! -s "$PIN_FILE" ]]; then
  echo "official plugins pin is missing or empty: $PIN_FILE" >&2
  exit 1
fi
PINNED="$(tr -d '[:space:]' < "$PIN_FILE")"
if [[ ! "$PINNED" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.-]+)?$ ]]; then
  echo "official plugins pin must be a release tag such as v0.1.5, got: $PINNED" >&2
  exit 1
fi

REF="${OVERRIDE:-$PINNED}"

NOTES_FILE="$RELEASE_NOTES_DIR/$VERSION.md"
if [[ -f "$NOTES_FILE" ]]; then
  DECLARED="$(grep -E '^\*\*Official plugins' "$NOTES_FILE" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9-]+)*' | head -n1 || true)"
  if [[ -n "$DECLARED" && "$DECLARED" != "$REF" ]]; then
    echo "release notes $NOTES_FILE declare official plugins $DECLARED, but the release would package $REF" >&2
    exit 1
  fi
fi

printf '%s\n' "$REF"
