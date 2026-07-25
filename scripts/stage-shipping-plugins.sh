#!/usr/bin/env bash
# Copies a plugin tree into a destination, leaving out plugins whose manifest
# sets "development": true.
#
# The development tree keeps every plugin — a developer needs platform-test in
# the sidebar. Packaged builds must not have it, or a normal user sees a
# "Platform Test" tool and its self-test output in the status bar.
#
# usage: stage-shipping-plugins.sh <source-plugins-dir> <destination-dir>
set -euo pipefail

SOURCE="${1:?source plugins directory required}"
DESTINATION="${2:?destination directory required}"

if [[ ! -d "$SOURCE" ]]; then
  echo "plugin source directory not found: $SOURCE" >&2
  exit 1
fi

rm -rf "$DESTINATION"
mkdir -p "$DESTINATION"

for plugin_dir in "$SOURCE"/*/; do
  [[ -d "$plugin_dir" ]] || continue
  plugin_name="$(basename "$plugin_dir")"
  manifest="$plugin_dir/plugin.json"

  if [[ -f "$manifest" ]] && command -v python3 >/dev/null 2>&1; then
    if python3 -c "import json,sys; sys.exit(0 if json.load(open(sys.argv[1])).get('development') else 1)" "$manifest"; then
      echo "  ⊘ excluding development plugin: $plugin_name"
      continue
    fi
  fi

  cp -R "$plugin_dir" "$DESTINATION/$plugin_name"
done

chmod -R a+rX "$DESTINATION"
