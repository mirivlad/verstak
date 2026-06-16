package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTempPlugin creates a minimal valid plugin directory for testing.
func createTempPlugin(t *testing.T, dir, id, name string) string {
	t.Helper()
	pluginDir := filepath.Join(dir, id)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"schemaVersion": 1,
		"id": "` + id + `",
		"name": "` + name + `",
		"version": "1.0.0",
		"apiVersion": "1.0",
		"provides": ["` + id + `.cap1"],
		"permissions": ["vault.read"]
	}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	return pluginDir
}

// TestDiscoverPlugins_EmptyDir tests discovery on a directory that does not exist.
// It should return empty results, not hang or error out.
func TestDiscoverPlugins_EmptyDir(t *testing.T) {
	plugins, errs := DiscoverPlugins([]string{"/tmp/nonexistent-plugin-dir-12345"})
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

// TestDiscoverPlugins_MissingDir tests discovery where the directory does not exist.
// It should not be treated as an error — just skip.
func TestDiscoverPlugins_MissingDir(t *testing.T) {
	plugins, errs := DiscoverPlugins([]string{"/tmp/missing-dir-99999"})
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors for missing dir, got %d", len(errs))
	}
}

// TestDiscoverPlugins_ValidPlugin tests discovery of a single valid plugin.
func TestDiscoverPlugins_ValidPlugin(t *testing.T) {
	dir := t.TempDir()
	createTempPlugin(t, dir, "test.plugin", "Test Plugin")

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Manifest.ID != "test.plugin" {
		t.Errorf("expected id 'test.plugin', got %q", plugins[0].Manifest.ID)
	}
	if !plugins[0].Enabled {
		t.Error("expected plugin to be enabled by default")
	}
	if len(errs) > 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

// TestDiscoverPlugins_NoManifest ensures subdirs without plugin.json are skipped.
func TestDiscoverPlugins_NoManifest(t *testing.T) {
	dir := t.TempDir()
	noManifestDir := filepath.Join(dir, "no-manifest")
	if err := os.MkdirAll(noManifestDir, 0755); err != nil {
		t.Fatal(err)
	}

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins (no manifest), got %d", len(plugins))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

// TestDiscoverPlugins_BrokenJSON ensures a corrupted manifest is reported as error
// but does not crash or hang discovery.
func TestDiscoverPlugins_BrokenJSON(t *testing.T) {
	dir := t.TempDir()
	pluginDir := filepath.Join(dir, "broken")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte("{invalid json}"), 0644); err != nil {
		t.Fatal(err)
	}

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins (broken manifest), got %d", len(plugins))
	}
	if len(errs) == 0 {
		t.Error("expected at least 1 error for broken manifest, got 0")
	}
}

// TestDiscoverPlugins_DuplicateID ensures duplicate plugin IDs are detected.
func TestDiscoverPlugins_DuplicateID(t *testing.T) {
	dir := t.TempDir()
	createTempPlugin(t, dir, "dup-one", "First")

	// Create a second plugin with the same ID (duplicate)
	pluginDir2 := filepath.Join(dir, "dup-two")
	if err := os.MkdirAll(pluginDir2, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"schemaVersion": 1,
		"id": "dup-one",
		"name": "Second",
		"version": "1.0.0",
		"apiVersion": "1.0",
		"provides": ["dup-one.cap1"],
		"permissions": ["vault.read"]
	}`
	if err := os.WriteFile(filepath.Join(pluginDir2, "plugin.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	plugins, errs := DiscoverPlugins([]string{dir})
	if len(plugins) != 1 {
		t.Errorf("expected 1 plugin (deduped), got %d", len(plugins))
	}
	hasDupError := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "duplicate") {
			hasDupError = true
		}
	}
	if !hasDupError {
		t.Error("expected duplicate plugin ID error")
	}
}

// TestDiscoverPlugins_MultipleDirs ensures discovery scans multiple directories.
func TestDiscoverPlugins_MultipleDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	createTempPlugin(t, dir1, "alpha.plugin", "Alpha")
	createTempPlugin(t, dir2, "beta.plugin", "Beta")

	plugins, errs := DiscoverPlugins([]string{dir1, dir2})
	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}
	if len(errs) > 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

// TestDiscoverPlugins_AllowsNonexistentDirs ensures that a mix of valid and
// nonexistent directories doesn't cause issues.
func TestDiscoverPlugins_AllowsNonexistentDirs(t *testing.T) {
	dir := t.TempDir()
	createTempPlugin(t, dir, "survivor.plugin", "Survivor")

	plugins, errs := DiscoverPlugins([]string{
		"/tmp/missing-aaa-88888",
		dir,
		"/tmp/missing-bbb-99999",
	})
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Manifest.ID != "survivor.plugin" {
		t.Errorf("expected 'survivor.plugin', got %q", plugins[0].Manifest.ID)
	}
	if len(errs) > 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(errs), errs)
	}
}

// TestFormatDiscoverySummary on empty list.
func TestFormatDiscoverySummary_Empty(t *testing.T) {
	s := FormatDiscoverySummary(nil)
	if s != "no plugins found" {
		t.Errorf("expected 'no plugins found', got %q", s)
	}
}

// TestFormatDiscoverySummary on populated list.
func TestFormatDiscoverySummary_Plugins(t *testing.T) {
	plugins := []Plugin{
		{
			Manifest: Manifest{ID: "test.one", Version: "1.0.0"},
		},
		{
			Manifest: Manifest{ID: "test.two", Version: "2.0.0"},
		},
	}
	s := FormatDiscoverySummary(plugins)
	if s != "2 plugin(s): test.one@1.0.0, test.two@2.0.0" {
		t.Errorf("unexpected summary: %q", s)
	}
}
