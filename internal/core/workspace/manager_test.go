package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestListWorkspacesReadsTopLevelPhysicalFolders(t *testing.T) {
	vaultDir := newVaultDir(t)
	mustMkdir(t, filepath.Join(vaultDir, "Project"))
	mustMkdir(t, filepath.Join(vaultDir, "Test"))
	mustMkdir(t, filepath.Join(vaultDir, ".verstak"))
	mustMkdir(t, filepath.Join(vaultDir, ".git"))
	mustWrite(t, filepath.Join(vaultDir, "readme.md"), "not a workspace")

	m := NewManager(vaultDir)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	workspaces, err := m.ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}

	got := workspaceNames(workspaces)
	want := []string{"Project", "Test"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("workspaces = %v, want %v", got, want)
	}
}

func TestListWorkspacesExcludesTopLevelSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs extra privileges on Windows")
	}
	vaultDir := newVaultDir(t)
	target := filepath.Join(t.TempDir(), "outside")
	mustMkdir(t, target)
	if err := os.Symlink(target, filepath.Join(vaultDir, "Linked")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	m := NewManager(vaultDir)
	workspaces, err := m.ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(workspaces) != 0 {
		t.Fatalf("expected symlink workspace to be excluded, got %+v", workspaces)
	}
}

func TestLoadDoesNotCreateOrMigrateFoldersFromOldWorkspaceJSON(t *testing.T) {
	vaultDir := newVaultDir(t)
	mustMkdir(t, filepath.Join(vaultDir, ".verstak"))
	oldTree := `{"schemaVersion":1,"nodes":[{"id":"old","type":"space","title":"Old Tree Workspace","path":"Old Tree Workspace"}],"currentNodeId":"old"}`
	mustWrite(t, filepath.Join(vaultDir, ".verstak", "workspace.json"), oldTree)

	m := NewManager(vaultDir)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Old Tree Workspace")); !os.IsNotExist(err) {
		t.Fatalf("Load created folder from old workspace.json, stat err=%v", err)
	}

	workspaces, err := m.ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(workspaces) != 0 {
		t.Fatalf("workspace.json tree should not be source of truth, got %+v", workspaces)
	}
}

func TestCreateWorkspaceCreatesFolderDefaultTemplateAndMetadataSnapshot(t *testing.T) {
	vaultDir := newVaultDir(t)
	m := NewManager(vaultDir)

	ws, err := m.CreateWorkspace("Project", "")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	if ws.Name != "Project" || ws.RootPath != "Project" {
		t.Fatalf("workspace = %+v, want Project root", ws)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Project")); err != nil {
		t.Fatalf("workspace folder missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Project", "Notes")); err != nil {
		t.Fatalf("default template notes folder missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Project", "Notes", "Overview.md")); !os.IsNotExist(err) {
		t.Fatalf("default template should not create overview file, stat err=%v", err)
	}

	meta, err := m.GetWorkspaceMetadata("Project")
	if err != nil {
		t.Fatalf("GetWorkspaceMetadata: %v", err)
	}
	if meta.WorkspaceName != "Project" {
		t.Fatalf("metadata workspaceName = %q", meta.WorkspaceName)
	}
	if meta.CreatedFromTemplate == nil {
		t.Fatal("metadata missing createdFromTemplate snapshot")
	}
	if meta.CreatedFromTemplate.TemplateID != "default" || meta.CreatedFromTemplate.TemplateName == "" || meta.CreatedFromTemplate.TemplateVersion == 0 || meta.CreatedFromTemplate.AppliedAt == "" {
		t.Fatalf("bad template snapshot: %+v", meta.CreatedFromTemplate)
	}
	if !meta.Features["files"] || !meta.Features["notes"] {
		t.Fatalf("features = %+v, want files and notes enabled", meta.Features)
	}
	if meta.Folders["notes"] != "Notes" {
		t.Fatalf("folders = %+v, want notes folder", meta.Folders)
	}
}

