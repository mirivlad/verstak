package appsettings

import (
	"os"
	"path/filepath"
	"reflect"
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
	if cfg.Language != "system" {
		t.Errorf("Language: got %q, want %q", cfg.Language, "system")
	}
	if cfg.SidebarWidth != DefaultSidebarWidth {
		t.Errorf("SidebarWidth: got %d, want %d", cfg.SidebarWidth, DefaultSidebarWidth)
	}
}

func TestUpdatePersistsSidebarAndExpandedFolders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	manager := NewManager(path)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}
	width := 320
	expanded := []string{"folder-a", "folder-b"}
	if err := manager.UpdateUIState(&width, &expanded); err != nil {
		t.Fatal(err)
	}

	reloaded := NewManager(path)
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	cfg := reloaded.Get()
	if cfg.SidebarWidth != 320 {
		t.Fatalf("SidebarWidth = %d, want 320", cfg.SidebarWidth)
	}
	if !reflect.DeepEqual(cfg.ExpandedFolderIDs, []string{"folder-a", "folder-b"}) {
		t.Fatalf("ExpandedFolderIDs = %#v", cfg.ExpandedFolderIDs)
	}

	expanded = []string{}
	if err := reloaded.UpdateUIState(nil, &expanded); err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Get().ExpandedFolderIDs; len(got) != 0 {
		t.Fatalf("ExpandedFolderIDs after clear = %#v", got)
	}
}

func TestLoad_MissingLanguageDefaultsToSystem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"theme":"dark","devMode":true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	m := NewManager(path)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := m.Get().Language; got != "system" {
		t.Fatalf("Language = %q, want system", got)
	}
}

func TestUpdateLanguagePersistsAndPreservesUnrelatedSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	m := NewManager(path)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	if err := m.Update(&Config{Theme: "light", DevMode: true}); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateLanguage("ru"); err != nil {
		t.Fatalf("UpdateLanguage: %v", err)
	}

	reloaded := NewManager(path)
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	cfg := reloaded.Get()
	if cfg.Language != "ru" || cfg.Theme != "light" || !cfg.DevMode {
		t.Fatalf("settings after language update = %+v", cfg)
	}
	if err := reloaded.UpdateLanguage("de"); err == nil {
		t.Fatal("UpdateLanguage(de) succeeded, want validation error")
	}
	if got := reloaded.Get().Language; got != "ru" {
		t.Fatalf("Language after rejected update = %q, want ru", got)
	}
}

func TestUpdateLanguageNotifiesTheDesktopShellAfterPersisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	manager := NewManager(path)
	if err := manager.Load(); err != nil {
		t.Fatal(err)
	}

	var received []string
	manager.SetLanguageChangedHandler(func(language string) {
		received = append(received, language)
	})
	if err := manager.UpdateLanguage(LanguageRussian); err != nil {
		t.Fatalf("UpdateLanguage: %v", err)
	}
	if got := manager.Get().Language; got != LanguageRussian {
		t.Fatalf("persisted language = %q, want %q", got, LanguageRussian)
	}
	if !reflect.DeepEqual(received, []string{LanguageRussian}) {
		t.Fatalf("language notifications = %#v, want [%q]", received, LanguageRussian)
	}
	if err := manager.UpdateLanguage("de"); err == nil {
		t.Fatal("UpdateLanguage(de) succeeded, want validation failure")
	}
	if !reflect.DeepEqual(received, []string{LanguageRussian}) {
		t.Fatalf("language notification fired after rejected update: %#v", received)
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

func TestBrowserReceiverTokenPersistsAndRotates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	manager := NewManager(path)
	if err := manager.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	firstToken, err := manager.EnsureBrowserReceiverToken()
	if err != nil {
		t.Fatalf("EnsureBrowserReceiverToken: %v", err)
	}
	if firstToken == "" {
		t.Fatal("EnsureBrowserReceiverToken returned an empty token")
	}

	reloaded := NewManager(path)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	persistedToken, err := reloaded.EnsureBrowserReceiverToken()
	if err != nil {
		t.Fatalf("EnsureBrowserReceiverToken after reload: %v", err)
	}
	if persistedToken != firstToken {
		t.Fatalf("persisted token = %q, want %q", persistedToken, firstToken)
	}

	rotatedToken, err := reloaded.RotateBrowserReceiverToken()
	if err != nil {
		t.Fatalf("RotateBrowserReceiverToken: %v", err)
	}
	if rotatedToken == "" || rotatedToken == firstToken {
		t.Fatalf("rotated token = %q, want new non-empty token", rotatedToken)
	}
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

func TestUpdateSync(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	m := NewManager(path)
	if err := m.Load(); err != nil {
		t.Fatal(err)
	}
	if err := m.Update(&Config{DevMode: true}); err != nil {
		t.Fatalf("Update dev mode: %v", err)
	}

	if err := m.UpdateSync(SyncSettings{
		Enabled:      true,
		ServerURL:    "https://sync.example",
		DeviceID:     "device-1",
		DeviceName:   "Desktop",
		SyncInterval: 15,
		LastStatus:   "connected",
		LastSyncAt:   "2026-06-27T00:00:00Z",
		LastError:    "previous",
	}); err != nil {
		t.Fatalf("UpdateSync: %v", err)
	}

	reloaded := NewManager(path)
	if err := reloaded.Load(); err != nil {
		t.Fatal(err)
	}
	cfg := reloaded.Get()
	if !cfg.Sync.Enabled ||
		cfg.Sync.ServerURL != "https://sync.example" ||
		cfg.Sync.DeviceID != "device-1" ||
		cfg.Sync.DeviceName != "Desktop" ||
		cfg.Sync.SyncInterval != 15 ||
		cfg.Sync.LastStatus != "connected" ||
		cfg.Sync.LastSyncAt != "2026-06-27T00:00:00Z" ||
		cfg.Sync.LastError != "previous" {
		t.Fatalf("sync settings = %+v", cfg.Sync)
	}
	if !cfg.DevMode {
		t.Fatal("UpdateSync changed DevMode")
	}
}

func TestAppSettings_NotInsideVault(t *testing.T) {
	// App settings path should be under ~/.config/verstak/, not inside vault
	path := DefaultConfigPath()
	if filepath.Base(filepath.Dir(path)) != "verstak" {
		t.Errorf("app settings path should be under .config/verstak, got %s", path)
	}
}
