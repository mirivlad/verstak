package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/vault"
)

// newTestVault creates a vault in a temp directory for testing.
func newTestVault(t *testing.T) (*vault.Vault, string) {
	t.Helper()
	tmpDir := t.TempDir()
	v := vault.NewVault(nil)
	if err := v.CreateVault(tmpDir); err != nil {
		t.Fatalf("failed to create test vault: %v", err)
	}
	return v, tmpDir
}

func newTestStorage(t *testing.T) (*Storage, string) {
	t.Helper()
	v, dir := newTestVault(t)
	return New(v), dir
}

// ─── Settings tests ──────────────────────────────────────────

func TestWriteReadPluginSettings(t *testing.T) {
	s, _ := newTestStorage(t)

	data := map[string]interface{}{
		"theme": "dark",
		"lang":  "en",
		"count": float64(42),
	}

	if err := s.WritePluginSettings("my-plugin", data); err != nil {
		t.Fatalf("WritePluginSettings: %v", err)
	}

	got, err := s.ReadPluginSettings("my-plugin")
	if err != nil {
		t.Fatalf("ReadPluginSettings: %v", err)
	}

	if got["theme"] != "dark" {
		t.Errorf("theme = %v, want dark", got["theme"])
	}
	if got["lang"] != "en" {
		t.Errorf("lang = %v, want en", got["lang"])
	}
	if got["count"] != float64(42) {
		t.Errorf("count = %v, want 42", got["count"])
	}
}

func TestReadPluginSettings_NotFound(t *testing.T) {
	s, _ := newTestStorage(t)

	got, err := s.ReadPluginSettings("unknown-plugin")
	if err != nil {
		t.Fatalf("ReadPluginSettings: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestReadPluginSettings_Corrupt(t *testing.T) {
	s, dir := newTestStorage(t)

	// Write corrupt JSON into the settings file
	settingsDir := filepath.Join(dir, "VerstakVault", ".verstak", "plugin-settings", "bad-plugin")
	os.MkdirAll(settingsDir, 0o755)
	os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte("{not json!!"), 0o644)

	_, err := s.ReadPluginSettings("bad-plugin")
	if err == nil {
		t.Fatal("expected error for corrupt settings.json, got nil")
	}
}

func TestWritePluginSetting_SingleKey(t *testing.T) {
	s, _ := newTestStorage(t)

	// Write a single key
	if err := s.WritePluginSetting("my-plugin", "key1", "value1"); err != nil {
		t.Fatalf("WritePluginSetting: %v", err)
	}

	// Read it back
	val, err := s.ReadPluginSetting("my-plugin", "key1")
	if err != nil {
		t.Fatalf("ReadPluginSetting: %v", err)
	}
	if val != "value1" {
		t.Errorf("key1 = %v, want value1", val)
	}

	// Write another key, first should be preserved
	if err := s.WritePluginSetting("my-plugin", "key2", float64(99)); err != nil {
		t.Fatalf("WritePluginSetting: %v", err)
	}

	settings, err := s.ReadPluginSettings("my-plugin")
	if err != nil {
		t.Fatalf("ReadPluginSettings: %v", err)
	}
	if settings["key1"] != "value1" {
		t.Errorf("key1 after second write = %v, want value1", settings["key1"])
	}
	if settings["key2"] != float64(99) {
		t.Errorf("key2 = %v, want 99", settings["key2"])
	}

	// Reading a missing key returns nil
	val, err = s.ReadPluginSetting("my-plugin", "missing")
	if err != nil {
		t.Fatalf("ReadPluginSetting: %v", err)
	}
	if val != nil {
		t.Errorf("missing key = %v, want nil", val)
	}
}

// ─── Data JSON tests ─────────────────────────────────────────

func TestPluginDataJSON_WriteRead(t *testing.T) {
	s, _ := newTestStorage(t)

	data := map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
		"meta":  map[string]interface{}{"version": float64(1)},
	}

	if err := s.WritePluginDataJSON("data-plugin", "mydata", data); err != nil {
		t.Fatalf("WritePluginDataJSON: %v", err)
	}

	got, err := s.ReadPluginDataJSON("data-plugin", "mydata")
	if err != nil {
		t.Fatalf("ReadPluginDataJSON: %v", err)
	}

	items, ok := got["items"].([]interface{})
	if !ok {
		t.Fatalf("items is not []interface{}, it's %T", got["items"])
	}
	if len(items) != 3 {
		t.Errorf("items len = %d, want 3", len(items))
	}

	// Ensure separate names don't collide
	got2, err := s.ReadPluginDataJSON("data-plugin", "other")
	if err != nil {
		t.Fatalf("ReadPluginDataJSON other: %v", err)
	}
	if len(got2) != 0 {
		t.Errorf("expected empty map for other, got %v", got2)
	}
}

// ─── Cache JSON tests ────────────────────────────────────────

func TestPluginCacheJSON_WriteRead(t *testing.T) {
	s, _ := newTestStorage(t)

	data := map[string]interface{}{
		"lastSync": "2025-01-01T00:00:00Z",
		"hitRate":  0.95,
	}

	if err := s.WritePluginCacheJSON("cache-plugin", "sync-state", data); err != nil {
		t.Fatalf("WritePluginCacheJSON: %v", err)
	}

	got, err := s.ReadPluginCacheJSON("cache-plugin", "sync-state")
	if err != nil {
		t.Fatalf("ReadPluginCacheJSON: %v", err)
	}

	if got["lastSync"] != "2025-01-01T00:00:00Z" {
		t.Errorf("lastSync = %v", got["lastSync"])
	}

	// Empty read for missing cache
	got2, err := s.ReadPluginCacheJSON("cache-plugin", "nope")
	if err != nil {
		t.Fatalf("ReadPluginCacheJSON nope: %v", err)
	}
	if len(got2) != 0 {
		t.Errorf("expected empty map, got %v", got2)
	}
}

// ─── Path traversal tests ────────────────────────────────────

func TestPathTraversal_Blocked(t *testing.T) {
	s, _ := newTestStorage(t)

	traversalIDs := []string{
		"..",
		"../evil",
		"foo/../../bar",
		"/absolute",
		`backslash\traverse`,
	}

	for _, id := range traversalIDs {
		t.Run(id, func(t *testing.T) {
			err := s.WritePluginSettings(id, map[string]interface{}{"x": 1})
			if err == nil {
				t.Errorf("WritePluginSettings(%q): expected error, got nil", id)
			}

			_, err = s.ReadPluginSettings(id)
			if err == nil {
				t.Errorf("ReadPluginSettings(%q): expected error, got nil", id)
			}
		})
	}

	// Empty pluginID should also be rejected
	err := s.WritePluginSettings("", map[string]interface{}{})
	if err == nil {
		t.Error("WritePluginSettings(\"\"): expected error, got nil")
	}
}

// ─── Atomic write tests ──────────────────────────────────────

func TestAtomicWrite(t *testing.T) {
	s, dir := newTestStorage(t)

	data := map[string]interface{}{
		"key": "value",
		"n":   float64(123),
	}

	if err := s.WritePluginSettings("atomic-plugin", data); err != nil {
		t.Fatalf("WritePluginSettings: %v", err)
	}

	// Verify no .tmp files remain in the settings directory
	settingsDir := filepath.Join(dir, "VerstakVault", ".verstak", "plugin-settings", "atomic-plugin")
	entries, err := os.ReadDir(settingsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" || (len(e.Name()) > 4 && e.Name()[:4] == ".tmp") {
			t.Errorf("leftover temp file found: %s", e.Name())
		}
	}
}