func TestWorkspaceMetadataDoesNotRequireLiveTemplate(t *testing.T) {
	vaultDir := newVaultDir(t)
	m := NewManager(vaultDir)
	if _, err := m.CreateWorkspace("ClientA", "client-project"); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}

	ClearTemplateRegistryForTest(t)

	meta, err := m.GetWorkspaceMetadata("ClientA")
	if err != nil {
		t.Fatalf("GetWorkspaceMetadata after registry clear: %v", err)
	}
	if meta.CreatedFromTemplate == nil || meta.CreatedFromTemplate.TemplateID != "client-project" {
		t.Fatalf("snapshot not preserved after registry clear: %+v", meta.CreatedFromTemplate)
	}
}

func TestMissingMetadataReturnsGenericWorkspaceMetadata(t *testing.T) {
	vaultDir := newVaultDir(t)
	mustMkdir(t, filepath.Join(vaultDir, "Loose"))

	m := NewManager(vaultDir)
	meta, err := m.GetWorkspaceMetadata("Loose")
	if err != nil {
		t.Fatalf("GetWorkspaceMetadata: %v", err)
	}
	if meta.WorkspaceName != "Loose" {
		t.Fatalf("workspaceName = %q", meta.WorkspaceName)
	}
	if meta.CreatedFromTemplate != nil {
		t.Fatalf("generic metadata should not invent a template snapshot: %+v", meta.CreatedFromTemplate)
	}
	if !meta.Features["files"] {
		t.Fatalf("generic metadata should enable files at minimum: %+v", meta.Features)
	}
}

func TestGetWorkspaceMetadataReturnsCanonicalFolderNameWhenStoredNameIsStale(t *testing.T) {
	vaultDir := newVaultDir(t)
	m := NewManager(vaultDir)
	if _, err := m.CreateWorkspace("Project", "default"); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}

	data, err := os.ReadFile(m.metadataPath("Project"))
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	meta.WorkspaceName = "OldName"
	staleData, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := os.WriteFile(m.metadataPath("Project"), staleData, 0o600); err != nil {
		t.Fatalf("write stale metadata: %v", err)
	}

	got, err := m.GetWorkspaceMetadata("Project")
	if err != nil {
		t.Fatalf("GetWorkspaceMetadata: %v", err)
	}
	if got.WorkspaceName != "Project" {
		t.Fatalf("workspaceName = %q, want canonical folder name Project", got.WorkspaceName)
	}
}

func TestRenameWorkspacePhysicallyRenamesFolderAndMetadata(t *testing.T) {
	vaultDir := newVaultDir(t)
	m := NewManager(vaultDir)
	if _, err := m.CreateWorkspace("Project", "default"); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}

	if err := m.RenameWorkspace("Project", "Renamed"); err != nil {
		t.Fatalf("RenameWorkspace: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Project")); !os.IsNotExist(err) {
		t.Fatalf("old folder still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Renamed")); err != nil {
		t.Fatalf("renamed folder missing: %v", err)
	}

	meta, err := m.GetWorkspaceMetadata("Renamed")
	if err != nil {
		t.Fatalf("metadata after rename: %v", err)
	}
	if meta.WorkspaceName != "Renamed" {
		t.Fatalf("metadata workspaceName = %q, want Renamed", meta.WorkspaceName)
	}
	if _, err := os.Stat(m.metadataPath("Project")); !os.IsNotExist(err) {
		t.Fatalf("old metadata key still exists or stat failed unexpectedly: %v", err)
	}
}

func TestTrashWorkspaceMovesFolderToTrashAndRemovesFromList(t *testing.T) {
	vaultDir := newVaultDir(t)
	m := NewManager(vaultDir)
	if _, err := m.CreateWorkspace("Project", "default"); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}

	result, err := m.TrashWorkspace("Project")
	if err != nil {
		t.Fatalf("TrashWorkspace: %v", err)
	}
	if result.OriginalPath != "Project" || result.TrashID == "" || result.TrashPath == "" {
		t.Fatalf("bad trash result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, "Project")); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists after trash, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(vaultDir, filepath.FromSlash(result.TrashPath))); err != nil {
		t.Fatalf("trashed workspace missing: %v", err)
	}

	workspaces, err := m.ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(workspaces) != 0 {
		t.Fatalf("workspace should be removed from list after trash, got %+v", workspaces)
	}
}

