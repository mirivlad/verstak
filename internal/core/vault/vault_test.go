package vault

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/verstak/verstak-desktop/internal/core/events"
)

func TestCreateVault_CreatesLayoutAndMeta(t *testing.T) {
	base := t.TempDir()
	bus := events.NewBus()
	v := NewVault(bus)

	err := v.CreateVault(base)
	if err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	vaultDir := filepath.Join(base, "VerstakVault")

	// Check vault.json exists
	metaPath := filepath.Join(vaultDir, ".verstak", "vault.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("vault.json not found: %v", err)
	}

	var meta VaultMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("failed to parse vault.json: %v", err)
	}

	if meta.SchemaVersion != 1 {
		t.Errorf("schemaVersion: got %d, want 1", meta.SchemaVersion)
	}
	if meta.VaultID == "" {
		t.Error("vaultId is empty")
	}
	if meta.App != "verstak" {
		t.Errorf("app: got %q, want %q", meta.App, "verstak")
	}

	// Check subdirectories
	expectedDirs := []string{
		".verstak/plugin-data",
		".verstak/plugin-settings",
		".verstak/plugin-cache",
		".verstak/trash",
		".verstak/logs",
	}
	for _, dir := range expectedDirs {
		full := filepath.Join(vaultDir, dir)
		info, err := os.Stat(full)
		if err != nil {
			t.Errorf("directory %s not found: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

func TestOpenVault_ReadsExistingVaultId(t *testing.T) {
	base := t.TempDir()
	bus := events.NewBus()
	v := NewVault(bus)

	// Create vault
	if err := v.CreateVault(base); err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	// Remember the vault ID
	meta := v.GetVaultMeta()
	if meta == nil {
		t.Fatal("meta is nil after CreateVault")
	}
	originalID := meta.VaultID

	// Close vault
	v.CloseVault()

	// Open vault
	vaultDir := filepath.Join(base, "VerstakVault")
	if err := v.OpenVault(vaultDir); err != nil {
		t.Fatalf("OpenVault failed: %v", err)
	}

	// Verify vault ID matches
	newMeta := v.GetVaultMeta()
	if newMeta == nil {
		t.Fatal("meta is nil after OpenVault")
	}
	if newMeta.VaultID != originalID {
		t.Errorf("vault ID mismatch: got %q, want %q", newMeta.VaultID, originalID)
	}
}

func TestOpenVault_CorruptJSON_Error(t *testing.T) {
	base := t.TempDir()
	vaultDir := filepath.Join(base, "VerstakVault")

	// Create vault directory with corrupt vault.json
	if err := os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755); err != nil {
		t.Fatalf("failed to create .verstak dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, ".verstak", "vault.json"), []byte("{corrupt"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt vault.json: %v", err)
	}

	bus := events.NewBus()
	v := NewVault(bus)

	err := v.OpenVault(vaultDir)
	if err == nil {
		t.Fatal("expected error for corrupt vault.json, got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse vault.json") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveSafePath_BlocksTraversal(t *testing.T) {
	base := t.TempDir()
	bus := events.NewBus()
	v := NewVault(bus)

	if err := v.CreateVault(base); err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	// Path traversal should be blocked
	_, err := v.ResolveSafePath("../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "path traversal detected") {
		t.Errorf("unexpected error: %v", err)
	}

	// Normal path should work
	result, err := v.ResolveSafePath("normal/path")
	if err != nil {
		t.Fatalf("ResolveSafePath failed for normal path: %v", err)
	}
	if !strings.Contains(result, "normal") {
		t.Errorf("unexpected result path: %s", result)
	}
}

func TestGetPluginDataPath_CreatesNamespace(t *testing.T) {
	base := t.TempDir()
	bus := events.NewBus()
	v := NewVault(bus)

	if err := v.CreateVault(base); err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	path := v.GetPluginDataPath("test-plugin")
	if !strings.Contains(path, "plugin-data") {
		t.Errorf("path does not contain plugin-data: %s", path)
	}
	if !strings.Contains(path, "test-plugin") {
		t.Errorf("path does not contain test-plugin: %s", path)
	}

	// Verify directory was created
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("plugin data directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("plugin data path is not a directory")
	}
}

func TestVaultStatus_Transitions(t *testing.T) {
	base := t.TempDir()
	bus := events.NewBus()
	v := NewVault(bus)

	// New vault → not-created
	if status := v.GetVaultStatus(); status != StatusNotCreated {
		t.Errorf("initial status: got %q, want %q", status, StatusNotCreated)
	}

	// Create → open
	if err := v.CreateVault(base); err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}
	if status := v.GetVaultStatus(); status != StatusOpen {
		t.Errorf("after create: got %q, want %q", status, StatusOpen)
	}

	// Close → closed
	v.CloseVault()
	if status := v.GetVaultStatus(); status != StatusClosed {
		t.Errorf("after close: got %q, want %q", status, StatusClosed)
	}
}

func TestVaultEvents_Published(t *testing.T) {
	base := t.TempDir()
	bus := events.NewBus()
	v := NewVault(bus)

	// Collect events
	var published []string
	bus.Subscribe("vault.created", func(e events.Event) {
		published = append(published, e.Name)
	})
	bus.Subscribe("vault.opened", func(e events.Event) {
		published = append(published, e.Name)
	})
	bus.Subscribe("vault.closed", func(e events.Event) {
		published = append(published, e.Name)
	})

	// Create
	if err := v.CreateVault(base); err != nil {
		t.Fatalf("CreateVault failed: %v", err)
	}

	// Close
	v.CloseVault()

	// Open
	vaultDir := filepath.Join(base, "VerstakVault")
	if err := v.OpenVault(vaultDir); err != nil {
		t.Fatalf("OpenVault failed: %v", err)
	}

	// Verify events
	expected := []string{"vault.created", "vault.closed", "vault.opened"}
	if len(published) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(published), published)
	}
	for i, name := range expected {
		if published[i] != name {
			t.Errorf("event %d: got %q, want %q", i, published[i], name)
		}
	}
}

func TestCreateVault_CreatesWorkspace(t *testing.T) {
	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "testvault")

	bus := events.NewBus()
	v := NewVault(bus)

	if err := v.CreateVault(vaultPath); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	wsPath := filepath.Join(v.GetVaultPath(), ".verstak", "workspace.json")
	data, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatalf("workspace.json not found: %v", err)
	}

	var ws struct {
		SchemaVersion int `json:"schemaVersion"`
		Nodes         []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Title    string `json:"title"`
			Status   string `json:"status"`
			ParentID string `json:"parentId"`
		} `json:"nodes"`
		CurrentNodeID string `json:"currentNodeId"`
	}
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("failed to parse workspace.json: %v", err)
	}

	if ws.SchemaVersion != 1 {
		t.Errorf("schemaVersion: got %d, want 1", ws.SchemaVersion)
	}
	if len(ws.Nodes) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(ws.Nodes))
	}
	if ws.Nodes[0].Type != "space" {
		t.Errorf("root type: got %q, want %q", ws.Nodes[0].Type, "space")
	}
	if ws.Nodes[0].Title != "My Workspace" {
		t.Errorf("root title: got %q, want %q", ws.Nodes[0].Title, "My Workspace")
	}
	if ws.Nodes[0].Status != "active" {
		t.Errorf("root status: got %q, want %q", ws.Nodes[0].Status, "active")
	}
	if ws.CurrentNodeID != ws.Nodes[0].ID {
		t.Errorf("currentNodeId should be root node id")
	}
}

