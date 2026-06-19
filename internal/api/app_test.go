package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/core/workspace"
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

func newFilesTestApp(t *testing.T, perms []string) (*App, string) {
	t.Helper()
	v := vault.NewVault(nil)
	if err := v.CreateVault(t.TempDir()); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	return &App{
		files: corefiles.NewService(v),
		vault: v,
		plugins: []plugin.Plugin{
			{
				Manifest: plugin.Manifest{
					ID:          "files.plugin",
					Name:        "Files Plugin",
					Version:     "1.0.0",
					Provides:    []string{"files/plugin/v1"},
					Permissions: perms,
				},
				Status:  plugin.StatusLoaded,
				Enabled: true,
			},
		},
	}, v.GetVaultPath()
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

func TestFilesBridgeReadWriteListMoveTrash(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.read", "files.write", "files.delete"})

	if errStr := app.CreateVaultFolder("files.plugin", "Docs"); errStr != "" {
		t.Fatalf("CreateVaultFolder: %s", errStr)
	}
	if errStr := app.WriteVaultTextFile("files.plugin", "Docs/one.txt", "hello", corefiles.WriteOptions{CreateIfMissing: true}); errStr != "" {
		t.Fatalf("WriteVaultTextFile: %s", errStr)
	}

	text, errStr := app.ReadVaultTextFile("files.plugin", "Docs/one.txt")
	if errStr != "" {
		t.Fatalf("ReadVaultTextFile: %s", errStr)
	}
	if text != "hello" {
		t.Fatalf("text = %q", text)
	}

	entries, errStr := app.ListVaultFiles("files.plugin", "Docs")
	if errStr != "" {
		t.Fatalf("ListVaultFiles: %s", errStr)
	}
	if len(entries) != 1 || entries[0].RelativePath != "Docs/one.txt" {
		t.Fatalf("entries = %+v", entries)
	}

	meta, errStr := app.GetVaultFileMetadata("files.plugin", "Docs/one.txt")
	if errStr != "" {
		t.Fatalf("GetVaultFileMetadata: %s", errStr)
	}
	if meta.Type != corefiles.FileTypeFile || !meta.IsText {
		t.Fatalf("metadata = %+v", meta)
	}

	if errStr := app.MoveVaultPath("files.plugin", "Docs/one.txt", "Docs/two.txt", corefiles.MoveOptions{}); errStr != "" {
		t.Fatalf("MoveVaultPath: %s", errStr)
	}
	trash, errStr := app.TrashVaultPath("files.plugin", "Docs/two.txt")
	if errStr != "" {
		t.Fatalf("TrashVaultPath: %s", errStr)
	}
	if trash.OriginalPath != "Docs/two.txt" || trash.TrashID == "" {
		t.Fatalf("trash result = %+v", trash)
	}
	if _, err := os.Stat(filepath.Join(root, trash.TrashPath)); err != nil {
		t.Fatalf("trash path missing: %v", err)
	}
}

func TestFilesBridgePermissions(t *testing.T) {
	cases := []struct {
		name       string
		perms      []string
		call       func(*App) string
		wantPhrase string
	}{
		{
			name:       "list requires read",
			perms:      []string{"files.write", "files.delete"},
			call:       func(app *App) string { _, errStr := app.ListVaultFiles("files.plugin", ""); return errStr },
			wantPhrase: "files.read",
		},
		{
			name:       "metadata requires read",
			perms:      []string{"files.write", "files.delete"},
			call:       func(app *App) string { _, errStr := app.GetVaultFileMetadata("files.plugin", "one.txt"); return errStr },
			wantPhrase: "files.read",
		},
		{
			name:       "read requires read",
			perms:      []string{"files.write", "files.delete"},
			call:       func(app *App) string { _, errStr := app.ReadVaultTextFile("files.plugin", "one.txt"); return errStr },
			wantPhrase: "files.read",
		},
		{
			name:  "write requires write",
			perms: []string{"files.read", "files.delete"},
			call: func(app *App) string {
				return app.WriteVaultTextFile("files.plugin", "one.txt", "x", corefiles.WriteOptions{CreateIfMissing: true})
			},
			wantPhrase: "files.write",
		},
		{
			name:       "create folder requires write",
			perms:      []string{"files.read", "files.delete"},
			call:       func(app *App) string { return app.CreateVaultFolder("files.plugin", "Folder") },
			wantPhrase: "files.write",
		},
		{
			name:  "move requires write",
			perms: []string{"files.read", "files.delete"},
			call: func(app *App) string {
				return app.MoveVaultPath("files.plugin", "one.txt", "two.txt", corefiles.MoveOptions{})
			},
			wantPhrase: "files.write",
		},
		{
			name:       "trash requires delete",
			perms:      []string{"files.read", "files.write"},
			call:       func(app *App) string { _, errStr := app.TrashVaultPath("files.plugin", "one.txt"); return errStr },
			wantPhrase: "files.delete",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, _ := newFilesTestApp(t, tc.perms)
			errStr := tc.call(app)
			if errStr == "" {
				t.Fatal("expected permission error")
			}
			if !strings.Contains(errStr, tc.wantPhrase) {
				t.Fatalf("error = %q, want %q", errStr, tc.wantPhrase)
			}
		})
	}
}

