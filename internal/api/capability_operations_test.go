package api

import (
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

func TestResolvePluginCapabilityOperationChecksDependencyAndPublishedOperation(t *testing.T) {
	app := newBridgeTestApp(t)
	provider, err := app.findPlugin("bridge.plugin")
	if err != nil {
		t.Fatal(err)
	}
	provider.Manifest.CapabilityOperations = map[string]map[string]string{
		"bridge/cap/v1": {"echo": "bridge.command"},
	}
	app.plugins = append(app.plugins, plugin.Plugin{
		Manifest: plugin.Manifest{
			ID:               "cap.consumer",
			Name:             "Capability Consumer",
			Version:          "1.0.0",
			Provides:         []string{"consumer/cap/v1"},
			OptionalRequires: []string{"bridge/cap/v1"},
			Permissions:      []string{"storage.namespace"},
		},
		Status:  plugin.StatusLoaded,
		Enabled: true,
	})

	resolved, errStr := app.ResolvePluginCapabilityOperation("cap.consumer", "bridge/cap/v1", "echo")
	if errStr != "" {
		t.Fatalf("ResolvePluginCapabilityOperation: %s", errStr)
	}
	if resolved["pluginId"] != "bridge.plugin" || resolved["commandId"] != "bridge.command" {
		t.Fatalf("resolution = %#v", resolved)
	}
	if _, errStr := app.ResolvePluginCapabilityOperation("cap.consumer", "bridge/cap/v1", "missing"); errStr == "" {
		t.Fatal("expected missing operation error")
	}
	if _, errStr := app.ResolvePluginCapabilityOperation("no.storage", "bridge/cap/v1", "echo"); errStr == "" {
		t.Fatal("expected undeclared capability dependency error")
	}
}
