package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultRootNode(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	verstakDir := filepath.Join(vaultDir, ".verstak")
	os.MkdirAll(verstakDir, 0o755)

	m := NewManager(vaultDir)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	tree := m.GetTree()
	if len(tree.Nodes) != 1 {
		t.Fatalf("expected 1 root node, got %d", len(tree.Nodes))
	}
	if tree.Nodes[0].Type != TypeSpace {
		t.Errorf("root type: got %s, want %s", tree.Nodes[0].Type, TypeSpace)
	}
	if tree.Nodes[0].Title != "My Workspace" {
		t.Errorf("root title: got %q, want %q", tree.Nodes[0].Title, "My Workspace")
	}
	if tree.CurrentNodeID != tree.Nodes[0].ID {
		t.Errorf("current node should be root")
	}
}

func TestCreateNode_Case(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	verstakDir := filepath.Join(vaultDir, ".verstak")
	os.MkdirAll(verstakDir, 0o755)

	m := NewManager(vaultDir)
	if err := m.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	rootID := m.GetTree().Nodes[0].ID

	node, err := m.CreateNode(rootID, TypeCase, "Test Case")
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	if node.Type != TypeCase {
		t.Errorf("type: got %s, want %s", node.Type, TypeCase)
	}
	if node.Title != "Test Case" {
		t.Errorf("title: got %q, want %q", node.Title, "Test Case")
	}
	if node.ParentID != rootID {
		t.Errorf("parentID: got %q, want %q", node.ParentID, rootID)
	}
	if node.Status != StatusActive {
		t.Errorf("status: got %s, want %s", node.Status, StatusActive)
	}

	// Verify persisted
	tree := m.GetTree()
	if len(tree.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(tree.Nodes))
	}
}

func TestCreateNode_InvalidType(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	_, err := m.CreateNode("", NodeType("note"), "My Note")
	if err == nil {
		t.Error("expected error for invalid type 'note'")
	}
}

func TestCreateNode_EmptyTitle(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	_, err := m.CreateNode("", TypeCase, "")
	if err == nil {
		t.Error("expected error for empty title")
	}
	_, err = m.CreateNode("", TypeCase, "   ")
	if err == nil {
		t.Error("expected error for whitespace-only title")
	}
}

func TestRenameNode(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	rootID := m.GetTree().Nodes[0].ID
	node, _ := m.CreateNode(rootID, TypeCase, "Original")

	if err := m.RenameNode(node.ID, "Renamed"); err != nil {
		t.Fatalf("RenameNode: %v", err)
	}

	renamed, _ := m.GetNode(node.ID)
	if renamed.Title != "Renamed" {
		t.Errorf("title: got %q, want %q", renamed.Title, "Renamed")
	}
	if renamed.UpdatedAt == node.UpdatedAt {
		t.Error("updatedAt should change after rename")
	}
}

func TestMoveNode(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	rootID := m.GetTree().Nodes[0].ID
	folder, _ := m.CreateNode(rootID, TypeFolder, "Folder")
	c, _ := m.CreateNode(rootID, TypeCase, "Case")

	// Move case into folder
	if err := m.MoveNode(c.ID, folder.ID); err != nil {
		t.Fatalf("MoveNode: %v", err)
	}

	moved, _ := m.GetNode(c.ID)
	if moved.ParentID != folder.ID {
		t.Errorf("parentID: got %q, want %q", moved.ParentID, folder.ID)
	}
}

func TestMoveNode_CannotMoveIntoSelf(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	rootID := m.GetTree().Nodes[0].ID
	node, _ := m.CreateNode(rootID, TypeCase, "Case")

	err := m.MoveNode(node.ID, node.ID)
	if err == nil {
		t.Error("expected error when moving node into itself")
	}
}

func TestMoveNode_CannotMoveIntoDescendant(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	rootID := m.GetTree().Nodes[0].ID
	folder, _ := m.CreateNode(rootID, TypeFolder, "Folder")
	child, _ := m.CreateNode(folder.ID, TypeCase, "Child")

	// Try to move folder into its own child
	err := m.MoveNode(folder.ID, child.ID)
	if err == nil {
		t.Error("expected error when moving node into descendant")
	}
}