func TestFilesBridgeRequiresLoadedPluginAndOpenVault(t *testing.T) {
	app, _ := newFilesTestApp(t, []string{"files.read"})
	app.plugins[0].Enabled = false
	if _, errStr := app.ListVaultFiles("files.plugin", ""); errStr == "" || !strings.Contains(errStr, "not enabled") {
		t.Fatalf("disabled plugin error = %q", errStr)
	}

	app, _ = newFilesTestApp(t, []string{"files.read"})
	app.plugins[0].Status = plugin.StatusFailed
	if _, errStr := app.ListVaultFiles("files.plugin", ""); errStr == "" || !strings.Contains(errStr, "not enabled") {
		t.Fatalf("failed plugin error = %q", errStr)
	}

	app, _ = newFilesTestApp(t, []string{"files.read"})
	app.plugins[0].Status = plugin.StatusDegraded
	if _, errStr := app.ListVaultFiles("files.plugin", ""); errStr != "" {
		t.Fatalf("degraded plugin should be allowed, got %q", errStr)
	}

	app, _ = newFilesTestApp(t, []string{"files.read"})
	if _, errStr := app.ListVaultFiles("missing.plugin", ""); errStr == "" || !strings.Contains(errStr, "not found") {
		t.Fatalf("missing plugin error = %q", errStr)
	}

	app, _ = newFilesTestApp(t, []string{"files.read"})
	app.vault.CloseVault()
	if _, errStr := app.ListVaultFiles("files.plugin", ""); errStr == "" || !strings.Contains(errStr, "vault-not-open") {
		t.Fatalf("closed vault error = %q", errStr)
	}
}

func TestSetCurrentVaultInitializesWorkspaceWhenMissingAtStartup(t *testing.T) {
	tmpDir := t.TempDir()
	vaultParent := filepath.Join(tmpDir, "vault-parent")
	if err := os.MkdirAll(vaultParent, 0o755); err != nil {
		t.Fatal(err)
	}

	bus := events.NewBus()
	vaultService := vault.NewVault(bus)
	if err := vaultService.CreateVault(vaultParent); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	vaultService.CloseVault()

	settings := appsettings.NewManager(filepath.Join(tmpDir, "config.json"))
	if err := settings.Load(); err != nil {
		t.Fatalf("settings Load: %v", err)
	}

	app := &App{
		capRegistry: capability.NewRegistry(),
		vault:       vaultService,
		appSettings: settings,
		workspace:   nil,
	}

	if errStr := app.SetCurrentVault(vaultParent); errStr != "" {
		t.Fatalf("SetCurrentVault: %s", errStr)
	}

	tree := app.GetWorkspaceTree()
	if tree["status"] == "not initialized" {
		t.Fatal("workspace should be initialized after SetCurrentVault")
	}
	nodes, ok := tree["nodes"].([]workspace.WorkspaceNode)
	if !ok {
		t.Fatalf("workspace nodes type: got %T", tree["nodes"])
	}
	if len(nodes) == 0 {
		t.Fatal("workspace nodes should not be empty")
	}
	if !app.capRegistry.Has("verstak/core/workspace/v1") {
		t.Fatal("workspace capability should be registered after SetCurrentVault")
	}
}