func TestCreateAndRenameConflictsAreExplicit(t *testing.T) {
	vaultDir := newVaultDir(t)
	mustMkdir(t, filepath.Join(vaultDir, "Existing"))
	mustMkdir(t, filepath.Join(vaultDir, "Other"))
	m := NewManager(vaultDir)

	if _, err := m.CreateWorkspace("Existing", ""); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("create conflict error = %v, want conflict", err)
	}
	if err := m.RenameWorkspace("Existing", "Other"); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("rename conflict error = %v, want conflict", err)
	}
}

func TestInvalidWorkspaceNamesRejected(t *testing.T) {
	vaultDir := newVaultDir(t)
	m := NewManager(vaultDir)

	names := []string{"", "   ", "A/B", `A\B`, "/abs", `C:\abs`, "..", "a..b", "bad\x00name", ".verstak", ".Verstak", ".git"}
	for _, name := range names {
		if _, err := m.CreateWorkspace(name, ""); err == nil {
			t.Fatalf("CreateWorkspace(%q) succeeded, want invalid name error", name)
		}
	}
}

func TestCompatibilityTreeIsDerivedFromTopLevelFolders(t *testing.T) {
	vaultDir := newVaultDir(t)
	mustMkdir(t, filepath.Join(vaultDir, "Project"))
	mustMkdir(t, filepath.Join(vaultDir, "Project", "Nested"))
	mustMkdir(t, filepath.Join(vaultDir, "Test"))

	m := NewManager(vaultDir)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	tree := m.GetTree()
	if len(tree.Nodes) != 2 {
		t.Fatalf("nodes = %+v, want 2 top-level workspaces", tree.Nodes)
	}
	if tree.Nodes[0].ID != "Project" || tree.Nodes[0].Title != "Project" || tree.Nodes[0].Path != "" {
		t.Fatalf("first compatibility node = %+v, want derived workspace without persisted path mapping", tree.Nodes[0])
	}
	for _, node := range tree.Nodes {
		if node.ParentID != "" {
			t.Fatalf("compatibility tree should be flat, got child node %+v", node)
		}
		if node.ID == "Nested" || node.Title == "Nested" {
			t.Fatalf("nested folders must not become workspace nodes: %+v", tree.Nodes)
		}
	}
}

func TestMoveNodeCompatibilityDoesNotCreateNestedWorkspaceModel(t *testing.T) {
	vaultDir := newVaultDir(t)
	mustMkdir(t, filepath.Join(vaultDir, "Project"))
	mustMkdir(t, filepath.Join(vaultDir, "Test"))

	m := NewManager(vaultDir)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	err := m.MoveNode("Project", "Test")
	if err == nil || !strings.Contains(err.Error(), "top-level only") {
		t.Fatalf("MoveNode error = %v, want top-level only", err)
	}
	if _, statErr := os.Stat(filepath.Join(vaultDir, "Test", "Project")); !os.IsNotExist(statErr) {
		t.Fatalf("MoveNode created nested mapped workspace, stat err=%v", statErr)
	}
}

func TestMetadataFileShape(t *testing.T) {
	vaultDir := newVaultDir(t)
	m := NewManager(vaultDir)
	if _, err := m.CreateWorkspace("Project", "default"); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}

	data, err := os.ReadFile(m.metadataPath("Project"))
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("metadata JSON: %v", err)
	}
	if raw["workspaceName"] != "Project" {
		t.Fatalf("workspaceName = %v", raw["workspaceName"])
	}
	if _, ok := raw["createdFromTemplate"].(map[string]interface{}); !ok {
		t.Fatalf("createdFromTemplate missing in raw metadata: %s", data)
	}
}

func newVaultDir(t *testing.T) string {
	t.Helper()
	vaultDir := filepath.Join(t.TempDir(), "vault")
	mustMkdir(t, vaultDir)
	mustMkdir(t, filepath.Join(vaultDir, ".verstak", "trash"))
	return vaultDir
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func workspaceNames(workspaces []Workspace) []string {
	names := make([]string, len(workspaces))
	for i, ws := range workspaces {
		names[i] = ws.Name
	}
	return names
}
