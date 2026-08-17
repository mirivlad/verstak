package workspacetree

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestReplaceFolderAppearanceUsesExactValues(t *testing.T) {
	vaultDir := t.TempDir()
	folderID := uuid.NewString()
	svc := NewService(vaultDir, nil)

	if err := svc.ReplaceFolderAppearance(folderID, &FolderAppearance{Icon: "star", Color: "#112233"}); err != nil {
		t.Fatalf("initial replace: %v", err)
	}
	if err := svc.ReplaceFolderAppearance(folderID, &FolderAppearance{Color: "#445566"}); err != nil {
		t.Fatalf("replace clearing icon: %v", err)
	}
	got, err := svc.GetFolderAppearance(folderID)
	if err != nil {
		t.Fatalf("read replacement: %v", err)
	}
	if got.Icon != "" || got.Color != "#445566" {
		t.Fatalf("replacement should be exact, got %#v", got)
	}

	if err := svc.ReplaceFolderAppearance(folderID, &FolderAppearance{}); err != nil {
		t.Fatalf("clear appearance: %v", err)
	}
	if _, err := os.Stat(folderAppearancePath(vaultDir, folderID)); !os.IsNotExist(err) {
		t.Fatalf("empty replacement should remove metadata file, stat err=%v", err)
	}
}

func TestInitializeMigratesLegacyFolderAppearanceWithoutOverwritingCore(t *testing.T) {
	vaultDir := t.TempDir()
	creator := NewService(vaultDir, nil)
	if err := creator.Initialize(); err != nil {
		t.Fatalf("initialize creator: %v", err)
	}
	folder, err := creator.CreateFolder("", "Legacy Folder", nil)
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}

	legacyPath := legacyFolderAppearancePath(vaultDir)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := legacyFolderAppearanceFile{Folders: map[string]legacyFolderAppearance{
		folder.ID: {IconID: "archive", ColorID: "#123456"},
	}}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	migrated := NewService(vaultDir, nil)
	if err := migrated.Initialize(); err != nil {
		t.Fatalf("initialize migration: %v", err)
	}
	got, err := migrated.GetFolderAppearance(folder.ID)
	if err != nil {
		t.Fatalf("read migrated appearance: %v", err)
	}
	if got.Icon != "archive" || got.Color != "#123456" {
		t.Fatalf("unexpected migrated appearance: %#v", got)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy source should be retained for recoverability: %v", err)
	}

	if err := migrated.ReplaceFolderAppearance(folder.ID, &FolderAppearance{Icon: "star", Color: "#abcdef"}); err != nil {
		t.Fatalf("write core appearance: %v", err)
	}
	legacy.Folders[folder.ID] = legacyFolderAppearance{IconID: "folder", ColorID: "#654321"}
	data, _ = json.Marshal(legacy)
	if err := os.WriteFile(legacyPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	again := NewService(vaultDir, nil)
	if err := again.Initialize(); err != nil {
		t.Fatalf("repeat initialize: %v", err)
	}
	got, err = again.GetFolderAppearance(folder.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Icon != "star" || got.Color != "#abcdef" {
		t.Fatalf("existing core appearance must win, got %#v", got)
	}
}

func TestLegacyFolderAppearanceIsBestEffort(t *testing.T) {
	vaultDir := t.TempDir()
	legacyPath := legacyFolderAppearancePath(vaultDir)
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := NewService(vaultDir, nil)
	if err := svc.Initialize(); err != nil {
		t.Fatalf("cosmetic legacy corruption must not block vault startup: %v", err)
	}
}
