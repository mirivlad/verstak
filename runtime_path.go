package main

import "path/filepath"

func bundledWebView2Path(executablePath string) string {
	return filepath.Join(filepath.Dir(executablePath), "webview2")
}
