package api

import (
	"log"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/buildinfo"
	"github.com/verstak/verstak-desktop/internal/core/diagnostics"
	"github.com/verstak/verstak-desktop/internal/shell/debug"
)

// DiagnosticsInfo is what the settings screen shows before anything is written:
// where the log is, and whether there is one at all.
type DiagnosticsInfo struct {
	LogPath   string `json:"logPath"`
	LogDir    string `json:"logDir"`
	Verbose   bool   `json:"verbose"`
	LastError string `json:"lastError,omitempty"`
}

// GetDiagnosticsInfo reports where this run is writing its log.
func (a *App) GetDiagnosticsInfo() DiagnosticsInfo {
	return DiagnosticsInfo{
		LogPath: debug.SessionLogPath(),
		LogDir:  debug.LogDir(),
		Verbose: debug.IsEnabled(),
	}
}

// CollectDiagnostics writes a report a user can read before sending it, and
// returns where it went. What it contains is decided in the diagnostics package;
// what it must never contain is the vault's contents.
func (a *App) CollectDiagnostics() (string, string) {
	now := time.Now()
	report := diagnostics.Render(a.diagnosticsInput(), now)
	path, err := diagnostics.Write(debug.LogDir(), report, now)
	if err != nil {
		log.Printf("[api] CollectDiagnostics: %v", err)
		return "", err.Error()
	}
	log.Printf("[api] diagnostics written to %s", path)
	return path, ""
}

func (a *App) diagnosticsInput() diagnostics.Input {
	build := buildinfo.Get()
	input := diagnostics.Input{
		AppVersion: build.Version,
		Commit:     build.Commit,
		BuiltAt:    build.BuildDate,
		LogPath:    debug.SessionLogPath(),
	}
	input.LogTail = diagnostics.ReadLogTail(input.LogPath)

	if a.appSettings != nil {
		settings := a.appSettings.Get()
		input.Language = settings.Language
		input.SyncDevice = settings.Sync.DeviceID
		input.LastWarning = settings.Sync.LastError
	}
	if a.vault != nil {
		input.VaultPath = a.vault.GetVaultPath()
		input.VaultOpen = input.VaultPath != ""
	}
	if a.syncSvc != nil {
		// Deliberately not the API key: the report is meant to be sent to
		// somebody, and that key is what lets them sync as this device.
		serverURL, _, _, lastSyncAt, err := a.syncSvc.GetState()
		if err == nil {
			input.SyncServer = serverURL
			input.LastSyncAt = lastSyncAt
		}
	}
	for _, item := range a.plugins {
		input.Plugins = append(input.Plugins, diagnostics.PluginState{
			ID:      item.Manifest.ID,
			Name:    item.Manifest.Name,
			Version: item.Manifest.Version,
			Source:  item.Manifest.Source,
			Status:  string(item.Status),
			Enabled: item.Enabled,
			Error:   item.Error,
		})
	}
	return input
}
