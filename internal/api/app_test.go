package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/verstak/verstak-desktop/internal/core/appsettings"
	"github.com/verstak/verstak-desktop/internal/core/browserreceiver"
	"github.com/verstak/verstak-desktop/internal/core/capability"
	"github.com/verstak/verstak-desktop/internal/core/contribution"
	"github.com/verstak/verstak-desktop/internal/core/events"
	corefiles "github.com/verstak/verstak-desktop/internal/core/files"
	"github.com/verstak/verstak-desktop/internal/core/plugin"
	"github.com/verstak/verstak-desktop/internal/core/storage"
	syncsvc "github.com/verstak/verstak-desktop/internal/core/sync"
	"github.com/verstak/verstak-desktop/internal/core/vault"
	"github.com/verstak/verstak-desktop/internal/core/workspace"
)

func newLocalHTTPTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen local test server: %v", err)
	}

	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	return server
}

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

type testExternalOpenService struct {
	open func(path string) error
}

func newTestExternalOpenService(open func(path string) error) *testExternalOpenService {
	return &testExternalOpenService{open: open}
}

func (s *testExternalOpenService) OpenPath(path string) error {
	return s.open(path)
}

func (s *testExternalOpenService) ShowInFolder(path string, _ bool) error {
	return s.open(path)
}

