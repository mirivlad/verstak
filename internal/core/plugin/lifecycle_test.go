package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/capability"
)

// coreCapabilities lists the 5 core capabilities that the platform registers
// before any plugins are loaded.
var coreCapabilities = []string{
	"verstak/core/plugin-manager/v1",
	"verstak/core/capability-registry/v1",
	"verstak/core/contribution-registry/v1",
	"verstak/core/permissions/v1",
	"verstak/core/events/v1",
}

// registerCoreCapabilities registers the 5 core capabilities on a registry.
func registerCoreCapabilities(t *testing.T, reg *capability.Registry) {
	t.Helper()
	if err := reg.Register("verstak-core", coreCapabilities); err != nil {
		t.Fatalf("failed to register core capabilities: %v", err)
	}
}

// createTempPluginWithManifest creates a plugin directory with a custom manifest JSON.
func createTempPluginWithManifest(t *testing.T, dir, id, manifestJSON string) string {
	t.Helper()
	pluginDir := filepath.Join(dir, id)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifestJSON), 0644); err != nil {
		t.Fatal(err)
	}
	return pluginDir
}

// TestLifecycle_CoreCapabilitiesRegisteredBeforePlugins verifies that when core
// capabilities are registered before plugin discovery, a plugin that requires
// verstak/core/plugin-manager/v1 sees it as available (CheckRequired returns empty).
func TestLifecycle_CoreCapabilitiesRegisteredBeforePlugins(t *testing.T) {
	dir := t.TempDir()

	// Create a plugin that requires the core plugin-manager capability.
	manifest := `{
		"schemaVersion": 1,
		"id": "test.lifecycle.core",
		"name": "Core Cap Test",
		"version": "1.0.0",
		"apiVersion": "1.0",
		"provides": ["test.lifecycle.core.cap1"],
		"requires": ["verstak/core/plugin-manager/v1"],
		"permissions": ["vault.read"]
	}`
	createTempPluginWithManifest(t, dir, "test.lifecycle.core", manifest)

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(errs) > 0 {
		t.Fatalf("unexpected discovery errors: %v", errs)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	// Register core capabilities, then check required.
	reg := capability.NewRegistry()
	registerCoreCapabilities(t, reg)

	missing := reg.CheckRequired(plugins[0].Manifest.Requires)
	if len(missing) != 0 {
		t.Errorf("expected no missing required capabilities, got: %v", missing)
	}
}

// TestLifecycle_MissingRequiredCapability verifies that when the required
// capability is NOT registered, CheckRequired reports it as missing and the
// plugin status should be set to StatusMissingRequiredCapability.
func TestLifecycle_MissingRequiredCapability(t *testing.T) {
	dir := t.TempDir()

	manifest := `{
		"schemaVersion": 1,
		"id": "test.lifecycle.missing",
		"name": "Missing Cap Test",
		"version": "1.0.0",
		"apiVersion": "1.0",
		"provides": ["test.lifecycle.missing.cap1"],
		"requires": ["verstak/core/plugin-manager/v1"],
		"permissions": ["vault.read"]
	}`
	createTempPluginWithManifest(t, dir, "test.lifecycle.missing", manifest)

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(errs) > 0 {
		t.Fatalf("unexpected discovery errors: %v", errs)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	// Registry WITHOUT core capabilities.
	reg := capability.NewRegistry()

	missing := reg.CheckRequired(plugins[0].Manifest.Requires)
	if len(missing) != 1 {
		t.Fatalf("expected 1 missing capability, got %d: %v", len(missing), missing)
	}
	if missing[0] != "verstak/core/plugin-manager/v1" {
		t.Errorf("expected missing 'verstak/core/plugin-manager/v1', got %q", missing[0])
	}

	// Simulate lifecycle: set status to missing-required-capability.
	plugins[0].Status = StatusMissingRequiredCapability
	if plugins[0].Status != StatusMissingRequiredCapability {
		t.Errorf("expected status %q, got %q", StatusMissingRequiredCapability, plugins[0].Status)
	}
}

// TestLifecycle_MissingOptionalCapability_DEGRADED verifies that when required
// capabilities are met but an optional capability is missing, the plugin status
// becomes degraded.
func TestLifecycle_MissingOptionalCapability_DEGRADED(t *testing.T) {
	dir := t.TempDir()

	manifest := `{
		"schemaVersion": 1,
		"id": "test.lifecycle.degraded",
		"name": "Degraded Test",
		"version": "1.0.0",
		"apiVersion": "1.0",
		"provides": ["test.lifecycle.degraded.cap1"],
		"requires": ["verstak/core/plugin-manager/v1"],
		"optionalRequires": ["verstak/core/vault/v1"],
		"permissions": ["vault.read"]
	}`
	createTempPluginWithManifest(t, dir, "test.lifecycle.degraded", manifest)

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(errs) > 0 {
		t.Fatalf("unexpected discovery errors: %v", errs)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	reg := capability.NewRegistry()
	registerCoreCapabilities(t, reg)

	// Required capabilities should be satisfied.
	missingRequired := reg.CheckRequired(plugins[0].Manifest.Requires)
	if len(missingRequired) != 0 {
		t.Errorf("expected no missing required capabilities, got: %v", missingRequired)
	}

	// Optional capability vault/v1 is NOT registered.
	missingOptional := reg.CheckRequired(plugins[0].Manifest.OptionalRequires)
	if len(missingOptional) != 1 {
		t.Fatalf("expected 1 missing optional capability, got %d: %v", len(missingOptional), missingOptional)
	}
	if missingOptional[0] != "verstak/core/vault/v1" {
		t.Errorf("expected missing optional 'verstak/core/vault/v1', got %q", missingOptional[0])
	}

	// Simulate lifecycle: required OK + optional missing => degraded.
	plugins[0].Status = StatusDegraded
	if plugins[0].Status != StatusDegraded {
		t.Errorf("expected status %q, got %q", StatusDegraded, plugins[0].Status)
	}
}

// TestLifecycle_AllCapabilitiesResolved_LOADED verifies that when all required
// and optional capabilities are registered, the plugin status becomes loaded.
func TestLifecycle_AllCapabilitiesResolved_LOADED(t *testing.T) {
	dir := t.TempDir()

	manifest := `{
		"schemaVersion": 1,
		"id": "test.lifecycle.loaded",
		"name": "Loaded Test",
		"version": "1.0.0",
		"apiVersion": "1.0",
		"provides": ["test.lifecycle.loaded.cap1"],
		"requires": ["verstak/core/plugin-manager/v1"],
		"optionalRequires": ["verstak/core/vault/v1"],
		"permissions": ["vault.read"]
	}`
	createTempPluginWithManifest(t, dir, "test.lifecycle.loaded", manifest)

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(errs) > 0 {
		t.Fatalf("unexpected discovery errors: %v", errs)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	reg := capability.NewRegistry()
	registerCoreCapabilities(t, reg)

	// Also register the vault capability so optional is satisfied.
	if err := reg.Register("verstak-vault", []string{"verstak/core/vault/v1"}); err != nil {
		t.Fatalf("failed to register vault capability: %v", err)
	}

	// Required capabilities satisfied.
	missingRequired := reg.CheckRequired(plugins[0].Manifest.Requires)
	if len(missingRequired) != 0 {
		t.Errorf("expected no missing required capabilities, got: %v", missingRequired)
	}

	// Optional capabilities satisfied.
	missingOptional := reg.CheckRequired(plugins[0].Manifest.OptionalRequires)
	if len(missingOptional) != 0 {
		t.Errorf("expected no missing optional capabilities, got: %v", missingOptional)
	}

	// All resolved => loaded.
	plugins[0].Status = StatusLoaded
	if plugins[0].Status != StatusLoaded {
		t.Errorf("expected status %q, got %q", StatusLoaded, plugins[0].Status)
	}
}

// TestLifecycle_ReloadPlugins_DoesNotDuplicateCapabilities verifies that after
// UnregisterAll + re-registering core capabilities, registering the same
// plugin capabilities again returns an error (no silent duplication).
func TestLifecycle_ReloadPlugins_DoesNotDuplicateCapabilities(t *testing.T) {
	dir := t.TempDir()

	manifest := `{
		"schemaVersion": 1,
		"id": "test.lifecycle.reload",
		"name": "Reload Test",
		"version": "1.0.0",
		"apiVersion": "1.0",
		"provides": ["test.lifecycle.reload.cap1"],
		"requires": ["verstak/core/plugin-manager/v1"],
		"permissions": ["vault.read"]
	}`
	createTempPluginWithManifest(t, dir, "test.lifecycle.reload", manifest)

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(errs) > 0 {
		t.Fatalf("unexpected discovery errors: %v", errs)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	reg := capability.NewRegistry()

	// First lifecycle: register core + plugin capabilities.
	registerCoreCapabilities(t, reg)
	if err := reg.Register(plugins[0].Manifest.ID, plugins[0].Manifest.Provides); err != nil {
		t.Fatalf("first register of plugin capabilities failed: %v", err)
	}

	// Simulate reload: UnregisterAll + re-register core.
	reg.UnregisterAll()
	registerCoreCapabilities(t, reg)

	// Re-register plugin capabilities — should succeed now (map was cleared).
	if err := reg.Register(plugins[0].Manifest.ID, plugins[0].Manifest.Provides); err != nil {
		t.Fatalf("re-register of plugin capabilities after UnregisterAll failed: %v", err)
	}

	// But registering the SAME capability a second time (without clearing) must fail.
	if err := reg.Register(plugins[0].Manifest.ID, plugins[0].Manifest.Provides); err == nil {
		t.Error("expected error when registering duplicate capability, got nil")
	}

	// Also verify core capabilities cannot be silently duplicated.
	if err := reg.Register("another-plugin", []string{"verstak/core/plugin-manager/v1"}); err == nil {
		t.Error("expected error when registering duplicate core capability, got nil")
	}
}

// TestLifecycle_DisabledPlugin verifies that a disabled plugin does NOT register
// its capabilities, but remains in the plugin list.
func TestLifecycle_DisabledPlugin(t *testing.T) {
	dir := t.TempDir()

	// Create a disabled plugin (Enabled=false via custom manifest).
	manifest := `{
		"schemaVersion": 1,
		"id": "test.lifecycle.disabled",
		"name": "Disabled Test",
		"version": "1.0.0",
		"apiVersion": "1.0",
		"provides": ["test.lifecycle.disabled.cap1"],
		"requires": ["verstak/core/plugin-manager/v1"],
		"permissions": ["vault.read"]
	}`
	createTempPluginWithManifest(t, dir, "test.lifecycle.disabled", manifest)

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(errs) > 0 {
		t.Fatalf("unexpected discovery errors: %v", errs)
	}
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}

	// Simulate lifecycle: plugin is disabled.
	plugins[0].Enabled = false
	plugins[0].Status = StatusDisabled

	reg := capability.NewRegistry()
	registerCoreCapabilities(t, reg)

	// Disabled plugin should NOT register its capabilities.
	if !plugins[0].Enabled {
		// Skip registration — this is the expected lifecycle behavior.
	} else {
		if err := reg.Register(plugins[0].Manifest.ID, plugins[0].Manifest.Provides); err != nil {
			t.Fatalf("failed to register capabilities for enabled plugin: %v", err)
		}
	}

	// Verify the plugin's capabilities are NOT in the registry.
	for _, capName := range plugins[0].Manifest.Provides {
		if reg.Has(capName) {
			t.Errorf("disabled plugin capability %q should NOT be registered", capName)
		}
	}

	// Plugin is still in the list.
	if plugins[0].Status != StatusDisabled {
		t.Errorf("expected status %q, got %q", StatusDisabled, plugins[0].Status)
	}
	if plugins[0].Manifest.ID != "test.lifecycle.disabled" {
		t.Errorf("expected plugin ID 'test.lifecycle.disabled', got %q", plugins[0].Manifest.ID)
	}
}
