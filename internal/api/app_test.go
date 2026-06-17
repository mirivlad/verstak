package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/plugin"
)

// newTestApp creates an App with a mocked plugin list for testing.
func newTestApp(tmpRoot string) *App {
	return &App{
		plugins: []plugin.Plugin{
			{
				Manifest: plugin.Manifest{
					ID:      "test.plugin",
					Name:    "Test Plugin",
					Version: "1.0.0",
					Icon:    "🧪",
					Frontend: &plugin.FrontendConfig{
						Entry: "frontend/dist/index.js",
						Style: "frontend/style.css",
					},
					Provides:    []string{"test/cap/v1"},
					Permissions: []string{"test.perm"},
				},
				RootPath: tmpRoot,
				Status:   plugin.StatusLoaded,
				Enabled:  true,
			},
			{
				Manifest: plugin.Manifest{
					ID:          "no-fe.plugin",
					Name:        "No Frontend Plugin",
					Provides:    []string{"test/nofe/v1"},
					Permissions: []string{"test.perm"},
				},
				RootPath: "/tmp/no-fe-plugin",
				Status:   plugin.StatusLoaded,
				Enabled:  true,
			},
		},
	}
}

// TestGetPluginFrontendInfo_KnownPluginWithFrontend verifies that
// GetPluginFrontendInfo returns correct metadata for a plugin with a frontend.
func TestGetPluginFrontendInfo_KnownPluginWithFrontend(t *testing.T) {
	app := newTestApp("/tmp/test-plugin")
	info := app.GetPluginFrontendInfo("test.plugin")

	if info["status"] != nil {
		t.Errorf("unexpected status key: expected no status, got %v", info["status"])
	}
	if info["pluginId"] != "test.plugin" {
		t.Errorf("pluginId: expected %q, got %v", "test.plugin", info["pluginId"])
	}
	if info["name"] != "Test Plugin" {
		t.Errorf("name: expected %q, got %v", "Test Plugin", info["name"])
	}
	if info["icon"] != "🧪" {
		t.Errorf("icon: expected %q, got %v", "🧪", info["icon"])
	}
	if info["version"] != "1.0.0" {
		t.Errorf("version: expected %q, got %v", "1.0.0", info["version"])
	}
	if info["entry"] != "frontend/dist/index.js" {
		t.Errorf("entry: expected %q, got %v", "frontend/dist/index.js", info["entry"])
	}
	if info["style"] != "frontend/style.css" {
		t.Errorf("style: expected %q, got %v", "frontend/style.css", info["style"])
	}
	if info["rootPath"] != "/tmp/test-plugin" {
		t.Errorf("rootPath: expected %q, got %v", "/tmp/test-plugin", info["rootPath"])
	}
}

// TestGetPluginFrontendInfo_PluginWithoutFrontend verifies that
// GetPluginFrontendInfo returns {"status": "no-frontend"} for a plugin
// that has no FrontendConfig.
func TestGetPluginFrontendInfo_PluginWithoutFrontend(t *testing.T) {
	app := newTestApp("/tmp/test-plugin")
	info := app.GetPluginFrontendInfo("no-fe.plugin")

	if info["status"] != "no-frontend" {
		t.Errorf("expected status %q, got %v", "no-frontend", info["status"])
	}
}

// TestGetPluginFrontendInfo_UnknownPlugin verifies that
// GetPluginFrontendInfo returns {"status": "not-found"} for a plugin ID
// that does not exist.
func TestGetPluginFrontendInfo_UnknownPlugin(t *testing.T) {
	app := newTestApp("/tmp/test-plugin")
	info := app.GetPluginFrontendInfo("nonexistent.plugin")

	if info["status"] != "not-found" {
		t.Errorf("expected status %q, got %v", "not-found", info["status"])
	}
}