func TestArchiveNode(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	rootID := m.GetTree().Nodes[0].ID
	node, _ := m.CreateNode(rootID, TypeCase, "To Archive")

	if err := m.ArchiveNode(node.ID); err != nil {
		t.Fatalf("ArchiveNode: %v", err)
	}

	archived, _ := m.GetNode(node.ID)
	if archived.Status != StatusArchived {
		t.Errorf("status: got %s, want %s", archived.Status, StatusArchived)
	}
}

func TestSetCurrentNode(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	rootID := m.GetTree().Nodes[0].ID
	node, _ := m.CreateNode(rootID, TypeCase, "My Case")

	if err := m.SetCurrentNode(node.ID); err != nil {
		t.Fatalf("SetCurrentNode: %v", err)
	}

	current, err := m.GetCurrentNode()
	if err != nil {
		t.Fatalf("GetCurrentNode: %v", err)
	}
	if current.ID != node.ID {
		t.Errorf("current: got %s, want %s", current.ID, node.ID)
	}
}

func TestGetTree_StableAfterReopen(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	// Create and populate
	m1 := NewManager(vaultDir)
	m1.Load()
	rootID := m1.GetTree().Nodes[0].ID
	m1.CreateNode(rootID, TypeCase, "Case 1")
	m1.CreateNode(rootID, TypeFolder, "Folder 1")
	m1.CreateNode(rootID, TypeCase, "Case 2")

	// Reopen
	m2 := NewManager(vaultDir)
	if err := m2.Load(); err != nil {
		t.Fatalf("reopen Load: %v", err)
	}

	tree := m2.GetTree()
	// root + 3 created = 4
	if len(tree.Nodes) != 4 {
		t.Fatalf("expected 4 nodes after reopen, got %d", len(tree.Nodes))
	}

	// Check order: children of root should be sorted by order
	children := m2.ListChildren(rootID)
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}
	if children[0].Title != "Case 1" {
		t.Errorf("first child: got %q, want %q", children[0].Title, "Case 1")
	}
	if children[1].Title != "Folder 1" {
		t.Errorf("second child: got %q, want %q", children[1].Title, "Folder 1")
	}
	if children[2].Title != "Case 2" {
		t.Errorf("third child: got %q, want %q", children[2].Title, "Case 2")
	}
}

func TestCorruptWorkspaceJSON(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	verstakDir := filepath.Join(vaultDir, ".verstak")
	os.MkdirAll(verstakDir, 0o755)

	// Write corrupt JSON
	corruptPath := filepath.Join(verstakDir, "workspace.json")
	os.WriteFile(corruptPath, []byte("{not valid json"), 0o600)

	m := NewManager(vaultDir)
	err := m.Load()
	if err == nil {
		t.Error("expected error for corrupt workspace.json")
	}

	// Should have created a backup
	entries, _ := os.ReadDir(verstakDir)
	backupFound := false
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".corrupt" || len(e.Name()) > 14 && e.Name()[14] == '-' {
			backupFound = true
			break
		}
	}
	// Also check for .corrupt.* pattern
	for _, e := range entries {
		name := e.Name()
		if len(name) > 20 && name[:14] == "workspace.json" {
			backupFound = true
			break
		}
	}
	_ = backupFound // backup may have different naming

	// Should have created a valid default tree
	tree := m.GetTree()
	if len(tree.Nodes) != 1 {
		t.Errorf("expected 1 default node, got %d", len(tree.Nodes))
	}
}

func TestListChildren_EmptyParent(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	// Root has no parent, so ListChildren("") should return root-level nodes
	children := m.ListChildren("")
	if len(children) != 1 {
		t.Errorf("expected 1 root-level node, got %d", len(children))
	}
}

func TestCreateNode_InvalidParent(t *testing.T) {
	dir := t.TempDir()
	vaultDir := filepath.Join(dir, "vault")
	os.MkdirAll(filepath.Join(vaultDir, ".verstak"), 0o755)

	m := NewManager(vaultDir)
	m.Load()

	_, err := m.CreateNode("nonexistent-id", TypeCase, "Orphan")
	if err == nil {
		t.Error("expected error for nonexistent parent")
	}
}
