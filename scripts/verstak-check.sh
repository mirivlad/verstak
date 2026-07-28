#!/usr/bin/env bash
# One command that verifies every Verstak repository present next to this one.
#
# Each repository keeps its own entry point; this script only knows where to
# knock. A repository that is not checked out is reported and skipped, not
# treated as a failure — the desktop repo can be cloned on its own.
#
# Usage:
#   scripts/verstak-check.sh              # every repository
#   scripts/verstak-check.sh desktop      # one or more repositories by suffix
#   VERSTAK_CHECK_E2E=0 scripts/verstak-check.sh
#     skips the Playwright suite, which needs ~5 minutes and a browser.
#
# The e2e suite used to be opt-in. Four specs were red for days because of it:
# nobody ran the check that would have said so.
set -uo pipefail

DESKTOP_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKSPACE="$(cd "$DESKTOP_ROOT/.." && pwd)"

FAILED=0
SKIPPED=()
PASSED=()
BROKEN=()

section() { printf '\n\033[1m── %s ──\033[0m\n' "$1"; }

run_step() {
  local label="$1"; shift
  if "$@"; then
    PASSED+=("$label")
    printf '  \033[32m✅\033[0m %s\n' "$label"
  else
    BROKEN+=("$label")
    FAILED=1
    printf '  \033[31m❌\033[0m %s\n' "$label"
  fi
}

# check_repo <directory-name> <label> <command...>
check_repo() {
  local dir="$1"; local label="$2"; shift 2
  local path="$WORKSPACE/$dir"
  if [ ! -d "$path" ]; then
    SKIPPED+=("$dir (not checked out)")
    printf '  \033[33m∅\033[0m %s — not checked out at %s\n' "$label" "$path"
    return
  fi
  run_step "$label" env -C "$path" "$@"
}

wanted() {
  local target="$1"; shift
  # No selectors given means every repository is wanted.
  [ "$#" -eq 0 ] && return 0
  for selector in "$@"; do
    case "$target" in *"$selector"*) return 0 ;; esac
  done
  return 1
}

SELECTORS=("$@")

echo "=== verstak check ==="
echo "workspace: $WORKSPACE"

if wanted verstak-desktop "${SELECTORS[@]}"; then
  section "verstak-desktop"
  check_repo verstak-desktop "desktop: static checks" bash scripts/check.sh
  check_repo verstak-desktop "desktop: go + contract tests" bash scripts/test.sh
  if [ "${VERSTAK_CHECK_E2E:-1}" = "1" ]; then
    check_repo verstak-desktop "desktop: playwright e2e" npm --prefix frontend run test:e2e
  else
    SKIPPED+=("desktop: playwright e2e (VERSTAK_CHECK_E2E=0)")
    printf '  \033[33m∅\033[0m desktop: playwright e2e — skipped by VERSTAK_CHECK_E2E=0\n'
  fi
fi

if wanted verstak-official-plugins "${SELECTORS[@]}"; then
  section "verstak-official-plugins"
  check_repo verstak-official-plugins "plugins: manifests, locales, smokes" bash scripts/check.sh
fi

if wanted verstak-sdk "${SELECTORS[@]}"; then
  section "verstak-sdk"
  check_repo verstak-sdk "sdk: types and schemas" bash scripts/check.sh
  check_repo verstak-sdk "sdk: unit tests" bash scripts/test.sh
fi

if wanted verstak-sync-server "${SELECTORS[@]}"; then
  section "verstak-sync-server"
  check_repo verstak-sync-server "sync-server: go vet" go vet ./...
  check_repo verstak-sync-server "sync-server: go test" go test ./...
fi

if wanted verstak-browser-extension "${SELECTORS[@]}"; then
  section "verstak-browser-extension"
  check_repo verstak-browser-extension "extension: tests" npm test --silent
fi

section "summary"
printf '  passed:  %d\n' "${#PASSED[@]}"
printf '  failed:  %d\n' "${#BROKEN[@]}"
printf '  skipped: %d\n' "${#SKIPPED[@]}"
for item in "${SKIPPED[@]:-}"; do [ -n "$item" ] && printf '    ∅ %s\n' "$item"; done
for item in "${BROKEN[@]:-}"; do [ -n "$item" ] && printf '    ❌ %s\n' "$item"; done

echo ""
if [ "$FAILED" -eq 0 ]; then
  echo "✅ verstak check passed"
else
  echo "❌ verstak check failed"
fi
exit "$FAILED"
