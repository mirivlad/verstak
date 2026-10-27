package api

import (
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"testing"
)

func TestLocalDataRequiresCurrentStoragePermissionAndCannotGrantProcessAccess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("APPDATA", home)
	t.Setenv("LOCALAPPDATA", home)
	a, r := processBridge(t)
	for _, id := range []string{"missing.plugin", "disabled.plugin", "no.storage"} {
		if _, err := a.ReadPluginLocalDataJSON(id, "trust"); err == nil {
			t.Fatal("unauthorized read", id)
		}
		if err := a.WritePluginLocalDataJSON(id, "trust", map[string]interface{}{"approved": true}); err == "" {
			t.Fatal("unauthorized write", id)
		}
	}
	a.plugins = append(a.plugins, plugin.Plugin{Manifest: plugin.Manifest{ID: "metadata-only", Permissions: []string{"storage.namespace"}}, Enabled: true, Status: plugin.StatusLoaded})
	if err := a.WritePluginLocalDataJSON("metadata-only", "trust", map[string]interface{}{"approved": true}); err != "" {
		t.Fatal(err)
	}
	if _, err := a.PluginProcessRun("metadata-only", "attempt", r); err == nil {
		t.Fatal("local data granted process access")
	}
	got, err := a.ReadPluginLocalDataJSON("bridge.plugin", "trust")
	if err != nil || len(got) != 0 {
		t.Fatal("namespace crossed", got, err)
	}
}