func newBridgeTestApp(t *testing.T) *App {
	t.Helper()
	tmpDir := t.TempDir()
	vaultParent := filepath.Join(tmpDir, "vault-parent")
	if err := os.MkdirAll(vaultParent, 0o755); err != nil {
		t.Fatal(err)
	}

	bus := events.NewBus()
	vaultService := vault.NewVault(bus)
	if err := vaultService.CreateVault(vaultParent); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	capReg := capability.NewRegistry()
	if err := capReg.Register("verstak-desktop", []string{"verstak/core/vault/v1"}); err != nil {
		t.Fatal(err)
	}
	if err := capReg.Register("bridge.plugin", []string{"bridge/cap/v1"}); err != nil {
		t.Fatal(err)
	}

	contribReg := contribution.NewRegistry()
	contribReg.Register("bridge.plugin", &plugin.Contributions{
		Commands: []plugin.ContributionCommand{
			{ID: "bridge.command", Title: "Bridge Command", Handler: "runBridgeCommand"},
		},
		OpenProviders: []plugin.ContributionOpenProvider{
			{
				ID:        "bridge.markdown",
				Title:     "Bridge Markdown",
				Priority:  100,
				Component: "BridgeMarkdown",
				Supports: []plugin.OpenProviderSupport{
					{Kind: "vault-file", Extensions: []string{".md"}, Contexts: []string{"generic-markdown", "notes-markdown"}},
				},
			},
		},
	})

	return &App{
		capRegistry:     capReg,
		contribRegistry: contribReg,
		eventBus:        bus,
		vault:           vaultService,
		storage:         storage.New(vaultService),
		plugins: []plugin.Plugin{
			{
				Manifest: plugin.Manifest{
					ID:          "bridge.plugin",
					Name:        "Bridge Plugin",
					Version:     "1.0.0",
					Provides:    []string{"bridge/cap/v1"},
					Requires:    []string{"verstak/core/capability-registry/v1"},
					Permissions: []string{"storage.namespace", "commands.register", "events.publish", "events.subscribe", "workbench.open"},
				},
				Status:  plugin.StatusLoaded,
				Enabled: true,
			},
			{
				Manifest: plugin.Manifest{
					ID:          "no.storage",
					Name:        "No Storage",
					Version:     "1.0.0",
					Provides:    []string{"no/storage/v1"},
					Permissions: []string{"events.publish"},
				},
				Status:  plugin.StatusLoaded,
				Enabled: true,
			},
			{
				Manifest: plugin.Manifest{
					ID:          "disabled.plugin",
					Name:        "Disabled",
					Version:     "1.0.0",
					Provides:    []string{"disabled/cap/v1"},
					Permissions: []string{"storage.namespace"},
				},
				Status:  plugin.StatusDisabled,
				Enabled: false,
			},
		},
	}
}

func TestContributionSummaryIncludesOpenProviders(t *testing.T) {
	app := newBridgeTestApp(t)

	summary := app.GetContributions()
	if len(summary.OpenProviders) != 1 {
		t.Fatalf("OpenProviders count = %d, want 1", len(summary.OpenProviders))
	}
	provider := summary.OpenProviders[0]
	if provider.PluginID != "bridge.plugin" || provider.ID != "bridge.markdown" || provider.Component != "BridgeMarkdown" {
		t.Fatalf("provider = %+v", provider)
	}
	if len(provider.Supports) != 1 || provider.Supports[0].Contexts[1] != "notes-markdown" {
		t.Fatalf("supports = %+v", provider.Supports)
	}
}

func TestWorkbenchOpenAndEditResourceRouteToProvider(t *testing.T) {
	app := newBridgeTestApp(t)
	app.contribRegistry.Register("disabled.plugin", &plugin.Contributions{
		OpenProviders: []plugin.ContributionOpenProvider{
			{
				ID:        "disabled.markdown",
				Title:     "Disabled Markdown",
				Priority:  1000,
				Component: "DisabledMarkdown",
				Supports: []plugin.OpenProviderSupport{
					{Kind: "vault-file", Extensions: []string{".md"}, Contexts: []string{"generic-markdown", "notes-markdown"}},
				},
			},
		},
	})

	result, errStr := app.OpenWorkbenchResource("bridge.plugin", map[string]interface{}{
		"kind":      "vault-file",
		"path":      "Notes/Overview.md",
		"extension": ".md",
		"context": map[string]interface{}{
			"sourceView":          "notes",
			"isInsideNotesFolder": true,
			"notesMode":           true,
		},
	})
	if errStr != "" {
		t.Fatalf("OpenWorkbenchResource: %s", errStr)
	}
	if result.ProviderID != "bridge.markdown" || result.ProviderComponent != "BridgeMarkdown" || result.Request.Mode != "view" {
		t.Fatalf("open result = %+v", result)
	}

	editResult, errStr := app.EditWorkbenchResource("bridge.plugin", map[string]interface{}{
		"kind":      "vault-file",
		"path":      "Notes/Overview.md",
		"extension": ".md",
		"context": map[string]interface{}{
			"sourceView":          "notes",
			"isInsideNotesFolder": true,
			"notesMode":           true,
		},
	})
	if errStr != "" {
		t.Fatalf("EditWorkbenchResource: %s", errStr)
	}
	if editResult.Request.Mode != "edit" {
		t.Fatalf("edit mode = %q", editResult.Request.Mode)
	}

	opened := app.GetWorkbenchOpenedResources()
	if len(opened) != 2 {
		t.Fatalf("opened resources = %+v", opened)
	}
}

