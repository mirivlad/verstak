package api

import (
	"strings"
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

func TestResolveDealCapabilityOperationRejectsInvalidScopeAndHostVersion(t *testing.T) {
	app := newBridgeTestApp(t)
	if err := app.capRegistry.Register("notes.provider", []string{"verstak/notes/v2"}); err != nil {
		t.Fatal(err)
	}
	app.plugins = append(app.plugins,
		plugin.Plugin{
			Manifest: plugin.Manifest{
				ID:                   "notes.provider",
				Name:                 "Notes Provider",
				Version:              "1.0.0",
				APIVersion:           "0.1.0",
				Provides:             []string{"verstak/notes/v2"},
				Permissions:          []string{"commands.register"},
				CapabilityOperations: map[string]map[string]string{"verstak/notes/v2": {"list": "notes.list"}},
			},
			Status: plugin.StatusLoaded, Enabled: true,
		},
		plugin.Plugin{
			Manifest: plugin.Manifest{
				ID: "notes.consumer", Name: "Notes Consumer", Version: "1.0.0", APIVersion: "0.1.0",
				Provides: []string{"notes/consumer/v1"}, OptionalRequires: []string{"verstak/notes/v2"}, Permissions: []string{"storage.namespace"},
			},
			Status: plugin.StatusLoaded, Enabled: true,
		},
	)

	if _, errStr := app.ResolveDealCapabilityOperation("notes.consumer", "verstak/notes/v2", "list", map[string]interface{}{
		"scope": map[string]interface{}{"kind": "deal", "workspaceId": "not-a-uuid"},
	}); !strings.Contains(errStr, "workspaceId") {
		t.Fatalf("invalid scope error = %q, want workspaceId", errStr)
	}

	resolved, errStr := app.ResolveDealCapabilityOperation("notes.consumer", "verstak/notes/v2", "list", map[string]interface{}{
		"scope": map[string]interface{}{"kind": "deal", "workspaceId": "11111111-1111-4111-8111-111111111111"},
	})
	if errStr != "" || resolved["commandId"] != "notes.list" {
		t.Fatalf("resolved = %#v, error = %q", resolved, errStr)
	}

	provider, err := app.findPlugin("notes.provider")
	if err != nil {
		t.Fatal(err)
	}
	provider.Manifest.APIVersion = "0.2.0"
	if _, errStr := app.ResolveDealCapabilityOperation("notes.consumer", "verstak/notes/v2", "list", map[string]interface{}{
		"scope": map[string]interface{}{"kind": "deal", "workspaceId": "11111111-1111-4111-8111-111111111111"},
	}); !strings.Contains(errStr, "host API") {
		t.Fatalf("incompatible host API error = %q", errStr)
	}
}
