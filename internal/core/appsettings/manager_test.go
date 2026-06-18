package appsettings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultCreation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	m := NewManager(path)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	cfg := m.Get()
	if cfg.SchemaVersion != 1 {
		t.Errorf("SchemaVersion: got %d, want 1", cfg.SchemaVersion)
	}
	if cfg.Theme != "dark" {
		t.Errorf("Theme: got %q, want %q", cfg.Theme, "dark")
	}
	if cfg.CurrentVaultPath != "" {
		t.Errorf("CurrentVaultPath: got %q, want empty", cfg.CurrentVaultPath)
	}
}

func TestLoad_CorruptConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Write corrupt JSON
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := NewManager(path)
	err := m.Load()
	if err == nil {
		t.Fatal("expected error for corrupt config")
	}

	// Should have created defaults
	cfg := m.Get()
	if cfg.SchemaVersion != 1 {
		t.Errorf("SchemaVersion: got %d, want 1", cfg.SchemaVersion)
	}

	// Check backup exists
	// Just verify no panic
}

func TestSetCurrentVault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	m := NewManager(path)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}

	if err := m.SetCurrentVault("/home/user/vault1"); err != nil {
		t.Fatalf("SetCurrentVault: %v", err)
	}

	cfg := m.Get()
	if cfg.CurrentVaultPath != "/home/user/vault1" {
		t.Errorf("CurrentVaultPath: got %q, want %q", cfg.CurrentVaultPath, "/home/user/vault1")
	}
	if len(cfg.RecentVaults) != 1 || cfg.RecentVaults[0] != "/home/user/vault1" {
		t.Errorf("RecentVaults: got %v", cfg.RecentVaults)
	}
}

func TestRecentVaults_NoDuplicates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	m := NewManager(path)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}

	m.SetCurrentVault("/vault/a")
	m.SetCurrentVault("/vault/b")
	m.SetCurrentVault("/vault/a") // duplicate

	cfg := m.Get()
	count := 0
	for _, r := range cfg.RecentVaults {
		if r == "/vault/a" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 occurrence of /vault/a in recents, got %d: %v", count, cfg.RecentVaults)
	}
}

func TestUpdate_Patch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	m := NewManager(path)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}

	if err := m.Update(&Config{
		Theme:   "light",
		DevMode: true,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	cfg := m.Get()
	if cfg.Theme != "light" {
		t.Errorf("Theme: got %q, want %q", cfg.Theme, "light")
	}
	if !cfg.DevMode {
		t.Error("DevMode: got false, want true")
	}
}

func TestUpdate_WorkbenchPreferences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	m := NewManager(path)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}

	if err := m.Update(&Config{
		Workbench: WorkbenchPreferences{
			DefaultTextEditorProvider:          "editor.text",
			DefaultMarkdownEditorProvider:      "editor.markdown",
			DefaultNotesMarkdownEditorProvider: "editor.notes",
		},
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	reloaded := NewManager(path)
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	cfg := reloaded.Get()
	if cfg.Workbench.DefaultTextEditorProvider != "editor.text" ||
		cfg.Workbench.DefaultMarkdownEditorProvider != "editor.markdown" ||
		cfg.Workbench.DefaultNotesMarkdownEditorProvider != "editor.notes" {
		t.Fatalf("workbench preferences = %+v", cfg.Workbench)
	}
}

func TestAppSettings_NotInsideVault(t *testing.T) {
	// App settings path should be under ~/.config/verstak/, not inside vault
	path := DefaultConfigPath()
	if filepath.Base(filepath.Dir(path)) != "verstak" {
		t.Errorf("app settings path should be under .config/verstak, got %s", path)
	}
}
