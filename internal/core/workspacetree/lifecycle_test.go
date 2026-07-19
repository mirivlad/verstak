package workspacetree

import (
	"os"
	"path/filepath"
	"testing"
)

func noopRefresh() error { return nil }

func TestCreateFolderInRoot(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()

	f, err := svc.CreateFolder("", "Clients", noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if f.Name != "Clients" || f.Path != "Clients" || f.ParentID != "" {
		t.Fatalf("folder = %+v", f)
	}
	// Verify marker.
	if _, err := os.Stat(filepath.Join(vault, "Clients", ".verstak", "folder.json")); err != nil {
		t.Fatal("marker missing")
	}
}

func TestCreateNestedFolder(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()

	parent, _ := svc.CreateFolder("", "Clients", noopRefresh)
	child, err := svc.CreateFolder(parent.ID, "Active", noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID != parent.ID {
		t.Fatalf("parentID = %q, want %s", child.ParentID, parent.ID)
	}
}

func TestCreateWorkspaceInRoot(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()

	ws, err := svc.CreateWorkspace("", "Project", "default", noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "Project" || ws.RootPath != "Project" {
		t.Fatalf("workspace = %+v", ws)
	}
	if _, err := os.Stat(filepath.Join(vault, "Project", ".verstak", "workspace.json")); err != nil {
		t.Fatal("marker missing")
	}
}

func TestCreateWorkspaceInNestedFolder(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()

	f, _ := svc.CreateFolder("", "Clients", noopRefresh)
	f2, _ := svc.CreateFolder(f.ID, "Active", noopRefresh)
	ws, err := svc.CreateWorkspace(f2.ID, "Client1", "", noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if ws.RootPath != "Clients/Active/Client1" {
		t.Fatalf("RootPath = %q", ws.RootPath)
	}
}

func TestCreateFolderCollisionRejected(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	svc.CreateFolder("", "Test", noopRefresh)
	_, err := svc.CreateFolder("", "Test", noopRefresh)
	if err == nil {
		t.Fatal("expected collision error")
	}
}

func TestRenameFolder(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	f, _ := svc.CreateFolder("", "OldName", noopRefresh)

	updated, err := svc.RenameFolder(f.ID, "NewName", noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "NewName" || updated.ID != f.ID {
		t.Fatalf("renamed = %+v", updated)
	}
}

func TestRenameWorkspace(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	ws, _ := svc.CreateWorkspace("", "OldWS", "", noopRefresh)

	updated, err := svc.RenameWorkspace(ws.ID, "NewWS", noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "NewWS" || updated.ID != ws.ID {
		t.Fatalf("renamed = %+v", updated)
	}
}

func TestMoveFolder(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	src, _ := svc.CreateFolder("", "Src", noopRefresh)
	dst, _ := svc.CreateFolder("", "Dst", noopRefresh)

	moved, err := svc.MoveFolder(src.ID, dst.ID, noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if moved.Path != "Dst/Src" || moved.ID != src.ID {
		t.Fatalf("moved = %+v", moved)
	}
}

func TestMoveWorkspace(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	f, _ := svc.CreateFolder("", "Clients", noopRefresh)
	ws, _ := svc.CreateWorkspace("", "Project", "", noopRefresh)

	moved, err := svc.MoveWorkspace(ws.ID, f.ID, noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if moved.RootPath != "Clients/Project" {
		t.Fatalf("RootPath = %q", moved.RootPath)
	}
}

func TestMoveFolderToDescendantRejected(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	parent, _ := svc.CreateFolder("", "Parent", noopRefresh)
	child, _ := svc.CreateFolder(parent.ID, "Child", noopRefresh)

	_, err := svc.MoveFolder(parent.ID, child.ID, noopRefresh)
	if err == nil {
		t.Fatal("should reject moving into descendant")
	}
}

func TestCaseOnlyRename(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	f, _ := svc.CreateFolder("", "client", noopRefresh)

	updated, err := svc.RenameFolder(f.ID, "Client", noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Client" {
		t.Fatalf("name = %q", updated.Name)
	}
}

func TestTrashAndRestoreFolder(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	f, _ := svc.CreateFolder("", "ToTrash", noopRefresh)

	entry, err := svc.TrashFolder(f.ID, noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	if entry.EntityType != "folder" {
		t.Fatalf("entry = %+v", entry)
	}

	// Folder should be gone from tree.
	if _, ok := svc.GetFolderByID(f.ID); ok {
		t.Fatal("folder should be gone after trash")
	}

	// Restore.
	restored, err := svc.RestoreTreeTrash(entry.TrashID, "", noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	rf := restored.(ScannedFolder)
	if rf.ID != f.ID {
		t.Fatalf("restored ID = %q, want %q", rf.ID, f.ID)
	}
}

func TestTrashAndRestoreWorkspace(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	ws, _ := svc.CreateWorkspace("", "ToTrash", "", noopRefresh)

	entry, err := svc.TrashWorkspace(ws.ID, noopRefresh)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := svc.GetWorkspaceByID(ws.ID); ok {
		t.Fatal("workspace should be gone after trash")
	}

	restored, err := svc.RestoreTreeTrash(entry.TrashID, "", noopRefresh)
	if err != nil {
		t.Fatal(err)
	}
	rw := restored.(ScannedWorkspace)
	if rw.ID != ws.ID {
		t.Fatalf("restored ID = %q, want %q", rw.ID, ws.ID)
	}
}

func TestPurgeTrash(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	f, _ := svc.CreateFolder("", "PurgeMe", noopRefresh)
	entry, _ := svc.TrashFolder(f.ID, noopRefresh)

	if err := svc.PurgeTreeTrash(entry.TrashID); err != nil {
		t.Fatal(err)
	}

	list, _ := svc.ListTreeTrash()
	for _, e := range list {
		if e.TrashID == entry.TrashID {
			t.Fatal("purged entry should be gone")
		}
	}
}

func TestPurgeInvalidTrashID(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()

	if err := svc.PurgeTreeTrash(""); err == nil {
		t.Fatal("empty ID should be rejected")
	}
	if err := svc.PurgeTreeTrash("../escape"); err == nil {
		t.Fatal("path traversal should be rejected")
	}
}

func TestSelectedWorkspaceClearedOnTrash(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	ws, _ := svc.CreateWorkspace("", "Project", "", noopRefresh)
	svc.SetCurrentWorkspaceID(ws.ID)

	if svc.GetCurrentWorkspaceID() != ws.ID {
		t.Fatal("current workspace not set")
	}

	svc.TrashWorkspace(ws.ID, noopRefresh)

	if svc.GetCurrentWorkspaceID() != "" {
		t.Fatal("current workspace should be cleared on trash")
	}
}

func TestRenamePreservesUUID(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	f, _ := svc.CreateFolder("", "X", noopRefresh)
	oldID := f.ID

	updated, _ := svc.RenameFolder(f.ID, "Y", noopRefresh)
	if updated.ID != oldID {
		t.Fatalf("UUID changed: %q → %q", oldID, updated.ID)
	}
}

func TestMovePreservesUUID(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	svc.Initialize()
	f1, _ := svc.CreateFolder("", "A", noopRefresh)
	f2, _ := svc.CreateFolder("", "B", noopRefresh)
	oldID := f1.ID

	moved, _ := svc.MoveFolder(f1.ID, f2.ID, noopRefresh)
	if moved.ID != oldID {
		t.Fatalf("UUID changed: %q → %q", oldID, moved.ID)
	}
}
