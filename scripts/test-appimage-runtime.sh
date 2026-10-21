#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEST_ROOT="$(mktemp -d)"
trap 'rm -rf "$TEST_ROOT"' EXIT

python3 - "$ROOT" "$TEST_ROOT" <<'PY'
import ctypes
import os
from pathlib import Path
import shutil
import subprocess
import sys

root, temporary = map(Path, sys.argv[1:])
app = temporary / 'AppDir with spaces'
lib_dir = app / 'usr/lib'
bin_dir = app / 'usr/bin'
lib_dir.mkdir(parents=True)
bin_dir.mkdir()
source_path = '/usr/lib/x86_64-linux-gnu/webkit2gtk-4.1'
source = temporary / 'webkit.c'
source.write_text(f'''const char *helpers(void) {{ return "{source_path}"; }}
const char *injected(void) {{ return "{source_path}/injected-bundle"; }}
''')
library = lib_dir / 'libwebkit2gtk-4.1.so.0'
subprocess.run(['cc', '-shared', '-fPIC', '-o', str(library), str(source)], check=True)
original_size = library.stat().st_size
relocator = root / 'scripts/relocate-webkit-runtime.py'
subprocess.run([sys.executable, str(relocator), str(app), source_path], check=True)
assert library.stat().st_size == original_size, 'ELF offsets changed'
loaded = ctypes.CDLL(str(library))
loaded.helpers.restype = loaded.injected.restype = ctypes.c_char_p
assert loaded.helpers() == b'././webkit2gtk-4.1/'
assert loaded.injected().decode() == source_path + '/injected-bundle'
before = library.read_bytes()
failure = subprocess.run([sys.executable, str(relocator), str(app), '/not/the/compiled/directory'], capture_output=True)
assert failure.returncode != 0, 'unknown runtime path was silently accepted'
assert library.read_bytes() == before, 'failed relocation modified a library'

shutil.copyfile(root / 'packaging/linux/AppRun', app / 'AppRun')
(app / 'AppRun').chmod(0o755)
launcher = bin_dir / 'verstak'
launcher.write_text('''#!/bin/sh
printf '%s\\n' "$PWD" "$WEBKIT_INJECTED_BUNDLE_PATH" "$GIO_MODULE_DIR" "${WEBKIT_DISABLE_SANDBOX_THIS_IS_DANGEROUS-unset}"
''')
launcher.chmod(0o755)
output = subprocess.check_output([str(app / 'AppRun')], env={'PATH': os.environ['PATH']}, text=True).splitlines()
assert output == [str(lib_dir), str(lib_dir / 'webkit2gtk-4.1/injected-bundle'), str(lib_dir / 'gio/modules'), 'unset'], output
print('AppImage runtime relocation and launcher checked')
PY