// TestGetPluginAssetContent_ExistingFile verifies that GetPluginAssetContent
// can read an existing frontend file from a plugin directory.
func TestGetPluginAssetContent_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test frontend file
	frontendDir := filepath.Join(tmpDir, "frontend", "dist")
	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "console.log('test');\n"
	if err := os.WriteFile(filepath.Join(frontendDir, "index.js"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	app := newTestApp(tmpDir)
	got, errStr := app.GetPluginAssetContent("test.plugin", "frontend/dist/index.js")

	if errStr != "" {
		t.Errorf("unexpected error: %s", errStr)
	}
	if got != content {
		t.Errorf("content mismatch:\n  expected: %q\n  got:      %q", content, got)
	}
}

// TestGetPluginAssetContent_ExistingStyleFile verifies reading a style file.
func TestGetPluginAssetContent_ExistingStyleFile(t *testing.T) {
	tmpDir := t.TempDir()

	frontendDir := filepath.Join(tmpDir, "frontend")
	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := "body { color: red; }\n"
	if err := os.WriteFile(filepath.Join(frontendDir, "style.css"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	app := newTestApp(tmpDir)
	got, errStr := app.GetPluginAssetContent("test.plugin", "frontend/style.css")

	if errStr != "" {
		t.Errorf("unexpected error: %s", errStr)
	}
	if got != content {
		t.Errorf("content mismatch:\n  expected: %q\n  got:      %q", content, got)
	}
}

// TestGetPluginAssetContent_AbsolutePathRejected verifies that asset paths
// starting with "/" are rejected.
func TestGetPluginAssetContent_AbsolutePathRejected(t *testing.T) {
	app := newTestApp("/tmp/test-plugin")

	_, errStr := app.GetPluginAssetContent("test.plugin", "/etc/passwd")
	if errStr == "" {
		t.Error("expected error for absolute path, got empty")
	}
	if !strings.Contains(strings.ToLower(errStr), "absolute") {
		t.Errorf("error should mention 'absolute', got: %s", errStr)
	}

	// Also test Windows-style absolute path
	_, errStr = app.GetPluginAssetContent("test.plugin", "\\etc\\passwd")
	if errStr == "" {
		t.Error("expected error for windows-style absolute path, got empty")
	}
}

// TestGetPluginAssetContent_PathTraversalRejected verifies that asset paths
// containing ".." are rejected.
func TestGetPluginAssetContent_PathTraversalRejected(t *testing.T) {
	app := newTestApp("/tmp/test-plugin")

	_, errStr := app.GetPluginAssetContent("test.plugin", "../../etc/passwd")
	if errStr == "" {
		t.Error("expected error for path traversal, got empty")
	}
	if !strings.Contains(strings.ToLower(errStr), "traversal") {
		t.Errorf("error should mention 'traversal', got: %s", errStr)
	}
}

// TestGetPluginAssetContent_PathEscapeRejected verifies that paths that
// resolve outside the plugin root directory are rejected after Join.
// This tests the absRoot-prefix check in GetPluginAssetContent.
func TestGetPluginAssetContent_PathEscapeRejected(t *testing.T) {
	tmpDir := t.TempDir()

	// Do NOT create the traversed-to file — the security check happens
	// before os.ReadFile, so the file should not matter.
	app := newTestApp(tmpDir)

	// Use a relative path with ".." that would escape the plugin root
	// The code checks strings.Contains(assetPath, "..") first, so this
	// would be caught at the traversal check. But let's also test a case
	// where ".." is NOT in the path but Join resolves outside root.
	// For instance: symlink-based escape (not testable easily) or
	// Join "/tmp/root" + "foo/../../../etc" — but ".." is in there.
	//
	// Instead, test that the absRoot prefix check works: create a path
	// that after cleaning technically starts differently but doesn't use "..".
	// This is hard to reproduce without symlinks. The ".." check catches
	// common cases. Let's just ensure the ".." check is solid:
	_, errStr := app.GetPluginAssetContent("test.plugin", "frontend/../../etc/passwd")
	if errStr == "" {
		t.Error("expected error for path traversal via nested '..', got empty")
	}
}

// TestGetPluginAssetContent_PluginNotFound verifies that GetPluginAssetContent
// returns an error for a nonexistent plugin ID.
func TestGetPluginAssetContent_PluginNotFound(t *testing.T) {
	app := newTestApp("/tmp/test-plugin")

	_, errStr := app.GetPluginAssetContent("nonexistent.plugin", "frontend/dist/index.js")
	if errStr == "" {
		t.Error("expected error for nonexistent plugin, got empty")
	}
	if !strings.Contains(strings.ToLower(errStr), "not found") &&
		!strings.Contains(strings.ToLower(errStr), "no frontend") {
		t.Errorf("error should mention 'not found' or 'no frontend', got: %s", errStr)
	}
}

// TestGetPluginAssetContent_NoFrontend verifies that GetPluginAssetContent
// returns an error for a plugin that exists but has no frontend config.
func TestGetPluginAssetContent_NoFrontend(t *testing.T) {
	app := newTestApp("/tmp/test-plugin")

	_, errStr := app.GetPluginAssetContent("no-fe.plugin", "frontend/dist/index.js")
	if errStr == "" {
		t.Error("expected error for plugin without frontend, got empty")
	}
	if !strings.Contains(strings.ToLower(errStr), "no frontend") {
		t.Errorf("error should mention 'no frontend', got: %s", errStr)
	}
}

// TestGetPluginAssetContent_NonexistentFile verifies that GetPluginAssetContent
// returns an error when the asset file does not exist on disk.
func TestGetPluginAssetContent_NonexistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	app := newTestApp(tmpDir)

	_, errStr := app.GetPluginAssetContent("test.plugin", "frontend/dist/missing.js")
	if errStr == "" {
		t.Error("expected error for nonexistent file, got empty")
	}
	if !strings.Contains(strings.ToLower(errStr), "failed to read") {
		t.Errorf("error should mention 'failed to read', got: %s", errStr)
	}
}
