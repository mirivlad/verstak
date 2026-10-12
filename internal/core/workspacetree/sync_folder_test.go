package workspacetree

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateFolderFromSyncPreservesIdentityAndIsIdempotent(t *testing.T) {
	svc, root := newMetadataV2Service(t)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	parentID := testUUID("sync-folder-parent")
	parent, err := svc.CreateFolderFromSync("Работа", parentID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if parent.ID != parentID || parent.Path != "Работа" {
		t.Fatalf("folder identity/path not preserved: %+v", parent)
	}
	if again, err := svc.CreateFolderFromSync("Работа", parentID, nil); err != nil || again.ID != parentID {
		t.Fatalf("idempotent create: %+v, %v", again, err)
	}
	childID := testUUID("sync-folder-child")
	child, err := svc.CreateFolderFromSync("Работа/Подкаталог", childID, nil)
	if err != nil || child.ID != childID || child.ParentID != parentID {
		t.Fatalf("nested create: %+v, %v", child, err)
	}
	if _, err := os.Stat(filepath.Join(root, "Работа", ".verstak", "folder.json")); err != nil {
		t.Fatalf("folder marker: %v", err)
	}
}

func TestCreateFolderFromSyncRejectsUnrelatedLocalFolder(t *testing.T) {
	svc, root := newMetadataV2Service(t)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	local, err := svc.CreateFolder("", "Работа", nil)
	if err != nil {
		t.Fatal(err)
	}
	otherID := testUUID("unrelated-remote-folder")
	if _, err := svc.CreateFolderFromSync("Работа", otherID, nil); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("should not overwrite distinct local folder: %v", err)
	}
	if _, err := svc.CreateFolderFromSync("WrongPath", local.ID, nil); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("should not move local folder by create: %v", err)
	}
	if current, ok := svc.GetFolderByID(local.ID); !ok || current.Path != "Работа" {
		t.Fatalf("original folder changed: %+v, %v", current, ok)
	}
	if _, err := os.Stat(filepath.Join(root, "Работа", ".verstak", "folder.json")); err != nil {
		t.Fatal(err)
	}
}

func TestCreateFolderFromSyncRejectsUnsafePathAndUnknownParent(t *testing.T) {
	svc, _ := newMetadataV2Service(t)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"../escape", "Missing/Child", "a//b"} {
		if _, err := svc.CreateFolderFromSync(path, testUUID("sync-folder-reject"), nil); err == nil {
			t.Errorf("unsafe or unknown path %q accepted", path)
		}
	}
}