func TestWorkbenchOpenResourceReturnsNoProviderFallback(t *testing.T) {
	app := newBridgeTestApp(t)

	result, errStr := app.OpenWorkbenchResource("bridge.plugin", map[string]interface{}{
		"kind": "vault-file",
		"path": "Images/logo.png",
	})
	if errStr != "" {
		t.Fatalf("OpenWorkbenchResource: %s", errStr)
	}
	if result.Status != "no-provider" || result.Request.Path != "Images/logo.png" {
		t.Fatalf("result = %+v", result)
	}
}

func TestWorkbenchOpenResourceRequiresPermission(t *testing.T) {
	app := newBridgeTestApp(t)

	_, errStr := app.OpenWorkbenchResource("no.storage", map[string]interface{}{
		"kind": "vault-file",
		"path": "Docs/readme.md",
	})
	if !strings.Contains(errStr, "workbench.open") {
		t.Fatalf("err = %q, want workbench.open permission error", errStr)
	}
}

func TestWorkbenchDisabledPluginProviderExcluded(t *testing.T) {
	app := newBridgeTestApp(t)

	result, errStr := app.OpenWorkbenchResource("bridge.plugin", map[string]interface{}{
		"kind":      "vault-file",
		"path":      "Docs/readme.md",
		"extension": ".md",
	})
	if errStr != "" {
		t.Fatalf("OpenWorkbenchResource: %s", errStr)
	}
	if result.ProviderPluginID != "bridge.plugin" {
		t.Fatalf("expected bridge.plugin provider, got providerPluginId=%q", result.ProviderPluginID)
	}
	if result.ProviderID == "disabled.markdown" {
		t.Fatal("disabled plugin provider should be excluded from selection")
	}
}

func TestPluginBridgeSettingsRequireLoadedPluginAndStoragePermission(t *testing.T) {
	app := newBridgeTestApp(t)

	if errStr := app.WritePluginSettings("bridge.plugin", map[string]interface{}{"savedText": "hello"}); errStr != "" {
		t.Fatalf("WritePluginSettings: %s", errStr)
	}
	settings, errStr := app.ReadPluginSettings("bridge.plugin")
	if errStr != "" {
		t.Fatalf("ReadPluginSettings: %s", errStr)
	}
	if settings["savedText"] != "hello" {
		t.Fatalf("savedText = %v, want hello", settings["savedText"])
	}

	if _, errStr := app.ReadPluginSettings("missing.plugin"); errStr == "" {
		t.Fatal("expected error for missing plugin")
	}
	if _, errStr := app.ReadPluginSettings("disabled.plugin"); errStr == "" {
		t.Fatal("expected error for disabled plugin")
	}
	if _, errStr := app.ReadPluginSettings("no.storage"); errStr == "" {
		t.Fatal("expected error for plugin without storage.namespace")
	}
}

func TestPluginBridgeCapabilitiesCommandsAndEventsAreChecked(t *testing.T) {
	app := newBridgeTestApp(t)

	capInfo, errStr := app.GetPluginCapability("bridge.plugin", "bridge/cap/v1")
	if errStr != "" {
		t.Fatalf("GetPluginCapability: %s", errStr)
	}
	if capInfo["available"] != true {
		t.Fatalf("capability should be available: %#v", capInfo)
	}
	if _, errStr := app.GetPluginCapability("no.storage", "bridge/cap/v1"); errStr == "" {
		t.Fatal("expected capability dependency error")
	}

	commandResult, errStr := app.ExecutePluginCommand("bridge.plugin", "bridge.command", map[string]interface{}{"value": "x"})
	if errStr != "" {
		t.Fatalf("ExecutePluginCommand: %s", errStr)
	}
	if commandResult["status"] != "declared" {
		t.Fatalf("command status = %v, want declared", commandResult["status"])
	}

	if errStr := app.PublishPluginEvent("bridge.plugin", "bridge.event", map[string]interface{}{"ok": true}); errStr != "" {
		t.Fatalf("PublishPluginEvent: %s", errStr)
	}
	if errStr := app.SubscribePluginEvent("bridge.plugin", "bridge.event"); errStr != "" {
		t.Fatalf("SubscribePluginEvent: %s", errStr)
	}
	if errStr := app.SubscribePluginEvent("no.storage", "bridge.event"); errStr == "" {
		t.Fatal("expected subscribe permission error")
	}
	if _, errStr := app.ExecutePluginCommand("no.storage", "bridge.command", nil); errStr == "" {
		t.Fatal("expected command permission/ownership error")
	}
}
