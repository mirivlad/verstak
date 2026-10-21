#!/usr/bin/env python3
"""Keep release WebKitGTK helper processes inside the staged AppImage."""

import os
from pathlib import Path
import sys


def relocate(app_dir, runtime_dir):
    original = os.fsencode(os.path.normpath(runtime_dir)) + b"\0"
    relative = b"././webkit2gtk-4.1/"
    if len(relative) >= len(original):
        raise ValueError("WebKit helper path is too short to relocate in place")
    replacement = relative.ljust(len(original), b"\0")
    changed = 0
    for library in (app_dir / "usr/lib").glob("libwebkit2gtk-4.1.so*"):
        contents = library.read_bytes()
        if original not in contents:
            continue
        # Match the complete NUL-terminated directory, preserving the ELF size
        # and the longer injected-bundle path that shares this prefix.
        library.write_bytes(contents.replace(original, replacement))
        changed += 1
    if not changed:
        raise ValueError(f"no WebKit library references helper directory {runtime_dir}")
    return changed


if __name__ == "__main__":
    if len(sys.argv) != 3:
        sys.exit("usage: relocate-webkit-runtime.py <AppDir> <source-runtime-dir>")
    try:
        count = relocate(Path(sys.argv[1]), sys.argv[2])
    except (OSError, ValueError) as error:
        sys.exit(f"cannot relocate WebKit helpers: {error}")
    print(f"Relocated WebKit helpers in {count} library/libraries")