func TestOpenVault_WorkspaceLoads(t *testing.T) {
	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "testvault")

	bus := events.NewBus()
	v := NewVault(bus)

	if err := v.CreateVault(vaultPath); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	v.CloseVault()

	if err := v.OpenVault(vaultPath); err != nil {
		t.Fatalf("OpenVault: %v", err)
	}

	wsPath := filepath.Join(v.GetVaultPath(), ".verstak", "workspace.json")
	data, err := os.ReadFile(wsPath)
	if err != nil {
		t.Fatalf("workspace.json not found after reopen: %v", err)
	}

	var ws struct {
		Nodes []struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Title string `json:"title"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(data, &ws); err != nil {
		t.Fatalf("failed to parse workspace.json: %v", err)
	}
	if len(ws.Nodes) != 1 {
		t.Fatalf("expected 1 node after reopen, got %d", len(ws.Nodes))
	}
	if ws.Nodes[0].Type != "space" {
		t.Errorf("root type after reopen: got %q, want %q", ws.Nodes[0].Type, "space")
	}
}

func TestCreateVault_VaultPathNormalized(t *testing.T) {
	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "testvault")

	bus := events.NewBus()
	v := NewVault(bus)

	if err := v.CreateVault(vaultPath); err != nil {
		t.Fatalf("CreateVault: %v", err)
	}

	expectedPath := filepath.Join(vaultPath, "VerstakVault")
	if v.GetVaultPath() != expectedPath {
		t.Errorf("GetVaultPath: got %q, want %q", v.GetVaultPath(), expectedPath)
	}
}
