package pluginstate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/vault"
)

func TestLoad_DefaultCreation(t *testing.T) {
	dir := t.TempDir()
	v := vault.NewVault(nil)
	if err := v.CreateVault(dir); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	m := NewManager(v)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	state := m.Get()
	if state.SchemaVersion != 1 {
		t.Errorf("SchemaVersion: got %d, want 1", state.SchemaVersion)
	}
	if len(state.EnabledPlugins) != 0 {
		t.Errorf("EnabledPlugins: expected empty, got %v", state.EnabledPlugins)
	}
}

func TestEnableDisable(t *testing.T) {
	dir := t.TempDir()
	v := vault.NewVault(nil)
	if err := v.CreateVault(dir); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	m := NewManager(v)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}

	// Enable
	if err := m.EnablePlugin("test-plugin"); err != nil {
		t.Fatalf("EnablePlugin: %v", err)
	}
	if !m.IsEnabled("test-plugin") {
		t.Error("expected test-plugin to be enabled")
	}

	// Disable
	if err := m.DisablePlugin("test-plugin"); err != nil {
		t.Fatalf("DisablePlugin: %v", err)
	}
	if !m.IsDisabled("test-plugin") {
		t.Error("expected test-plugin to be disabled")
	}
	if m.IsEnabled("test-plugin") {
		t.Error("expected test-plugin to NOT be enabled after disable")
	}
}

func TestDisablePlugin_Persists(t *testing.T) {
	dir := t.TempDir()
	v := vault.NewVault(nil)
	if err := v.CreateVault(dir); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	m := NewManager(v)
	m.Load()

	m.EnablePlugin("test-plugin")
	m.DisablePlugin("test-plugin")

	// Re-load from disk
	m2 := NewManager(v)
	m2.Load()

	if m2.IsEnabled("test-plugin") {
		t.Error("disabled plugin should not be enabled after reload")
	}
	if !m2.IsDisabled("test-plugin") {
		t.Error("disabled plugin should be disabled after reload")
	}
}

func TestRecordDesiredPlugin(t *testing.T) {
	dir := t.TempDir()
	v := vault.NewVault(nil)
	if err := v.CreateVault(dir); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	m := NewManager(v)
	m.Load()

	if err := m.RecordDesiredPlugin("test-plugin", "1.0.0", "official"); err != nil {
		t.Fatalf("RecordDesiredPlugin: %v", err)
	}

	state := m.Get()
	if len(state.DesiredPlugins) != 1 {
		t.Fatalf("DesiredPlugins: expected 1, got %d", len(state.DesiredPlugins))
	}
	if state.DesiredPlugins[0].ID != "test-plugin" {
		t.Errorf("DesiredPlugin ID: got %q, want %q", state.DesiredPlugins[0].ID, "test-plugin")
	}
}

func TestMissingInstalled(t *testing.T) {
	dir := t.TempDir()
	v := vault.NewVault(nil)
	if err := v.CreateVault(dir); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	m := NewManager(v)
	m.Load()

	m.RecordDesiredPlugin("plugin-a", "1.0.0", "official")
	m.RecordDesiredPlugin("plugin-b", "2.0.0", "local")
	m.RecordDesiredPlugin("plugin-c", "3.0.0", "official")

	// Only plugin-a is installed
	missing := m.ListMissingInstalled([]string{"plugin-a"})
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing, got %d", len(missing))
	}

	ids := make(map[string]bool)
	for _, dp := range missing {
		ids[dp.ID] = true
	}
	if !ids["plugin-b"] || !ids["plugin-c"] {
		t.Errorf("expected plugin-b and plugin-c in missing, got %v", ids)
	}
}

func TestCorruptPluginsJSON(t *testing.T) {
	dir := t.TempDir()
	v := vault.NewVault(nil)
	if err := v.CreateVault(dir); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	// Create corrupt plugins.json
	vaultPath := v.GetVaultPath()
	pluginsPath := filepath.Join(vaultPath, ".verstak", "plugins.json")
	os.WriteFile(pluginsPath, []byte("{not json"), 0o644)

	m := NewManager(v)
	err := m.Load()
	if err == nil {
		t.Fatal("expected error for corrupt plugins.json")
	}

	// Should have created defaults
	state := m.Get()
	if state.SchemaVersion != 1 {
		t.Errorf("SchemaVersion: got %d, want 1", state.SchemaVersion)
	}
}

func TestVaultClosed_StateUnavailable(t *testing.T) {
	v := vault.NewVault(nil)
	// Don't open vault — state should fail

	m := NewManager(v)
	err := m.Load()
	if err == nil {
		t.Fatal("expected error when vault is not open")
	}
}