func newSyncFilesTestApp(t *testing.T, perms []string, deviceID string) (*App, string) {
	t.Helper()
	app, root := newFilesTestApp(t, perms)
	app.syncSvc = syncsvc.NewService(root, deviceID)
	app.appSettings = appsettings.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err := app.appSettings.Load(); err != nil {
		t.Fatalf("settings Load: %v", err)
	}
	cfg := app.appSettings.Get()
	cfg.Sync.DeviceID = deviceID
	if err := app.appSettings.UpdateSync(cfg.Sync); err != nil {
		t.Fatalf("settings UpdateSync: %v", err)
	}
	return app, root
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

func TestGetPluginLocalizationReadsDeclaredCatalog(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "locales"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "locales", "ru.json"), []byte(`{"greeting":"Привет","count":"Всего: {count}"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(root)
	app.plugins[0].Manifest.Localization = &plugin.LocalizationConfig{
		DefaultLocale: "en",
		Locales: map[string]string{
			"en": "locales/en.json",
			"ru": "locales/ru.json",
		},
	}

	catalog, errStr := app.GetPluginLocalization("test.plugin", "ru")
	if errStr != "" {
		t.Fatalf("GetPluginLocalization: %s", errStr)
	}
	if catalog["greeting"] != "Привет" || catalog["count"] != "Всего: {count}" {
		t.Fatalf("catalog = %#v", catalog)
	}
}

func TestGetPluginLocalizationRejectsInvalidCatalogs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "locales"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "locales", "bad.json"), []byte(`{"valid":"text","invalid":42}`), 0o644); err != nil {
		t.Fatal(err)
	}
	app := newTestApp(root)

	tests := []struct {
		name   string
		config *plugin.LocalizationConfig
		locale string
		want   string
	}{
		{"missing declaration", nil, "ru", "does not declare"},
		{"unknown locale", &plugin.LocalizationConfig{DefaultLocale: "en", Locales: map[string]string{"en": "locales/en.json"}}, "ru", "not declared"},
		{"traversal", &plugin.LocalizationConfig{DefaultLocale: "en", Locales: map[string]string{"en": "../outside.json"}}, "en", "traversal"},
		{"backslash", &plugin.LocalizationConfig{DefaultLocale: "en", Locales: map[string]string{"en": `locales\bad.json`}}, "en", "backslash"},
		{"non string value", &plugin.LocalizationConfig{DefaultLocale: "en", Locales: map[string]string{"en": "locales/bad.json"}}, "en", "parse"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			app.plugins[0].Manifest.Localization = tc.config
			_, errStr := app.GetPluginLocalization("test.plugin", tc.locale)
			if !strings.Contains(errStr, tc.want) {
				t.Fatalf("error = %q, want substring %q", errStr, tc.want)
			}
		})
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

	if err := os.WriteFile(filepath.Join(root, "Docs", "image.png"), []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}
	bytesResult, errStr := app.ReadVaultFileBytes("files.plugin", "Docs/image.png")
	if errStr != "" {
		t.Fatalf("ReadVaultFileBytes: %s", errStr)
	}
	if bytesResult.RelativePath != "Docs/image.png" || bytesResult.MimeHint != "image/png" || bytesResult.DataBase64 != "iVBORw==" {
		t.Fatalf("bytes result = %+v", bytesResult)
	}
	if errStr := app.WriteVaultFileBytes("files.plugin", "Docs/from-api.bin", "AQID", corefiles.WriteOptions{CreateIfMissing: true}); errStr != "" {
		t.Fatalf("WriteVaultFileBytes: %s", errStr)
	}
	writtenBytes, errStr := app.ReadVaultFileBytes("files.plugin", "Docs/from-api.bin")
	if errStr != "" {
		t.Fatalf("ReadVaultFileBytes written: %s", errStr)
	}
	if writtenBytes.DataBase64 != "AQID" || writtenBytes.Size != 3 {
		t.Fatalf("written bytes result = %+v", writtenBytes)
	}

	entries, errStr := app.ListVaultFiles("files.plugin", "Docs")
	if errStr != "" {
		t.Fatalf("ListVaultFiles: %s", errStr)
	}
	hasOne := false
	for _, entry := range entries {
		if entry.RelativePath == "Docs/one.txt" {
			hasOne = true
			break
		}
	}
	if !hasOne {
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
	trashEntries, errStr := app.ListVaultTrash("files.plugin")
	if errStr != "" {
		t.Fatalf("ListVaultTrash: %s", errStr)
	}
	if len(trashEntries) != 1 || trashEntries[0].OriginalPath != "Docs/two.txt" || trashEntries[0].TrashID != trash.TrashID {
		t.Fatalf("trash entries = %+v, want Docs/two.txt", trashEntries)
	}

	restored, errStr := app.RestoreVaultTrash("files.plugin", trash.TrashID, corefiles.RestoreOptions{})
	if errStr != "" {
		t.Fatalf("RestoreVaultTrash: %s", errStr)
	}
	if restored != "Docs/two.txt" {
		t.Fatalf("restored path = %q, want Docs/two.txt", restored)
	}
	if _, err := os.Stat(filepath.Join(root, "Docs", "two.txt")); err != nil {
		t.Fatalf("restored file missing: %v", err)
	}

	permanent, errStr := app.TrashVaultPath("files.plugin", "Docs/two.txt")
	if errStr != "" {
		t.Fatalf("TrashVaultPath before permanent delete: %s", errStr)
	}
	if errStr := app.DeleteVaultTrash("files.plugin", permanent.TrashID); errStr != "" {
		t.Fatalf("DeleteVaultTrash: %s", errStr)
	}
	trashEntries, errStr = app.ListVaultTrash("files.plugin")
	if errStr != "" {
		t.Fatalf("ListVaultTrash after permanent delete: %s", errStr)
	}
	if len(trashEntries) != 0 {
		t.Fatalf("trash entries after permanent delete = %+v, want none", trashEntries)
	}
}

func TestFilesBridgeWritePublishesFileChangedActivityEvent(t *testing.T) {
	app, _ := newFilesTestApp(t, []string{"files.write"})
	bus := events.NewBus()
	app.eventBus = bus

	if errStr := app.CreateVaultFolder("files.plugin", "Project"); errStr != "" {
		t.Fatalf("CreateVaultFolder Project: %s", errStr)
	}
	if errStr := app.CreateVaultFolder("files.plugin", "Project/Notes"); errStr != "" {
		t.Fatalf("CreateVaultFolder Project/Notes: %s", errStr)
	}

	var received []events.Event
	bus.Subscribe("file.changed", func(event events.Event) {
		received = append(received, event)
	})

	if errStr := app.WriteVaultTextFile("files.plugin", "Project/Notes/one.txt", "hello", corefiles.WriteOptions{CreateIfMissing: true}); errStr != "" {
		t.Fatalf("WriteVaultTextFile: %s", errStr)
	}

	if len(received) != 1 {
		t.Fatalf("received %d file.changed events, want 1", len(received))
	}
	payload, ok := received[0].Payload.(map[string]interface{})
	if !ok {
		t.Fatalf("payload = %#v, want map[string]interface{}", received[0].Payload)
	}
	if payload["path"] != "Project/Notes/one.txt" {
		t.Fatalf("payload path = %#v", payload["path"])
	}
	if payload["workspaceRootPath"] != "Project" {
		t.Fatalf("payload workspaceRootPath = %#v, want Project", payload["workspaceRootPath"])
	}
	if payload["pluginId"] != "files.plugin" {
		t.Fatalf("payload pluginId = %#v, want files.plugin", payload["pluginId"])
	}
	if received[0].Timestamp == "" {
		t.Fatal("event timestamp is empty")
	}
}

func TestActivityProviderRecordsFileChangedWithoutMountedView(t *testing.T) {
	app, _ := newFilesTestApp(t, []string{"files.write"})
	app.eventBus = events.NewBus()
	app.storage = storage.New(app.vault)
	app.contribRegistry = contribution.NewRegistry()
	app.plugins = append(app.plugins, plugin.Plugin{
		Manifest: plugin.Manifest{
			ID:          "verstak.activity",
			Name:        "Activity",
			Version:     "1.0.0",
			Provides:    []string{"activity.log"},
			Permissions: []string{"storage.namespace"},
		},
		Status:  plugin.StatusLoaded,
		Enabled: true,
	})

	if errStr := app.CreateVaultFolder("files.plugin", "Project"); errStr != "" {
		t.Fatalf("CreateVaultFolder Project: %s", errStr)
	}
	if errStr := app.CreateVaultFolder("files.plugin", "Project/Notes"); errStr != "" {
		t.Fatalf("CreateVaultFolder Project/Notes: %s", errStr)
	}

	app.contribRegistry.Register("verstak.activity", &plugin.Contributions{
		ActivityProviders: []plugin.ContributionActivityProvider{{
			ID:      "verstak.activity.log",
			Events:  []string{"file.changed"},
			Handler: "recordActivityEvent",
		}},
	})
	app.ensureActivityProviderSubscriptions()

	if errStr := app.WriteVaultTextFile("files.plugin", "Project/Notes/one.txt", "hello", corefiles.WriteOptions{CreateIfMissing: true}); errStr != "" {
		t.Fatalf("WriteVaultTextFile: %s", errStr)
	}

	settings, err := app.storage.ReadPluginSettings("verstak.activity")
	if err != nil {
		t.Fatalf("ReadPluginSettings: %v", err)
	}
	stored, ok := settings["events:workspace:Project"].([]interface{})
	if !ok {
		t.Fatalf("events:workspace:Project = %#v, want []interface{}", settings["events:workspace:Project"])
	}
	if len(stored) != 1 {
		t.Fatalf("stored %d activity events, want 1", len(stored))
	}
	activity, ok := stored[0].(map[string]interface{})
	if !ok {
		t.Fatalf("activity = %#v, want map[string]interface{}", stored[0])
	}
	if activity["type"] != "file.changed" {
		t.Fatalf("activity type = %#v, want file.changed", activity["type"])
	}
	if activity["workspaceRootPath"] != "Project" {
		t.Fatalf("activity workspaceRootPath = %#v, want Project", activity["workspaceRootPath"])
	}
	if activity["sourcePluginId"] != "files.plugin" {
		t.Fatalf("activity sourcePluginId = %#v, want files.plugin", activity["sourcePluginId"])
	}
}

func TestBrowserInboxRecordsCaptureWithoutMountedView(t *testing.T) {
	v := vault.NewVault(nil)
	if err := v.CreateVault(t.TempDir()); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	bus := events.NewBus()
	app := &App{
		eventBus: bus,
		storage:  storage.New(v),
		vault:    v,
		plugins: []plugin.Plugin{{
			Manifest: plugin.Manifest{
				ID:          "verstak.browser-inbox",
				Name:        "Browser Inbox",
				Permissions: []string{"storage.namespace"},
			},
			Status:  plugin.StatusLoaded,
			Enabled: true,
		}},
	}
	app.ensureBrowserInboxSubscriptions()

	if err := app.recordBrowserCapture(events.Event{
		Name:      "browser.capture.page",
		Timestamp: "2026-07-11T12:00:00Z",
		Payload: map[string]interface{}{
			"captureId":  "capture-background",
			"capturedAt": "2026-07-11T11:59:00Z",
			"kind":       "page",
			"url":        "https://example.com/article",
			"title":      "Background capture",
			"domain":     "example.com",
		},
	}); err != nil {
		t.Fatalf("RecordBrowserCapture: %v", err)
	}
	bus.Publish(events.Event{Name: browserInboxMutationEvent, Payload: map[string]interface{}{
		"pluginId": browserInboxPluginID, "action": "assign", "captureId": "capture-background", "workspaceRootPath": "Project",
	}})
	bus.Publish(events.Event{Name: browserInboxMutationEvent, Payload: map[string]interface{}{
		"pluginId": browserInboxPluginID, "action": "processed", "captureId": "capture-background", "processed": true,
	}})
	if err := app.recordBrowserCapture(events.Event{
		Name: "browser.capture.page",
		Payload: map[string]interface{}{
			"captureId": "capture-background",
			"title":     "Retried payload",
		},
	}); err != nil {
		t.Fatalf("RecordBrowserCapture retry: %v", err)
	}

	settings, err := app.storage.ReadPluginSettings("verstak.browser-inbox")
	if err != nil {
		t.Fatalf("ReadPluginSettings: %v", err)
	}
	stored, ok := settings["captures:global"].([]interface{})
	if !ok || len(stored) != 1 {
		t.Fatalf("captures:global = %#v, want one capture", settings["captures:global"])
	}
	capture, ok := stored[0].(map[string]interface{})
	if !ok {
		t.Fatalf("capture = %#v, want map[string]interface{}", stored[0])
	}
	if capture["captureId"] != "capture-background" || capture["title"] != "Background capture" || capture["workspaceRootPath"] != "Project" || capture["processed"] != true {
		t.Fatalf("stored capture = %#v", capture)
	}
}

func TestBrowserInboxRejectsCaptureWithoutOpenVault(t *testing.T) {
	v := vault.NewVault(nil)
	app := &App{
		storage: storage.New(v),
		vault:   v,
		plugins: []plugin.Plugin{{
			Manifest: plugin.Manifest{
				ID:          browserInboxPluginID,
				Permissions: []string{"storage.namespace"},
			},
			Status:  plugin.StatusLoaded,
			Enabled: true,
		}},
	}
	app.ensureBrowserInboxSubscriptions()

	err := app.recordBrowserCapture(events.Event{
		Name:    "browser.capture.page",
		Payload: map[string]interface{}{"captureId": "capture-no-vault"},
	})

	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("RecordBrowserCapture error = %v, want unavailable", err)
	}
}

func TestBrowserInboxSerializesConcurrentCapturesAndMutations(t *testing.T) {
	v := vault.NewVault(nil)
	if err := v.CreateVault(t.TempDir()); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	bus := events.NewBus()
	app := &App{
		eventBus: bus,
		storage:  storage.New(v),
		vault:    v,
		plugins: []plugin.Plugin{{
			Manifest: plugin.Manifest{
				ID:          browserInboxPluginID,
				Permissions: []string{"storage.namespace"},
			},
			Status:  plugin.StatusLoaded,
			Enabled: true,
		}},
	}
	app.ensureBrowserInboxSubscriptions()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.storage.WritePluginSetting(browserInboxPluginID, "domainBindings", map[string]interface{}{
			"example.com": "Project",
		}); err != nil {
			t.Errorf("WritePluginSetting(domainBindings): %v", err)
		}
	}()
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := app.recordBrowserCapture(events.Event{
				Name:      "browser.capture.page",
				Timestamp: "2026-07-11T12:00:00Z",
				Payload: map[string]interface{}{
					"captureId": fmt.Sprintf("capture-%02d", index),
					"kind":      "page",
					"url":       fmt.Sprintf("https://example.com/%d", index),
				},
			}); err != nil {
				t.Errorf("RecordBrowserCapture(%d): %v", index, err)
			}
		}(i)
	}
	wg.Wait()

	bus.Publish(events.Event{
		Name: browserInboxMutationEvent,
		Payload: map[string]interface{}{
			"pluginId":          browserInboxPluginID,
			"action":            "assign",
			"captureId":         "capture-00",
			"workspaceRootPath": "Project",
		},
	})
	bus.Publish(events.Event{
		Name: browserInboxMutationEvent,
		Payload: map[string]interface{}{
			"pluginId":  browserInboxPluginID,
			"action":    "delete",
			"captureId": "capture-01",
		},
	})

	settings, err := app.storage.ReadPluginSettings(browserInboxPluginID)
	if err != nil {
		t.Fatalf("ReadPluginSettings: %v", err)
	}
	stored, ok := settings[browserInboxGlobalKey].([]interface{})
	if !ok || len(stored) != 49 {
		t.Fatalf("captures:global contains %d captures, want 49", len(stored))
	}
	var assigned map[string]interface{}
	for _, item := range stored {
		capture := item.(map[string]interface{})
		if capture["captureId"] == "capture-00" {
			assigned = capture
		}
		if capture["captureId"] == "capture-01" {
			t.Fatal("deleted capture remained in storage")
		}
	}
	if assigned == nil || assigned["workspaceRootPath"] != "Project" {
		t.Fatalf("assigned capture = %#v", assigned)
	}
	if settings["domainBindings"] == nil {
		t.Fatal("concurrent domainBindings update was lost")
	}
}

func TestBrowserInboxMutationMigratesAndClearsLegacyCaptures(t *testing.T) {
	v := vault.NewVault(nil)
	if err := v.CreateVault(t.TempDir()); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	bus := events.NewBus()
	app := &App{
		eventBus: bus,
		storage:  storage.New(v),
		vault:    v,
		plugins: []plugin.Plugin{{
			Manifest: plugin.Manifest{ID: browserInboxPluginID, Permissions: []string{"storage.namespace"}},
			Status:   plugin.StatusLoaded,
			Enabled:  true,
		}},
	}
	if err := app.storage.WritePluginSettings(browserInboxPluginID, map[string]interface{}{
		browserInboxLegacyKey: []interface{}{map[string]interface{}{
			"captureId": "legacy-duplicate",
			"title":     "Stale legacy copy",
		}},
		browserInboxGlobalKey: []interface{}{map[string]interface{}{
			"captureId":         "legacy-duplicate",
			"title":             "Canonical copy",
			"workspaceRootPath": "Project",
			"processed":         true,
		}},
	}); err != nil {
		t.Fatalf("WritePluginSettings: %v", err)
	}
	app.ensureBrowserInboxSubscriptions()

	bus.Publish(events.Event{Name: browserInboxMutationEvent, Payload: map[string]interface{}{
		"pluginId": browserInboxPluginID,
		"action":   "migrate",
	}})

	settings, err := app.storage.ReadPluginSettings(browserInboxPluginID)
	if err != nil {
		t.Fatalf("ReadPluginSettings: %v", err)
	}
	legacy := settings[browserInboxLegacyKey].([]interface{})
	if len(legacy) != 0 {
		t.Fatalf("legacy captures = %#v, want empty", legacy)
	}
	stored := settings[browserInboxGlobalKey].([]interface{})
	if len(stored) != 1 {
		t.Fatalf("canonical captures = %#v, want one", stored)
	}
	capture := stored[0].(map[string]interface{})
	if capture["title"] != "Canonical copy" || capture["workspaceRootPath"] != "Project" || capture["processed"] != true {
		t.Fatalf("canonical capture = %#v", capture)
	}
}

func TestActivityFromEventRedactsBinaryPayload(t *testing.T) {
	activity := activityFromEvent(events.Event{
		Name:      "browser.capture.file",
		Timestamp: "2026-06-29T00:00:00Z",
		Payload: map[string]interface{}{
			"captureId":         "capture-binary",
			"title":             "logo.png",
			"workspaceRootPath": "Project",
			"fileDataBase64":    "iVBORw==",
			"fileText":          "preview",
		},
	})
	payload, ok := activity["payload"].(map[string]interface{})
	if !ok {
		t.Fatalf("payload = %#v, want map", activity["payload"])
	}
	if _, ok := payload["fileDataBase64"]; ok {
		t.Fatalf("activity payload leaked fileDataBase64: %#v", payload)
	}
	if payload["fileText"] != "preview" {
		t.Fatalf("activity payload fileText = %#v, want preview", payload["fileText"])
	}
}

func TestFilesBridgeOpenExternalUsesVaultPathPolicyAndPermission(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.openExternal"})
	filePath := filepath.Join(root, "Docs", "one.txt")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var opened []string
	app.externalOpen = newTestExternalOpenService(func(path string) error {
		opened = append(opened, path)
		return nil
	})

	if errStr := app.OpenVaultPathExternal("files.plugin", "Docs/one.txt"); errStr != "" {
		t.Fatalf("OpenVaultPathExternal: %s", errStr)
	}
	if len(opened) != 1 || opened[0] != filePath {
		t.Fatalf("opened = %#v, want %q", opened, filePath)
	}

	if errStr := app.OpenVaultPathExternal("files.plugin", ".verstak/vault.json"); errStr == "" || !strings.Contains(errStr, "reserved-path") {
		t.Fatalf("reserved path error = %q, want reserved-path", errStr)
	}
	if len(opened) != 1 {
		t.Fatalf("reserved path should not open, opened = %#v", opened)
	}
}

func TestFilesBridgeShowInFolderUsesVaultPathPolicyAndPermission(t *testing.T) {
	app, root := newFilesTestApp(t, []string{"files.openExternal"})
	filePath := filepath.Join(root, "Docs", "one.txt")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var shown []string
	app.externalOpen = newTestExternalOpenService(func(path string) error {
		shown = append(shown, path)
		return nil
	})

	if errStr := app.ShowVaultPathInFolder("files.plugin", "Docs/one.txt"); errStr != "" {
		t.Fatalf("ShowVaultPathInFolder: %s", errStr)
	}
	if len(shown) != 1 || shown[0] != filePath {
		t.Fatalf("shown = %#v, want %q", shown, filePath)
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
			name:       "read bytes requires read",
			perms:      []string{"files.write", "files.delete"},
			call:       func(app *App) string { _, errStr := app.ReadVaultFileBytes("files.plugin", "one.txt"); return errStr },
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
			name:  "write bytes requires write",
			perms: []string{"files.read", "files.delete"},
			call: func(app *App) string {
				return app.WriteVaultFileBytes("files.plugin", "one.bin", "AQID", corefiles.WriteOptions{CreateIfMissing: true})
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
		{
			name:  "restore requires delete",
			perms: []string{"files.read", "files.write"},
			call: func(app *App) string {
				_, errStr := app.RestoreVaultTrash("files.plugin", "missing", corefiles.RestoreOptions{})
				return errStr
			},
			wantPhrase: "files.delete",
		},
		{
			name:  "restore requires write",
			perms: []string{"files.read", "files.delete"},
			call: func(app *App) string {
				_, errStr := app.RestoreVaultTrash("files.plugin", "missing", corefiles.RestoreOptions{})
				return errStr
			},
			wantPhrase: "files.write",
		},
		{
			name:       "permanent trash delete requires delete",
			perms:      []string{"files.read", "files.write"},
			call:       func(app *App) string { return app.DeleteVaultTrash("files.plugin", "missing") },
			wantPhrase: "files.delete",
		},
		{
			name:       "open external requires openExternal",
			perms:      []string{"files.read", "files.write", "files.delete"},
			call:       func(app *App) string { return app.OpenVaultPathExternal("files.plugin", "one.txt") },
			wantPhrase: "files.openExternal",
		},
		{
			name:       "show in folder requires openExternal",
			perms:      []string{"files.read", "files.write", "files.delete"},
			call:       func(app *App) string { return app.ShowVaultPathInFolder("files.plugin", "one.txt") },
			wantPhrase: "files.openExternal",
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

func TestApplyRemoteFileOps(t *testing.T) {
	app, _ := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, "local-device")

	ops := []syncsvc.Op{
		{
			OpID:        "folder-create",
			DeviceID:    "remote-device",
			EntityType:  syncsvc.EntityFolder,
			EntityID:    "Docs",
			OpType:      syncsvc.OpCreate,
			PayloadJSON: `{"path":"Docs"}`,
		},
		{
			OpID:        "file-create",
			DeviceID:    "remote-device",
			EntityType:  syncsvc.EntityFile,
			EntityID:    "Docs/one.txt",
			OpType:      syncsvc.OpCreate,
			PayloadJSON: `{"path":"Docs/one.txt","content":"hello"}`,
		},
		{
			OpID:        "file-update",
			DeviceID:    "remote-device",
			EntityType:  syncsvc.EntityFile,
			EntityID:    "Docs/one.txt",
			OpType:      syncsvc.OpUpdate,
			PayloadJSON: `{"path":"Docs/one.txt","content":"updated"}`,
		},
		{
			OpID:        "binary-create",
			DeviceID:    "remote-device",
			EntityType:  syncsvc.EntityFile,
			EntityID:    "Docs/image.bin",
			OpType:      syncsvc.OpCreate,
			PayloadJSON: `{"path":"Docs/image.bin","dataBase64":"AQID"}`,
		},
		{
			OpID:        "file-move",
			DeviceID:    "remote-device",
			EntityType:  syncsvc.EntityFile,
			EntityID:    "Docs/one.txt",
			OpType:      syncsvc.OpMove,
			PayloadJSON: `{"fromPath":"Docs/one.txt","toPath":"Docs/two.txt"}`,
		},
	}
	for _, op := range ops {
		if err := app.applyRemoteOp(op); err != nil {
			t.Fatalf("applyRemoteOp(%s): %v", op.OpID, err)
		}
	}

	text, errStr := app.ReadVaultTextFile("files.plugin", "Docs/two.txt")
	if errStr != "" {
		t.Fatalf("ReadVaultTextFile: %s", errStr)
	}
	if text != "updated" {
		t.Fatalf("content = %q, want updated", text)
	}
	binaryBytes, errStr := app.ReadVaultFileBytes("files.plugin", "Docs/image.bin")
	if errStr != "" {
		t.Fatalf("ReadVaultFileBytes binary: %s", errStr)
	}
	if binaryBytes.DataBase64 != "AQID" {
		t.Fatalf("binary dataBase64 = %q, want AQID", binaryBytes.DataBase64)
	}
	if _, errStr := app.GetVaultFileMetadata("files.plugin", "Docs/one.txt"); !strings.Contains(errStr, "not-found") {
		t.Fatalf("old path metadata err = %q, want not-found", errStr)
	}

	if err := app.applyRemoteOp(syncsvc.Op{
		OpID:        "file-delete",
		DeviceID:    "remote-device",
		EntityType:  syncsvc.EntityFile,
		EntityID:    "Docs/two.txt",
		OpType:      syncsvc.OpDelete,
		PayloadJSON: `{"path":"Docs/two.txt"}`,
	}); err != nil {
		t.Fatalf("applyRemoteOp(file-delete): %v", err)
	}
	if _, errStr := app.GetVaultFileMetadata("files.plugin", "Docs/two.txt"); !strings.Contains(errStr, "not-found") {
		t.Fatalf("deleted path metadata err = %q, want not-found", errStr)
	}
}

func TestApplyRemoteFolderOps(t *testing.T) {
	app, _ := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, "local-device")

	ops := []syncsvc.Op{
		{
			OpID:        "folder-create",
			DeviceID:    "remote-device",
			EntityType:  syncsvc.EntityFolder,
			EntityID:    "Projects",
			OpType:      syncsvc.OpCreate,
			PayloadJSON: `{"path":"Projects"}`,
		},
		{
			OpID:        "folder-move",
			DeviceID:    "remote-device",
			EntityType:  syncsvc.EntityFolder,
			EntityID:    "Projects",
			OpType:      syncsvc.OpMove,
			PayloadJSON: `{"fromPath":"Projects","toPath":"Archive"}`,
		},
		{
			OpID:        "folder-delete",
			DeviceID:    "remote-device",
			EntityType:  syncsvc.EntityFolder,
			EntityID:    "Archive",
			OpType:      syncsvc.OpDelete,
			PayloadJSON: `{"path":"Archive"}`,
		},
	}
	for _, op := range ops {
		if err := app.applyRemoteOp(op); err != nil {
			t.Fatalf("applyRemoteOp(%s): %v", op.OpID, err)
		}
	}
	if _, errStr := app.GetVaultFileMetadata("files.plugin", "Archive"); !strings.Contains(errStr, "not-found") {
		t.Fatalf("deleted folder metadata err = %q, want not-found", errStr)
	}
}

func TestApplyRemoteOpSkipsLocalDevice(t *testing.T) {
	app, _ := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, "local-device")
	if errStr := app.WriteVaultTextFile("files.plugin", "local.txt", "keep", corefiles.WriteOptions{CreateIfMissing: true}); errStr != "" {
		t.Fatalf("WriteVaultTextFile: %s", errStr)
	}

	err := app.applyRemoteOp(syncsvc.Op{
		OpID:        "own-delete",
		DeviceID:    "local-device",
		EntityType:  syncsvc.EntityFile,
		EntityID:    "local.txt",
		OpType:      syncsvc.OpDelete,
		PayloadJSON: `{"path":"local.txt"}`,
	})
	if err != nil {
		t.Fatalf("applyRemoteOp: %v", err)
	}
	text, errStr := app.ReadVaultTextFile("files.plugin", "local.txt")
	if errStr != "" {
		t.Fatalf("ReadVaultTextFile: %s", errStr)
	}
	if text != "keep" {
		t.Fatalf("content = %q, want keep", text)
	}
}

func TestFileBridgeRecordsSyncOps(t *testing.T) {
	app, _ := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, "local-device")

	if errStr := app.CreateVaultFolder("files.plugin", "Docs"); errStr != "" {
		t.Fatalf("CreateVaultFolder: %s", errStr)
	}
	if errStr := app.WriteVaultTextFile("files.plugin", "Docs/one.txt", "hello", corefiles.WriteOptions{CreateIfMissing: true}); errStr != "" {
		t.Fatalf("WriteVaultTextFile create: %s", errStr)
	}
	if errStr := app.WriteVaultTextFile("files.plugin", "Docs/one.txt", "updated", corefiles.WriteOptions{Overwrite: true}); errStr != "" {
		t.Fatalf("WriteVaultTextFile update: %s", errStr)
	}
	if errStr := app.WriteVaultFileBytes("files.plugin", "Docs/image.bin", "AQID", corefiles.WriteOptions{CreateIfMissing: true}); errStr != "" {
		t.Fatalf("WriteVaultFileBytes create: %s", errStr)
	}
	if errStr := app.MoveVaultPath("files.plugin", "Docs/one.txt", "Docs/two.txt", corefiles.MoveOptions{}); errStr != "" {
		t.Fatalf("MoveVaultPath: %s", errStr)
	}
	if _, errStr := app.TrashVaultPath("files.plugin", "Docs/two.txt"); errStr != "" {
		t.Fatalf("TrashVaultPath: %s", errStr)
	}

	ops, err := app.syncSvc.GetUnpushedOps()
	if err != nil {
		t.Fatalf("GetUnpushedOps: %v", err)
	}
	if len(ops) != 6 {
		t.Fatalf("ops len = %d, want 6: %#v", len(ops), ops)
	}

	want := []struct {
		entityType string
		entityID   string
		opType     string
		payload    string
	}{
		{syncsvc.EntityFolder, "Docs", syncsvc.OpCreate, `"path":"Docs"`},
		{syncsvc.EntityFile, "Docs/one.txt", syncsvc.OpCreate, `"content":"hello"`},
		{syncsvc.EntityFile, "Docs/one.txt", syncsvc.OpUpdate, `"content":"updated"`},
		{syncsvc.EntityFile, "Docs/image.bin", syncsvc.OpCreate, `"dataBase64":"AQID"`},
		{syncsvc.EntityFile, "Docs/one.txt", syncsvc.OpMove, `"toPath":"Docs/two.txt"`},
		{syncsvc.EntityFile, "Docs/two.txt", syncsvc.OpDelete, `"path":"Docs/two.txt"`},
	}
	for i, w := range want {
		if ops[i].DeviceID != "local-device" || ops[i].EntityType != w.entityType || ops[i].EntityID != w.entityID || ops[i].OpType != w.opType {
			t.Fatalf("op[%d] = %+v, want device/local %s %s %s", i, ops[i], w.entityType, w.entityID, w.opType)
		}
		if !strings.Contains(ops[i].PayloadJSON, w.payload) {
			t.Fatalf("op[%d] payload = %s, want contains %s", i, ops[i].PayloadJSON, w.payload)
		}
	}
}

func TestSyncNowPushesLocalOpsAndAppliesPulledFileOps(t *testing.T) {
	app, root := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, "local-device")

	var pushedOps []struct {
		OpID              string `json:"op_id"`
		EntityType        string `json:"entity_type"`
		EntityID          string `json:"entity_id"`
		OpType            string `json:"op_type"`
		PayloadJSON       string `json:"payload_json"`
		ClientSequence    int    `json:"client_sequence"`
		LastSeenServerSeq int    `json:"last_seen_server_seq"`
		CreatedAt         string `json:"created_at"`
	}
	var pushedDeviceID string
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer device-token" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/sync/push":
			var req struct {
				DeviceID string `json:"device_id"`
				Ops      []struct {
					OpID              string `json:"op_id"`
					EntityType        string `json:"entity_type"`
					EntityID          string `json:"entity_id"`
					OpType            string `json:"op_type"`
					PayloadJSON       string `json:"payload_json"`
					ClientSequence    int    `json:"client_sequence"`
					LastSeenServerSeq int    `json:"last_seen_server_seq"`
					CreatedAt         string `json:"created_at"`
				} `json:"ops"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			pushedDeviceID = req.DeviceID
			pushedOps = req.Ops
			accepted := make([]string, 0, len(req.Ops))
			for _, op := range req.Ops {
				accepted = append(accepted, op.OpID)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"accepted":  accepted,
				"count":     len(accepted),
				"conflicts": []map[string]interface{}{},
			})
		case "/api/v1/sync/pull":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"server_sequence": 2,
				"ops": []map[string]interface{}{
					{
						"op_id":           "remote-folder",
						"server_sequence": 1,
						"device_id":       "remote-device",
						"entity_type":     syncsvc.EntityFolder,
						"entity_id":       "Remote",
						"op_type":         syncsvc.OpCreate,
						"payload_json":    `{"path":"Remote"}`,
						"created_at":      "2026-06-27T00:00:00Z",
					},
					{
						"op_id":           "remote-file",
						"server_sequence": 2,
						"device_id":       "remote-device",
						"entity_type":     syncsvc.EntityFile,
						"entity_id":       "Remote/hello.txt",
						"op_type":         syncsvc.OpCreate,
						"payload_json":    `{"path":"Remote/hello.txt","content":"from remote"}`,
						"created_at":      "2026-06-27T00:00:01Z",
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if err := app.syncSvc.SetState(server.URL, ""); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	if err := syncsvc.SaveDeviceToken(root, "device-token"); err != nil {
		t.Fatalf("SaveDeviceToken: %v", err)
	}
	if errStr := app.CreateVaultFolder("files.plugin", "Local"); errStr != "" {
		t.Fatalf("CreateVaultFolder: %s", errStr)
	}
	if errStr := app.WriteVaultTextFile("files.plugin", "Local/one.txt", "local", corefiles.WriteOptions{CreateIfMissing: true}); errStr != "" {
		t.Fatalf("WriteVaultTextFile: %s", errStr)
	}

	result, err := app.syncNow()
	if err != nil {
		t.Fatalf("syncNow: %v", err)
	}
	if result["pushed"] != 2 || result["pulled"] != 2 || result["serverSequence"] != 2 {
		t.Fatalf("sync result = %#v", result)
	}
	if pushedDeviceID != "local-device" {
		t.Fatalf("pushed device = %q, want local-device", pushedDeviceID)
	}
	if len(pushedOps) != 2 {
		t.Fatalf("pushed ops len = %d, want 2", len(pushedOps))
	}
	for i, op := range pushedOps {
		if op.LastSeenServerSeq != 0 {
			t.Fatalf("pushed op[%d] last seen = %d, want 0", i, op.LastSeenServerSeq)
		}
	}

	text, errStr := app.ReadVaultTextFile("files.plugin", "Remote/hello.txt")
	if errStr != "" {
		t.Fatalf("ReadVaultTextFile remote: %s", errStr)
	}
	if text != "from remote" {
		t.Fatalf("remote content = %q, want from remote", text)
	}
	unpushed, err := app.syncSvc.GetUnpushedOps()
	if err != nil {
		t.Fatalf("GetUnpushedOps: %v", err)
	}
	if len(unpushed) != 0 {
		t.Fatalf("unpushed len = %d, want 0: %#v", len(unpushed), unpushed)
	}
	_, _, lastPullSeq, _, err := app.syncSvc.GetState()
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if lastPullSeq != 2 {
		t.Fatalf("last pull seq = %d, want 2", lastPullSeq)
	}
}

func TestSyncConfigurePairsCurrentVaultID(t *testing.T) {
	app, _ := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, "local-device")
	meta := app.vault.GetVaultMeta()
	if meta == nil || meta.VaultID == "" {
		t.Fatal("test vault must have an ID")
	}

	var pairedVaultID string
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/client/pair" {
			http.NotFound(w, r)
			return
		}
		var request struct {
			VaultID string `json:"vault_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		pairedVaultID = request.VaultID
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"device_id":    "paired-device",
			"device_token": "paired-token",
		})
	}))
	defer server.Close()

	if err := app.syncConfigure(server.URL, "alice", "secret"); err != nil {
		t.Fatalf("syncConfigure: %v", err)
	}
	if pairedVaultID != meta.VaultID {
		t.Fatalf("paired vault ID = %q, want %q", pairedVaultID, meta.VaultID)
	}
}

func TestSyncNowHydratesLegacyVaultDeviceID(t *testing.T) {
	app, root := newSyncFilesTestApp(t, []string{"files.read", "files.write", "files.delete"}, "wrong-global-device")
	var pushedDeviceID string
	server := newLocalHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer vault-token" {
			http.Error(w, "missing auth", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/client/me":
			_ = json.NewEncoder(w).Encode(map[string]string{"device_id": "vault-device"})
		case "/api/v1/sync/push":
			var request struct {
				DeviceID string `json:"device_id"`
				Ops      []struct {
					OpID string `json:"op_id"`
				} `json:"ops"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			pushedDeviceID = request.DeviceID
			accepted := make([]string, 0, len(request.Ops))
			for _, op := range request.Ops {
				accepted = append(accepted, op.OpID)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"accepted":  accepted,
				"count":     len(accepted),
				"conflicts": []map[string]interface{}{},
			})
		case "/api/v1/sync/pull":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"server_sequence": 0,
				"ops":             []map[string]interface{}{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	app.syncSvc = syncsvc.NewService(root, "")
	if err := app.syncSvc.SetState(server.URL, ""); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	if err := syncsvc.SaveDeviceToken(root, "vault-token"); err != nil {
		t.Fatalf("SaveDeviceToken: %v", err)
	}
	if err := app.syncSvc.RecordOp(syncsvc.EntityFile, "Docs/one.txt", syncsvc.OpCreate, map[string]string{"path": "Docs/one.txt"}); err != nil {
		t.Fatalf("RecordOp: %v", err)
	}

	if _, err := app.syncNow(); err != nil {
		t.Fatalf("syncNow: %v", err)
	}
	if pushedDeviceID != "vault-device" {
		t.Fatalf("pushed device ID = %q, want vault-device", pushedDeviceID)
	}
	if app.syncSvc.GetDeviceID() != "vault-device" {
		t.Fatalf("persisted device ID = %q, want vault-device", app.syncSvc.GetDeviceID())
	}
}

func TestVaultTransitionsRebindSyncService(t *testing.T) {
	settings := appsettings.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err := settings.Load(); err != nil {
		t.Fatalf("settings Load: %v", err)
	}
	app := &App{
		capRegistry: capability.NewRegistry(),
		vault:       vault.NewVault(nil),
		appSettings: settings,
	}

	if err := app.CreateVault(t.TempDir()); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}
	if app.syncSvc == nil {
		t.Fatal("CreateVault must initialize the sync service")
	}

	vaultB := vault.NewVault(nil)
	if err := vaultB.CreateVault(t.TempDir()); err != nil {
		t.Fatalf("Create vault B: %v", err)
	}
	primeSyncState(t, vaultB.GetVaultPath(), "device-b", "https://sync-b.example.test", 22)
	if err := app.OpenVault(vaultB.GetVaultPath()); err != nil {
		t.Fatalf("OpenVault B: %v", err)
	}
	assertSyncState(t, app.syncSvc, "device-b", "https://sync-b.example.test", 22)

	vaultC := vault.NewVault(nil)
	if err := vaultC.CreateVault(t.TempDir()); err != nil {
		t.Fatalf("Create vault C: %v", err)
	}
	primeSyncState(t, vaultC.GetVaultPath(), "device-c", "https://sync-c.example.test", 33)
	if errStr := app.SetCurrentVault(vaultC.GetVaultPath()); errStr != "" {
		t.Fatalf("SetCurrentVault C: %s", errStr)
	}
	assertSyncState(t, app.syncSvc, "device-c", "https://sync-c.example.test", 33)
}

