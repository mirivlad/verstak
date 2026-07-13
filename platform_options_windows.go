//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

func applyPlatformOptions(app *options.App) {
	executablePath, err := os.Executable()
	if err != nil {
		return
	}

	runtimePath := bundledWebView2Path(executablePath)
	if _, err := os.Stat(filepath.Join(runtimePath, "msedgewebview2.exe")); err != nil {
		return
	}

	// Fixed Version WebView2 runtimes from v120 need these permissions on
	// unpackaged Windows 10 applications. The archive is extracted by the user,
	// so the command can update permissions without an installer or download.
	for _, sid := range []string{"*S-1-15-2-2", "*S-1-15-2-1"} {
		_ = exec.Command("icacls", runtimePath, "/grant", sid+":(OI)(CI)(RX)", "/T", "/C").Run()
	}

	app.Windows = &windows.Options{
		WebviewBrowserPath: runtimePath,
	}
}
