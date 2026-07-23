package workspacetree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyPathFromSyncMovesFolderByStableKey(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	source, _ := svc.CreateFolder("", "Source", noopRefresh)
	target, _ := svc.CreateFolder("", "Target", noopRefresh)

	if err := svc.ApplyPathFromSync(
		"folder:"+source.ID,
		"Source",
		"Target/Source",
		noopRefresh,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(vault, "Target", "Source")); err != nil {
		t.Fatalf("folder was not moved: %v", err)
	}
	if moved, ok := svc.GetFolderByID(source.ID); !ok || moved.ParentID != target.ID {
		t.Fatalf("moved folder = %+v, found=%v", moved, ok)
	}
}

func TestApplyPathFromSyncMovesWorkspaceByStableKey(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	target, _ := svc.CreateFolder("", "Target", noopRefresh)
	deal, _ := svc.CreateWorkspace("", "Deal", "", noopRefresh)

	if err := svc.ApplyPathFromSync(
		"workspace:"+deal.ID,
		"Deal",
		"Target/Deal",
		noopRefresh,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(vault, "Target", "Deal")); err != nil {
		t.Fatalf("workspace was not moved: %v", err)
	}
	if moved, ok := svc.GetWorkspaceByID(deal.ID); !ok || moved.RootPath != "Target/Deal" {
		t.Fatalf("moved workspace = %+v, found=%v", moved, ok)
	}
	if _, ok := svc.GetFolderByID(target.ID); !ok {
		t.Fatal("target folder identity was lost")
	}
}

func TestApplyPathFromSyncRejectsStalePreviousPathAndUnsafeParent(t *testing.T) {
	vault := t.TempDir()
	svc := NewService(vault, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatal(err)
	}
	source, _ := svc.CreateFolder("", "Source", noopRefresh)

	if err := svc.ApplyPathFromSync("folder:"+source.ID, "Elsewhere", "Renamed", noopRefresh); err == nil {
		t.Fatal("expected stale previous path conflict")
	}
	if err := svc.ApplyPathFromSync("folder:"+source.ID, "Source", "Missing/Source", noopRefresh); err == nil {
		t.Fatal("expected missing semantic parent error")
	}
	if _, err := os.Stat(filepath.Join(vault, "Source")); err != nil {
		t.Fatalf("rejected sync path moved source: %v", err)
	}
}