func primeSyncState(t *testing.T, vaultRoot, deviceID, serverURL string, lastPullSeq int) {
	t.Helper()
	service := syncsvc.NewService(vaultRoot, deviceID)
	if err := service.SetState(serverURL, ""); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	if err := service.SetLastPullSeq(lastPullSeq); err != nil {
		t.Fatalf("SetLastPullSeq: %v", err)
	}
}

func assertSyncState(t *testing.T, service *syncsvc.Service, wantDeviceID, wantServerURL string, wantLastPullSeq int) {
	t.Helper()
	if service == nil {
		t.Fatal("sync service is nil")
	}
	serverURL, _, lastPullSeq, _, err := service.GetState()
	if err != nil {
		t.Fatalf("GetState: %v", err)
	}
	if service.GetDeviceID() != wantDeviceID || serverURL != wantServerURL || lastPullSeq != wantLastPullSeq {
		t.Fatalf("sync state = device=%q server=%q cursor=%d, want device=%q server=%q cursor=%d",
			service.GetDeviceID(), serverURL, lastPullSeq, wantDeviceID, wantServerURL, wantLastPullSeq)
	}
}

func TestAppSettingsLanguageAPI(t *testing.T) {
	settings := appsettings.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err := settings.Load(); err != nil {
		t.Fatal(err)
	}
	if err := settings.Update(&appsettings.Config{Theme: "light", DevMode: true}); err != nil {
		t.Fatal(err)
	}
	app := &App{appSettings: settings}

	if got := app.GetAppSettings()["language"]; got != "system" {
		t.Fatalf("initial language = %#v, want system", got)
	}
	if errStr := app.UpdateAppSettings(map[string]interface{}{"language": "ru"}); errStr != "" {
		t.Fatalf("UpdateAppSettings language: %s", errStr)
	}
	got := app.GetAppSettings()
	if got["language"] != "ru" || got["theme"] != "light" || got["devMode"] != true {
		t.Fatalf("settings after language update = %#v", got)
	}
	if errStr := app.UpdateAppSettings(map[string]interface{}{"language": "de"}); errStr == "" {
		t.Fatal("UpdateAppSettings accepted unsupported language")
	}
	if got := app.GetAppSettings()["language"]; got != "ru" {
		t.Fatalf("language after rejected update = %#v, want ru", got)
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
	if nodes[0].Path != "" {
		t.Fatalf("compatibility node should not expose workspace path mapping: %+v", nodes[0])
	}
	if !app.capRegistry.Has("verstak/core/workspace/v1") {
		t.Fatal("workspace capability should be registered after SetCurrentVault")
	}
}

func TestSetCurrentVaultStartsLiveFileWatcher(t *testing.T) {
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

	received := make(chan events.Event, 4)
	bus.Subscribe("file.changed", func(event events.Event) {
		received <- event
	})

	app := &App{
		capRegistry: capability.NewRegistry(),
		eventBus:    bus,
		vault:       vaultService,
		appSettings: settings,
	}
	if errStr := app.SetCurrentVault(vaultParent); errStr != "" {
		t.Fatalf("SetCurrentVault: %s", errStr)
	}
	t.Cleanup(func() {
		if app.fileWatcher != nil {
			app.fileWatcher.Stop()
		}
	})

	if err := os.WriteFile(filepath.Join(vaultService.GetVaultPath(), "external.md"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-received:
		payload, ok := event.Payload.(map[string]interface{})
		if !ok {
			t.Fatalf("payload type = %T, want map", event.Payload)
		}
		if payload["path"] != "external.md" || payload["operation"] != "external.create" {
			t.Fatalf("event payload = %#v", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watcher event")
	}
}

func TestWorkspaceAPIUsesTopLevelFoldersAndMetadataSnapshot(t *testing.T) {
	app, vaultDir := newFilesTestApp(t, []string{"files.read"})
	app.workspace = workspace.NewManager(vaultDir)
	if err := app.workspace.Load(); err != nil {
		t.Fatalf("workspace Load: %v", err)
	}

	ws, errStr := app.CreateWorkspace("Project", "client-project")
	if errStr != "" {
		t.Fatalf("CreateWorkspace: %s", errStr)
	}
	if ws.RootPath != "Project" {
		t.Fatalf("workspace = %+v, want rootPath Project", ws)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Project", "Notes")); err != nil {
		t.Fatalf("template notes folder missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Project", "Notes", "Overview.md")); !os.IsNotExist(err) {
		t.Fatalf("template should not create overview file, stat err=%v", err)
	}

	meta, errStr := app.GetWorkspaceMetadata("Project")
	if errStr != "" {
		t.Fatalf("GetWorkspaceMetadata: %s", errStr)
	}
	if meta.CreatedFromTemplate == nil || meta.CreatedFromTemplate.TemplateID != "client-project" {
		t.Fatalf("metadata snapshot = %+v", meta.CreatedFromTemplate)
	}

	if errStr := app.RenameWorkspace("Project", "Renamed"); errStr != "" {
		t.Fatalf("RenameWorkspace: %s", errStr)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Renamed")); err != nil {
		t.Fatalf("renamed folder missing: %v", err)
	}

	result, errStr := app.TrashWorkspace("Renamed")
	if errStr != "" {
		t.Fatalf("TrashWorkspace: %s", errStr)
	}
	if result.TrashPath == "" {
		t.Fatalf("trash result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Renamed")); !os.IsNotExist(err) {
		t.Fatalf("workspace should be moved out of top level, stat err=%v", err)
	}
}

func TestWorkspaceAPIListsSelectableTemplates(t *testing.T) {
	app, vaultDir := newFilesTestApp(t, []string{"files.read"})
	app.workspace = workspace.NewManager(vaultDir)
	if err := app.workspace.Load(); err != nil {
		t.Fatalf("workspace Load: %v", err)
	}

	templates, errStr := app.ListWorkspaceTemplates()
	if errStr != "" {
		t.Fatalf("ListWorkspaceTemplates: %s", errStr)
	}
	if len(templates) != 5 {
		t.Fatalf("templates = %+v, want 5 selectable templates", templates)
	}
	if templates[0].ID != "default" || templates[1].ID != "project" || templates[4].ID != "minimal" {
		t.Fatalf("template order = %+v", templates)
	}
}

func TestWorkspaceAPIPublishesLifecycleEvents(t *testing.T) {
	app, vaultDir := newFilesTestApp(t, []string{"files.read"})
	app.workspace = workspace.NewManager(vaultDir)
	app.eventBus = events.NewBus()
	if err := app.workspace.Load(); err != nil {
		t.Fatalf("workspace Load: %v", err)
	}

	received := map[string]map[string]interface{}{}
	for _, eventName := range []string{"workspace.created", "workspace.selected", "workspace.renamed", "workspace.trashed"} {
		name := eventName
		app.eventBus.Subscribe(name, func(event events.Event) {
			payload, ok := event.Payload.(map[string]interface{})
			if !ok {
				t.Fatalf("%s payload type = %T", name, event.Payload)
			}
			received[name] = payload
		})
	}

	if _, errStr := app.CreateWorkspace("Project", "client-project"); errStr != "" {
		t.Fatalf("CreateWorkspace: %s", errStr)
	}
	if errStr := app.SetCurrentWorkspace("Project"); errStr != "" {
		t.Fatalf("SetCurrentWorkspace: %s", errStr)
	}
	if errStr := app.RenameWorkspace("Project", "Renamed"); errStr != "" {
		t.Fatalf("RenameWorkspace: %s", errStr)
	}
	if _, errStr := app.TrashWorkspace("Renamed"); errStr != "" {
		t.Fatalf("TrashWorkspace: %s", errStr)
	}

	if got := received["workspace.created"]["workspaceRootPath"]; got != "Project" {
		t.Fatalf("workspace.created workspaceRootPath = %#v, want Project", got)
	}
	if got := received["workspace.created"]["templateId"]; got != "client-project" {
		t.Fatalf("workspace.created templateId = %#v, want client-project", got)
	}
	if got := received["workspace.selected"]["workspaceRootPath"]; got != "Project" {
		t.Fatalf("workspace.selected workspaceRootPath = %#v, want Project", got)
	}
	if got := received["workspace.renamed"]["workspaceRootPath"]; got != "Renamed" {
		t.Fatalf("workspace.renamed workspaceRootPath = %#v, want Renamed", got)
	}
	if got := received["workspace.renamed"]["previousWorkspaceRootPath"]; got != "Project" {
		t.Fatalf("workspace.renamed previousWorkspaceRootPath = %#v, want Project", got)
	}
	if got := received["workspace.trashed"]["workspaceRootPath"]; got != "Renamed" {
		t.Fatalf("workspace.trashed workspaceRootPath = %#v, want Renamed", got)
	}
	if got := received["workspace.trashed"]["trashPath"]; got == "" {
		t.Fatalf("workspace.trashed trashPath = %#v, want non-empty", got)
	}
}

func TestMoveWorkspaceNodeCompatibilityIsUnsupported(t *testing.T) {
	app, vaultDir := newFilesTestApp(t, []string{"files.read"})
	app.workspace = workspace.NewManager(vaultDir)
	if err := app.workspace.Load(); err != nil {
		t.Fatalf("workspace Load: %v", err)
	}
	if _, errStr := app.CreateWorkspace("Project", "default"); errStr != "" {
		t.Fatalf("CreateWorkspace Project: %s", errStr)
	}
	if _, errStr := app.CreateWorkspace("Test", "default"); errStr != "" {
		t.Fatalf("CreateWorkspace Test: %s", errStr)
	}

	errStr := app.MoveWorkspaceNode("Project", "Test")
	if errStr == "" || !strings.Contains(errStr, "top-level only") {
		t.Fatalf("MoveWorkspaceNode error = %q, want top-level only", errStr)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Test", "Project")); !os.IsNotExist(err) {
		t.Fatalf("MoveWorkspaceNode created nested mapped workspace, stat err=%v", err)
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
		FileActions: []plugin.ContributionAction{
			{ID: "bridge.file.action", Label: "Bridge File Action", Icon: "zap", Handler: "bridge.command"},
		},
		NoteActions: []plugin.ContributionAction{
			{ID: "bridge.note.action", Label: "Bridge Note Action", Icon: "zap", Handler: "bridge.command"},
		},
		ContextMenuEntries: []plugin.ContributionContextMenuEntry{
			{ID: "bridge.file.context", Label: "Bridge File Context", Context: "file", Group: "open", Handler: "bridge.command"},
		},
		StatusBarItems: []plugin.ContributionStatusBarItem{
			{ID: "bridge.status", Label: "Bridge Ready", Position: "right", Handler: "openBridgeStatus"},
		},
		SearchProviders: []plugin.ContributionSearchProvider{
			{ID: "bridge.search", Label: "Bridge Search", Handler: "searchVault"},
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
			{
				Manifest: plugin.Manifest{
					ID:          "secrets.plugin",
					Name:        "Secrets Plugin",
					Version:     "1.0.0",
					Provides:    []string{"secret-store", "secrets.read-ui", "secrets.write-ui"},
					Permissions: []string{"secrets.read", "secrets.write", "workbench.open"},
				},
				Status:  plugin.StatusLoaded,
				Enabled: true,
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

func TestContributionSummaryIncludesStatusBarItems(t *testing.T) {
	app := newBridgeTestApp(t)

	summary := app.GetContributions()
	if len(summary.StatusBarItems) != 1 {
		t.Fatalf("StatusBarItems count = %d, want 1", len(summary.StatusBarItems))
	}
	item := summary.StatusBarItems[0]
	if item.PluginID != "bridge.plugin" || item.ID != "bridge.status" || item.Label != "Bridge Ready" || item.Position != "right" || item.Handler != "openBridgeStatus" {
		t.Fatalf("status item = %+v", item)
	}
}

func TestContributionSummaryIncludesSearchProviders(t *testing.T) {
	app := newBridgeTestApp(t)

	summary := app.GetContributions()
	if len(summary.SearchProviders) != 1 {
		t.Fatalf("SearchProviders count = %d, want 1", len(summary.SearchProviders))
	}
	provider := summary.SearchProviders[0]
	if provider.PluginID != "bridge.plugin" || provider.ID != "bridge.search" || provider.Label != "Bridge Search" || provider.Handler != "searchVault" {
		t.Fatalf("search provider = %+v", provider)
	}
}

func TestContributionSummaryIncludesActionsAndContextMenus(t *testing.T) {
	app := newBridgeTestApp(t)

	summary := app.GetContributions()
	if len(summary.FileActions) != 1 {
		t.Fatalf("FileActions count = %d, want 1", len(summary.FileActions))
	}
	fileAction := summary.FileActions[0]
	if fileAction.PluginID != "bridge.plugin" || fileAction.ID != "bridge.file.action" || fileAction.Label != "Bridge File Action" || fileAction.Handler != "bridge.command" {
		t.Fatalf("file action = %+v", fileAction)
	}
	if len(summary.NoteActions) != 1 {
		t.Fatalf("NoteActions count = %d, want 1", len(summary.NoteActions))
	}
	noteAction := summary.NoteActions[0]
	if noteAction.PluginID != "bridge.plugin" || noteAction.ID != "bridge.note.action" || noteAction.Label != "Bridge Note Action" || noteAction.Handler != "bridge.command" {
		t.Fatalf("note action = %+v", noteAction)
	}
	if len(summary.ContextMenuEntries) != 1 {
		t.Fatalf("ContextMenuEntries count = %d, want 1", len(summary.ContextMenuEntries))
	}
	menuEntry := summary.ContextMenuEntries[0]
	if menuEntry.PluginID != "bridge.plugin" || menuEntry.ID != "bridge.file.context" || menuEntry.Context != "file" || menuEntry.Handler != "bridge.command" {
		t.Fatalf("context menu entry = %+v", menuEntry)
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

func TestPluginSecretsRequirePermissionsAndUnlock(t *testing.T) {
	app := newBridgeTestApp(t)

	status, errStr := app.PluginSecretsStatus("secrets.plugin")
	if errStr != "" {
		t.Fatalf("PluginSecretsStatus: %s", errStr)
	}
	if status["unlocked"] == true {
		t.Fatalf("new secret session should be locked: %+v", status)
	}
	if status["initialized"] == true {
		t.Fatalf("new secret session should not be initialized: %+v", status)
	}

	if errStr := app.PluginSecretsUnlock("no.storage", "master password"); !strings.Contains(errStr, "secrets.read") {
		t.Fatalf("PluginSecretsUnlock err = %q, want secrets.read permission error", errStr)
	}
	if _, errStr := app.PluginSecretsList("secrets.plugin"); !strings.Contains(errStr, "locked") {
		t.Fatalf("PluginSecretsList before unlock err = %q, want locked", errStr)
	}

	if errStr := app.PluginSecretsUnlock("secrets.plugin", "123123"); !strings.Contains(errStr, "at least 8 characters") {
		t.Fatalf("weak PluginSecretsUnlock err = %q, want minimum length error", errStr)
	}

	if errStr := app.PluginSecretsUnlock("secrets.plugin", "master password"); errStr != "" {
		t.Fatalf("PluginSecretsUnlock: %s", errStr)
	}
	status, errStr = app.PluginSecretsStatus("secrets.plugin")
	if errStr != "" {
		t.Fatalf("PluginSecretsStatus after unlock: %s", errStr)
	}
	if status["unlocked"] != true {
		t.Fatalf("secret session not unlocked: %+v", status)
	}
	if status["initialized"] != true {
		t.Fatalf("secret session not initialized: %+v", status)
	}

	writeResult, errStr := app.PluginSecretsWrite("secrets.plugin", map[string]interface{}{
		"id":       "client-a.database",
		"title":    "Client A Database",
		"value":    "workspace-secret-value",
		"username": "app",
		"scope": map[string]interface{}{
			"kind":              "workspace",
			"workspaceRootPath": "ClientA",
		},
	})
	if errStr != "" {
		t.Fatalf("PluginSecretsWrite: %s", errStr)
	}
	if writeResult["id"] != "client-a.database" || writeResult["value"] != nil {
		t.Fatalf("write result = %+v", writeResult)
	}

	list, errStr := app.PluginSecretsList("secrets.plugin")
	if errStr != "" {
		t.Fatalf("PluginSecretsList: %s", errStr)
	}
	if len(list) != 1 {
		t.Fatalf("PluginSecretsList len = %d, want 1: %+v", len(list), list)
	}
	if list[0]["value"] != nil {
		t.Fatalf("PluginSecretsList leaked value: %+v", list[0])
	}
	scope, _ := list[0]["scope"].(map[string]interface{})
	if scope["kind"] != "workspace" || scope["workspaceRootPath"] != "ClientA" {
		t.Fatalf("list scope = %+v", scope)
	}

	readResult, errStr := app.PluginSecretsRead("secrets.plugin", "client-a.database")
	if errStr != "" {
		t.Fatalf("PluginSecretsRead: %s", errStr)
	}
	if readResult["value"] != "workspace-secret-value" {
		t.Fatalf("read result = %+v", readResult)
	}

	link, errStr := app.PluginSecretsCopyLink("secrets.plugin", "client-a.database")
	if errStr != "" {
		t.Fatalf("PluginSecretsCopyLink: %s", errStr)
	}
	if link != "[Client A Database](verstak-secret://client-a.database)" {
		t.Fatalf("link = %q", link)
	}

	if errStr := app.PluginSecretsDelete("secrets.plugin", "client-a.database"); errStr != "" {
		t.Fatalf("PluginSecretsDelete: %s", errStr)
	}
	list, errStr = app.PluginSecretsList("secrets.plugin")
	if errStr != "" {
		t.Fatalf("PluginSecretsList after delete: %s", errStr)
	}
	if len(list) != 0 {
		t.Fatalf("PluginSecretsList after delete = %+v, want empty", list)
	}
}

func TestPluginSecretsRejectWrongMasterPasswordAcrossSessions(t *testing.T) {
	app := newBridgeTestApp(t)
	if errStr := app.PluginSecretsUnlock("secrets.plugin", "master password"); errStr != "" {
		t.Fatalf("PluginSecretsUnlock first: %s", errStr)
	}

	next := newBridgeTestApp(t)
	next.vault = app.vault
	if errStr := next.PluginSecretsUnlock("secrets.plugin", "wrong password"); !strings.Contains(errStr, "invalid master password") {
		t.Fatalf("wrong unlock err = %q, want invalid master password", errStr)
	}
	if errStr := next.PluginSecretsUnlock("secrets.plugin", "master password"); errStr != "" {
		t.Fatalf("PluginSecretsUnlock correct: %s", errStr)
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

func TestPluginSyncBridgeRequiresDeclaredPermissions(t *testing.T) {
	app := newBridgeTestApp(t)
	app.plugins = append(app.plugins,
		plugin.Plugin{
			Manifest: plugin.Manifest{
				ID:          "sync.local",
				Name:        "Sync Local",
				Version:     "1.0.0",
				Provides:    []string{"sync/local/v1"},
				Permissions: []string{"sync.participate"},
			},
			Status:  plugin.StatusLoaded,
			Enabled: true,
		},
	)

	status, errStr := app.PluginSyncStatus("sync.local")
	if errStr != "" {
		t.Fatalf("PluginSyncStatus: %s", errStr)
	}
	if status == nil {
		t.Fatal("PluginSyncStatus returned nil status")
	}

	if _, errStr := app.PluginSyncStatus("no.storage"); !strings.Contains(errStr, "sync.participate") {
		t.Fatalf("PluginSyncStatus err = %q, want sync.participate permission error", errStr)
	}
	if _, errStr := app.PluginSyncNow("sync.local"); !strings.Contains(errStr, "network.remote") {
		t.Fatalf("PluginSyncNow err = %q, want network.remote permission error", errStr)
	}

	app.syncSvc = syncsvc.NewService(app.vaultPath(), "local-device")
	app.appSettings = appsettings.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err := app.appSettings.Load(); err != nil {
		t.Fatalf("settings Load: %v", err)
	}
	cfg := app.appSettings.Get()
	cfg.Sync.Enabled = true
	cfg.Sync.ServerURL = "https://sync.example.test"
	cfg.Sync.DeviceID = "device-1"
	cfg.Sync.DeviceName = "test-device"
	cfg.Sync.LastStatus = "connected"
	if err := app.appSettings.UpdateSync(cfg.Sync); err != nil {
		t.Fatalf("settings UpdateSync: %v", err)
	}
	if err := syncsvc.SaveDeviceToken(app.vaultPath(), "secret-token"); err != nil {
		t.Fatalf("SaveDeviceToken: %v", err)
	}

	if errStr := app.PluginSyncResetKey("sync.local"); errStr != "" {
		t.Fatalf("PluginSyncResetKey: %s", errStr)
	}
	if token := syncsvc.LoadDeviceToken(app.vaultPath()); token != "" {
		t.Fatalf("device token = %q, want cleared", token)
	}
	cfg = app.appSettings.Get()
	if cfg.Sync.DeviceID != "" || cfg.Sync.DeviceName != "" || cfg.Sync.LastStatus != "disconnected" {
		t.Fatalf("sync settings after reset = %#v, want cleared device and disconnected status", cfg.Sync)
	}
	if cfg.Sync.ServerURL != "https://sync.example.test" {
		t.Fatalf("server URL = %q, want preserved", cfg.Sync.ServerURL)
	}
	status, errStr = app.PluginSyncStatus("sync.local")
	if errStr != "" {
		t.Fatalf("PluginSyncStatus after reset: %s", errStr)
	}
	if status.Configured || status.ServerURL != "https://sync.example.test" || status.StatusLabel != "disconnected" {
		t.Fatalf("status after reset = %#v, want server preserved, not configured, disconnected", status)
	}
	if errStr := app.PluginSyncResetKey("no.storage"); !strings.Contains(errStr, "sync.participate") {
		t.Fatalf("PluginSyncResetKey err = %q, want sync.participate permission error", errStr)
	}
}

func TestPluginBrowserReceiverPairingRequiresPermissionAndRotatesToken(t *testing.T) {
	app := newBridgeTestApp(t)
	app.appSettings = appsettings.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err := app.appSettings.Load(); err != nil {
		t.Fatalf("settings Load: %v", err)
	}
	initialToken, err := app.appSettings.EnsureBrowserReceiverToken()
	if err != nil {
		t.Fatalf("EnsureBrowserReceiverToken: %v", err)
	}
	app.browserReceiver = browserreceiver.NewWithOptions(app.eventBus, browserreceiver.Options{
		RequireToken:  true,
		ReceiverToken: initialToken,
	})
	app.plugins = append(app.plugins, plugin.Plugin{
		Manifest: plugin.Manifest{
			ID:          "browser.local",
			Name:        "Browser Local",
			Version:     "1.0.0",
			Permissions: []string{"browser.receiver.manage"},
		},
		Status:  plugin.StatusLoaded,
		Enabled: true,
	})

	pairing, errStr := app.PluginBrowserReceiverPairing("browser.local")
	if errStr != "" {
		t.Fatalf("PluginBrowserReceiverPairing: %s", errStr)
	}
	if pairing["receiverToken"] != initialToken {
		t.Fatalf("pairing token = %q, want %q", pairing["receiverToken"], initialToken)
	}
	if pairing["receiverUrl"] != browserreceiver.DefaultCaptureURL {
		t.Fatalf("pairing receiver URL = %q, want %q", pairing["receiverUrl"], browserreceiver.DefaultCaptureURL)
	}
	if _, errStr := app.PluginBrowserReceiverPairing("no.storage"); !strings.Contains(errStr, "browser.receiver.manage") {
		t.Fatalf("missing permission error = %q", errStr)
	}

	rotated, errStr := app.PluginRotateBrowserReceiverToken("browser.local")
	if errStr != "" {
		t.Fatalf("PluginRotateBrowserReceiverToken: %s", errStr)
	}
	if rotated["receiverToken"] == "" || rotated["receiverToken"] == initialToken {
		t.Fatalf("rotated pairing token = %q, want new non-empty token", rotated["receiverToken"])
	}
}

func TestPluginSyncStatusReportsPersistedError(t *testing.T) {
	app := newBridgeTestApp(t)
	app.plugins = append(app.plugins,
		plugin.Plugin{
			Manifest: plugin.Manifest{
				ID:          "sync.local",
				Name:        "Sync Local",
				Version:     "1.0.0",
				Provides:    []string{"sync/local/v1"},
				Permissions: []string{"sync.participate"},
			},
			Status:  plugin.StatusLoaded,
			Enabled: true,
		},
	)
	app.syncSvc = syncsvc.NewService(app.vaultPath(), "local-device")
	app.appSettings = appsettings.NewManager(filepath.Join(t.TempDir(), "config.json"))
	if err := app.appSettings.Load(); err != nil {
		t.Fatalf("settings Load: %v", err)
	}
	if err := app.syncSvc.SetState("https://sync.example.test", ""); err != nil {
		t.Fatalf("SetState: %v", err)
	}
	cfg := app.appSettings.Get()
	cfg.Sync.Enabled = true
	cfg.Sync.ServerURL = "https://sync.example.test"
	cfg.Sync.DeviceID = "device-1"
	cfg.Sync.LastStatus = "error"
	cfg.Sync.LastError = "push: server unavailable"
	if err := app.appSettings.UpdateSync(cfg.Sync); err != nil {
		t.Fatalf("settings UpdateSync: %v", err)
	}
	if err := syncsvc.SaveDeviceToken(app.vaultPath(), "secret-token"); err != nil {
		t.Fatalf("SaveDeviceToken: %v", err)
	}

	status, errStr := app.PluginSyncStatus("sync.local")
	if errStr != "" {
		t.Fatalf("PluginSyncStatus: %s", errStr)
	}
	if status.StatusLabel != "error" || status.LastError != "push: server unavailable" {
		t.Fatalf("status = %#v, want persisted sync error", status)
	}
	cfg = app.appSettings.Get()
	if cfg.Sync.LastStatus != "error" || cfg.Sync.LastError != "push: server unavailable" {
		t.Fatalf("persisted sync settings = %#v, want error preserved", cfg.Sync)
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

func TestSubscribePluginEventRegistersBackendEventBridge(t *testing.T) {
	app := newBridgeTestApp(t)
	emitted := make(chan map[string]interface{}, 1)
	originalEmit := emitFrontendEvent
	emitFrontendEvent = func(_ context.Context, eventName string, data ...interface{}) {
		if eventName != pluginEventRuntimeName {
			t.Errorf("eventName = %q, want %q", eventName, pluginEventRuntimeName)
		}
		if len(data) != 1 {
			t.Errorf("data length = %d, want 1", len(data))
			return
		}
		payload, ok := data[0].(map[string]interface{})
		if !ok {
			t.Errorf("data[0] type = %T, want map[string]interface{}", data[0])
			return
		}
		emitted <- payload
	}
	t.Cleanup(func() {
		emitFrontendEvent = originalEmit
	})

	if errStr := app.SubscribePluginEvent("bridge.plugin", "browser.capture.page"); errStr != "" {
		t.Fatalf("SubscribePluginEvent: %s", errStr)
	}
	if !app.eventBus.HasSubscribers("browser.capture.page") {
		t.Fatal("expected backend event bus subscriber")
	}

	app.eventBus.Publish(events.Event{
		Name:      "browser.capture.page",
		Timestamp: "2026-06-27T00:00:00.000Z",
		Payload:   map[string]interface{}{"url": "https://example.com"},
	})

	event := <-emitted
	if event["name"] != "browser.capture.page" {
		t.Fatalf("event name = %v, want browser.capture.page", event["name"])
	}
	if event["timestamp"] != "2026-06-27T00:00:00.000Z" {
		t.Fatalf("event timestamp = %v, want documented timestamp", event["timestamp"])
	}
	payload, ok := event["payload"].(map[string]interface{})
	if !ok {
		t.Fatalf("event payload type = %T, want map[string]interface{}", event["payload"])
	}
	if payload["url"] != "https://example.com" {
		t.Fatalf("payload url = %v, want https://example.com", payload["url"])
	}
}
