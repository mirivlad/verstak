package workspacetree

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareImportedWorkspace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "staged-workspace")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	prepared, err := PrepareImportedWorkspace(dir, "Сайт", "default")
	if err != nil {
		t.Fatal(err)
	}
	if prepared.ID == "" {
		t.Fatal("workspace ID is empty")
	}
	if _, err := os.Stat(filepath.Join(dir, ".verstak", "workspace.json")); err != nil {
		t.Fatalf("workspace marker: %v", err)
	}
	for _, name := range []string{"Notes", "Files"} {
		if info, err := os.Stat(filepath.Join(dir, name)); err != nil || !info.IsDir() {
			t.Fatalf("template directory %s: %v", name, err)
		}
	}

	var metadata map[string]any
	if err := json.Unmarshal(prepared.RegistryJSON, &metadata); err != nil {
		t.Fatalf("registry JSON: %v", err)
	}
	if metadata["workspaceId"] != prepared.ID || metadata["workspaceName"] != "Сайт" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestPrepareImportedWorkspaceRejectsUnsupportedTemplate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "staged-workspace")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := PrepareImportedWorkspace(dir, "Сайт", "admin"); err == nil {
		t.Fatal("expected unsupported template error")
	}
}
