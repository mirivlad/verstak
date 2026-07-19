package workspacetree

import (
	"os"
	"path/filepath"
	"testing"
)

func requireVaultInit(t *testing.T, vault string) {
	t.Helper()
	verstakDir := filepath.Join(vault, ".verstak")
	if err := os.MkdirAll(verstakDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(verstakDir, "vault.json"), []byte(`{"schemaVersion":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreWorkspaceAppearsInTreeImmediately(t *testing.T) {
	vault := t.TempDir()
	requireVaultInit(t, vault)
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	ws, _ := svc.CreateWorkspace("", "TestWS", "minimal", func() error { return nil })
	entry, _ := svc.TrashWorkspace(ws.ID, func() error { return nil })
	_, err := svc.RestoreTreeTrash(entry.TrashID, "", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	tree := svc.GetTree()
	if len(tree.Roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(tree.Roots))
	}
	if _, ok := svc.GetWorkspaceByID(ws.ID); !ok {
		t.Fatal("GetWorkspaceByID failed")
	}
}

func TestRestoreFolderAppearsInTreeImmediately(t *testing.T) {
	vault := t.TempDir()
	requireVaultInit(t, vault)
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	f, _ := svc.CreateFolder("", "TestFolder", func() error { return nil })
	entry, _ := svc.TrashFolder(f.ID, func() error { return nil })
	_, err := svc.RestoreTreeTrash(entry.TrashID, "", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := svc.GetFolderByID(f.ID); !ok {
		t.Fatal("GetFolderByID failed")
	}
}

func TestRestoreNestedWorkspaceToOriginalParent(t *testing.T) {
	vault := t.TempDir()
	requireVaultInit(t, vault)
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	f, _ := svc.CreateFolder("", "Clients", func() error { return nil })
	ws, _ := svc.CreateWorkspace(f.ID, "ProjectA", "minimal", func() error { return nil })
	t.Logf("Created nested: %s", ws.RootPath)

	entry, _ := svc.TrashWorkspace(ws.ID, func() error { return nil })

	// Verify parent UUID in metadata.
	if _, err := os.Stat(filepath.Join(vault, ".verstak", "trash", "tree", entry.TrashID, "metadata.json")); err != nil {
		t.Fatalf("trash metadata missing: %v", err)
	}

	_, err := svc.RestoreTreeTrash(entry.TrashID, "", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	ws2, ok := svc.GetWorkspaceByID(ws.ID)
	if !ok {
		t.Fatal("GetWorkspaceByID failed")
	}
	if ws2.RootPath != "Clients/ProjectA" {
		t.Fatalf("got path=%q, want Clients/ProjectA", ws2.RootPath)
	}
}

func TestRestoreRenamedParentUsesCurrentPath(t *testing.T) {
	vault := t.TempDir()
	requireVaultInit(t, vault)
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	f, _ := svc.CreateFolder("", "Clients", func() error { return nil })
	ws, _ := svc.CreateWorkspace(f.ID, "ProjectA", "minimal", func() error { return nil })
	entry, _ := svc.TrashWorkspace(ws.ID, func() error { return nil })

	// Rename parent.
	if _, err := svc.RenameFolder(f.ID, "Customers", func() error { return nil }); err != nil {
		t.Fatal(err)
	}

	_, err := svc.RestoreTreeTrash(entry.TrashID, "", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	ws2, _ := svc.GetWorkspaceByID(ws.ID)
	if ws2.RootPath != "Customers/ProjectA" {
		t.Fatalf("got path=%q, want Customers/ProjectA", ws2.RootPath)
	}
}

func TestRestoreRootEntityStaysInRoot(t *testing.T) {
	vault := t.TempDir()
	requireVaultInit(t, vault)
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	ws, _ := svc.CreateWorkspace("", "RootWS", "minimal", func() error { return nil })
	entry, _ := svc.TrashWorkspace(ws.ID, func() error { return nil })
	_, err := svc.RestoreTreeTrash(entry.TrashID, "", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	ws2, _ := svc.GetWorkspaceByID(ws.ID)
	if ws2.RootPath != "RootWS" {
		t.Fatalf("got path=%q, want RootWS", ws2.RootPath)
	}
}

func TestRestoreNestedFolderToOriginalParent(t *testing.T) {
	vault := t.TempDir()
	requireVaultInit(t, vault)
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	rootF, _ := svc.CreateFolder("", "Archive", func() error { return nil })
	childF, _ := svc.CreateFolder(rootF.ID, "Old", func() error { return nil })
	entry, _ := svc.TrashFolder(childF.ID, func() error { return nil })
	_, err := svc.RestoreTreeTrash(entry.TrashID, "", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	f2, ok := svc.GetFolderByID(childF.ID)
	if !ok {
		t.Fatal("GetFolderByID failed")
	}
	if f2.Path != "Archive/Old" {
		t.Fatalf("got path=%q, want Archive/Old", f2.Path)
	}
}
