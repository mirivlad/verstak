package workspacetree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotWriteReadRoundtrip(t *testing.T) {
	vault := t.TempDir()
	f1 := testUUID("snap-f")
	ws1 := testUUID("snap-ws")
	original := SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Folders:       map[string]SnapshotEntry{f1: {Path: "Clients"}},
		Workspaces:    map[string]SnapshotEntry{ws1: {Path: "Project"}},
	}
	if err := WriteSnapshot(vault, &original); err != nil {
		t.Fatal(err)
	}
	read, err := ReadSnapshot(vault)
	if err != nil {
		t.Fatal(err)
	}
	if read == nil {
		t.Fatal("snapshot not found")
	}
	if read.Folders[f1].Path != "Clients" {
		t.Fatalf("folder path = %q", read.Folders[f1].Path)
	}
}

func TestSnapshotMissingFileReturnsNil(t *testing.T) {
	vault := t.TempDir()
	snap, err := ReadSnapshot(vault)
	if err != nil {
		t.Fatal(err)
	}
	if snap != nil {
		t.Fatal("expected nil snapshot for missing file")
	}
}

func TestSnapshotAtomicWrite(t *testing.T) {
	vault := t.TempDir()
	snap := SemanticSnapshot{
		SchemaVersion: snapshotSchemaVersion,
		Folders:       map[string]SnapshotEntry{testUUID("atom-f"): {Path: "A"}},
	}
	if err := WriteSnapshot(vault, &snap); err != nil {
		t.Fatal(err)
	}
	tmpPath := filepath.Join(vault, ".verstak", "cache", "workspace-tree.json.tmp")
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatal("tmp file should not exist after atomic write")
	}
}

func TestRemoveSnapshot(t *testing.T) {
	vault := t.TempDir()
	snap := SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Workspaces: map[string]SnapshotEntry{testUUID("rm-ws"): {Path: "A"}}}
	if err := WriteSnapshot(vault, &snap); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSnapshot(vault); err != nil {
		t.Fatal(err)
	}
	read, _ := ReadSnapshot(vault)
	if read != nil {
		t.Fatal("snapshot should be gone")
	}
}

func TestSnapshotCorruptedJSONReturnsNil(t *testing.T) {
	vault := t.TempDir()
	path := filepath.Join(vault, ".verstak", "cache", "workspace-tree.json")
	mustMkdirAll(t, filepath.Dir(path))
	mustWriteFile(t, path, "not json")
	snap, err := ReadSnapshot(vault)
	if err != nil {
		t.Fatal(err)
	}
	if snap != nil {
		t.Fatal("corrupted snapshot should return nil")
	}
}

func TestSnapshotFailedTempWriteDoesNotDestroyPrevious(t *testing.T) {
	vault := t.TempDir()
	snap := SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Workspaces: map[string]SnapshotEntry{testUUID("keep"): {Path: "A"}}}
	if err := WriteSnapshot(vault, &snap); err != nil {
		t.Fatal(err)
	}
	// Read it back to confirm.
	first, _ := ReadSnapshot(vault)
	if first == nil || len(first.Workspaces) != 1 {
		t.Fatal("initial snapshot corrupted")
	}
	// Now write a new snapshot successfully.
	snap2 := SemanticSnapshot{SchemaVersion: snapshotSchemaVersion,
		Workspaces: map[string]SnapshotEntry{testUUID("keep"): {Path: "B"}}}
	if err := WriteSnapshot(vault, &snap2); err != nil {
		t.Fatal(err)
	}
	second, _ := ReadSnapshot(vault)
	if second.Workspaces[testUUID("keep")].Path != "B" {
		t.Fatal("second snapshot path not updated")
	}
}

func TestSnapshotUnsupportedSchemaReturnsNil(t *testing.T) {
	vault := t.TempDir()
	path := filepath.Join(vault, ".verstak", "cache", "workspace-tree.json")
	mustMkdirAll(t, filepath.Dir(path))
	mustWriteFile(t, path, `{"schemaVersion":99,"workspaces":{},"folders":{}}`)
	snap, err := ReadSnapshot(vault)
	if err != nil {
		t.Fatal(err)
	}
	if snap != nil {
		t.Fatal("unsupported schema should return nil")
	}
}

func TestSnapshotDeletedCacheCanBeRebuilt(t *testing.T) {
	vault := t.TempDir()
	createWS(t, vault, "Project", testUUID("rebuild"))
	// First scan and write.
	scan, _ := Scan(vault, nil)
	snap := NewSnapshotFromScan(scan)
	WriteSnapshot(vault, &snap)
	// Delete.
	RemoveSnapshot(vault)
	// Rebuild.
	scan2, _ := Scan(vault, nil)
	snap2 := NewSnapshotFromScan(scan2)
	if err := WriteSnapshot(vault, &snap2); err != nil {
		t.Fatal(err)
	}
	read, _ := ReadSnapshot(vault)
	if read == nil {
		t.Fatal("snapshot should be rebuilt")
	}
}
