#!/usr/bin/env bash
# Generates a throwaway vault for measurements and GUI probing.
#
# Nobody should benchmark against their working vault, and nobody should have
# to hand-build a plausible one. This writes a tree with the shape Verstak
# actually produces — Deals holding Notes/ and Files/, some grouped into
# folders — at a size you choose.
#
#   scripts/make-synthetic-vault.sh <path> [deals] [files-per-section]
#
#   scripts/make-synthetic-vault.sh /tmp/verstak-bench            # ~2000 files
#   scripts/make-synthetic-vault.sh /tmp/verstak-bench 200 25     # ~10000 files
#
# The result is a plain file tree. That is enough for the sync scanner, which
# works on any directory. To open it in the application, create an empty vault
# through the interface first and point this at the created VerstakVault
# directory — the startup reconciler picks the Deals up.
set -euo pipefail

TARGET="${1:?usage: make-synthetic-vault.sh <path> [deals] [files-per-section]}"
DEALS="${2:-40}"
PER_SECTION="${3:-25}"
FOLDERS=("Clients" "Projects" "Archive")

if [ -e "$TARGET" ] && [ -n "$(ls -A "$TARGET" 2>/dev/null)" ]; then
  echo "refusing to write into a non-empty directory: $TARGET" >&2
  echo "remove it first, or choose another path" >&2
  exit 1
fi

mkdir -p "$TARGET/.verstak"

lorem() {
  # Deterministic filler, so repeated runs produce identical trees and a
  # measurement is not comparing different data.
  local seed="$1"
  printf '# %s\n\nParagraph one for %s.\n\n- point a\n- point b\n\nClosing note.\n' "$seed" "$seed"
}

created=0
for ((d = 0; d < DEALS; d++)); do
  folder="${FOLDERS[$((d % ${#FOLDERS[@]}))]}"
  # Every fourth Deal sits at the vault root rather than inside a folder, so
  # the tree has both shapes.
  if [ $((d % 4)) -eq 3 ]; then
    deal="$TARGET/Deal-$(printf '%03d' "$d")"
  else
    deal="$TARGET/$folder/Deal-$(printf '%03d' "$d")"
  fi

  mkdir -p "$deal/Notes" "$deal/Files"
  lorem "Deal $d overview" > "$deal/Notes/Overview.md"
  created=$((created + 1))

  for ((f = 0; f < PER_SECTION; f++)); do
    lorem "note $d-$f" > "$deal/Notes/note-$(printf '%03d' "$f").md"
    lorem "file $d-$f" > "$deal/Files/document-$(printf '%03d' "$f").md"
    created=$((created + 2))
  done
done

total="$(find "$TARGET" -type f | wc -l)"
bytes="$(du -sh "$TARGET" | cut -f1)"

echo "synthetic vault: $TARGET"
echo "  deals:  $DEALS"
echo "  files:  $total ($bytes on disk)"
echo ""
echo "Measure the current per-operation sync cost against it with:"
echo "  VERSTAK_BENCH_VAULT=$TARGET go test ./internal/core/sync -run TestRecordCostOnSyntheticVault -v"
